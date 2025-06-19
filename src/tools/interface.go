package tools

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
)

type Tools interface {
	Tool() mcp.Tool
	Handler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error)
}
