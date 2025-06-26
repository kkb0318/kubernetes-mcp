package tools

import (
	"fmt"

	openapi_v2 "github.com/google/gnostic-models/openapiv2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/openapi"
	restclient "k8s.io/client-go/rest"
)

type fakeDiscoveryClient struct {
	apiResourceLists []*metav1.APIResourceList
}

var _ discovery.DiscoveryInterface = (*fakeDiscoveryClient)(nil)

func (f *fakeDiscoveryClient) ServerPreferredResources() ([]*metav1.APIResourceList, error) {
	return f.apiResourceLists, nil
}

func (f *fakeDiscoveryClient) ServerResourcesForGroupVersion(groupVersion string) (*metav1.APIResourceList, error) {
	for _, list := range f.apiResourceLists {
		if list == nil {
			continue
		}
		if list.GroupVersion == groupVersion {
			return list, nil
		}
	}
	return nil, fmt.Errorf("no APIResourceList for groupVersion %q", groupVersion)
}
func (f *fakeDiscoveryClient) ServerGroupsAndResources() ([]*metav1.APIGroup, []*metav1.APIResourceList, error) {
	return nil, nil, nil
}

func (f *fakeDiscoveryClient) ServerGroups() (*metav1.APIGroupList, error) {
	return nil, fmt.Errorf("not implemented")
}
func (f *fakeDiscoveryClient) ServerResources() ([]*metav1.APIResourceList, error) {
	return nil, fmt.Errorf("not implemented")
}
func (f *fakeDiscoveryClient) ServerGroupResources() ([]*metav1.APIGroupList, error) {
	return nil, fmt.Errorf("not implemented")
}
func (f *fakeDiscoveryClient) ServerVersion() (*version.Info, error) {
	return nil, fmt.Errorf("not implemented")
}
func (f *fakeDiscoveryClient) OpenAPISchema() (*openapi_v2.Document, error) {
	return nil, fmt.Errorf("not implemented")
}
func (f *fakeDiscoveryClient) OpenAPIV3() openapi.Client {
	return nil
}
func (f *fakeDiscoveryClient) RESTClient() restclient.Interface {
	return nil
}
func (f *fakeDiscoveryClient) SwaggerSchema(version string) (*metav1.APIResourceList, error) {
	return nil, fmt.Errorf("not implemented")
}
func (f *fakeDiscoveryClient) ServerPreferredNamespacedResources() ([]*metav1.APIResourceList, error) {
	return nil, fmt.Errorf("not implemented")
}
func (f *fakeDiscoveryClient) ServerPreferredResourcesForGroupVersion(groupVersion string) (*metav1.APIResourceList, error) {
	return f.ServerResourcesForGroupVersion(groupVersion)
}
func (f *fakeDiscoveryClient) Fresh() bool {
	return true
}
func (f *fakeDiscoveryClient) Invalidate() {
}

func (f *fakeDiscoveryClient) WithLegacy() discovery.DiscoveryInterface {
	return nil
}

// FakeMultiClusterClient implements MultiClusterClientInterface for testing
type FakeMultiClusterClient struct {
	client Client
}

func NewFakeMultiClusterClient(client Client) *FakeMultiClusterClient {
	return &FakeMultiClusterClient{client: client}
}

func (f *FakeMultiClusterClient) GetClient(context string) (Client, error) {
	return f.client, nil
}

func (f *FakeMultiClusterClient) GetDefaultContext() string {
	return "test-context"
}

func (f *FakeMultiClusterClient) ListContexts() ([]string, error) {
	return []string{"test-context", "other-context"}, nil
}

// Compile-time verification that FakeMultiClusterClient implements MultiClusterClientInterface
var _ MultiClusterClientInterface = (*FakeMultiClusterClient)(nil)
