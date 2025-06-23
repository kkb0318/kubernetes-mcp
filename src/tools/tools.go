package tools

import (
	"github.com/mark3labs/mcp-go/server"
)

// RegisterTools は MCPServer に対してツールをまとめて登録します
func RegisterTools(s *server.MCPServer, client Client) {
	tools := []Tools{
		NewListTool(client),
		NewLogTool(client),
		NewDescribeTool(client),
		NewListEventsTool(client),
	}
	for _, t := range tools {
		s.AddTool(t.Tool(), t.Handler)
	}
}
