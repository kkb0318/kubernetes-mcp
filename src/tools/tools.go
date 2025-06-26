package tools

import (
	"github.com/mark3labs/mcp-go/server"
)

// RegisterTools は MCPServer に対してツールをまとめて登録します
func RegisterTools(s *server.MCPServer, multiClient MultiClusterClientInterface) {
	tools := []Tools{
		NewListTool(multiClient),
		NewLogTool(multiClient),
		NewDescribeTool(multiClient),
		NewListEventsTool(multiClient),
	}
	for _, t := range tools {
		s.AddTool(t.Tool(), t.Handler)
	}
}
