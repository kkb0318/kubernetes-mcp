package tools

import (
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

type Client interface {
	DynamicClient() (dynamic.Interface, error)
	DiscoClient() (discovery.DiscoveryInterface, error)
	RESTMapper() (meta.RESTMapper, error)
	Clientset() (*kubernetes.Clientset, error)
	ResourceInterface(gvr schema.GroupVersionResource, namespaced bool, ns string) (dynamic.ResourceInterface, error)
}

// MultiClusterClientInterface for managing multiple cluster connections.
type MultiClusterClientInterface interface {
	GetClient(context string) (Client, error)
	GetDefaultContext() string
	ListContexts() ([]string, error)
}
