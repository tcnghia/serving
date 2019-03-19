// +build e2e

package e2e

import (
	"os"
	"os/signal"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CleanupOnInterrupt will execute the function cleanup if an interrupt signal is caught
func CleanupOnInterrupt(cleanup func()) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for range c {
			cleanup()
			os.Exit(1)
		}
	}()
}

// TearDown delete the resources.
func TearDown(t *testing.T, clients *Clients, name string) {
	clients.KubeClient.Kube.Apps().Deployments(TestNamespace).Delete(name, &metav1.DeleteOptions{})
	clients.KubeClient.Kube.CoreV1().Services(TestNamespace).Delete(name, &metav1.DeleteOptions{})
	clients.IstioClient.VirtualServices.Delete(name, &metav1.DeleteOptions{})
}
