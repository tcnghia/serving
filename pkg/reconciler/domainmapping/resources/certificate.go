/*
Copyright 2020 The Knative Authors

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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/networking/pkg/apis/networking"
	networkingv1alpha1 "knative.dev/networking/pkg/apis/networking/v1alpha1"
	"knative.dev/pkg/kmeta"
	"knative.dev/serving/pkg/apis/serving"
	"knative.dev/serving/pkg/apis/serving/v1alpha1"
)

// Certificate returns the name for the Certificate
// child resource for the given Route.
func Certificate(dm kmeta.Accessor) string {
	return "dm-" + string(dm.GetUID())
}

// MakeCertificate creates an array of Certificate for the Route to request TLS certificates.
// domainTagMap is an one-to-one mapping between domain and tag, for major domain (tag-less),
// the value is an empty string
// Returns one certificate for each domain
func MakeCertificate(dm *v1alpha1.DomainMapping, certClass string) *networkingv1alpha1.Certificate {
	dnsName := dm.Status.URL.Host

	// k8s supports cert name only up to 63 chars and so is constructed as route-[UID]-[tag digest]
	// where route-[UID] will take 42 characters and leaves 20 characters for tag digest (need to include `-`).
	// We use https://golang.org/pkg/hash/adler32/#Checksum to compute the digest which returns a uint32.
	// We represent the digest in unsigned integer format with maximum value of 4,294,967,295 which are 10 digits.
	// The "-[tag digest]" is computed only if there's a tag
	certName := Certificate(dm)

	return &networkingv1alpha1.Certificate{
		ObjectMeta: metav1.ObjectMeta{
			Name: certName,
			// TODO(tcnghia): DM has no namespace?
			Namespace:       "default",
			OwnerReferences: []metav1.OwnerReference{*kmeta.NewControllerRef(dm)},
			Annotations: kmeta.FilterMap(kmeta.UnionMaps(map[string]string{
				networking.CertificateClassAnnotationKey: certClass,
			}, dm.Annotations), func(key string) bool {
				return key == corev1.LastAppliedConfigAnnotation
			}),
			Labels: map[string]string{
				serving.RouteLabelKey: dm.Name,
			},
		},
		Spec: networkingv1alpha1.CertificateSpec{
			DNSNames:   []string{dnsName},
			SecretName: certName,
		},
	}
}
