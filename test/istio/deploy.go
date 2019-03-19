// +build e2e

package e2e

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func makeSelector(name string) *metav1.LabelSelector {
	return &metav1.LabelSelector{
		MatchLabels: map[string]string{
			"app": name,
		},
	}
}

func makeLabels(name string) map[string]string {
	return map[string]string{
		"app": name,
	}
}

func makeContainers() []corev1.Container {
	return []corev1.Container{{
		Name:  "user-container",
		Image: "tcnghia/helloworld-go:latest",

		ReadinessProbe: &corev1.Probe{
			Handler: corev1.Handler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: "?prime=10",
					Port: intstr.FromInt(8080),
				},
			},
			InitialDelaySeconds: 1,
			PeriodSeconds:       1,
		},
		Resources: corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("10m"),
				corev1.ResourceMemory: resource.MustParse("50Mi"),
			},
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("10m"),
				corev1.ResourceMemory: resource.MustParse("20Mi"),
			},
		},
	}}
}

func makeDeployment(name string) *appsv1.Deployment {
	one := int32(1)
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: TestNamespace,
			Name:      name,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &one,
			Selector: makeSelector(name),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: makeLabels(name),
					Annotations: map[string]string{
						"sidecar.istio.io/inject": "true",
					},
				},
				Spec: corev1.PodSpec{
					Containers: makeContainers(),
				},
			},
		},
	}
}
