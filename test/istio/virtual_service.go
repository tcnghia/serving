// +build e2e

package e2e

import (
	"fmt"
	"regexp"

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
				Match: makeMatch(name + "." + TestNamespace + "." + domain),
				Route: makeRoute(name),
			}, {
				Match: makeMatch(name + "." + TestNamespace + ".svc.cluster.local"),
				Route: makeRoute(name),
			}, {
				Match: makeMatch(name + "." + TestNamespace + ".svc"),
				Route: makeRoute(name),
			}, {
				Match: makeMatch(name + "." + TestNamespace),
				Route: makeRoute(name),
			}},
		},
	}
}

func makeMatch(host string) []v1alpha3.HTTPMatchRequest {
	return []v1alpha3.HTTPMatchRequest{{
		Authority: &v1alpha1.StringMatch{
			Regex: hostRegExp(host),
		},
	}}
}

// hostRegExp returns an ECMAScript regular expression to match either host or host:<any port>
func hostRegExp(host string) string {
	// Should only match 1..65535, but for simplicity it matches 0-99999
	portMatch := `(?::\d{1,5})?`

	return fmt.Sprintf("^%s%s$", regexp.QuoteMeta(host), portMatch)
}

func makeRoute(name string) []v1alpha3.DestinationWeight {
	return []v1alpha3.DestinationWeight{{
		Destination: v1alpha3.Destination{
			Host: revisionServiceName(name) + "." + TestNamespace + ".svc.cluster.local",
			Port: v1alpha3.PortSelector{
				Number: 80,
			},
		},
		Weight: 100,
	}}
}
