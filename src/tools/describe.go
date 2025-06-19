package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type DescribeResourceInput struct {
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	Namespace string `json:"namespace,omitempty"`
}

type DescribeTool struct {
	client Client
}

func NewDescribeTool(client Client) *DescribeTool {
	return &DescribeTool{client: client}
}

func (d *DescribeTool) Tool() mcp.Tool {
	return mcp.NewTool("describe_resource",
		mcp.WithDescription("Describe a specific Kubernetes resource by kind and name, similar to 'kubectl describe'"),
		mcp.WithString("kind",
			mcp.Required(),
			mcp.Description("Kind of the Kubernetes resource, e.g., Pod, Deployment, Service, ConfigMap, or any CRD"),
		),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the resource to describe"),
		),
		mcp.WithString("namespace",
			mcp.Description("Kubernetes namespace of the resource (leave empty to search all namespaces, use 'default' for default namespace)"),
		),
	)
}

func (d *DescribeTool) Handler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	input, err := parseAndValidateDescribeParams(req.Params.Arguments)
	if err != nil {
		return nil, err
	}

	gvrMatch, err := d.discoverResourceByKind(input.Kind)
	if err != nil {
		return nil, err
	}

	resource, err := d.getResource(ctx, gvrMatch, input)
	if err != nil {
		return nil, err
	}

	describeOutput := d.formatResourceDescription(resource)

	out, err := json.Marshal(describeOutput)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal describe output: %w", err)
	}

	return mcp.NewToolResultText(string(out)), nil
}

func (d *DescribeTool) discoverResourceByKind(kind string) (*gvrMatch, error) {
	discoClient, err := d.client.DiscoClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create discovery client: %w", err)
	}

	apiResourceLists, err := discoClient.ServerPreferredResources()
	if err != nil {
		return nil, fmt.Errorf("failed to discover resources: %w", err)
	}

	return findGVRByKind(apiResourceLists, kind)
}

func (d *DescribeTool) getResource(ctx context.Context, gvrMatch *gvrMatch, input *DescribeResourceInput) (*unstructured.Unstructured, error) {
	ri, err := d.client.ResourceInterface(*gvrMatch.ToGroupVersionResource(), gvrMatch.namespaced, input.Namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource interface: %w", err)
	}

	resource, err := ri.Get(ctx, input.Name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get resource %s/%s: %w", input.Kind, input.Name, err)
	}

	return resource, nil
}

func (d *DescribeTool) formatResourceDescription(resource *unstructured.Unstructured) map[string]interface{} {
	description := map[string]interface{}{
		"name":      resource.GetName(),
		"namespace": resource.GetNamespace(),
		"kind":      resource.GetKind(),
		"labels":    resource.GetLabels(),
		"annotations": resource.GetAnnotations(),
		"creationTimestamp": resource.GetCreationTimestamp(),
		"resourceVersion": resource.GetResourceVersion(),
		"uid":       resource.GetUID(),
	}

	if spec, found, err := unstructured.NestedMap(resource.Object, "spec"); found && err == nil {
		description["spec"] = spec
	}

	if status, found, err := unstructured.NestedMap(resource.Object, "status"); found && err == nil {
		description["status"] = status
	}

	if ownerRefs := resource.GetOwnerReferences(); len(ownerRefs) > 0 {
		description["ownerReferences"] = ownerRefs
	}

	if finalizers := resource.GetFinalizers(); len(finalizers) > 0 {
		description["finalizers"] = finalizers
	}

	return description
}

func parseAndValidateDescribeParams(args map[string]any) (*DescribeResourceInput, error) {
	input := &DescribeResourceInput{}

	if kindVal, ok := args["kind"].(string); ok && kindVal != "" {
		input.Kind = kindVal
	} else {
		return nil, errors.New("kind must be provided and be a string")
	}

	if nameVal, ok := args["name"].(string); ok && nameVal != "" {
		input.Name = nameVal
	} else {
		return nil, errors.New("name must be provided and be a string")
	}

	if ns, ok := args["namespace"].(string); ok {
		input.Namespace = ns
	}
	if input.Namespace == "" {
		input.Namespace = metav1.NamespaceAll
	}

	return input, nil
}