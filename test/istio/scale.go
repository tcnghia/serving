// +build e2e

package e2e

import (
	"fmt"
	"io/ioutil"
	"net/http"
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

func CreateSvcDeployVirtualService(t *testing.T, clients *Clients, name string) error {
	t.Logf("Creating Service, Deployment, VirtualService %s\n", name)
	_, err := clients.KubeClient.Kube.Apps().Deployments(TestNamespace).Create(makeDeployment(name))
	if err != nil {
		t.Errorf("Error creating Deployment %v", err)
	}
	_, err = clients.KubeClient.Kube.CoreV1().Services(TestNamespace).Create(makeRevisionService(name))
	if err != nil {
		t.Errorf("Error creating Revision Service %v", err)
	}
	_, err = clients.KubeClient.Kube.CoreV1().Services(TestNamespace).Create(makeRouteService(name))
	if err != nil {
		t.Errorf("Error creating Route Service %v", err)
	}
	xipDomain := getXipDomain(clients)
	_, err = clients.IstioClient.VirtualServices.Create(makeVirtualService(name, xipDomain))
	if err != nil {
		t.Errorf("Error creating VirtualService %v", err)
	}
	return nil
}

func Probe20Times(t *testing.T, domain string) int {
	tries := 20
	count := 0
	for i := 0; i < tries; i++ {
		resp, err := http.Get(fmt.Sprintf("http://%s", domain))
		if err == nil && resp != nil && resp.StatusCode == http.StatusOK {
			count = count + 1
		}
		time.Sleep(1 * time.Second)
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
			name := test.SubServiceNameForTest(t, fmt.Sprintf("%d", i))

			// Send it to our cleanup logic (below)
			cleanupCh <- name
			_ = CreateSvcDeployVirtualService(t, clients, name)
			// if err := WaitForPod(t, name, timeout); err != nil {

			// }
			start := time.Now()
			domain := fmt.Sprintf("%s.%s.%s", name, TestNamespace, xipDomain)
			msg, err := WaitFor200(t, domain, timeout)
			count := Probe20Times(t, domain)
			t.Logf("[latency]\t[%s]\t%v\t[200:%d]\t%v", name, time.Since(start), count, msg)
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
			t.Logf("Added %v to cleanup routine.\n", name)
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
