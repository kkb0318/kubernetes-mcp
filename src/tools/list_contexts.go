package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

// ListContextsTool は Kubernetes の context 一覧を返すツールです
type ListContextsTool struct {
	multiClient MultiClusterClientInterface
}

// NewListContextsTool は新しい ListContextsTool インスタンスを作成します
func NewListContextsTool(multiClient MultiClusterClientInterface) *ListContextsTool {
	return &ListContextsTool{multiClient: multiClient}
}

// Tool は MCP ツールの定義を返します
func (t *ListContextsTool) Tool() mcp.Tool {
	return mcp.Tool{
		Name:        "list_contexts",
		Description: "List all available Kubernetes contexts from kubeconfig",
		InputSchema: mcp.ToolInputSchema{
			Type:       "object",
			Properties: map[string]interface{}{},
		},
	}
}

// ContextInfo は context の情報を表す構造体です
type ContextInfo struct {
	Name      string `json:"name"`
	IsCurrent bool   `json:"is_current"`
}

// ListContextsResponse は list_contexts の応答を表す構造体です
type ListContextsResponse struct {
	Contexts       []ContextInfo `json:"contexts"`
	CurrentContext string        `json:"current_context"`
	Total          int           `json:"total"`
}

// Handler は list_contexts リクエストを処理します
func (t *ListContextsTool) Handler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Get all contexts
	contexts, err := t.multiClient.ListContexts()
	if err != nil {
		return nil, fmt.Errorf("error listing contexts: %w", err)
	}

	// Get current context
	currentContext := t.multiClient.GetDefaultContext()

	// Build response
	contextInfos := make([]ContextInfo, len(contexts))
	for i, contextName := range contexts {
		contextInfos[i] = ContextInfo{
			Name:      contextName,
			IsCurrent: contextName == currentContext,
		}
	}

	response := ListContextsResponse{
		Contexts:       contextInfos,
		CurrentContext: currentContext,
		Total:          len(contexts),
	}

	// Convert to JSON
	jsonBytes, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("error marshaling response: %w", err)
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}

// Compile-time verification that ListContextsTool implements Tools interface
var _ Tools = (*ListContextsTool)(nil)