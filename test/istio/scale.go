// +build e2e

package e2e

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/knative/serving/pkg/pool"
	"github.com/knative/serving/test"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	IstioNamespace      = "istio-system"
	IstioIngressgateway = "istio-ingressgateway"
	WaitInterval        = 2 * time.Second
)

func getXipDomain(clients *Clients) string {
	svc, _ := clients.KubeClient.Kube.CoreV1().Services(IstioNamespace).Get(IstioIngressgateway, metav1.GetOptions{})
	return svc.Status.LoadBalancer.Ingress[0].IP + ".xip.io"
}

func WaitForEndpoints(clients *Clients, endpoints string, timeout time.Duration) error {
	return wait.PollImmediate(WaitInterval, timeout, func() (bool, error) {
		ep, err := clients.KubeClient.Kube.CoreV1().Endpoints(TestNamespace).Get(endpoints, metav1.GetOptions{})
		if err != nil || len(ep.Subsets) == 0 || len(ep.Subsets[0].Addresses) == 0 {
			return false, nil
		}
		return true, nil
	})
}

func CreateSvcDeployVirtualService(t *testing.T, clients *Clients, name string) error {
	// t.Logf("Creating Service, Deployment, VirtualService %s\n", name)
	if _, err := clients.KubeClient.Kube.Apps().Deployments(TestNamespace).Create(makeDeployment(name)); err != nil {
		t.Fatalf("Error creating Deployment %v", err)
	}
	if _, err := clients.KubeClient.Kube.CoreV1().Services(TestNamespace).Create(makeRevisionService(name)); err != nil {
		t.Fatalf("Error creating Revision Service %v", err)
	}
	// Wait for the Endpoint to have available replicas
	if err := WaitForEndpoints(clients, revisionServiceName(name), 2*time.Minute); err != nil {
		t.Errorf("Error waiting for Revision Service to be ready %v", err)
	}
	if _, err := clients.KubeClient.Kube.CoreV1().Services(TestNamespace).Create(makeRouteService(name)); err != nil {
		t.Fatalf("Error creating Route Service %v", err)
	}
	xipDomain := getXipDomain(clients)
	if _, err := clients.IstioClient.VirtualServices.Create(makeVirtualService(name, xipDomain)); err != nil {
		t.Fatalf("Error creating VirtualService %v", err)
	}
	return nil
}

// http.Get, ignoring DNS.
func httpGet(domain string) (*http.Response, error) {
	i := 0
	for {
		resp, err := http.Get(fmt.Sprintf("http://%s", domain))
		if err == nil {
			return resp, err
		}
		msg := err.Error()
		// Retrying for DNS.
		if strings.Contains(msg, "no such host") || strings.Contains(msg, ":53") {
			i = i + 1
			if i <= 10 {
				time.Sleep(1 * time.Second)
				continue
			}
		}
		return resp, err
	}
}

func ProbeNTimes(t *testing.T, domain string) map[int]int {
	tries := 40
	count := make(map[int]int)
	for i := 0; i < tries; i++ {
		resp, err := httpGet(domain)
		if err == nil && resp != nil {
			count[resp.StatusCode] = count[resp.StatusCode] + 1
		}
		if resp == nil {
			t.Logf("Unexpected error: %v", err)
		}
		// Every half seconds.
		time.Sleep(500 * time.Millisecond)
	}
	return count
}

func WaitFor200(t *testing.T, domain string, timeout time.Duration) (string, error) {
	var msg = ""
	err := wait.PollImmediate(WaitInterval, timeout, func() (bool, error) {
		resp, err := http.Get(fmt.Sprintf("http://%s", domain))
		if resp == nil {
			return false, nil
		}
		defer resp.Body.Close()
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		msg = string(bodyBytes)
		if err != nil || resp.StatusCode != http.StatusOK {
			return false, nil
		}
		return true, nil
	})
	return msg, err
}

func IstioScaleToWithin(t *testing.T, scale int, timeout time.Duration) {
	clients := Setup(t)

	cleanupCh := make(chan string, scale)
	defer close(cleanupCh)

	wg := pool.NewWithCapacity(50 /* maximum in-flight creates */, scale /* capacity */)
	xipDomain := getXipDomain(clients)
	for i := 0; i < scale; i++ {
		// https://golang.org/doc/faq#closures_and_goroutines
		i := i

		wg.Go(func() error {
			name := test.SubServiceNameForTest(t, fmt.Sprintf("%03d", i))

			// Send it to our cleanup logic (below)
			cleanupCh <- name
			start := time.Now()
			_ = CreateSvcDeployVirtualService(t, clients, name)
			created := time.Now()
			createSecs := int(created.Sub(start).Seconds())
			domain := fmt.Sprintf("%s.%s.%s", name, TestNamespace, xipDomain)
			msg, err := WaitFor200(t, domain, timeout)
			readied := time.Now()
			readySecs := int(readied.Sub(created).Seconds())
			counts := ProbeNTimes(t, domain)
			done := time.Now()
			doneSecs := int(done.Sub(readied).Seconds())
			t.Logf("[%s]\t%d\t%d\t%d\t[%v]\t%v", name, createSecs, readySecs, doneSecs, counts, msg)
			return err
		})
	}
	// Wait for all of the service creations to complete (possibly in failure),
	// and signal the done channel.
	doneCh := make(chan error)
	go func() {
		defer close(doneCh)
		if err := wg.Wait(); err != nil {
			doneCh <- err
		}
	}()
	for {
		select {
		case name := <-cleanupCh:
			// t.Logf("Added %v to cleanup routine.\n", name)
			CleanupOnInterrupt(func() { TearDown(t, clients, name) })
			defer TearDown(t, clients, name)
		case err := <-doneCh:
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			return
		}
	}
}
