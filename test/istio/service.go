// +build e2e

package e2e

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func revisionServiceName(name string) string {
	return name + "-rev"
}

func routeServiceName(name string) string {
	return name
}

func makeRouteService(name string) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      routeServiceName(name),
			Namespace: TestNamespace,
		},
		Spec: corev1.ServiceSpec{
			Type:         corev1.ServiceTypeExternalName,
			ExternalName: "istio-ingressgateway.istio-system.svc.cluster.local",
		},
	}
}

func makeRevisionService(name string) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      revisionServiceName(name),
			Namespace: TestNamespace,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{{
				Name:       "http",
				Port:       80,
				TargetPort: intstr.FromString("queue-port"),
			}, {
				Name:       "metrics",
				Port:       9090,
				TargetPort: intstr.FromString("queue-metrics"),
			}},
			Selector: makeLabels(name),
		},
	}
}
