package models

import (
	"fmt"
	"time"
)

// LLMProviderType defines supported LLM providers for rental keys
type LLMProviderType string

const (
	LLMProviderTypeOpenAI    LLMProviderType = "openai"
	LLMProviderTypeAnthropic LLMProviderType = "anthropic"
	LLMProviderTypeGoogleAI  LLMProviderType = "google_ai"
)

// ValidLLMProviderTypes returns all valid LLM provider types
func ValidLLMProviderTypes() []LLMProviderType {
	return []LLMProviderType{
		LLMProviderTypeOpenAI,
		LLMProviderTypeAnthropic,
		LLMProviderTypeGoogleAI,
	}
}

// IsValidLLMProviderType checks if the given provider type is valid
func IsValidLLMProviderType(t LLMProviderType) bool {
	for _, valid := range ValidLLMProviderTypes() {
		if t == valid {
			return true
		}
	}
	return false
}

// ProvisionerType defines how the rental key was created
type ProvisionerType string

const (
	ProvisionerTypeManual        ProvisionerType = "manual"
	ProvisionerTypeAutoOpenAI    ProvisionerType = "auto_openai"
	ProvisionerTypeAutoAnthropic ProvisionerType = "auto_anthropic"
	ProvisionerTypeAutoGoogle    ProvisionerType = "auto_google"
)

// ValidProvisionerTypes returns all valid provisioner types
func ValidProvisionerTypes() []ProvisionerType {
	return []ProvisionerType{
		ProvisionerTypeManual,
		ProvisionerTypeAutoOpenAI,
		ProvisionerTypeAutoAnthropic,
		ProvisionerTypeAutoGoogle,
	}
}

// IsValidProvisionerType checks if the given provisioner type is valid
func IsValidProvisionerType(t ProvisionerType) bool {
	for _, valid := range ValidProvisionerTypes() {
		if t == valid {
			return true
		}
	}
	return false
}

// MultimodalUsage tracks token usage across different modalities
type MultimodalUsage struct {
	// Text tokens
	PromptTokens     int64 `json:"prompt_tokens"`
	CompletionTokens int64 `json:"completion_tokens"`

	// Image tokens
	ImageInputTokens  int64 `json:"image_input_tokens"`
	ImageOutputTokens int64 `json:"image_output_tokens"`

	// Audio tokens
	AudioInputTokens  int64 `json:"audio_input_tokens"`
	AudioOutputTokens int64 `json:"audio_output_tokens"`

	// Video tokens
	VideoInputTokens  int64 `json:"video_input_tokens"`
	VideoOutputTokens int64 `json:"video_output_tokens"`
}

// TotalTokens returns the sum of all token types
func (u *MultimodalUsage) TotalTokens() int64 {
	return u.PromptTokens + u.CompletionTokens +
		u.ImageInputTokens + u.ImageOutputTokens +
		u.AudioInputTokens + u.AudioOutputTokens +
		u.VideoInputTokens + u.VideoOutputTokens
}

// TotalInputTokens returns the sum of all input tokens
func (u *MultimodalUsage) TotalInputTokens() int64 {
	return u.PromptTokens + u.ImageInputTokens + u.AudioInputTokens + u.VideoInputTokens
}

// TotalOutputTokens returns the sum of all output tokens
func (u *MultimodalUsage) TotalOutputTokens() int64 {
	return u.CompletionTokens + u.ImageOutputTokens + u.AudioOutputTokens + u.VideoOutputTokens
}

// Add adds another MultimodalUsage to this one
func (u *MultimodalUsage) Add(other MultimodalUsage) {
	u.PromptTokens += other.PromptTokens
	u.CompletionTokens += other.CompletionTokens
	u.ImageInputTokens += other.ImageInputTokens
	u.ImageOutputTokens += other.ImageOutputTokens
	u.AudioInputTokens += other.AudioInputTokens
	u.AudioOutputTokens += other.AudioOutputTokens
	u.VideoInputTokens += other.VideoInputTokens
	u.VideoOutputTokens += other.VideoOutputTokens
}

// IsEmpty returns true if all token counts are zero
func (u *MultimodalUsage) IsEmpty() bool {
	return u.TotalTokens() == 0
}

// RentalKeyResource represents a rental API key for LLM providers
// The actual API key value is NEVER exposed to users - only used internally by the system
type RentalKeyResource struct {
	BaseResource

	// Provider information
	Provider       LLMProviderType        `json:"provider"`
	ProviderConfig map[string]interface{} `json:"provider_config,omitempty"`

	// Encrypted API key - NEVER serialized to JSON, never exposed via API
	EncryptedAPIKey string `json:"-"`

	// Usage limits (nil means unlimited)
	DailyRequestLimit *int   `json:"daily_request_limit,omitempty"`
	MonthlyTokenLimit *int64 `json:"monthly_token_limit,omitempty"`

	// Current usage counters (reset periodically)
	RequestsToday    int       `json:"requests_today"`
	TokensThisMonth  int64     `json:"tokens_this_month"`
	LastUsageResetAt time.Time `json:"last_usage_reset_at"`

	// Total usage statistics (multimodal)
	TotalRequests int64           `json:"total_requests"`
	TotalUsage    MultimodalUsage `json:"total_usage"`
	TotalCost     float64         `json:"total_cost"`

	// Timestamps
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`

	// Billing and management
	PricingPlanID   string          `json:"pricing_plan_id,omitempty"`
	CreatedBy       string          `json:"created_by,omitempty"`
	ProvisionerType ProvisionerType `json:"provisioner_type"`
}

// NewRentalKeyResource creates a new rental key resource
func NewRentalKeyResource(ownerID, name string, provider LLMProviderType) *RentalKeyResource {
	now := time.Now()
	return &RentalKeyResource{
		BaseResource: BaseResource{
			Type:      ResourceTypeRentalKey,
			OwnerID:   ownerID,
			Name:      name,
			Status:    ResourceStatusActive,
			Metadata:  make(map[string]interface{}),
			CreatedAt: now,
			UpdatedAt: now,
		},
		Provider:         provider,
		ProviderConfig:   make(map[string]interface{}),
		ProvisionerType:  ProvisionerTypeManual,
		LastUsageResetAt: now,
		TotalUsage:       MultimodalUsage{},
	}
}

// Validate validates the rental key resource
func (r *RentalKeyResource) Validate() error {
	if err := r.BaseResource.Validate(); err != nil {
		return err
	}

	if !IsValidLLMProviderType(r.Provider) {
		return &ValidationError{
			Field:   "provider",
			Message: fmt.Sprintf("invalid provider type: %s", r.Provider),
		}
	}

	if !IsValidProvisionerType(r.ProvisionerType) {
		return &ValidationError{
			Field:   "provisioner_type",
			Message: fmt.Sprintf("invalid provisioner type: %s", r.ProvisionerType),
		}
	}

	// Note: EncryptedAPIKey validation happens at repository level
	// because it should be set during creation with encryption

	return nil
}

// RecordUsage records a single LLM usage event
func (r *RentalKeyResource) RecordUsage(usage MultimodalUsage, cost float64) {
	now := time.Now()

	r.RequestsToday++
	r.TokensThisMonth += usage.TotalTokens()
	r.TotalRequests++
	r.TotalUsage.Add(usage)
	r.TotalCost += cost
	r.LastUsedAt = &now
	r.UpdatedAt = now
}

// CheckLimits checks if usage limits are exceeded
func (r *RentalKeyResource) CheckLimits() error {
	if r.DailyRequestLimit != nil && r.RequestsToday >= *r.DailyRequestLimit {
		return ErrDailyLimitExceeded
	}
	if r.MonthlyTokenLimit != nil && r.TokensThisMonth >= *r.MonthlyTokenLimit {
		return ErrMonthlyTokenLimitExceeded
	}
	return nil
}

// CanMakeRequest checks if a request can be made (status and limits)
func (r *RentalKeyResource) CanMakeRequest() error {
	if !r.IsActive() {
		return ErrRentalKeySuspended
	}
	return r.CheckLimits()
}

// ResetDailyUsage resets the daily request counter
func (r *RentalKeyResource) ResetDailyUsage() {
	r.RequestsToday = 0
}

// ResetMonthlyUsage resets the monthly token counter
func (r *RentalKeyResource) ResetMonthlyUsage() {
	r.TokensThisMonth = 0
	r.LastUsageResetAt = time.Now()
}

// GetDailyUsagePercent returns the daily usage percentage (0-100)
func (r *RentalKeyResource) GetDailyUsagePercent() float64 {
	if r.DailyRequestLimit == nil || *r.DailyRequestLimit <= 0 {
		return 0
	}
	return float64(r.RequestsToday) / float64(*r.DailyRequestLimit) * 100
}

// GetMonthlyUsagePercent returns the monthly usage percentage (0-100)
func (r *RentalKeyResource) GetMonthlyUsagePercent() float64 {
	if r.MonthlyTokenLimit == nil || *r.MonthlyTokenLimit <= 0 {
		return 0
	}
	return float64(r.TokensThisMonth) / float64(*r.MonthlyTokenLimit) * 100
}

// GetRemainingDailyRequests returns remaining daily requests (nil if unlimited)
func (r *RentalKeyResource) GetRemainingDailyRequests() *int {
	if r.DailyRequestLimit == nil {
		return nil
	}
	remaining := *r.DailyRequestLimit - r.RequestsToday
	if remaining < 0 {
		remaining = 0
	}
	return &remaining
}

// GetRemainingMonthlyTokens returns remaining monthly tokens (nil if unlimited)
func (r *RentalKeyResource) GetRemainingMonthlyTokens() *int64 {
	if r.MonthlyTokenLimit == nil {
		return nil
	}
	remaining := *r.MonthlyTokenLimit - r.TokensThisMonth
	if remaining < 0 {
		remaining = 0
	}
	return &remaining
}

// Suspend suspends the rental key
func (r *RentalKeyResource) Suspend() {
	r.Status = ResourceStatusSuspended
	r.UpdatedAt = time.Now()
}

// Activate activates the rental key
func (r *RentalKeyResource) Activate() {
	r.Status = ResourceStatusActive
	r.UpdatedAt = time.Now()
}

// RentalKeyUsageRecord represents a single usage record for billing and analytics
type RentalKeyUsageRecord struct {
	ID          string `json:"id"`
	RentalKeyID string `json:"rental_key_id"`
	Model       string `json:"model"`

	// Multimodal usage
	Usage MultimodalUsage `json:"usage"`

	// Cost and context
	EstimatedCost float64 `json:"estimated_cost"`
	ExecutionID   string  `json:"execution_id,omitempty"`
	WorkflowID    string  `json:"workflow_id,omitempty"`
	NodeID        string  `json:"node_id,omitempty"`

	// Status
	Status         string `json:"status"` // success, failed, rate_limited
	ErrorMessage   string `json:"error_message,omitempty"`
	ResponseTimeMs int    `json:"response_time_ms,omitempty"`

	CreatedAt time.Time `json:"created_at"`
}

// NewRentalKeyUsageRecord creates a new usage record
func NewRentalKeyUsageRecord(rentalKeyID, model string, usage MultimodalUsage) *RentalKeyUsageRecord {
	return &RentalKeyUsageRecord{
		RentalKeyID: rentalKeyID,
		Model:       model,
		Usage:       usage,
		Status:      "success",
		CreatedAt:   time.Now(),
	}
}

// SetFailed marks the usage record as failed
func (r *RentalKeyUsageRecord) SetFailed(err error) {
	r.Status = "failed"
	if err != nil {
		r.ErrorMessage = err.Error()
	}
}

// SetRateLimited marks the usage record as rate limited
func (r *RentalKeyUsageRecord) SetRateLimited(err error) {
	r.Status = "rate_limited"
	if err != nil {
		r.ErrorMessage = err.Error()
	}
}
