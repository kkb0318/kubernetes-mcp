package tools

import (
	"context"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/apimachinery/pkg/watch"
)

type FakeDescribeResourceInterface struct {
	resource *unstructured.Unstructured
}

func (f *FakeDescribeResourceInterface) Create(ctx context.Context, obj *unstructured.Unstructured, options metav1.CreateOptions, subresources ...string) (*unstructured.Unstructured, error) {
	return nil, nil
}

func (f *FakeDescribeResourceInterface) Update(ctx context.Context, obj *unstructured.Unstructured, options metav1.UpdateOptions, subresources ...string) (*unstructured.Unstructured, error) {
	return nil, nil
}

func (f *FakeDescribeResourceInterface) UpdateStatus(ctx context.Context, obj *unstructured.Unstructured, options metav1.UpdateOptions) (*unstructured.Unstructured, error) {
	return nil, nil
}

func (f *FakeDescribeResourceInterface) Delete(ctx context.Context, name string, options metav1.DeleteOptions, subresources ...string) error {
	return nil
}

func (f *FakeDescribeResourceInterface) DeleteCollection(ctx context.Context, options metav1.DeleteOptions, listOptions metav1.ListOptions) error {
	return nil
}

func (f *FakeDescribeResourceInterface) Get(ctx context.Context, name string, options metav1.GetOptions, subresources ...string) (*unstructured.Unstructured, error) {
	return f.resource, nil
}

func (f *FakeDescribeResourceInterface) List(ctx context.Context, opts metav1.ListOptions) (*unstructured.UnstructuredList, error) {
	return nil, nil
}

func (f *FakeDescribeResourceInterface) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	return nil, nil
}

func (f *FakeDescribeResourceInterface) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, options metav1.PatchOptions, subresources ...string) (*unstructured.Unstructured, error) {
	return nil, nil
}

func (f *FakeDescribeResourceInterface) Apply(ctx context.Context, name string, obj *unstructured.Unstructured, options metav1.ApplyOptions, subresources ...string) (*unstructured.Unstructured, error) {
	return nil, nil
}

func (f *FakeDescribeResourceInterface) ApplyStatus(ctx context.Context, name string, obj *unstructured.Unstructured, options metav1.ApplyOptions) (*unstructured.Unstructured, error) {
	return nil, nil
}

type FakeDescribeKubernetesClient struct {
	resource *unstructured.Unstructured
}

func (f FakeDescribeKubernetesClient) DynamicClient() (dynamic.Interface, error) {
	return nil, nil
}

func (f FakeDescribeKubernetesClient) DiscoClient() (discovery.DiscoveryInterface, error) {
	fakeDisco := &fakeDiscoveryClient{
		apiResourceLists: []*metav1.APIResourceList{
			{
				GroupVersion: "apps/v1",
				APIResources: []metav1.APIResource{
					{Kind: "Deployment", Name: "deployments", Namespaced: true},
				},
			},
			{
				GroupVersion: "v1",
				APIResources: []metav1.APIResource{
					{Kind: "Pod", Name: "pods", Namespaced: true},
				},
			},
		},
	}
	return fakeDisco, nil
}

func (f FakeDescribeKubernetesClient) Clientset() (*kubernetes.Clientset, error) {
	return nil, nil
}

func (f FakeDescribeKubernetesClient) RESTMapper() (meta.RESTMapper, error) {
	return nil, nil
}

func (f FakeDescribeKubernetesClient) ResourceInterface(gvr schema.GroupVersionResource, namespaced bool, ns string) (dynamic.ResourceInterface, error) {
	return &FakeDescribeResourceInterface{resource: f.resource}, nil
}

func TestDescribeTool_Tool(t *testing.T) {
	client := FakeDescribeKubernetesClient{}
	tool := NewDescribeTool(client)

	mcpTool := tool.Tool()

	assert.Equal(t, "describe_resource", mcpTool.Name)
	assert.Contains(t, mcpTool.Description, "Describe a specific Kubernetes resource")
}

func TestDescribeTool_Handler(t *testing.T) {
	testPod := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Pod",
			"metadata": map[string]interface{}{
				"name":      "test-pod",
				"namespace": "default",
				"labels": map[string]interface{}{
					"app": "test",
				},
				"annotations": map[string]interface{}{
					"kubernetes.io/change-cause": "test deployment",
				},
				"creationTimestamp": "2023-01-01T00:00:00Z",
				"resourceVersion":   "12345",
				"uid":               "test-uid-123",
			},
			"spec": map[string]interface{}{
				"containers": []interface{}{
					map[string]interface{}{
						"name":  "test-container",
						"image": "nginx:latest",
					},
				},
			},
			"status": map[string]interface{}{
				"phase": "Running",
				"conditions": []interface{}{
					map[string]interface{}{
						"type":   "Ready",
						"status": "True",
					},
				},
			},
		},
	}

	testCases := []struct {
		name        string
		client      Client
		request     map[string]any
		expectError bool
	}{
		{
			name:   "SuccessfulDescribe",
			client: FakeDescribeKubernetesClient{resource: testPod},
			request: map[string]any{
				"kind":      "Pod",
				"name":      "test-pod",
				"namespace": "default",
			},
			expectError: false,
		},
		{
			name:   "MissingKind",
			client: FakeDescribeKubernetesClient{resource: testPod},
			request: map[string]any{
				"name":      "test-pod",
				"namespace": "default",
			},
			expectError: true,
		},
		{
			name:   "MissingName",
			client: FakeDescribeKubernetesClient{resource: testPod},
			request: map[string]any{
				"kind":      "Pod",
				"namespace": "default",
			},
			expectError: true,
		},
		{
			name:   "EmptyKind",
			client: FakeDescribeKubernetesClient{resource: testPod},
			request: map[string]any{
				"kind":      "",
				"name":      "test-pod",
				"namespace": "default",
			},
			expectError: true,
		},
		{
			name:   "EmptyName",
			client: FakeDescribeKubernetesClient{resource: testPod},
			request: map[string]any{
				"kind":      "Pod",
				"name":      "",
				"namespace": "default",
			},
			expectError: true,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			tool := NewDescribeTool(tt.client)
			req := &mcp.CallToolRequest{}
			req.Params.Arguments = tt.request

			result, err := tool.Handler(context.TODO(), *req)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.NotEmpty(t, result.Content)
				textContent, ok := result.Content[0].(mcp.TextContent)
				assert.True(t, ok)
				assert.Equal(t, "text", textContent.Type)
			}
		})
	}
}

func TestParseAndValidateDescribeParams(t *testing.T) {
	testCases := []struct {
		name        string
		args        map[string]any
		expected    *DescribeResourceInput
		expectedErr bool
	}{
		{
			name: "ValidMinimal",
			args: map[string]any{
				"kind": "Pod",
				"name": "test-pod",
			},
			expected: &DescribeResourceInput{
				Kind:      "Pod",
				Name:      "test-pod",
				Namespace: metav1.NamespaceAll,
			},
			expectedErr: false,
		},
		{
			name: "ValidWithNamespace",
			args: map[string]any{
				"kind":      "Deployment",
				"name":      "test-deployment",
				"namespace": "default",
			},
			expected: &DescribeResourceInput{
				Kind:      "Deployment",
				Name:      "test-deployment",
				Namespace: "default",
			},
			expectedErr: false,
		},
		{
			name: "MissingKind",
			args: map[string]any{
				"name":      "test-pod",
				"namespace": "default",
			},
			expected:    nil,
			expectedErr: true,
		},
		{
			name: "MissingName",
			args: map[string]any{
				"kind":      "Pod",
				"namespace": "default",
			},
			expected:    nil,
			expectedErr: true,
		},
		{
			name: "EmptyKind",
			args: map[string]any{
				"kind":      "",
				"name":      "test-pod",
				"namespace": "default",
			},
			expected:    nil,
			expectedErr: true,
		},
		{
			name: "EmptyName",
			args: map[string]any{
				"kind":      "Pod",
				"name":      "",
				"namespace": "default",
			},
			expected:    nil,
			expectedErr: true,
		},
		{
			name: "EmptyNamespace",
			args: map[string]any{
				"kind":      "Pod",
				"name":      "test-pod",
				"namespace": "",
			},
			expected: &DescribeResourceInput{
				Kind:      "Pod",
				Name:      "test-pod",
				Namespace: metav1.NamespaceAll,
			},
			expectedErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := parseAndValidateDescribeParams(tc.args)

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

func TestDescribeTool_FormatResourceDescription(t *testing.T) {
	tool := NewDescribeTool(FakeDescribeKubernetesClient{})

	testPod := &unstructured.Unstructured{}
	testPod.SetName("test-pod")
	testPod.SetNamespace("default")
	testPod.SetKind("Pod")
	testPod.SetLabels(map[string]string{
		"app": "test",
	})
	testPod.SetAnnotations(map[string]string{
		"kubernetes.io/change-cause": "test deployment",
	})
	testPod.SetUID("test-uid-123")
	testPod.SetResourceVersion("12345")

	// Set spec and status using the Object field
	testPod.Object = map[string]interface{}{
		"metadata": map[string]interface{}{
			"name":      "test-pod",
			"namespace": "default",
			"labels": map[string]interface{}{
				"app": "test",
			},
			"annotations": map[string]interface{}{
				"kubernetes.io/change-cause": "test deployment",
			},
			"uid":             "test-uid-123",
			"resourceVersion": "12345",
		},
		"kind": "Pod",
		"spec": map[string]interface{}{
			"containers": []interface{}{
				map[string]interface{}{
					"name":  "test-container",
					"image": "nginx:latest",
				},
			},
		},
		"status": map[string]interface{}{
			"phase": "Running",
			"conditions": []interface{}{
				map[string]interface{}{
					"type":   "Ready",
					"status": "True",
				},
			},
		},
	}

	result := tool.formatResourceDescription(testPod)

	assert.Equal(t, "test-pod", result["name"])
	assert.Equal(t, "default", result["namespace"])
	assert.Equal(t, "Pod", result["kind"])
	assert.NotNil(t, result["labels"])
	assert.NotNil(t, result["annotations"])
	assert.Equal(t, "test-uid-123", string(result["uid"].(types.UID)))
	assert.Equal(t, "12345", result["resourceVersion"])
	assert.NotNil(t, result["spec"])
	assert.NotNil(t, result["status"])

	// Check spec content
	spec, ok := result["spec"].(map[string]interface{})
	assert.True(t, ok)
	containers, ok := spec["containers"].([]interface{})
	assert.True(t, ok)
	assert.Len(t, containers, 1)

	// Check status content
	status, ok := result["status"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "Running", status["phase"])
	assert.NotNil(t, status["conditions"])
}

func TestDescribeTool_FormatResourceDescriptionWithoutSpecStatus(t *testing.T) {
	tool := NewDescribeTool(FakeDescribeKubernetesClient{})

	// Create a resource without spec and status
	testConfigMap := &unstructured.Unstructured{}
	testConfigMap.SetName("test-configmap")
	testConfigMap.SetNamespace("default")
	testConfigMap.SetKind("ConfigMap")
	testConfigMap.SetUID("configmap-uid-123")

	testConfigMap.Object = map[string]interface{}{
		"metadata": map[string]interface{}{
			"name":      "test-configmap",
			"namespace": "default",
			"uid":       "configmap-uid-123",
		},
		"kind": "ConfigMap",
		"data": map[string]interface{}{
			"key1": "value1",
			"key2": "value2",
		},
	}

	result := tool.formatResourceDescription(testConfigMap)

	assert.Equal(t, "test-configmap", result["name"])
	assert.Equal(t, "default", result["namespace"])
	assert.Equal(t, "ConfigMap", result["kind"])
	assert.Equal(t, "configmap-uid-123", string(result["uid"].(types.UID)))

	// spec and status should not be present since they don't exist in the resource
	_, specExists := result["spec"]
	_, statusExists := result["status"]
	assert.False(t, specExists)
	assert.False(t, statusExists)
}

