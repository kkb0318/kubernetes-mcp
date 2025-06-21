package main

import (
	"fmt"
	"os"

	"github.com/mark3labs/mcp-go/server"
	"github.com/kkb0318/kubernetes-mcp/src/client"
	"github.com/kkb0318/kubernetes-mcp/src/tools"
)

const Version = "0.1.0"

func main() {
	s := server.NewMCPServer(
		"MCP k8s Server",
		Version,
		server.WithToolCapabilities(false),
	)
	
	k8s, err := client.NewKubernetesClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating Kubernetes client: %v\n", err)
		os.Exit(1)
	}
	
	tools.RegisterTools(s, k8s)

	if err := server.ServeStdio(s); err != nil {
		fmt.Fprintf(os.Stderr, "Error starting MCP server: %v\n", err)
		os.Exit(1)
	}
}
