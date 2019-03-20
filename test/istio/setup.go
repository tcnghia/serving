// +build e2e

package e2e

import (
	"testing"

	istioversioned "github.com/knative/pkg/client/clientset/versioned"
	istiotyped "github.com/knative/pkg/client/clientset/versioned/typed/istio/v1alpha3"
	"github.com/knative/pkg/test"
	pkgTest "github.com/knative/pkg/test"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const TestNamespace = "serving-tests"

type IstioClients struct {
	VirtualServices istiotyped.VirtualServiceInterface
}

// Clients holds instances of interfaces for making requests to Knative Serving.
type Clients struct {
	KubeClient  *pkgTest.KubeClient
	IstioClient *IstioClients
	Dynamic     dynamic.Interface
}

func Setup(t *testing.T) *Clients {
	clients, err := NewClients(pkgTest.Flags.Kubeconfig, pkgTest.Flags.Cluster, TestNamespace)
	if err != nil {
		t.Fatalf("Couldn't initialize clients: %v", err)
	}
	return clients
}

func NewClients(configPath string, clusterName string, namespace string) (*Clients, error) {
	clients := &Clients{}
	cfg, err := buildClientConfig(configPath, clusterName)
	if err != nil {
		return nil, err
	}
	clients.KubeClient, err = test.NewKubeClient(configPath, clusterName)
	if err != nil {
		return nil, err
	}
	clients.IstioClient, err = newIstioClients(cfg, VirtualServiceNamespace)
	if err != nil {
		return nil, err
	}
	clients.Dynamic, err = dynamic.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	return clients, nil

}

// NewIstioClients instantiates and returns the serving clientset required to make requests to the
// knative serving cluster.
func newIstioClients(cfg *rest.Config, namespace string) (*IstioClients, error) {
	cs, err := istioversioned.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	return &IstioClients{
		VirtualServices: cs.NetworkingV1alpha3().VirtualServices(namespace),
	}, nil
}

func buildClientConfig(kubeConfigPath string, clusterName string) (*rest.Config, error) {
	overrides := clientcmd.ConfigOverrides{}
	// Override the cluster name if provided.
	if clusterName != "" {
		overrides.Context.Cluster = clusterName
	}
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeConfigPath},
		&overrides).ClientConfig()
}
