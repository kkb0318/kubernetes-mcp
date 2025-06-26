package client

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/kkb0318/kubernetes-mcp/src/tools"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// MultiClusterClient manages connections to multiple Kubernetes clusters using contexts.
type MultiClusterClient struct {
	clients        map[string]*KubernetesClient
	defaultContext string
	kubeconfig     string
	mu             sync.RWMutex
}

// NewMultiClusterClient creates a new MultiClusterClient that can manage multiple cluster connections.
func NewMultiClusterClient() (*MultiClusterClient, error) {
	var kubeconfig string
	if kubeconfigEnv := os.Getenv("KUBECONFIG"); kubeconfigEnv != "" {
		kubeconfig = kubeconfigEnv
	} else {
		// Try multiple ways to find home directory
		home := os.Getenv("HOME")
		if home == "" {
			if userHome, err := os.UserHomeDir(); err == nil {
				home = userHome
			}
		}
		kubeconfig = filepath.Join(home, ".kube", "config")
	}

	// Get the default context from kubeconfig
	defaultContext, err := getDefaultContext(kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to get default context: %w", err)
	}

	return &MultiClusterClient{
		clients:        make(map[string]*KubernetesClient),
		defaultContext: defaultContext,
		kubeconfig:     kubeconfig,
	}, nil
}

// GetClient returns a Kubernetes client for the specified context.
// If context is empty, it uses the default context.
// Clients are cached to avoid recreating connections.
func (m *MultiClusterClient) GetClient(context string) (tools.Client, error) {
	if context == "" {
		context = m.defaultContext
	}

	m.mu.RLock()
	if client, exists := m.clients[context]; exists {
		m.mu.RUnlock()
		return &ClientWrapper{client: client, context: context}, nil
	}
	m.mu.RUnlock()

	// Create new client for this context
	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check in case another goroutine created it while we were waiting for the lock
	if client, exists := m.clients[context]; exists {
		return &ClientWrapper{client: client, context: context}, nil
	}

	client, err := m.createClientForContext(context)
	if err != nil {
		return nil, fmt.Errorf("failed to create client for context '%s': %w", context, err)
	}

	m.clients[context] = client
	return &ClientWrapper{client: client, context: context}, nil
}

// createClientForContext creates a new KubernetesClient for the specified context.
func (m *MultiClusterClient) createClientForContext(context string) (*KubernetesClient, error) {
	// First try in-cluster config (if running inside a pod)
	if config, err := rest.InClusterConfig(); err == nil {
		return &KubernetesClient{config: config}, nil
	}

	// Fall back to kubeconfig with specific context
	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: m.kubeconfig},
		&clientcmd.ConfigOverrides{CurrentContext: context},
	).ClientConfig()

	if err != nil {
		return nil, fmt.Errorf("failed to load kubeconfig for context '%s': %w", context, err)
	}

	return &KubernetesClient{config: config}, nil
}

// GetDefaultContext returns the default context name.
func (m *MultiClusterClient) GetDefaultContext() string {
	return m.defaultContext
}

// ListContexts returns all available contexts from the kubeconfig.
func (m *MultiClusterClient) ListContexts() ([]string, error) {
	config, err := clientcmd.LoadFromFile(m.kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to load kubeconfig: %w", err)
	}

	contexts := make([]string, 0, len(config.Contexts))
	for contextName := range config.Contexts {
		contexts = append(contexts, contextName)
	}

	return contexts, nil
}

// getDefaultContext extracts the default context from kubeconfig.
func getDefaultContext(kubeconfig string) (string, error) {
	config, err := clientcmd.LoadFromFile(kubeconfig)
	if err != nil {
		return "", fmt.Errorf("failed to load kubeconfig: %w", err)
	}

	if config.CurrentContext == "" {
		// If no current context is set, try to find any available context
		for contextName := range config.Contexts {
			return contextName, nil
		}
		return "", fmt.Errorf("no contexts found in kubeconfig")
	}

	return config.CurrentContext, nil
}

// ClientWrapper wraps a KubernetesClient and implements the tools.Client interface.
type ClientWrapper struct {
	client  *KubernetesClient
	context string
}

// DynamicClient returns the dynamic client for this context.
func (c *ClientWrapper) DynamicClient() (dynamic.Interface, error) {
	return c.client.DynamicClient()
}

// DiscoClient returns the discovery client for this context.
func (c *ClientWrapper) DiscoClient() (discovery.DiscoveryInterface, error) {
	return c.client.DiscoClient()
}

// Clientset returns the typed clientset for this context.
func (c *ClientWrapper) Clientset() (*kubernetes.Clientset, error) {
	return c.client.Clientset()
}

// RESTMapper returns the REST mapper for this context.
func (c *ClientWrapper) RESTMapper() (meta.RESTMapper, error) {
	return c.client.RESTMapper()
}

// ResourceInterface returns the resource interface for this context.
func (c *ClientWrapper) ResourceInterface(gvr schema.GroupVersionResource, namespaced bool, ns string) (dynamic.ResourceInterface, error) {
	return c.client.ResourceInterface(gvr, namespaced, ns)
}

// GetContext returns the context name for this client.
func (c *ClientWrapper) GetContext() string {
	return c.context
}

// Compile-time verification that ClientWrapper implements tools.Client interface
var _ tools.Client = (*ClientWrapper)(nil)

// Compile-time verification that MultiClusterClient implements tools.MultiClusterClientInterface
var _ tools.MultiClusterClientInterface = (*MultiClusterClient)(nil)

