package tools

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
)

// FakeListContextsMultiClusterClient implements MultiClusterClientInterface for testing list_contexts functionality
type FakeListContextsMultiClusterClient struct {
	contexts       []string
	currentContext string
	listError      error
}

func (f *FakeListContextsMultiClusterClient) ListContexts() ([]string, error) {
	if f.listError != nil {
		return nil, f.listError
	}
	return f.contexts, nil
}

func (f *FakeListContextsMultiClusterClient) GetDefaultContext() string {
	return f.currentContext
}

func (f *FakeListContextsMultiClusterClient) GetClient(context string) (Client, error) {
	return nil, nil
}

func TestListContextsTool_Tool(t *testing.T) {
	multiClient := &FakeListContextsMultiClusterClient{}
	tool := NewListContextsTool(multiClient)

	mcpTool := tool.Tool()

	assert.Equal(t, "list_contexts", mcpTool.Name)
	assert.Equal(t, "List all available Kubernetes contexts from kubeconfig", mcpTool.Description)
	assert.Equal(t, "object", mcpTool.InputSchema.Type)
	assert.NotNil(t, mcpTool.InputSchema.Properties)
}

func TestListContextsTool_Handler(t *testing.T) {
	testCases := []struct {
		name           string
		contexts       []string
		currentContext string
		listError      error
		expectedErr    bool
		validate       func(*testing.T, *ListContextsResponse)
	}{
		{
			name:           "successful listing with multiple contexts",
			contexts:       []string{"context1", "context2", "context3"},
			currentContext: "context2",
			expectedErr:    false,
			validate: func(t *testing.T, response *ListContextsResponse) {
				assert.Equal(t, 3, response.Total)
				assert.Equal(t, "context2", response.CurrentContext)
				assert.Len(t, response.Contexts, 3)

				// Check that contexts are properly mapped
				contextMap := make(map[string]bool)
				for _, ctx := range response.Contexts {
					contextMap[ctx.Name] = ctx.IsCurrent
				}

				assert.True(t, contextMap["context2"], "context2 should be marked as current")
				assert.False(t, contextMap["context1"], "context1 should not be marked as current")
				assert.False(t, contextMap["context3"], "context3 should not be marked as current")
			},
		},
		{
			name:           "single context",
			contexts:       []string{"single-context"},
			currentContext: "single-context",
			expectedErr:    false,
			validate: func(t *testing.T, response *ListContextsResponse) {
				assert.Equal(t, 1, response.Total)
				assert.Equal(t, "single-context", response.CurrentContext)
				assert.Len(t, response.Contexts, 1)
				assert.Equal(t, "single-context", response.Contexts[0].Name)
				assert.True(t, response.Contexts[0].IsCurrent)
			},
		},
		{
			name:           "no contexts",
			contexts:       []string{},
			currentContext: "",
			expectedErr:    false,
			validate: func(t *testing.T, response *ListContextsResponse) {
				assert.Equal(t, 0, response.Total)
				assert.Equal(t, "", response.CurrentContext)
				assert.Len(t, response.Contexts, 0)
			},
		},
		{
			name:           "current context not in list",
			contexts:       []string{"context1", "context2"},
			currentContext: "different-context",
			expectedErr:    false,
			validate: func(t *testing.T, response *ListContextsResponse) {
				assert.Equal(t, 2, response.Total)
				assert.Equal(t, "different-context", response.CurrentContext)
				assert.Len(t, response.Contexts, 2)

				// All contexts should be marked as not current
				for _, ctx := range response.Contexts {
					assert.False(t, ctx.IsCurrent)
				}
			},
		},
		{
			name:        "error listing contexts",
			listError:   assert.AnError,
			expectedErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			multiClient := &FakeListContextsMultiClusterClient{
				contexts:       tc.contexts,
				currentContext: tc.currentContext,
				listError:      tc.listError,
			}

			tool := NewListContextsTool(multiClient)

			req := &mcp.CallToolRequest{}
			req.Params.Arguments = map[string]interface{}{}

			result, err := tool.Handler(context.Background(), *req)

			if tc.expectedErr {
				assert.Error(t, err)
				assert.Nil(t, result)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, result)

			// Parse the JSON response
			textContent, ok := result.Content[0].(mcp.TextContent)
			assert.True(t, ok)
			assert.Equal(t, "text", textContent.Type)
			
			var response ListContextsResponse
			err = json.Unmarshal([]byte(textContent.Text), &response)
			assert.NoError(t, err)

			if tc.validate != nil {
				tc.validate(t, &response)
			}
		})
	}
}

func TestListContextsTool_JSONMarshaling(t *testing.T) {
	testCases := []struct {
		name     string
		response ListContextsResponse
	}{
		{
			name: "complete response",
			response: ListContextsResponse{
				Contexts: []ContextInfo{
					{Name: "context1", IsCurrent: true},
					{Name: "context2", IsCurrent: false},
				},
				CurrentContext: "context1",
				Total:          2,
			},
		},
		{
			name: "empty response",
			response: ListContextsResponse{
				Contexts:       []ContextInfo{},
				CurrentContext: "",
				Total:          0,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test marshaling
			jsonBytes, err := json.MarshalIndent(tc.response, "", "  ")
			assert.NoError(t, err)

			// Test unmarshaling
			var unmarshaled ListContextsResponse
			err = json.Unmarshal(jsonBytes, &unmarshaled)
			assert.NoError(t, err)

			// Verify round-trip consistency
			assert.Equal(t, tc.response, unmarshaled)
		})
	}
}

func TestContextInfo_JSONTags(t *testing.T) {
	ctx := ContextInfo{
		Name:      "test-context",
		IsCurrent: true,
	}

	jsonBytes, err := json.Marshal(ctx)
	assert.NoError(t, err)

	expected := `{"name":"test-context","is_current":true}`
	assert.JSONEq(t, expected, string(jsonBytes))
}

func TestNewListContextsTool(t *testing.T) {
	multiClient := &FakeListContextsMultiClusterClient{}
	tool := NewListContextsTool(multiClient)

	assert.NotNil(t, tool)
	assert.Equal(t, multiClient, tool.multiClient)
}

func TestListContextsTool_ImplementsInterface(t *testing.T) {
	multiClient := &FakeListContextsMultiClusterClient{}
	tool := NewListContextsTool(multiClient)

	// Verify that ListContextsTool implements Tools interface
	var _ Tools = tool
}