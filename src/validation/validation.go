package validation

import (
	"fmt"
	"regexp"
	"strings"
)

// ValidateResourceName validates Kubernetes resource names
func ValidateResourceName(name string) error {
	if name == "" {
		return fmt.Errorf("resource name cannot be empty")
	}
	
	// Kubernetes resource names must follow DNS subdomain naming conventions
	if len(name) > 253 {
		return fmt.Errorf("resource name too long (max 253 characters)")
	}
	
	// Must contain only lowercase alphanumeric characters, '-', or '.'
	validName := regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$`)
	if !validName.MatchString(name) {
		return fmt.Errorf("invalid resource name: must contain only lowercase alphanumeric characters, '-', or '.'")
	}
	
	return nil
}

// ValidateNamespace validates Kubernetes namespace names
func ValidateNamespace(namespace string) error {
	if namespace == "" {
		return nil // Empty namespace is valid (uses default)
	}
	
	if len(namespace) > 63 {
		return fmt.Errorf("namespace name too long (max 63 characters)")
	}
	
	// Must contain only lowercase alphanumeric characters or '-'
	validNamespace := regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)
	if !validNamespace.MatchString(namespace) {
		return fmt.Errorf("invalid namespace name: must contain only lowercase alphanumeric characters or '-'")
	}
	
	// Reserved namespaces
	reserved := []string{"kube-system", "kube-public", "kube-node-lease"}
	for _, r := range reserved {
		if namespace == r {
			return nil // Reserved namespaces are valid
		}
	}
	
	return nil
}

// ValidateLabelSelector validates Kubernetes label selectors
func ValidateLabelSelector(selector string) error {
	if selector == "" {
		return nil // Empty selector is valid
	}
	
	// Basic validation for label selector format
	// This is a simplified check; Kubernetes has more complex rules
	parts := strings.Split(selector, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		
		// Check for basic key=value or key!=value format
		if !strings.Contains(part, "=") && !strings.Contains(part, "!=") && !strings.Contains(part, " in ") && !strings.Contains(part, " notin ") {
			return fmt.Errorf("invalid label selector format: %s", part)
		}
	}
	
	return nil
}

// ValidateKind validates Kubernetes resource kinds
func ValidateKind(kind string) error {
	if kind == "" {
		return fmt.Errorf("resource kind cannot be empty")
	}
	
	if kind == "all" {
		return nil // Special case for discovery
	}
	
	// Allow both uppercase (Kind) and lowercase (resource names) formats
	// Must contain only alphanumeric characters and start with letter
	validKind := regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9]*$`)
	if !validKind.MatchString(kind) {
		return fmt.Errorf("invalid resource kind: must start with letter and contain only alphanumeric characters")
	}
	
	return nil
}