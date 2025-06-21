package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/kkb0318/kubernetes-mcp/src/validation"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type KubectlLogsInput struct {
	Name       string `json:"name"`
	Namespace  string `json:"namespace"`
	Container  string `json:"container,omitempty"`
	Tail       int64  `json:"tail,omitempty"`
	Since      string `json:"since,omitempty"`
	SinceTime  string `json:"sinceTime,omitempty"`
	Timestamps bool   `json:"timestamps,omitempty"`
	Previous   bool   `json:"previous,omitempty"`
}

// LogTool handles fetching logs based on the input parameters.
type LogTool struct {
	client Client
}

// NewLogTool creates a new LogTool with the provided Kubernetes client.
func NewLogTool(client Client) *LogTool {
	return &LogTool{
		client: client,
	}
}

// Tool returns the MCP tool definition for fetching pod logs.
func (l *LogTool) Tool() mcp.Tool {
	return mcp.NewTool("get_pod_logs",
		mcp.WithDescription("Get logs from a Kubernetes pod with various filtering options"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the pod to get logs from"),
		),
		mcp.WithString("namespace",
			mcp.Description("Kubernetes namespace of the pod (defaults to 'default' if not specified)"),
		),
		mcp.WithString("container",
			mcp.Description("Container name within the pod (optional)"),
		),
		mcp.WithNumber("tail",
			mcp.Description("Number of lines to show from the end of the logs (defaults to 100 if not specified, use 0 for all logs)"),
		),
		mcp.WithString("since",
			mcp.Description("Return logs newer than a relative duration like 5s, 2m, or 3h (optional)"),
		),
		mcp.WithString("sinceTime",
			mcp.Description("Return logs after a specific time (RFC3339 format, e.g., 2025-06-20T10:00:00Z) (optional)"),
		),
		mcp.WithBoolean("timestamps",
			mcp.Description("Include timestamps in the log output (optional)"),
		),
		mcp.WithBoolean("previous",
			mcp.Description("Get logs from the previous container instance if it crashed (optional)"),
		),
	)
}

// Handler fetches logs based on the provided request parameters.
func (l *LogTool) Handler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	input, err := l.parseAndValidateLogsParams(req.Params.Arguments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse and validate list params: %w", err)
	}

	clientset, err := l.client.Clientset()
	if err != nil {
		return nil, fmt.Errorf("failed to get clientset: %w", err)
	}

	// First, get the pod to check its status
	pod, err := clientset.CoreV1().Pods(input.Namespace).Get(ctx, input.Name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get pod %s/%s: %w", input.Namespace, input.Name, err)
	}

	logs := make(map[string]any)
	logs["podStatus"] = map[string]any{
		"phase":   pod.Status.Phase,
		"reason":  pod.Status.Reason,
		"message": pod.Status.Message,
	}

	// Check container statuses
	containerStatuses := make([]map[string]any, 0)
	for _, containerStatus := range pod.Status.ContainerStatuses {
		status := map[string]any{
			"name":         containerStatus.Name,
			"ready":        containerStatus.Ready,
			"restartCount": containerStatus.RestartCount,
		}

		if containerStatus.State.Waiting != nil {
			status["state"] = "waiting"
			status["reason"] = containerStatus.State.Waiting.Reason
			status["message"] = containerStatus.State.Waiting.Message
		} else if containerStatus.State.Running != nil {
			status["state"] = "running"
			status["startedAt"] = containerStatus.State.Running.StartedAt
		} else if containerStatus.State.Terminated != nil {
			status["state"] = "terminated"
			status["reason"] = containerStatus.State.Terminated.Reason
			status["message"] = containerStatus.State.Terminated.Message
			status["exitCode"] = containerStatus.State.Terminated.ExitCode
		}

		containerStatuses = append(containerStatuses, status)
	}
	logs["containerStatuses"] = containerStatuses

	// Try to get current logs
	logOptions := &corev1.PodLogOptions{
		Container:    input.Container,
		SinceSeconds: sinceSeconds(input.Since),
		SinceTime:    sinceTime(input.SinceTime),
		Timestamps:   input.Timestamps,
		Previous:     input.Previous,
	}

	// Only set TailLines if it's greater than 0
	if input.Tail > 0 {
		logOptions.TailLines = &input.Tail
	}

	podLogs := clientset.CoreV1().Pods(input.Namespace).GetLogs(input.Name, logOptions)
	podLogString, err := podLogs.Stream(ctx)
	if err != nil {
		// If getting current logs fails and we haven't tried previous logs, try previous
		if !input.Previous {
			logOptions.Previous = true
			// Ensure TailLines is set for previous logs too
			if input.Tail > 0 {
				logOptions.TailLines = &input.Tail
			}
			podLogs = clientset.CoreV1().Pods(input.Namespace).GetLogs(input.Name, logOptions)
			podLogString, err = podLogs.Stream(ctx)
			if err != nil {
				logs["error"] = fmt.Sprintf("failed to get both current and previous logs: %v", err)
				logs["logs"] = ""
			} else {
				defer podLogString.Close()
				logBytes, readErr := io.ReadAll(podLogString)
				if readErr != nil {
					logs["error"] = fmt.Sprintf("failed to read previous logs: %v", readErr)
					logs["logs"] = ""
				} else {
					logs["logs"] = string(logBytes)
					logs["source"] = "previous"
				}
			}
		} else {
			logs["error"] = fmt.Sprintf("failed to stream pod logs: %v", err)
			logs["logs"] = ""
		}
	} else {
		defer podLogString.Close()
		logBytes, readErr := io.ReadAll(podLogString)
		if readErr != nil {
			logs["error"] = fmt.Sprintf("failed to read pod logs: %v", readErr)
			logs["logs"] = ""
		} else {
			logs["logs"] = string(logBytes)
			logs["source"] = "current"
		}
	}

	out, err := json.Marshal(logs)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal logs: %w", err)
	}

	return mcp.NewToolResultText(string(out)), nil
}

// sinceSeconds parses the 'since' duration string into seconds.
func sinceSeconds(since string) *int64 {
	if since == "" {
		return nil
	}
	duration, err := time.ParseDuration(since)
	if err != nil {
		return nil
	}
	seconds := int64(duration.Seconds())
	return &seconds
}

// sinceTime parses the 'sinceTime' string into metav1.Time.
func sinceTime(sinceTime string) *metav1.Time {
	if sinceTime == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339, sinceTime)
	if err != nil {
		return nil
	}
	return &metav1.Time{Time: t}
}

// parseAndValidateLogsParams validates and parses the input parameters.
func (l *LogTool) parseAndValidateLogsParams(args map[string]any) (*KubectlLogsInput, error) {
	input := &KubectlLogsInput{}

	if name, ok := args["name"]; ok && name != nil {
		input.Name = name.(string)
		if err := validation.ValidateResourceName(input.Name); err != nil {
			return nil, fmt.Errorf("invalid pod name: %w", err)
		}
	}

	if namespace, ok := args["namespace"]; ok && namespace != nil {
		input.Namespace = namespace.(string)
		if err := validation.ValidateNamespace(input.Namespace); err != nil {
			return nil, fmt.Errorf("invalid namespace: %w", err)
		}
	}

	if container, ok := args["container"]; ok && container != nil {
		input.Container = container.(string)
	}

	if tail, ok := args["tail"]; ok && tail != nil {
		input.Tail = int64(tail.(float64))
	} else {
		// Default to 100 lines if not specified
		input.Tail = 100
	}

	if since, ok := args["since"]; ok && since != nil {
		input.Since = since.(string)
	}

	if sinceTime, ok := args["sinceTime"]; ok && sinceTime != nil {
		input.SinceTime = sinceTime.(string)
	}

	if timestamps, ok := args["timestamps"]; ok && timestamps != nil {
		input.Timestamps = timestamps.(bool)
	}

	if previous, ok := args["previous"]; ok && previous != nil {
		input.Previous = previous.(bool)
	}

	if input.Namespace == "" {
		input.Namespace = metav1.NamespaceDefault
	}

	if input.Name == "" {
		return nil, fmt.Errorf("name must be provided")
	}

	return input, nil
}
