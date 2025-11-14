package handlers

// The CategoryNode, PolicyListItem, PolicyDetail, and ElementInfo structs represent HTTP
// responses and are shared across multiple handlers, so they're defined in a separate file.

// CategoryNode represents a category tree node.
type CategoryNode struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Children    []*CategoryNode `json:"children"`
	PolicyCount int             `json:"policyCount"`
}

// PolicyListItem represents a summary of a policy under a category.
type PolicyListItem struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	State       string `json:"state"`
	Section     string `json:"section"`
}

// PolicyDetail contains details for a single policy.
type PolicyDetail struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Section     string        `json:"section"`
	State       string        `json:"state"`
	Elements    []ElementInfo `json:"elements"`
	RegistryKey string        `json:"registryKey"`
}

// ElementInfo contains metadata for elements within a policy.
type ElementInfo struct {
	ID           string                 `json:"id"`
	Type         string                 `json:"type"`
	Label        string                 `json:"label"`
	Required     bool                   `json:"required"`
	DefaultValue interface{}            `json:"defaultValue,omitempty"`
	Options      []EnumOptionInfo       `json:"options,omitempty"`
	MinValue     *uint32                `json:"minValue,omitempty"`
	MaxValue     *uint32                `json:"maxValue,omitempty"`
	MaxLength    *int                   `json:"maxLength,omitempty"`
	Description  string                 `json:"description,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// EnumOptionInfo represents enum options.
type EnumOptionInfo struct {
	Index       int    `json:"index"`
	DisplayName string `json:"displayName"`
}
