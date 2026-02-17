package models

// Page represents a paginated list response.
type Page[T any] struct {
	Items []*T `json:"items"`
	Total int  `json:"total"`
}

// ListOptions configures list/search operations.
type ListOptions struct {
	Limit      int               `json:"limit,omitempty"`
	Offset     int               `json:"offset,omitempty"`
	Sort       string            `json:"sort,omitempty"`
	Order      string            `json:"order,omitempty"`
	Search     string            `json:"search,omitempty"`
	Filters    map[string]string `json:"filters,omitempty"`
	WorkflowID string            `json:"workflow_id,omitempty"`
}
