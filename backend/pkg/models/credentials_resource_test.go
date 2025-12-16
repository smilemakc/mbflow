package models

import (
	"testing"
	"time"
)

func TestNewCredentialsResource(t *testing.T) {
	cred := NewCredentialsResource("owner-123", "My API Key", CredentialTypeAPIKey)

	if cred.OwnerID != "owner-123" {
		t.Errorf("OwnerID = %q, want %q", cred.OwnerID, "owner-123")
	}
	if cred.Name != "My API Key" {
		t.Errorf("Name = %q, want %q", cred.Name, "My API Key")
	}
	if cred.CredentialType != CredentialTypeAPIKey {
		t.Errorf("CredentialType = %q, want %q", cred.CredentialType, CredentialTypeAPIKey)
	}
	if cred.Status != ResourceStatusActive {
		t.Errorf("Status = %q, want %q", cred.Status, ResourceStatusActive)
	}
	if cred.Type != ResourceTypeCredentials {
		t.Errorf("Type = %q, want %q", cred.Type, ResourceTypeCredentials)
	}
	if cred.UsageCount != 0 {
		t.Errorf("UsageCount = %d, want 0", cred.UsageCount)
	}
}

func TestIsValidCredentialType(t *testing.T) {
	tests := []struct {
		credType CredentialType
		want     bool
	}{
		{CredentialTypeAPIKey, true},
		{CredentialTypeBasicAuth, true},
		{CredentialTypeOAuth2, true},
		{CredentialTypeServiceAccount, true},
		{CredentialTypeCustom, true},
		{CredentialType("invalid"), false},
		{CredentialType(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.credType), func(t *testing.T) {
			if got := IsValidCredentialType(tt.credType); got != tt.want {
				t.Errorf("IsValidCredentialType(%q) = %v, want %v", tt.credType, got, tt.want)
			}
		})
	}
}

func TestCredentialsResource_Validate(t *testing.T) {
	tests := []struct {
		name    string
		setup   func() *CredentialsResource
		wantErr bool
	}{
		{
			name: "valid api_key",
			setup: func() *CredentialsResource {
				c := NewCredentialsResource("owner", "name", CredentialTypeAPIKey)
				c.EncryptedData["api_key"] = "encrypted-value"
				return c
			},
			wantErr: false,
		},
		{
			name: "api_key missing key",
			setup: func() *CredentialsResource {
				return NewCredentialsResource("owner", "name", CredentialTypeAPIKey)
			},
			wantErr: true,
		},
		{
			name: "valid basic_auth",
			setup: func() *CredentialsResource {
				c := NewCredentialsResource("owner", "name", CredentialTypeBasicAuth)
				c.EncryptedData["username"] = "encrypted-user"
				c.EncryptedData["password"] = "encrypted-pass"
				return c
			},
			wantErr: false,
		},
		{
			name: "basic_auth missing password",
			setup: func() *CredentialsResource {
				c := NewCredentialsResource("owner", "name", CredentialTypeBasicAuth)
				c.EncryptedData["username"] = "encrypted-user"
				return c
			},
			wantErr: true,
		},
		{
			name: "valid oauth2",
			setup: func() *CredentialsResource {
				c := NewCredentialsResource("owner", "name", CredentialTypeOAuth2)
				c.EncryptedData["client_id"] = "encrypted-id"
				c.EncryptedData["client_secret"] = "encrypted-secret"
				return c
			},
			wantErr: false,
		},
		{
			name: "valid service_account",
			setup: func() *CredentialsResource {
				c := NewCredentialsResource("owner", "name", CredentialTypeServiceAccount)
				c.EncryptedData["json_key"] = "encrypted-json"
				return c
			},
			wantErr: false,
		},
		{
			name: "valid custom",
			setup: func() *CredentialsResource {
				c := NewCredentialsResource("owner", "name", CredentialTypeCustom)
				c.EncryptedData["custom_field"] = "encrypted-value"
				return c
			},
			wantErr: false,
		},
		{
			name: "custom empty data",
			setup: func() *CredentialsResource {
				return NewCredentialsResource("owner", "name", CredentialTypeCustom)
			},
			wantErr: true,
		},
		{
			name: "invalid type",
			setup: func() *CredentialsResource {
				c := NewCredentialsResource("owner", "name", CredentialType("invalid"))
				return c
			},
			wantErr: true,
		},
		{
			name: "missing owner",
			setup: func() *CredentialsResource {
				c := NewCredentialsResource("", "name", CredentialTypeAPIKey)
				c.EncryptedData["api_key"] = "value"
				return c
			},
			wantErr: true,
		},
		{
			name: "missing name",
			setup: func() *CredentialsResource {
				c := NewCredentialsResource("owner", "", CredentialTypeAPIKey)
				c.EncryptedData["api_key"] = "value"
				return c
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := tt.setup()
			err := c.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCredentialsResource_IsExpired(t *testing.T) {
	c := NewCredentialsResource("owner", "name", CredentialTypeAPIKey)

	// No expiration
	if c.IsExpired() {
		t.Error("IsExpired() = true, want false (no expiration set)")
	}

	// Future expiration
	future := time.Now().Add(time.Hour)
	c.ExpiresAt = &future
	if c.IsExpired() {
		t.Error("IsExpired() = true, want false (future expiration)")
	}

	// Past expiration
	past := time.Now().Add(-time.Hour)
	c.ExpiresAt = &past
	if !c.IsExpired() {
		t.Error("IsExpired() = false, want true (past expiration)")
	}
}

func TestCredentialsResource_IncrementUsage(t *testing.T) {
	c := NewCredentialsResource("owner", "name", CredentialTypeAPIKey)

	if c.UsageCount != 0 {
		t.Errorf("Initial UsageCount = %d, want 0", c.UsageCount)
	}
	if c.LastUsedAt != nil {
		t.Error("Initial LastUsedAt should be nil")
	}

	c.IncrementUsage()

	if c.UsageCount != 1 {
		t.Errorf("UsageCount after increment = %d, want 1", c.UsageCount)
	}
	if c.LastUsedAt == nil {
		t.Error("LastUsedAt should be set after increment")
	}

	c.IncrementUsage()
	if c.UsageCount != 2 {
		t.Errorf("UsageCount after second increment = %d, want 2", c.UsageCount)
	}
}

func TestCredentialsResource_GetAPIKey(t *testing.T) {
	c := NewCredentialsResource("owner", "name", CredentialTypeAPIKey)
	c.DecryptedData = map[string]string{"api_key": "my-secret-key"}

	if got := c.GetAPIKey(); got != "my-secret-key" {
		t.Errorf("GetAPIKey() = %q, want %q", got, "my-secret-key")
	}

	// Wrong type
	c.CredentialType = CredentialTypeBasicAuth
	if got := c.GetAPIKey(); got != "" {
		t.Errorf("GetAPIKey() with wrong type = %q, want empty", got)
	}
}

func TestCredentialsResource_GetBasicAuth(t *testing.T) {
	c := NewCredentialsResource("owner", "name", CredentialTypeBasicAuth)
	c.DecryptedData = map[string]string{
		"username": "myuser",
		"password": "mypass",
	}

	username, password := c.GetBasicAuth()
	if username != "myuser" {
		t.Errorf("GetBasicAuth() username = %q, want %q", username, "myuser")
	}
	if password != "mypass" {
		t.Errorf("GetBasicAuth() password = %q, want %q", password, "mypass")
	}

	// Wrong type
	c.CredentialType = CredentialTypeAPIKey
	username, password = c.GetBasicAuth()
	if username != "" || password != "" {
		t.Error("GetBasicAuth() with wrong type should return empty strings")
	}
}

func TestCredentialsResource_GetOAuth2(t *testing.T) {
	c := NewCredentialsResource("owner", "name", CredentialTypeOAuth2)
	c.DecryptedData = map[string]string{
		"client_id":     "my-client-id",
		"client_secret": "my-client-secret",
		"access_token":  "access-token",
		"refresh_token": "refresh-token",
	}

	oauth := c.GetOAuth2()
	if oauth == nil {
		t.Fatal("GetOAuth2() returned nil")
	}
	if oauth.ClientID != "my-client-id" {
		t.Errorf("ClientID = %q, want %q", oauth.ClientID, "my-client-id")
	}
	if oauth.ClientSecret != "my-client-secret" {
		t.Errorf("ClientSecret = %q, want %q", oauth.ClientSecret, "my-client-secret")
	}
}

func TestCredentialsResource_GetServiceAccountJSON(t *testing.T) {
	c := NewCredentialsResource("owner", "name", CredentialTypeServiceAccount)
	jsonKey := `{"type":"service_account","project_id":"test"}`
	c.DecryptedData = map[string]string{"json_key": jsonKey}

	if got := c.GetServiceAccountJSON(); got != jsonKey {
		t.Errorf("GetServiceAccountJSON() = %q, want %q", got, jsonKey)
	}
}

func TestCredentialsResource_GetCustomValue(t *testing.T) {
	c := NewCredentialsResource("owner", "name", CredentialTypeCustom)
	c.DecryptedData = map[string]string{
		"key1": "value1",
		"key2": "value2",
	}

	if got := c.GetCustomValue("key1"); got != "value1" {
		t.Errorf("GetCustomValue(key1) = %q, want %q", got, "value1")
	}
	if got := c.GetCustomValue("nonexistent"); got != "" {
		t.Errorf("GetCustomValue(nonexistent) = %q, want empty", got)
	}
}

func TestCredentialsResource_ToJSON(t *testing.T) {
	c := NewCredentialsResource("owner", "name", CredentialTypeBasicAuth)
	c.DecryptedData = map[string]string{
		"username": "user",
		"password": "pass",
	}

	jsonStr, err := c.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() error = %v", err)
	}

	if jsonStr == "" || jsonStr == "{}" {
		t.Error("ToJSON() returned empty or empty object")
	}
}

func TestCredentialsResource_ClearDecryptedData(t *testing.T) {
	c := NewCredentialsResource("owner", "name", CredentialTypeAPIKey)
	c.DecryptedData = map[string]string{"api_key": "secret"}

	c.ClearDecryptedData()

	if c.DecryptedData != nil {
		t.Error("ClearDecryptedData() did not clear data")
	}
}
