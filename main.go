package main

import (
	"fmt"

	"github.com/mark3labs/mcp-go/server"
	"github.com/kkb0318/kubernetes-mcp/src/client"
	"github.com/kkb0318/kubernetes-mcp/src/tools"
)

func main() {
	s := server.NewMCPServer(
		"MCP k8s Server",
		"0.1.0",
		server.WithToolCapabilities(false),
	)
	k8s, err := client.NewKubernetesClient()
	if err != nil {
		fmt.Printf("Error starting MCP server: %v\n", err)
	}
	tools.RegisterTools(s, k8s)

	if err := server.ServeStdio(s); err != nil {
		fmt.Printf("Error starting MCP server: %v\n", err)
	}
}
