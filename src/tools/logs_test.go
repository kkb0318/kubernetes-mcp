package tools

import (
	"context"
	"errors"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

type FakeLogClient struct {
	clientset *kubernetes.Clientset
	err       error
}

func (f *FakeLogClient) Clientset() (*kubernetes.Clientset, error) {
	return f.clientset, f.err
}

func (f *FakeLogClient) DynamicClient() (dynamic.Interface, error) {
	return nil, nil
}

func (f *FakeLogClient) DiscoClient() (discovery.DiscoveryInterface, error) {
	return nil, nil
}

func (f *FakeLogClient) RESTMapper() (meta.RESTMapper, error) {
	return nil, nil
}

func (f *FakeLogClient) ResourceInterface(gvr schema.GroupVersionResource, namespaced bool, ns string) (dynamic.ResourceInterface, error) {
	return nil, nil
}

func TestLogTool_Handler_ClientsetError(t *testing.T) {
	client := &FakeLogClient{
		clientset: nil,
		err:       errors.New("clientset error"),
	}

	tool := NewLogTool(client)

	req := mcp.CallToolRequest{
		Params: struct {
			Name      string         `json:"name"`
			Arguments map[string]any `json:"arguments,omitempty"`
			Meta      *struct {
				ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
			} `json:"_meta,omitempty"`
		}{
			Arguments: map[string]any{
				"name":      "test-pod",
				"namespace": "default",
			},
		},
	}

	actualResult, actualErr := tool.Handler(context.Background(), req)

	assert.Error(t, actualErr)
	assert.Contains(t, actualErr.Error(), "failed to get clientset: clientset error")
	assert.Nil(t, actualResult)
}

func TestParseAndValidateLogsParams(t *testing.T) {
	testCases := []struct {
		name        string
		args        map[string]any
		expectedErr bool
	}{
		{
			name: "ValidParams",
			args: map[string]any{
				"name":       "test-pod",
				"namespace":  "default",
				"container":  "test-container",
				"tail":       float64(100),
				"since":      "1h",
				"sinceTime":  "2025-06-20T10:00:00Z",
				"timestamps": true,
				"previous":   false,
				"follow":     false,
			},
			expectedErr: false,
		},
		{
			name: "MissingName",
			args: map[string]any{
				"namespace": "default",
			},
			expectedErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := parseAndValidateLogsParams(tc.args)

			if tc.expectedErr {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			}
		})
	}
}

func TestSinceSeconds(t *testing.T) {
	testCases := []struct {
		name     string
		since    string
		expected *int64
	}{
		{
			name:     "EmptyString",
			since:    "",
			expected: nil,
		},
		{
			name:     "ValidDuration",
			since:    "1h",
			expected: func() *int64 { v := int64(3600); return &v }(),
		},
		{
			name:     "InvalidDuration",
			since:    "invalid",
			expected: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := sinceSeconds(tc.since)
			if tc.expected == nil {
				assert.Nil(t, result)
			} else {
				assert.NotNil(t, result)
				assert.Equal(t, *tc.expected, *result)
			}
		})
	}
}

func TestSinceTime(t *testing.T) {
	testCases := []struct {
		name      string
		sinceTime string
		expected  bool
	}{
		{
			name:      "EmptyString",
			sinceTime: "",
			expected:  false,
		},
		{
			name:      "ValidTime",
			sinceTime: "2025-06-20T10:00:00Z",
			expected:  true,
		},
		{
			name:      "InvalidTime",
			sinceTime: "invalid-time",
			expected:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := sinceTime(tc.sinceTime)
			if tc.expected {
				assert.NotNil(t, result)
			} else {
				assert.Nil(t, result)
			}
		})
	}
}

func TestLogTool_Tool(t *testing.T) {
	client := &FakeLogClient{}
	tool := NewLogTool(client)

	mcpTool := tool.Tool()

	assert.Equal(t, "get_pod_logs", mcpTool.Name)
	assert.Contains(t, mcpTool.Description, "Get logs from a Kubernetes pod")
}
