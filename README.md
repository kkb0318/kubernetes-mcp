# Kubernetes MCP Server

[![tests](https://github.com/kkb0318/kubernetes-mcp/actions/workflows/test.yml/badge.svg?branch=main)](https://github.com/kkb0318/kubernetes-mcp/actions/workflows/test.yml)
[![codecov](https://codecov.io/gh/kkb0318/kubernetes-mcp/graph/badge.svg?token=RPOAC26LAH)](https://codecov.io/gh/kkb0318/kubernetes-mcp)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

https://github.com/user-attachments/assets/89df70b0-65d1-461c-b4ab-84b2087136fa

A Model Context Protocol (MCP) server that provides safe, read-only access to Kubernetes resources for debugging and inspection. Built with security in mind, it offers comprehensive cluster visibility without modification capabilities.

## Features

- **🔒 Read-only security**: Safely inspect Kubernetes resources without modification capabilities
- **🎯 CRD support**: Works seamlessly with any Custom Resource Definitions in your cluster
- **🌐 Multi-cluster support**: Switch between different Kubernetes contexts seamlessly
- **🔍 Smart discovery**: Find resources by API group substring (e.g., "flux" for FluxCD, "argo" for ArgoCD)
- **⚡ High performance**: Efficient resource querying with filtering and pagination
- **🛠️ Comprehensive toolset**:
  - `list_resources`: List and filter Kubernetes resources with advanced options
  - `describe_resource`: Get detailed information about specific resources
  - `get_pod_logs`: Retrieve pod logs with sophisticated filtering capabilities
  - `list_events`: List and filter Kubernetes events for debugging and monitoring
  - `list_contexts`: List all available Kubernetes contexts from kubeconfig

## 🚀 Quick Start

### Prerequisites

- Kubernetes cluster access with a valid kubeconfig file
- Go 1.24+ (for building from source)

### Installation Options

#### Option 1: Install with Go (Recommended)

```bash
go install github.com/kkb0318/kubernetes-mcp@latest
```

The binary will be available at `$GOPATH/bin/kubernetes-mcp` (or `$HOME/go/bin/kubernetes-mcp` if `GOPATH` is not set).

#### Option 2: Build from Source

```bash
git clone https://github.com/kkb0318/kubernetes-mcp.git
cd kubernetes-mcp
go build -o kubernetes-mcp .
```

## ⚙️ Configuration

### MCP Server Setup

Add the server to your MCP configuration:

#### Basic Configuration
Uses `~/.kube/config` automatically:
```json
{
  "mcpServers": {
    "kubernetes": {
      "command": "/path/to/kubernetes-mcp"
    }
  }
}
```

#### Custom Kubeconfig
```json
{
  "mcpServers": {
    "kubernetes": {
      "command": "/path/to/kubernetes-mcp",
      "env": {
        "KUBECONFIG": "/path/to/your/kubeconfig"
      }
    }
  }
}
```

> **Note**: Replace `/path/to/kubernetes-mcp` with your actual binary path.

### Standalone Usage

```bash
# Default kubeconfig (~/.kube/config)
./kubernetes-mcp

# Custom kubeconfig path
KUBECONFIG=/path/to/your/kubeconfig ./kubernetes-mcp
```

**Important**: Ensure you have appropriate read permissions for the Kubernetes resources you want to inspect.

## 🛠️ Available Tools

### `list_resources`
List and filter Kubernetes resources with advanced capabilities.

| Parameter | Type | Description |
|-----------|------|-------------|
| `context` | optional | Kubernetes context name from kubeconfig (leave empty for current context) |
| `kind` | **required** | Resource type (Pod, Deployment, Service, etc.) or "all" for discovery |
| `groupFilter` | optional | Filter by API group substring for project-specific resources |
| `namespace` | optional | Target namespace (defaults to all namespaces) |
| `labelSelector` | optional | Filter by labels (e.g., "app=nginx") |
| `fieldSelector` | optional | Filter by fields (e.g., "metadata.name=my-pod") |
| `limit` | optional | Maximum number of resources to return |
| `timeoutSeconds` | optional | Request timeout (default: 30s) |
| `showDetails` | optional | Return full resource objects instead of summary |

**Examples:**
```json
// List pods with label selector
{
  "kind": "Pod",
  "namespace": "default",
  "labelSelector": "app=nginx"
}

// List pods from a specific cluster context
{
  "kind": "Pod",
  "context": "production-cluster",
  "namespace": "default"
}

// Discover FluxCD resources
{
  "kind": "all",
  "groupFilter": "flux"
}
```

### `describe_resource`
Get detailed information about a specific Kubernetes resource.

| Parameter | Type | Description |
|-----------|------|-------------|
| `context` | optional | Kubernetes context name from kubeconfig (leave empty for current context) |
| `kind` | **required** | Resource type (Pod, Deployment, etc.) |
| `name` | **required** | Resource name |
| `namespace` | optional | Target namespace |

**Example:**
```json
{
  "kind": "Pod",
  "name": "nginx-pod",
  "namespace": "default"
}
```

### `get_pod_logs`
Retrieve pod logs with sophisticated filtering options.

| Parameter | Type | Description |
|-----------|------|-------------|
| `context` | optional | Kubernetes context name from kubeconfig (leave empty for current context) |
| `name` | **required** | Pod name |
| `namespace` | optional | Pod namespace (defaults to "default") |
| `container` | optional | Specific container name |
| `tail` | optional | Number of lines from the end (default: 100) |
| `since` | optional | Duration like "5s", "2m", "3h" |
| `sinceTime` | optional | RFC3339 timestamp |
| `timestamps` | optional | Include timestamps in output |
| `previous` | optional | Get logs from previous container instance |

**Example:**
```json
{
  "name": "nginx-pod",
  "namespace": "default",
  "tail": 50,
  "since": "5m",
  "timestamps": true
}
```

### `list_events`
List and filter Kubernetes events with advanced filtering options for debugging and monitoring.

| Parameter | Type | Description |
|-----------|------|-------------|
| `context` | optional | Kubernetes context name from kubeconfig (leave empty for current context) |
| `namespace` | optional | Target namespace (leave empty for all namespaces) |
| `object` | optional | Filter by object name (e.g., pod name, deployment name) |
| `eventType` | optional | Filter by event type: "Normal" or "Warning" (case-insensitive) |
| `reason` | optional | Filter by event reason (e.g., "Pulled", "Failed", "FailedScheduling") |
| `since` | optional | Duration like "5s", "2m", "1h" |
| `sinceTime` | optional | RFC3339 timestamp (e.g., "2025-06-20T10:00:00Z") |
| `limit` | optional | Maximum number of events to return (default: 100) |
| `timeoutSeconds` | optional | Request timeout (default: 30s) |

**Examples:**
```json
// List recent warning events
{
  "eventType": "Warning",
  "since": "30m"
}

// List events for a specific pod
{
  "object": "nginx-pod",
  "namespace": "default"
}

// List failed scheduling events
{
  "reason": "FailedScheduling",
  "limit": 50
}
```

### `list_contexts`
List all available Kubernetes contexts from your kubeconfig file.

**Parameters:**
None - this tool takes no parameters.

**Example Response:**
```json
{
  "contexts": [
    {
      "name": "production-cluster",
      "is_current": false
    },
    {
      "name": "staging-cluster", 
      "is_current": true
    },
    {
      "name": "development-cluster",
      "is_current": false
    }
  ],
  "current_context": "staging-cluster",
  "total": 3
}
```

**Use Case:**
Perfect for multi-cluster workflows where you need to:
- Discover available Kubernetes contexts
- Identify the current active context
- Plan operations across multiple clusters

## 🌟 Advanced Features

### 🌐 Multi-Cluster Support
Seamlessly work with multiple Kubernetes clusters using context switching:

- **Context Parameter**: All tools now support an optional `context` parameter to specify which cluster to query
- **Automatic Discovery**: Uses your existing kubeconfig file and automatically discovers available contexts
- **Default Context**: When no context is specified, uses the current context from your kubeconfig
- **Cached Connections**: Efficiently manages connections to multiple clusters with connection caching

**Multi-cluster Examples:**
```json
// Query production cluster
{
  "kind": "Pod",
  "context": "production-cluster",
  "namespace": "default"
}

// Get logs from staging environment
{
  "name": "api-server",
  "context": "staging-cluster",
  "namespace": "api"
}

// Compare resources across environments (use multiple calls)
{
  "kind": "Deployment",
  "context": "production-cluster",
  "namespace": "app"
}
```

### 🎯 Custom Resource Definition (CRD) Support
Automatically discovers and works with any CRDs in your cluster. Simply use the CRD's Kind name with `list_resources` or `describe_resource` tools.

### 🔍 Smart Resource Discovery
Use the `groupFilter` parameter to discover resources by API group substring:

| Filter | Discovers | Examples |
|--------|-----------|----------|
| `"flux"` | FluxCD resources | HelmReleases, Kustomizations, GitRepositories |
| `"argo"` | ArgoCD resources | Applications, AppProjects, ApplicationSets |
| `"istio"` | Istio resources | VirtualServices, DestinationRules, Gateways |
| `"cert-manager"` | cert-manager resources | Certificates, Issuers, ClusterIssuers |

### 🔒 Security & Safety
Built with security as a primary concern:
- ✅ **Read-only access** - No resource creation, modification, or deletion
- ✅ **Production safe** - Secure for use in production environments
- ✅ **Minimal permissions** - Only requires read access to cluster resources
- ✅ **No destructive operations** - Cannot harm your cluster

---

## 🤝 Contributing

We welcome contributions! Please ensure all changes maintain the read-only nature of the server and include appropriate tests.

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
