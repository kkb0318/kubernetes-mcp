package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/kkb0318/kubernetes-mcp/src/validation"
	"github.com/mark3labs/mcp-go/mcp"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
)

// ListEventsInput represents the input parameters for listing Kubernetes events.
type ListEventsInput struct {
	Namespace      string `json:"namespace,omitempty"`
	Object         string `json:"object,omitempty"`
	EventType      string `json:"eventType,omitempty"`
	Reason         string `json:"reason,omitempty"`
	Since          string `json:"since,omitempty"`
	SinceTime      string `json:"sinceTime,omitempty"`
	Limit          int64  `json:"limit,omitempty"`
	TimeoutSeconds int64  `json:"timeoutSeconds,omitempty"`
}

// EventInfo represents formatted event information for better readability.
type EventInfo struct {
	FirstTimestamp metav1.Time `json:"firstTimestamp"`
	LastTimestamp  metav1.Time `json:"lastTimestamp"`
	Count          int32       `json:"count"`
	Type           string      `json:"type"`
	Reason         string      `json:"reason"`
	Object         string      `json:"object"`
	Message        string      `json:"message"`
	Source         string      `json:"source,omitempty"`
	Namespace      string      `json:"namespace,omitempty"`
}

// ListEventsTool provides functionality to list Kubernetes events with advanced filtering.
type ListEventsTool struct {
	client Client
}

// NewListEventsTool creates a new ListEventsTool instance with the provided Kubernetes client.
func NewListEventsTool(client Client) *ListEventsTool {
	return &ListEventsTool{client: client}
}

// Tool returns the MCP tool definition for listing Kubernetes events.
func (l *ListEventsTool) Tool() mcp.Tool {
	return mcp.NewTool("list_events",
		mcp.WithDescription("List Kubernetes events with advanced filtering options for debugging and monitoring"),
		mcp.WithString("namespace",
			mcp.Description("Kubernetes namespace to list events from (leave empty for all namespaces, use 'default' for default namespace)"),
		),
		mcp.WithString("object",
			mcp.Description("Filter events by the name of the Kubernetes object (e.g., pod name, deployment name)"),
		),
		mcp.WithString("eventType",
			mcp.Description("Filter by event type: 'Normal' or 'Warning' (case-insensitive)"),
		),
		mcp.WithString("reason",
			mcp.Description("Filter by event reason (e.g., 'Pulled', 'Failed', 'FailedScheduling', 'Killing')"),
		),
		mcp.WithString("since",
			mcp.Description("Return events newer than a relative duration like '5s', '2m', '1h', '24h' (optional)"),
		),
		mcp.WithString("sinceTime",
			mcp.Description("Return events after a specific time (RFC3339 format, e.g., 2025-06-20T10:00:00Z) (optional)"),
		),
		mcp.WithNumber("limit",
			mcp.Description("Maximum number of events to return (default: 100, use 0 for no limit)"),
		),
		mcp.WithNumber("timeoutSeconds",
			mcp.Description("Timeout for the list operation in seconds (default: 30)"),
		),
	)
}

// Handler processes requests to list Kubernetes events with filtering options.
func (l *ListEventsTool) Handler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	input, err := l.parseAndValidateEventsParams(req.Params.Arguments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse and validate events params: %w", err)
	}

	clientset, err := l.client.Clientset()
	if err != nil {
		return nil, fmt.Errorf("failed to get clientset: %w", err)
	}

	listOptions := l.buildListOptions(input)

	var eventList *corev1.EventList
	if input.Namespace == "" {
		// List events from all namespaces
		eventList, err = clientset.CoreV1().Events("").List(ctx, listOptions)
	} else {
		// List events from specific namespace
		eventList, err = clientset.CoreV1().Events(input.Namespace).List(ctx, listOptions)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to list events: %w", err)
	}

	// Filter events based on input parameters
	filteredEvents := l.filterEvents(eventList.Items, input)

	// Convert to EventInfo format for better readability
	eventInfos := l.convertToEventInfos(filteredEvents)

	result := map[string]any{
		"events":    eventInfos,
		"total":     len(eventInfos),
		"namespace": input.Namespace,
		"filters": map[string]any{
			"object":         input.Object,
			"eventType":      input.EventType,
			"reason":         input.Reason,
			"since":          input.Since,
			"sinceTime":      input.SinceTime,
		},
	}

	out, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal events: %w", err)
	}

	return mcp.NewToolResultText(string(out)), nil
}

// buildListOptions creates metav1.ListOptions from the input parameters.
func (l *ListEventsTool) buildListOptions(input *ListEventsInput) metav1.ListOptions {
	listOptions := metav1.ListOptions{}

	// Build field selector for object if specified
	if input.Object != "" {
		listOptions.FieldSelector = fields.OneTermEqualSelector("involvedObject.name", input.Object).String()
	}

	// Set limit
	listOptions.Limit = input.Limit

	// Set timeout
	listOptions.TimeoutSeconds = &input.TimeoutSeconds

	return listOptions
}

// filterEvents applies additional filtering based on input parameters.
func (l *ListEventsTool) filterEvents(events []corev1.Event, input *ListEventsInput) []corev1.Event {
	var filteredEvents []corev1.Event

	for _, event := range events {
		// Filter by event type if specified
		if input.EventType != "" {
			if !strings.EqualFold(event.Type, input.EventType) {
				continue
			}
		}

		// Filter by reason if specified
		if input.Reason != "" {
			if !strings.Contains(strings.ToLower(event.Reason), strings.ToLower(input.Reason)) {
				continue
			}
		}

		// Filter by time if specified
		if !l.isEventWithinTimeRange(&event, input) {
			continue
		}

		filteredEvents = append(filteredEvents, event)
	}

	return filteredEvents
}

// isEventWithinTimeRange checks if the event falls within the specified time range.
func (l *ListEventsTool) isEventWithinTimeRange(event *corev1.Event, input *ListEventsInput) bool {
	var cutoffTime time.Time
	var err error

	// Parse since duration
	if input.Since != "" {
		duration, parseErr := time.ParseDuration(input.Since)
		if parseErr == nil {
			cutoffTime = time.Now().Add(-duration)
		}
	}

	// Parse sinceTime (overrides since duration if both specified)
	if input.SinceTime != "" {
		cutoffTime, err = time.Parse(time.RFC3339, input.SinceTime)
		if err != nil {
			// If parsing fails, ignore the time filter
			return true
		}
	}

	// If no time filter specified, include all events
	if cutoffTime.IsZero() {
		return true
	}

	// Check if event's last timestamp is after the cutoff time
	return event.LastTimestamp.Time.After(cutoffTime)
}

// convertToEventInfos converts raw events to formatted EventInfo structs.
func (l *ListEventsTool) convertToEventInfos(events []corev1.Event) []EventInfo {
	var eventInfos []EventInfo

	for _, event := range events {
		eventInfo := EventInfo{
			FirstTimestamp: event.FirstTimestamp,
			LastTimestamp:  event.LastTimestamp,
			Count:          event.Count,
			Type:           event.Type,
			Reason:         event.Reason,
			Message:        event.Message,
			Namespace:      event.Namespace,
		}

		// Format involved object information
		if event.InvolvedObject.Kind != "" && event.InvolvedObject.Name != "" {
			eventInfo.Object = fmt.Sprintf("%s/%s", event.InvolvedObject.Kind, event.InvolvedObject.Name)
			if event.InvolvedObject.Namespace != "" && event.InvolvedObject.Namespace != event.Namespace {
				eventInfo.Object = fmt.Sprintf("%s/%s/%s", event.InvolvedObject.Namespace, event.InvolvedObject.Kind, event.InvolvedObject.Name)
			}
		}

		// Format source information
		if event.Source.Component != "" {
			eventInfo.Source = event.Source.Component
			if event.Source.Host != "" {
				eventInfo.Source = fmt.Sprintf("%s (%s)", event.Source.Component, event.Source.Host)
			}
		}

		eventInfos = append(eventInfos, eventInfo)
	}

	return eventInfos
}

// parseAndValidateEventsParams validates and extracts parameters from request arguments.
func (l *ListEventsTool) parseAndValidateEventsParams(args map[string]any) (*ListEventsInput, error) {
	input := &ListEventsInput{}

	if ns, ok := args["namespace"].(string); ok && ns != "" {
		input.Namespace = ns
		if err := validation.ValidateNamespace(input.Namespace); err != nil {
			return nil, fmt.Errorf("invalid namespace: %w", err)
		}
	}

	if obj, ok := args["object"].(string); ok && obj != "" {
		input.Object = obj
		if err := validation.ValidateResourceName(input.Object); err != nil {
			return nil, fmt.Errorf("invalid object: %w", err)
		}
	}

	if eventType, ok := args["eventType"].(string); ok && eventType != "" {
		input.EventType = eventType
		if !strings.EqualFold(eventType, "Normal") && !strings.EqualFold(eventType, "Warning") {
			return nil, fmt.Errorf("invalid eventType: must be 'Normal' or 'Warning' (case-insensitive)")
		}
	}

	if reason, ok := args["reason"].(string); ok && reason != "" {
		input.Reason = reason
	}

	if since, ok := args["since"].(string); ok && since != "" {
		input.Since = since
		if _, err := time.ParseDuration(since); err != nil {
			return nil, fmt.Errorf("invalid since duration format: %w", err)
		}
	}

	if sinceTime, ok := args["sinceTime"].(string); ok && sinceTime != "" {
		input.SinceTime = sinceTime
		if _, err := time.Parse(time.RFC3339, sinceTime); err != nil {
			return nil, fmt.Errorf("invalid sinceTime format (expected RFC3339): %w", err)
		}
	}

	if limit, ok := args["limit"].(float64); ok && limit >= 0 {
		input.Limit = int64(limit)
	} else {
		input.Limit = 100
	}

	if timeoutSeconds, ok := args["timeoutSeconds"].(float64); ok && timeoutSeconds > 0 {
		input.TimeoutSeconds = int64(timeoutSeconds)
	} else {
		input.TimeoutSeconds = 30
	}

	return input, nil
}
