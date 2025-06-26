package tools

import (
	"context"
	"os"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/yaml"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

func NewFakeKubernetesClient() {
}

type FakeKubernetesClient struct{}

func (f FakeKubernetesClient) DynamicClient() (dynamic.Interface, error) {
	return nil, nil
}

func (f FakeKubernetesClient) DiscoClient() (discovery.DiscoveryInterface, error) {
	fakeDisco := &fakeDiscoveryClient{
		apiResourceLists: []*metav1.APIResourceList{
			{
				GroupVersion: "apps/v1",
				APIResources: []metav1.APIResource{
					{Kind: "Deployment", Name: "deployments", Namespaced: true},
				},
			},
		},
	}
	return fakeDisco, nil
}
func (f FakeKubernetesClient) Clientset() (*kubernetes.Clientset, error) {
	return nil, nil
}
func (f FakeKubernetesClient) RESTMapper() (meta.RESTMapper, error) {
	return nil, nil
}
func (f FakeKubernetesClient) ResourceInterface(gvr schema.GroupVersionResource, namespaced bool, ns string) (dynamic.ResourceInterface, error) {
	scheme := runtime.NewScheme()
	_ = appsv1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	depUnstr := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "apps/v1",
			"kind":       "Deployment",
			"metadata": map[string]any{
				"name":      "foo-deployment",
				"namespace": "default",
			},
		},
	}

	fakeDynClient := fake.NewSimpleDynamicClient(scheme, depUnstr)
	ri := fakeDynClient.Resource(gvr).Namespace(ns)
	return ri, nil
}

func TestResource(t *testing.T) {
	testCases := []struct {
		name     string
		input    Client
		request  map[string]any
		expected *mcp.CallToolResult
	}{
		{
			name:  "Success",
			input: FakeKubernetesClient{},
			request: map[string]any{
				"namespace": "default",
				"kind":      "deployments",
			},
			expected: &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Annotated: mcp.Annotated{
							Annotations: nil,
						},
						Type: "text",
						Text: "[{\"name\":\"foo-deployment\",\"namespace\":\"default\",\"kind\":\"Deployment\"}]",
					},
				},
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			multiClient := NewFakeMultiClusterClient(tt.input)
			l := NewListTool(multiClient)
			req := &mcp.CallToolRequest{}
			req.Params.Arguments = tt.request
			actual, err := l.Handler(context.TODO(), *req)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestFindGVR(t *testing.T) {

	tests := []struct {
		inputKind      string
		datapath       string
		expected       *schema.GroupVersionResource
		expectingError bool
	}{
		{
			inputKind: "hr",
			datapath:  "testdata/apiresources.yaml",
			expected: &schema.GroupVersionResource{
				Group:    "helm.toolkit.fluxcd.io",
				Version:  "v2",
				Resource: "helmreleases",
			},
			expectingError: false,
		},
		{
			inputKind: "HelmRelease",
			datapath:  "testdata/apiresources.yaml",
			expected: &schema.GroupVersionResource{
				Group:    "helm.toolkit.fluxcd.io",
				Version:  "v2",
				Resource: "helmreleases",
			},
			expectingError: false,
		},
		{
			inputKind: "helmreleases",
			datapath:  "testdata/apiresources.yaml",
			expected: &schema.GroupVersionResource{
				Group:    "helm.toolkit.fluxcd.io",
				Version:  "v2",
				Resource: "helmreleases",
			},
			expectingError: false,
		},
		{
			inputKind: "pod",
			datapath:  "testdata/apiresources.yaml",
			expected: &schema.GroupVersionResource{
				Group:    "",
				Version:  "v1",
				Resource: "pods",
			},
			expectingError: false,
		},
		{
			inputKind:      "unknownkind",
			datapath:       "testdata/apiresources.yaml",
			expected:       &schema.GroupVersionResource{},
			expectingError: true,
		},
		{
			inputKind: "sa",
			datapath:  "testdata/apiresources.yaml",
			expected: &schema.GroupVersionResource{
				Group:    "",
				Version:  "v1",
				Resource: "serviceaccounts",
			},
			expectingError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.inputKind, func(t *testing.T) {
			// arrange
			data, err := os.ReadFile(tt.datapath)
			if err != nil {
				t.Fatalf("Failed to read %s: %v", tt.datapath, err)
			}
			var apiResLists []*metav1.APIResourceList
			if err := yaml.Unmarshal(data, &apiResLists); err != nil {
				t.Fatalf("Failed to unmarshal Yaml: %v", err)
			}

			// act
			actual, err := findGVRByKind(apiResLists, tt.inputKind)

			// assert
			if tt.expectingError {
				assert.Error(t, err)
			} else {
				assert.Equal(t, tt.expected, actual.ToGroupVersionResource())
			}
		})
	}
}

func TestFindGVRsByGroupSubstring(t *testing.T) {
	tests := []struct {
		name           string
		groupSubstring string
		datapath       string
		expectedGVRs   []*schema.GroupVersionResource
		expectingError bool
	}{
		{
			name:           "Helm group matches",
			groupSubstring: "helm.toolkit.fluxcd.io",
			datapath:       "testdata/apiresources.yaml",
			expectedGVRs: []*schema.GroupVersionResource{
				{
					Group:    "helm.toolkit.fluxcd.io",
					Version:  "v2",
					Resource: "helmreleases",
				},
			},
			expectingError: false,
		},
		{
			name:           "flux group matches",
			groupSubstring: "fluxcd",
			datapath:       "testdata/apiresources.yaml",
			expectedGVRs: []*schema.GroupVersionResource{
				{
					Group:    "helm.toolkit.fluxcd.io",
					Version:  "v2",
					Resource: "helmreleases",
				},
				{
					Group:    "kustomize.toolkit.fluxcd.io",
					Version:  "v1",
					Resource: "kustomizations",
				},
				{
					Group:    "notification.toolkit.fluxcd.io",
					Version:  "v1",
					Resource: "receivers",
				},
				{
					Group:    "notification.toolkit.fluxcd.io",
					Version:  "v1beta3",
					Resource: "providers",
				},
				{
					Group:    "notification.toolkit.fluxcd.io",
					Version:  "v1beta3",
					Resource: "alerts",
				},
				{
					Group:    "source.toolkit.fluxcd.io",
					Version:  "v1",
					Resource: "gitrepositories",
				},
				{
					Group:    "source.toolkit.fluxcd.io",
					Version:  "v1",
					Resource: "buckets",
				},
				{
					Group:    "source.toolkit.fluxcd.io",
					Version:  "v1",
					Resource: "helmrepositories",
				},
				{
					Group:    "source.toolkit.fluxcd.io",
					Version:  "v1",
					Resource: "helmcharts",
				},
				{
					Group:    "source.toolkit.fluxcd.io",
					Version:  "v1beta2",
					Resource: "ocirepositories",
				},
			},
			expectingError: false,
		},
		{
			name:           "No group matches",
			groupSubstring: "nonexistent.group",
			datapath:       "testdata/apiresources.yaml",
			expectedGVRs:   nil,
			expectingError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// arrange
			data, err := os.ReadFile(tt.datapath)
			if err != nil {
				t.Fatalf("failed to read %s: %v", tt.datapath, err)
			}
			var apiResLists []*metav1.APIResourceList
			if err := yaml.Unmarshal(data, &apiResLists); err != nil {
				t.Fatalf("failed to unmarshal YAML: %v", err)
			}

			// act
			actualGVRs, err := findGVRsByGroupSubstring(apiResLists, tt.groupSubstring)

			// assert
			if tt.expectingError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedGVRs, actualGVRs.ToGroupVersionResources())
		})
	}
}

func TestParseAndValidateListParams(t *testing.T) {
	testCases := []struct {
		name        string
		args        map[string]any
		expected    *ListResourcesInput
		expectedErr bool
	}{
		{
			name: "MinimalValid",
			args: map[string]any{
				"kind": "pods",
			},
			expected: &ListResourcesInput{
				Kind:           "pods",
				Namespace:      metav1.NamespaceAll,
				TimeoutSeconds: 30,
			},
			expectedErr: false,
		},
		{
			name: "FullValid",
			args: map[string]any{
				"kind":           "deployments",
				"namespace":      "default",
				"labelSelector":  "app=nginx",
				"fieldSelector":  "metadata.name=my-deployment",
				"limit":          float64(10),
				"timeoutSeconds": float64(60),
				"showDetails":    true,
			},
			expected: &ListResourcesInput{
				Kind:           "deployments",
				Namespace:      "default",
				LabelSelector:  "app=nginx",
				FieldSelector:  "metadata.name=my-deployment",
				Limit:          10,
				TimeoutSeconds: 60,
				ShowDetails:    true,
			},
			expectedErr: false,
		},
		{
			name: "MissingKind",
			args: map[string]any{
				"namespace": "default",
			},
			expected:    nil,
			expectedErr: true,
		},
		{
			name: "EmptyKind",
			args: map[string]any{
				"kind": "",
			},
			expected:    nil,
			expectedErr: true,
		},
		{
			name: "WithShowDetails",
			args: map[string]any{
				"kind":        "pods",
				"showDetails": true,
			},
			expected: &ListResourcesInput{
				Kind:           "pods",
				Namespace:      metav1.NamespaceAll,
				TimeoutSeconds: 30,
				ShowDetails:    true,
			},
			expectedErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := parseAndValidateListParams(tc.args)

			if tc.expectedErr {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expected, result)
			}
		})
	}
}

func TestListTool_Tool(t *testing.T) {
	client := FakeKubernetesClient{}
	multiClient := NewFakeMultiClusterClient(client)
	tool := NewListTool(multiClient)

	mcpTool := tool.Tool()

	assert.Equal(t, "list_resources", mcpTool.Name)
	assert.Contains(t, mcpTool.Description, "List Kubernetes resources")
}

func TestExtractResourceStatus(t *testing.T) {
	client := FakeKubernetesClient{}
	multiClient := NewFakeMultiClusterClient(client)
	tool := NewListTool(multiClient)

	// Create a mock unstructured object with status
	obj := &unstructured.Unstructured{}
	obj.SetName("test-pod")
	obj.SetNamespace("default")
	obj.SetKind("Pod")

	// Set status using unstructured.SetNestedField to avoid deep copy issues
	obj.Object = map[string]interface{}{
		"metadata": map[string]interface{}{
			"name":      "test-pod",
			"namespace": "default",
		},
		"kind": "Pod",
		"status": map[string]interface{}{
			"phase": "Running",
			"conditions": []interface{}{
				map[string]interface{}{
					"type":   "Ready",
					"status": "True",
				},
			},
			"containerStatuses": []interface{}{
				map[string]interface{}{
					"name":         "container1",
					"ready":        true,
					"restartCount": int64(0),
				},
			},
		},
	}

	result := tool.extractResourceStatus(obj)

	assert.Equal(t, "test-pod", result.Name)
	assert.Equal(t, "default", result.Namespace)
	assert.Equal(t, "Pod", result.Kind)
	assert.NotNil(t, result.Status)

	// Check that status contains the expected fields
	statusMap, ok := result.Status.(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "Running", statusMap["phase"])
	assert.NotNil(t, statusMap["conditions"])
	assert.NotNil(t, statusMap["containerStatuses"])
}

func TestBuildListOptions(t *testing.T) {
	testCases := []struct {
		name     string
		input    *ListResourcesInput
		expected metav1.ListOptions
	}{
		{
			name: "FullOptions",
			input: &ListResourcesInput{
				Kind:           "deployments",
				Namespace:      "default",
				LabelSelector:  "app=nginx",
				FieldSelector:  "metadata.name=test",
				Limit:          10,
				TimeoutSeconds: 60,
			},
			expected: metav1.ListOptions{
				LabelSelector:  "app=nginx",
				FieldSelector:  "metadata.name=test",
				Limit:          10,
				TimeoutSeconds: func() *int64 { v := int64(60); return &v }(),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client := FakeKubernetesClient{}
			multiClient := NewFakeMultiClusterClient(client)
			tool := NewListTool(multiClient)
			result := tool.buildListOptions(tc.input)

			assert.Equal(t, tc.expected.LabelSelector, result.LabelSelector)
			assert.Equal(t, tc.expected.FieldSelector, result.FieldSelector)
			assert.Equal(t, tc.expected.Limit, result.Limit)
			if tc.expected.TimeoutSeconds != nil {
				assert.NotNil(t, result.TimeoutSeconds)
				assert.Equal(t, *tc.expected.TimeoutSeconds, *result.TimeoutSeconds)
			}
		})
	}
}
