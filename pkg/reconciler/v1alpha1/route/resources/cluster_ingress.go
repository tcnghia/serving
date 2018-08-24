/*
Copyright 2018 The Knative Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package resources

import (
	"sort"
	"time"

	"github.com/knative/pkg/kmeta"
	"github.com/knative/serving/pkg/activator"
	"github.com/knative/serving/pkg/apis/networking/v1alpha1"
	servingv1alpha1 "github.com/knative/serving/pkg/apis/serving/v1alpha1"
	"github.com/knative/serving/pkg/reconciler"
	revisionresources "github.com/knative/serving/pkg/reconciler/v1alpha1/revision/resources"
	"github.com/knative/serving/pkg/reconciler/v1alpha1/route/resources/names"
	"github.com/knative/serving/pkg/reconciler/v1alpha1/route/traffic"
	"github.com/knative/serving/pkg/system"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// MakeClusterIngress creates an Istio ClusterIngress to set up routing rules.  Such ClusterIngress specifies
// which Gateways and Hosts that it applies to, as well as the routing rules.
func MakeClusterIngress(u *servingv1alpha1.Route, tc *traffic.TrafficConfig) *v1alpha1.ClusterIngress {
	return &v1alpha1.ClusterIngress{
		ObjectMeta: metav1.ObjectMeta{
			Name:            names.ClusterIngress(u),
			Namespace:       u.Namespace,
			Labels:          map[string]string{"route": u.Name},
			OwnerReferences: []metav1.OwnerReference{*kmeta.NewControllerRef(u)},
		},
		Spec: makeClusterIngressSpec(u, tc.Targets),
	}
}

func makeClusterIngressSpec(u *servingv1alpha1.Route, targets map[string][]traffic.RevisionTarget) v1alpha1.ClusterIngressSpec {
	domain := u.Status.Domain
	names := []string{}
	for name := range targets {
		names = append(names, name)
	}
	// Sort the names to give things a deterministic ordering.
	sort.Strings(names)
	// The routes are matching rule based on domain name to traffic split targets.
	rules := []v1alpha1.ClusterIngressRule{}
	for _, name := range names {
		rules = append(rules, *makeClusterIngressRule(getRouteDomains(name, u, domain), u.Namespace, targets[name]))
	}
	return v1alpha1.ClusterIngressSpec{
		Rules: rules,
	}
}

func makeClusterIngressRule(domains []string, ns string, targets []traffic.RevisionTarget) *v1alpha1.ClusterIngressRule {
	active, inactive := groupInactiveTargets(targets)
	splits := []v1alpha1.ClusterIngressBackendSplit{}
	for _, t := range active {
		if t.Percent == 0 {
			// Don't include 0% routes.
			continue
		}
		splits = append(splits, v1alpha1.ClusterIngressBackendSplit{
			Backend: &v1alpha1.ClusterIngressBackend{
				ServiceNamespace: ns,
				ServiceName:      reconciler.GetServingK8SServiceNameForObj(t.TrafficTarget.RevisionName),
				ServicePort:      intstr.FromInt(int(revisionresources.ServicePort)),
			},
			Weight: float32(t.Percent) / 100,
		})
	}
	defaultTimeout := time.Second * 60
	path := v1alpha1.HTTPClusterIngressPath{
		Backends: splits,
		Timeout:  defaultTimeout,
		Retries: &v1alpha1.HTTPRetry{
			Attempts:      3,
			PerTryTimeout: defaultTimeout,
		},
	}
	return &v1alpha1.ClusterIngressRule{
		Hosts: domains,
		ClusterIngressRuleValue: v1alpha1.ClusterIngressRuleValue{
			HTTP: &v1alpha1.HTTPClusterIngressRuleValue{
				Paths: []v1alpha1.HTTPClusterIngressPath{
					*addInactive(&path, ns, inactive),
				},
			},
		},
	}
}

func addInactive(r *v1alpha1.HTTPClusterIngressPath, ns string, inactive []traffic.RevisionTarget) *v1alpha1.HTTPClusterIngressPath {
	if len(inactive) == 0 {
		// No need to change
		return r
	}
	totalInactivePercent := 0
	maxInactiveTarget := traffic.RevisionTarget{}

	for _, t := range inactive {
		totalInactivePercent += t.Percent
		if t.Percent >= maxInactiveTarget.Percent {
			maxInactiveTarget = t
		}
	}
	if totalInactivePercent == 0 {
		// However, we shouldn't need to include 0% route anyway.
		return r
	}
	r.Backends = append(r.Backends, v1alpha1.ClusterIngressBackendSplit{
		Backend: &v1alpha1.ClusterIngressBackend{
			ServiceNamespace: system.Namespace,
			ServiceName:      activator.K8sServiceName,
			ServicePort:      intstr.FromInt(int(revisionresources.ServicePort)),
		},
		Weight: float32(totalInactivePercent) / 100,
	})
	r.AppendHeaders = map[string]string{
		activator.RevisionHeaderName:      maxInactiveTarget.RevisionName,
		activator.ConfigurationHeader:     maxInactiveTarget.ConfigurationName,
		activator.RevisionHeaderNamespace: ns,
	}
	return r
}
