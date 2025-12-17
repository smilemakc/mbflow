package models

import (
	"testing"
)

func TestNewRentalKeyResource(t *testing.T) {
	ownerID := "user-123"
	name := "Test Rental Key"
	provider := LLMProviderTypeOpenAI

	key := NewRentalKeyResource(ownerID, name, provider)

	if key == nil {
		t.Fatal("expected non-nil RentalKeyResource")
	}
	if key.OwnerID != ownerID {
		t.Errorf("expected ownerID %s, got %s", ownerID, key.OwnerID)
	}
	if key.Name != name {
		t.Errorf("expected name %s, got %s", name, key.Name)
	}
	if key.Provider != provider {
		t.Errorf("expected provider %s, got %s", provider, key.Provider)
	}
	if key.Type != ResourceTypeRentalKey {
		t.Errorf("expected type %s, got %s", ResourceTypeRentalKey, key.Type)
	}
	if key.Status != ResourceStatusActive {
		t.Errorf("expected status %s, got %s", ResourceStatusActive, key.Status)
	}
	if key.ProvisionerType != ProvisionerTypeManual {
		t.Errorf("expected provisioner type %s, got %s", ProvisionerTypeManual, key.ProvisionerType)
	}
	if key.CreatedAt.IsZero() {
		t.Error("expected non-zero CreatedAt")
	}
}

func TestRentalKeyResource_Validate(t *testing.T) {
	tests := []struct {
		name    string
		key     *RentalKeyResource
		wantErr bool
	}{
		{
			name: "valid key",
			key: &RentalKeyResource{
				BaseResource: BaseResource{
					ID:      "key-123",
					Name:    "Test Key",
					OwnerID: "user-123",
				},
				Provider:        LLMProviderTypeOpenAI,
				ProvisionerType: ProvisionerTypeManual,
			},
			wantErr: false,
		},
		{
			name: "missing name",
			key: &RentalKeyResource{
				BaseResource: BaseResource{
					ID:      "key-123",
					Name:    "",
					OwnerID: "user-123",
				},
				Provider:        LLMProviderTypeOpenAI,
				ProvisionerType: ProvisionerTypeManual,
			},
			wantErr: true,
		},
		{
			name: "missing owner",
			key: &RentalKeyResource{
				BaseResource: BaseResource{
					ID:      "key-123",
					Name:    "Test Key",
					OwnerID: "",
				},
				Provider:        LLMProviderTypeOpenAI,
				ProvisionerType: ProvisionerTypeManual,
			},
			wantErr: true,
		},
		{
			name: "invalid provider",
			key: &RentalKeyResource{
				BaseResource: BaseResource{
					ID:      "key-123",
					Name:    "Test Key",
					OwnerID: "user-123",
				},
				Provider:        "invalid",
				ProvisionerType: ProvisionerTypeManual,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.key.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRentalKeyResource_CheckLimits(t *testing.T) {
	tests := []struct {
		name            string
		dailyLimit      *int
		monthlyLimit    *int64
		requestsToday   int
		tokensThisMonth int64
		wantErr         bool
		wantErrType     error
	}{
		{
			name:            "no limits set",
			dailyLimit:      nil,
			monthlyLimit:    nil,
			requestsToday:   1000,
			tokensThisMonth: 1000000,
			wantErr:         false,
		},
		{
			name:          "daily limit under",
			dailyLimit:    intPtr(100),
			requestsToday: 50,
			wantErr:       false,
		},
		{
			name:          "daily limit at",
			dailyLimit:    intPtr(100),
			requestsToday: 100,
			wantErr:       true,
			wantErrType:   ErrDailyLimitExceeded,
		},
		{
			name:          "daily limit over",
			dailyLimit:    intPtr(100),
			requestsToday: 150,
			wantErr:       true,
			wantErrType:   ErrDailyLimitExceeded,
		},
		{
			name:            "monthly limit under",
			monthlyLimit:    int64Ptr(100000),
			tokensThisMonth: 50000,
			wantErr:         false,
		},
		{
			name:            "monthly limit at",
			monthlyLimit:    int64Ptr(100000),
			tokensThisMonth: 100000,
			wantErr:         true,
			wantErrType:     ErrMonthlyTokenLimitExceeded,
		},
		{
			name:            "monthly limit over",
			monthlyLimit:    int64Ptr(100000),
			tokensThisMonth: 150000,
			wantErr:         true,
			wantErrType:     ErrMonthlyTokenLimitExceeded,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := &RentalKeyResource{
				DailyRequestLimit: tt.dailyLimit,
				MonthlyTokenLimit: tt.monthlyLimit,
				RequestsToday:     tt.requestsToday,
				TokensThisMonth:   tt.tokensThisMonth,
			}
			err := key.CheckLimits()
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckLimits() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != tt.wantErrType {
				t.Errorf("CheckLimits() error = %v, wantErrType %v", err, tt.wantErrType)
			}
		})
	}
}

func TestRentalKeyResource_RecordUsage(t *testing.T) {
	key := NewRentalKeyResource("user-123", "Test Key", LLMProviderTypeOpenAI)

	usage := MultimodalUsage{
		PromptTokens:      100,
		CompletionTokens:  50,
		ImageInputTokens:  10,
		ImageOutputTokens: 5,
	}

	key.RecordUsage(usage, 0.005)

	// Check counters incremented
	if key.RequestsToday != 1 {
		t.Errorf("expected RequestsToday 1, got %d", key.RequestsToday)
	}
	if key.TotalRequests != 1 {
		t.Errorf("expected TotalRequests 1, got %d", key.TotalRequests)
	}

	// Check token totals
	expectedTokens := int64(165) // 100 + 50 + 10 + 5
	if key.TokensThisMonth != expectedTokens {
		t.Errorf("expected TokensThisMonth %d, got %d", expectedTokens, key.TokensThisMonth)
	}

	// Check multimodal usage
	if key.TotalUsage.PromptTokens != 100 {
		t.Errorf("expected PromptTokens 100, got %d", key.TotalUsage.PromptTokens)
	}
	if key.TotalUsage.CompletionTokens != 50 {
		t.Errorf("expected CompletionTokens 50, got %d", key.TotalUsage.CompletionTokens)
	}

	// Check cost
	if key.TotalCost != 0.005 {
		t.Errorf("expected TotalCost 0.005, got %f", key.TotalCost)
	}

	// Check LastUsedAt is set
	if key.LastUsedAt == nil {
		t.Error("expected LastUsedAt to be set")
	}

	// Record another usage
	key.RecordUsage(usage, 0.003)
	if key.RequestsToday != 2 {
		t.Errorf("expected RequestsToday 2, got %d", key.RequestsToday)
	}
	if key.TotalCost != 0.008 {
		t.Errorf("expected TotalCost 0.008, got %f", key.TotalCost)
	}
}

func TestMultimodalUsage_TotalTokens(t *testing.T) {
	usage := &MultimodalUsage{
		PromptTokens:      100,
		CompletionTokens:  50,
		ImageInputTokens:  10,
		ImageOutputTokens: 5,
		AudioInputTokens:  20,
		AudioOutputTokens: 15,
		VideoInputTokens:  30,
		VideoOutputTokens: 25,
	}

	expected := int64(255) // 100+50+10+5+20+15+30+25
	total := usage.TotalTokens()

	if total != expected {
		t.Errorf("expected TotalTokens %d, got %d", expected, total)
	}
}

func TestMultimodalUsage_Add(t *testing.T) {
	usage1 := &MultimodalUsage{
		PromptTokens:     100,
		CompletionTokens: 50,
	}
	usage2 := MultimodalUsage{
		PromptTokens:      200,
		CompletionTokens:  100,
		ImageInputTokens:  10,
		ImageOutputTokens: 5,
	}

	usage1.Add(usage2)

	if usage1.PromptTokens != 300 {
		t.Errorf("expected PromptTokens 300, got %d", usage1.PromptTokens)
	}
	if usage1.CompletionTokens != 150 {
		t.Errorf("expected CompletionTokens 150, got %d", usage1.CompletionTokens)
	}
	if usage1.ImageInputTokens != 10 {
		t.Errorf("expected ImageInputTokens 10, got %d", usage1.ImageInputTokens)
	}
}

func TestRentalKeyResource_ResetDailyUsage(t *testing.T) {
	key := NewRentalKeyResource("user-123", "Test Key", LLMProviderTypeOpenAI)
	key.RequestsToday = 100

	key.ResetDailyUsage()

	if key.RequestsToday != 0 {
		t.Errorf("expected RequestsToday 0, got %d", key.RequestsToday)
	}
}

func TestRentalKeyResource_ResetMonthlyUsage(t *testing.T) {
	key := NewRentalKeyResource("user-123", "Test Key", LLMProviderTypeOpenAI)
	key.TokensThisMonth = 100000

	key.ResetMonthlyUsage()

	if key.TokensThisMonth != 0 {
		t.Errorf("expected TokensThisMonth 0, got %d", key.TokensThisMonth)
	}
}

func TestIsValidLLMProviderType(t *testing.T) {
	tests := []struct {
		provider LLMProviderType
		valid    bool
	}{
		{LLMProviderTypeOpenAI, true},
		{LLMProviderTypeAnthropic, true},
		{LLMProviderTypeGoogleAI, true},
		{"invalid", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(string(tt.provider), func(t *testing.T) {
			got := IsValidLLMProviderType(tt.provider)
			if got != tt.valid {
				t.Errorf("IsValidLLMProviderType() = %v, want %v", got, tt.valid)
			}
		})
	}
}

func TestRentalKeyResource_CanMakeRequest(t *testing.T) {
	tests := []struct {
		name    string
		key     *RentalKeyResource
		wantErr error
	}{
		{
			name: "active key with no limits",
			key: &RentalKeyResource{
				BaseResource: BaseResource{
					Status: ResourceStatusActive,
				},
			},
			wantErr: nil,
		},
		{
			name: "suspended key",
			key: &RentalKeyResource{
				BaseResource: BaseResource{
					Status: ResourceStatusSuspended,
				},
			},
			wantErr: ErrRentalKeySuspended,
		},
		{
			name: "active key at daily limit",
			key: &RentalKeyResource{
				BaseResource: BaseResource{
					Status: ResourceStatusActive,
				},
				DailyRequestLimit: intPtr(100),
				RequestsToday:     100,
			},
			wantErr: ErrDailyLimitExceeded,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.key.CanMakeRequest()
			if err != tt.wantErr {
				t.Errorf("CanMakeRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRentalKeyResource_GetUsagePercent(t *testing.T) {
	key := NewRentalKeyResource("user-123", "Test Key", LLMProviderTypeOpenAI)
	key.DailyRequestLimit = intPtr(100)
	key.MonthlyTokenLimit = int64Ptr(1000)
	key.RequestsToday = 50
	key.TokensThisMonth = 250

	dailyPercent := key.GetDailyUsagePercent()
	if dailyPercent != 50.0 {
		t.Errorf("expected daily percent 50.0, got %f", dailyPercent)
	}

	monthlyPercent := key.GetMonthlyUsagePercent()
	if monthlyPercent != 25.0 {
		t.Errorf("expected monthly percent 25.0, got %f", monthlyPercent)
	}
}

func TestNewRentalKeyUsageRecord(t *testing.T) {
	rentalKeyID := "key-123"
	model := "gpt-4"
	usage := MultimodalUsage{
		PromptTokens:     100,
		CompletionTokens: 50,
	}

	record := NewRentalKeyUsageRecord(rentalKeyID, model, usage)

	if record.RentalKeyID != rentalKeyID {
		t.Errorf("expected RentalKeyID %s, got %s", rentalKeyID, record.RentalKeyID)
	}
	if record.Model != model {
		t.Errorf("expected Model %s, got %s", model, record.Model)
	}
	if record.Status != "success" {
		t.Errorf("expected Status success, got %s", record.Status)
	}
}

// Helper functions
func intPtr(i int) *int {
	return &i
}

func int64Ptr(i int64) *int64 {
	return &i
}
