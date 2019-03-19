// +build e2e

package e2e

import (
	"github.com/knative/pkg/apis/istio/common/v1alpha1"
	"github.com/knative/pkg/apis/istio/v1alpha3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func makeVirtualService(name string, domain string) *v1alpha3.VirtualService {
	return &v1alpha3.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: TestNamespace,
		},
		Spec: v1alpha3.VirtualServiceSpec{
			Hosts: []string{
				name + "." + TestNamespace + "." + domain,
			},
			Gateways: []string{
				"knative-ingress-gateway.knative-serving.svc.cluster.local",
				"mesh",
			},
			Http: []v1alpha3.HTTPRoute{{
				Match: []v1alpha3.HTTPMatchRequest{{
					Authority: &v1alpha1.StringMatch{
						Exact: name + "." + TestNamespace + "." + domain,
					},
				}},
				Route: []v1alpha3.DestinationWeight{{
					Destination: v1alpha3.Destination{
						Host: name + "." + TestNamespace + ".svc.cluster.local",
						Port: v1alpha3.PortSelector{
							Number: 80,
						},
					},
					Weight: 100,
				}},
			}},
		},
	}
}
