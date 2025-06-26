package tools

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

type FakeEventsClient struct {
	clientset *kubernetes.Clientset
	err       error
}

func (f *FakeEventsClient) Clientset() (*kubernetes.Clientset, error) {
	return f.clientset, f.err
}

func (f *FakeEventsClient) DynamicClient() (dynamic.Interface, error) {
	return nil, nil
}

func (f *FakeEventsClient) DiscoClient() (discovery.DiscoveryInterface, error) {
	return nil, nil
}

func (f *FakeEventsClient) RESTMapper() (meta.RESTMapper, error) {
	return nil, nil
}

func (f *FakeEventsClient) ResourceInterface(gvr schema.GroupVersionResource, namespaced bool, ns string) (dynamic.ResourceInterface, error) {
	return nil, nil
}

func TestListEventsTool_Handler_ClientsetError(t *testing.T) {
	client := &FakeEventsClient{
		clientset: nil,
		err:       errors.New("clientset error"),
	}

	multiClient := NewFakeMultiClusterClient(client)
	tool := NewListEventsTool(multiClient)

	req := mcp.CallToolRequest{
		Params: struct {
			Name      string         `json:"name"`
			Arguments map[string]any `json:"arguments,omitempty"`
			Meta      *struct {
				ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
			} `json:"_meta,omitempty"`
		}{
			Arguments: map[string]any{
				"namespace": "default",
			},
		},
	}

	actualResult, actualErr := tool.Handler(context.Background(), req)

	assert.Error(t, actualErr)
	assert.Contains(t, actualErr.Error(), "failed to get clientset: clientset error")
	assert.Nil(t, actualResult)
}

func TestParseAndValidateEventsParams(t *testing.T) {
	testCases := []struct {
		name        string
		args        map[string]any
		expectedErr bool
		validate    func(*testing.T, *ListEventsInput)
	}{
		{
			name: "ValidParams",
			args: map[string]any{
				"namespace":      "default",
				"object":         "test-pod",
				"eventType":      "Warning",
				"reason":         "Failed",
				"since":          "1h",
				"sinceTime":      "2025-06-20T10:00:00Z",
				"limit":          float64(50),
				"timeoutSeconds": float64(60),
			},
			expectedErr: false,
			validate: func(t *testing.T, input *ListEventsInput) {
				assert.Equal(t, "default", input.Namespace)
				assert.Equal(t, "test-pod", input.Object)
				assert.Equal(t, "Warning", input.EventType)
				assert.Equal(t, "Failed", input.Reason)
				assert.Equal(t, "1h", input.Since)
				assert.Equal(t, "2025-06-20T10:00:00Z", input.SinceTime)
				assert.Equal(t, int64(50), input.Limit)
				assert.Equal(t, int64(60), input.TimeoutSeconds)
			},
		},
		{
			name: "MinimalParams",
			args: map[string]any{},
			expectedErr: false,
			validate: func(t *testing.T, input *ListEventsInput) {
				assert.Empty(t, input.Namespace)
				assert.Empty(t, input.Object)
				assert.Empty(t, input.EventType)
				assert.Empty(t, input.Reason)
				assert.Empty(t, input.Since)
				assert.Empty(t, input.SinceTime)
				assert.Equal(t, int64(100), input.Limit)     // Default value
				assert.Equal(t, int64(30), input.TimeoutSeconds) // Default value
			},
		},
		{
			name: "InvalidEventType",
			args: map[string]any{
				"eventType": "Invalid",
			},
			expectedErr: true,
		},
		{
			name: "InvalidNamespace",
			args: map[string]any{
				"namespace": "invalid-namespace-with-invalid-chars!",
			},
			expectedErr: true,
		},
		{
			name: "InvalidSinceDuration",
			args: map[string]any{
				"since": "invalid-duration",
			},
			expectedErr: true,
		},
		{
			name: "InvalidSinceTime",
			args: map[string]any{
				"sinceTime": "invalid-time-format",
			},
			expectedErr: true,
		},
		{
			name: "ValidEventTypeNormal",
			args: map[string]any{
				"eventType": "normal",
			},
			expectedErr: false,
			validate: func(t *testing.T, input *ListEventsInput) {
				assert.Equal(t, "normal", input.EventType)
			},
		},
		{
			name: "ValidEventTypeWarning",
			args: map[string]any{
				"eventType": "WARNING",
			},
			expectedErr: false,
			validate: func(t *testing.T, input *ListEventsInput) {
				assert.Equal(t, "WARNING", input.EventType)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client := &FakeEventsClient{}
			multiClient := NewFakeMultiClusterClient(client)
	tool := NewListEventsTool(multiClient)
			result, err := tool.parseAndValidateEventsParams(tc.args)

			if tc.expectedErr {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				if tc.validate != nil {
					tc.validate(t, result)
				}
			}
		})
	}
}

func TestListEventsTool_buildListOptions(t *testing.T) {
	client := &FakeEventsClient{}
	multiClient := NewFakeMultiClusterClient(client)
	tool := NewListEventsTool(multiClient)

	testCases := []struct {
		name     string
		input    *ListEventsInput
		validate func(*testing.T, metav1.ListOptions)
	}{
		{
			name: "WithObject",
			input: &ListEventsInput{
				Object:         "test-pod",
				Limit:          50,
				TimeoutSeconds: 60,
			},
			validate: func(t *testing.T, opts metav1.ListOptions) {
				assert.Equal(t, "involvedObject.name=test-pod", opts.FieldSelector)
				assert.Equal(t, int64(50), opts.Limit)
				assert.Equal(t, int64(60), *opts.TimeoutSeconds)
			},
		},
		{
			name: "DefaultValues",
			input: &ListEventsInput{},
			validate: func(t *testing.T, opts metav1.ListOptions) {
				assert.Empty(t, opts.FieldSelector)
				assert.Equal(t, int64(0), opts.Limit) // No limit since input defaults weren't set
				assert.Equal(t, int64(0), *opts.TimeoutSeconds) // No timeout since input defaults weren't set
			},
		},
		{
			name: "ZeroLimit",
			input: &ListEventsInput{
				Limit: 0,
			},
			validate: func(t *testing.T, opts metav1.ListOptions) {
				assert.Equal(t, int64(0), opts.Limit) // No default applied in buildListOptions
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			opts := tool.buildListOptions(tc.input)
			tc.validate(t, opts)
		})
	}
}

func TestListEventsTool_filterEvents(t *testing.T) {
	client := &FakeEventsClient{}
	multiClient := NewFakeMultiClusterClient(client)
	tool := NewListEventsTool(multiClient)

	// Create test events
	now := time.Now()
	oneHourAgo := now.Add(-1 * time.Hour)
	twoHoursAgo := now.Add(-2 * time.Hour)

	events := []corev1.Event{
		{
			Type:   "Normal",
			Reason: "Pulled",
			LastTimestamp: metav1.Time{Time: oneHourAgo},
		},
		{
			Type:   "Warning",
			Reason: "Failed",
			LastTimestamp: metav1.Time{Time: twoHoursAgo},
		},
		{
			Type:   "Warning",
			Reason: "FailedScheduling",
			LastTimestamp: metav1.Time{Time: oneHourAgo},
		},
	}

	testCases := []struct {
		name     string
		input    *ListEventsInput
		expected int
	}{
		{
			name:     "NoFilter",
			input:    &ListEventsInput{},
			expected: 3,
		},
		{
			name: "FilterByEventType",
			input: &ListEventsInput{
				EventType: "Warning",
			},
			expected: 2,
		},
		{
			name: "FilterByReason",
			input: &ListEventsInput{
				Reason: "Failed",
			},
			expected: 2, // Both "Failed" and "FailedScheduling" should match
		},
		{
			name: "FilterByExactReason",
			input: &ListEventsInput{
				Reason: "Pulled",
			},
			expected: 1,
		},
		{
			name: "FilterBySince",
			input: &ListEventsInput{
				Since: "90m", // 1.5 hours
			},
			expected: 2, // Only events from 1 hour ago
		},
		{
			name: "CombinedFilters",
			input: &ListEventsInput{
				EventType: "Warning",
				Since:     "90m",
			},
			expected: 1, // Only Warning events from 1 hour ago
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			filtered := tool.filterEvents(events, tc.input)
			assert.Len(t, filtered, tc.expected)
		})
	}
}

func TestListEventsTool_isEventWithinTimeRange(t *testing.T) {
	client := &FakeEventsClient{}
	multiClient := NewFakeMultiClusterClient(client)
	tool := NewListEventsTool(multiClient)

	now := time.Now()
	event := &corev1.Event{
		LastTimestamp: metav1.Time{Time: now.Add(-30 * time.Minute)},
	}

	testCases := []struct {
		name     string
		input    *ListEventsInput
		expected bool
	}{
		{
			name:     "NoTimeFilter",
			input:    &ListEventsInput{},
			expected: true,
		},
		{
			name: "WithinSinceRange",
			input: &ListEventsInput{
				Since: "1h",
			},
			expected: true,
		},
		{
			name: "OutsideSinceRange",
			input: &ListEventsInput{
				Since: "15m",
			},
			expected: false,
		},
		{
			name: "WithinSinceTimeRange",
			input: &ListEventsInput{
				SinceTime: now.Add(-45 * time.Minute).Format(time.RFC3339),
			},
			expected: true,
		},
		{
			name: "OutsideSinceTimeRange",
			input: &ListEventsInput{
				SinceTime: now.Add(-15 * time.Minute).Format(time.RFC3339),
			},
			expected: false,
		},
		{
			name: "InvalidSinceTime",
			input: &ListEventsInput{
				SinceTime: "invalid-time",
			},
			expected: true, // Should include all events if parsing fails
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tool.isEventWithinTimeRange(event, tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestListEventsTool_convertToEventInfos(t *testing.T) {
	client := &FakeEventsClient{}
	multiClient := NewFakeMultiClusterClient(client)
	tool := NewListEventsTool(multiClient)

	now := time.Now()
	events := []corev1.Event{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
			},
			FirstTimestamp: metav1.Time{Time: now.Add(-1 * time.Hour)},
			LastTimestamp:  metav1.Time{Time: now.Add(-30 * time.Minute)},
			Count:          3,
			Type:           "Warning",
			Reason:         "Failed",
			Message:        "Container failed to start",
			InvolvedObject: corev1.ObjectReference{
				Kind:      "Pod",
				Name:      "test-pod",
				Namespace: "default",
			},
			Source: corev1.EventSource{
				Component: "kubelet",
				Host:      "node1",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
			},
			FirstTimestamp: metav1.Time{Time: now.Add(-2 * time.Hour)},
			LastTimestamp:  metav1.Time{Time: now.Add(-1 * time.Hour)},
			Count:          1,
			Type:           "Normal",
			Reason:         "Pulled",
			Message:        "Container image pulled",
			InvolvedObject: corev1.ObjectReference{
				Kind: "Pod",
				Name: "test-pod",
			},
			Source: corev1.EventSource{
				Component: "kubelet",
			},
		},
	}

	eventInfos := tool.convertToEventInfos(events)

	assert.Len(t, eventInfos, 2)

	// Check first event
	assert.Equal(t, int32(3), eventInfos[0].Count)
	assert.Equal(t, "Warning", eventInfos[0].Type)
	assert.Equal(t, "Failed", eventInfos[0].Reason)
	assert.Equal(t, "Container failed to start", eventInfos[0].Message)
	assert.Equal(t, "default", eventInfos[0].Namespace)
	assert.Equal(t, "Pod/test-pod", eventInfos[0].Object)
	assert.Equal(t, "kubelet (node1)", eventInfos[0].Source)

	// Check second event
	assert.Equal(t, int32(1), eventInfos[1].Count)
	assert.Equal(t, "Normal", eventInfos[1].Type)
	assert.Equal(t, "Pulled", eventInfos[1].Reason)
	assert.Equal(t, "Container image pulled", eventInfos[1].Message)
	assert.Equal(t, "Pod/test-pod", eventInfos[1].Object)
	assert.Equal(t, "kubelet", eventInfos[1].Source)
}

func TestListEventsTool_Tool(t *testing.T) {
	client := &FakeEventsClient{}
	multiClient := NewFakeMultiClusterClient(client)
	tool := NewListEventsTool(multiClient)

	mcpTool := tool.Tool()

	assert.Equal(t, "list_events", mcpTool.Name)
	assert.Contains(t, mcpTool.Description, "List Kubernetes events")
	
	// Verify the tool has an input schema
	assert.NotNil(t, mcpTool.InputSchema)
	
	// Check that the tool schema has properties
	assert.NotNil(t, mcpTool.InputSchema.Properties)
	assert.True(t, len(mcpTool.InputSchema.Properties) > 0, "Tool should have input parameters")
}

func TestEventInfoJSONSerialization(t *testing.T) {
	now := time.Now()
	eventInfo := EventInfo{
		FirstTimestamp: metav1.Time{Time: now.Add(-1 * time.Hour)},
		LastTimestamp:  metav1.Time{Time: now},
		Count:          5,
		Type:           "Warning",
		Reason:         "Failed",
		Object:         "Pod/test-pod",
		Message:        "Container failed to start",
		Source:         "kubelet (node1)",
		Namespace:      "default",
	}

	// Test JSON serialization
	jsonData, err := json.Marshal(eventInfo)
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	// Test JSON deserialization
	var deserializedEventInfo EventInfo
	err = json.Unmarshal(jsonData, &deserializedEventInfo)
	assert.NoError(t, err)
	assert.Equal(t, eventInfo.Count, deserializedEventInfo.Count)
	assert.Equal(t, eventInfo.Type, deserializedEventInfo.Type)
	assert.Equal(t, eventInfo.Reason, deserializedEventInfo.Reason)
	assert.Equal(t, eventInfo.Object, deserializedEventInfo.Object)
	assert.Equal(t, eventInfo.Message, deserializedEventInfo.Message)
	assert.Equal(t, eventInfo.Source, deserializedEventInfo.Source)
	assert.Equal(t, eventInfo.Namespace, deserializedEventInfo.Namespace)
}