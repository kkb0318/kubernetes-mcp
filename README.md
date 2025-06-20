# Kubernetes MCP Server

[![tests](https://github.com/kkb0318/kubernetes-mcp/actions/workflows/test.yml/badge.svg?branch=main)](https://github.com/kkb0318/kubernetes-mcp/actions/workflows/test.yml)


https://github.com/user-attachments/assets/89df70b0-65d1-461c-b4ab-84b2087136fa


A Model Context Protocol (MCP) server for Kubernetes debugging and inspection. This server provides read-only access to Kubernetes resources without the ability to create or modify them, making it safe for debugging and monitoring purposes.

## Features

- **Read-only access**: Safely inspect Kubernetes resources without modification capabilities
- **CRD support**: Works with any Custom Resource Definitions (CRDs) in your cluster
- **Substring search**: Discover resources by API group substring (e.g., "flux" for FluxCD, "argo" for ArgoCD)
- **Built-in tools**:
  - `list_resources`: List and filter Kubernetes resources
  - `describe_resource`: Get detailed information about specific resources
  - `get_pod_logs`: Retrieve pod logs with advanced filtering

## Installation

### Prerequisites

- Access to a Kubernetes cluster (kubeconfig required)

### Option 1: Install with Go

If you have Go installed, this is the easiest way:

```bash
go install github.com/kkb0318/kubernetes-mcp@latest
```

The binary will be installed to `$GOPATH/bin/kubernetes-mcp` (or `$HOME/go/bin/kubernetes-mcp` if `GOPATH` is not set).

### Option 2: Build from source

If you prefer to build from source:

**Requirements:**
- Go 1.24 or later

```bash
git clone https://github.com/kkb0318/kubernetes-mcp.git
cd kubernetes-mcp
go build -o kubernetes-mcp .
```

## Usage

The server uses your default kubeconfig for cluster access. Ensure you have proper read permissions for the resources you want to inspect.

### Running the server

```bash
./kubernetes-mcp
```

## Available Tools

### 1. `list_resources`

List Kubernetes resources with filtering capabilities.

**Parameters:**
- `kind` (required): Resource type (Pod, Deployment, Service, etc.) or "all" for discovery
- `groupFilter` (optional): Filter by API group substring to discover project-specific resources
- `namespace` (optional): Target namespace (defaults to all namespaces)
- `labelSelector` (optional): Filter by labels (e.g., "app=nginx")
- `fieldSelector` (optional): Filter by fields (e.g., "metadata.name=my-pod")
- `limit` (optional): Maximum number of resources to return
- `timeoutSeconds` (optional): Request timeout (default: 30s)
- `showDetails` (optional): Return full resource objects instead of summary

**Example usage:**
```json
{
  "kind": "Pod",
  "namespace": "default",
  "labelSelector": "app=nginx"
}
```

**Discovery mode:**
```json
{
  "kind": "all",
  "groupFilter": "flux"
}
```

### 2. `describe_resource`

Get detailed information about a specific resource.

**Parameters:**
- `kind` (required): Resource type
- `name` (required): Resource name
- `namespace` (optional): Target namespace

**Example usage:**
```json
{
  "kind": "Pod",
  "name": "nginx-pod",
  "namespace": "default"
}
```

### 3. `get_pod_logs`

Retrieve pod logs with various filtering options.

**Parameters:**
- `name` (required): Pod name
- `namespace` (optional): Pod namespace (defaults to "default")
- `container` (optional): Specific container name
- `tail` (optional): Number of lines from the end (default: 100)
- `since` (optional): Duration like "5s", "2m", "3h"
- `sinceTime` (optional): RFC3339 timestamp
- `timestamps` (optional): Include timestamps
- `previous` (optional): Get logs from previous container instance

**Example usage:**
```json
{
  "name": "nginx-pod",
  "namespace": "default",
  "tail": 50,
  "since": "5m"
}
```

## Key Features

### CRD Support

The server automatically discovers and works with any Custom Resource Definitions in your cluster. Simply use the CRD's Kind name with the `list_resources` or `describe_resource` tools.

### Resource Discovery

Use the `groupFilter` parameter to discover resources by API group substring:

- `"flux"` - Discover FluxCD resources (HelmReleases, Kustomizations, etc.)
- `"argo"` - Discover ArgoCD resources (Applications, AppProjects, etc.)
- `"istio"` - Discover Istio resources (VirtualServices, DestinationRules, etc.)
- `"cert-manager"` - Discover cert-manager resources (Certificates, Issuers, etc.)

### Safety First

This server is designed for debugging and inspection only:
- No resource creation, modification, or deletion capabilities
- Read-only access to cluster resources
- Safe to use in production environments for monitoring

## Configuration

The server uses the standard Kubernetes client configuration:
- `~/.kube/config` for local development
- In-cluster configuration when running as a pod
- Respects `KUBECONFIG` environment variable

## Contributing

This project is open source and welcomes contributions. Please ensure all changes maintain the read-only nature of the server.

## License

[Add your license information here]

## Support

For issues and questions, please use the GitHub issue tracker.
