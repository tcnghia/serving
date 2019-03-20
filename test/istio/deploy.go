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

// makeQueueContainer creates the container spec for queue sidecar.
func makeQueueContainer(name string) corev1.Container {
	return corev1.Container{
		Name:  "queue-proxy",
		Image: "gcr.io/nghia-elafros-2/queue-7204c16e44715cd30f78443fb99e0f58@sha256:e4caf20d48b59e166f9330fcefed61fd9b9ec93df171f4f03feebd7dfc11d8cb",
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU: resource.MustParse("25m"),
			},
		},
		Ports: []corev1.ContainerPort{{
			ContainerPort: 8012,
			Name:          "queue-port",
			Protocol:      "TCP",
		}, {
			ContainerPort: 8022,
			Name:          "queueadm-port",
			Protocol:      "TCP",
		}, {
			ContainerPort: 9090,
			Name:          "queue-metrics",
			Protocol:      "TCP",
		}},
		ReadinessProbe: &corev1.Probe{
			Handler: corev1.Handler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: "/health",
					Port: intstr.FromInt(8022),
				},
			},
			InitialDelaySeconds: 1,
			PeriodSeconds:       1,
		},
		Env: []corev1.EnvVar{{
			Name:  "SERVING_NAMESPACE",
			Value: TestNamespace,
		}, {
			Name:  "SERVING_CONFIGURATION",
			Value: name,
		}, {
			Name:  "SERVING_REVISION",
			Value: name,
		}, {
			Name:  "SERVING_AUTOSCALER",
			Value: "autoscaler",
		}, {
			Name:  "SERVING_AUTOSCALER_PORT",
			Value: "8080",
		}, {
			Name:  "CONTAINER_CONCURRENCY",
			Value: "0",
		}, {
			Name:  "REVISION_TIMEOUT_SECONDS",
			Value: "300",
		}, {
			Name: "SERVING_POD",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.name",
				},
			},
		}, {
			Name: "SERVING_LOGGING_CONFIG",
			Value: `{
              "level": "info",
              "development": false,
              "outputPaths": ["stdout"],
              "errorOutputPaths": ["stderr"],
              "encoding": "json",
              "encoderConfig": {
                "timeKey": "ts",
                "levelKey": "level",
                "nameKey": "logger",
                "callerKey": "caller",
                "messageKey": "msg",
                "stacktraceKey": "stacktrace",
                "lineEnding": "",
                "levelEncoder": "",
                "timeEncoder": "iso8601",
                "durationEncoder": "",
                "callerEncoder": ""
              }
            }
`,
		}, {
			Name:  "SERVING_LOGGING_LEVEL",
			Value: "info",
		}, {
			Name:  "USER_PORT",
			Value: "8080",
		}, {
			Name:  "SYSTEM_NAMESPACE",
			Value: "knative-serving",
		}},
	}
}

func makeUserContainer(name string) corev1.Container {
	return corev1.Container{
		Name:  "user-container",
		Image: "tcnghia/helloworld-go:latest",
		Env: []corev1.EnvVar{{
			Name:  "TARGET",
			Value: name,
		}},
		Ports: []corev1.ContainerPort{{
			ContainerPort: 8080,
			Name:          "user-port",
			Protocol:      "TCP",
		}},
		ReadinessProbe: &corev1.Probe{
			Handler: corev1.Handler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: "/",
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
	}
}

func makeContainers(name string) []corev1.Container {
	return []corev1.Container{makeQueueContainer(name), makeUserContainer(name)}
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
					Containers: makeContainers(name),
				},
			},
		},
	}
}
