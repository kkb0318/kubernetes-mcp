package client

import (
	"fmt"
	"os"
	"path/filepath"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
)

type KubernetesClient struct {
	config *rest.Config
}

func NewKubernetesClient() (*KubernetesClient, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		// fallback to kubeconfig
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
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("failed to load kubeconfig: %w", err)
		}
	}
	return &KubernetesClient{config: config}, nil
}

func (k *KubernetesClient) DynamicClient() (dynamic.Interface, error) {
	return dynamic.NewForConfig(k.config)
}
func (k *KubernetesClient) DiscoClient() (discovery.DiscoveryInterface, error) {
	return discovery.NewDiscoveryClientForConfig(k.config)
}
func (k *KubernetesClient) Clientset() (*kubernetes.Clientset, error) {
	return kubernetes.NewForConfig(k.config)
}
func (k *KubernetesClient) RESTMapper() (meta.RESTMapper, error) {
	disco, err := k.DiscoClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create discovery client: %w", err)
	}
	return restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(disco)), nil
}

func (k *KubernetesClient) ResourceInterface(
	gvr schema.GroupVersionResource,
	namespaced bool,
	ns string,
) (dynamic.ResourceInterface, error) {
	dynClient, err := k.DynamicClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}

	if !namespaced {
		return dynClient.Resource(gvr), nil
	}
	return dynClient.Resource(gvr).Namespace(ns), nil
}
