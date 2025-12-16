package models

import (
	"encoding/json"
	"fmt"
	"time"
)

// CredentialType defines the type of credential
type CredentialType string

const (
	// CredentialTypeAPIKey represents a simple API key or token
	CredentialTypeAPIKey CredentialType = "api_key"
	// CredentialTypeBasicAuth represents username/password credentials
	CredentialTypeBasicAuth CredentialType = "basic_auth"
	// CredentialTypeOAuth2 represents OAuth2 credentials
	CredentialTypeOAuth2 CredentialType = "oauth2"
	// CredentialTypeServiceAccount represents a service account (e.g., Google Cloud JSON)
	CredentialTypeServiceAccount CredentialType = "service_account"
	// CredentialTypeCustom represents custom key-value pairs
	CredentialTypeCustom CredentialType = "custom"
)

// ValidCredentialTypes returns all valid credential types
func ValidCredentialTypes() []CredentialType {
	return []CredentialType{
		CredentialTypeAPIKey,
		CredentialTypeBasicAuth,
		CredentialTypeOAuth2,
		CredentialTypeServiceAccount,
		CredentialTypeCustom,
	}
}

// IsValidCredentialType checks if the given type is valid
func IsValidCredentialType(t CredentialType) bool {
	for _, valid := range ValidCredentialTypes() {
		if t == valid {
			return true
		}
	}
	return false
}

// CredentialsResource represents encrypted credentials for external services
type CredentialsResource struct {
	BaseResource
	CredentialType CredentialType    `json:"credential_type"`
	EncryptedData  map[string]string `json:"encrypted_data"` // All values are encrypted
	Provider       string            `json:"provider,omitempty"`
	ExpiresAt      *time.Time        `json:"expires_at,omitempty"`
	LastUsedAt     *time.Time        `json:"last_used_at,omitempty"`
	UsageCount     int64             `json:"usage_count"`
	PricingPlanID  string            `json:"pricing_plan_id,omitempty"`

	// DecryptedData holds decrypted values (not stored, populated on demand)
	// This field is only populated when explicitly requested
	DecryptedData map[string]string `json:"-"`
}

// APIKeyCredential represents API key credential data
type APIKeyCredential struct {
	APIKey string `json:"api_key"`
}

// BasicAuthCredential represents basic auth credential data
type BasicAuthCredential struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// OAuth2Credential represents OAuth2 credential data
type OAuth2Credential struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	AccessToken  string `json:"access_token,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	TokenURL     string `json:"token_url,omitempty"`
	Scopes       string `json:"scopes,omitempty"`
}

// ServiceAccountCredential represents service account credential data
type ServiceAccountCredential struct {
	JSONKey string `json:"json_key"` // Full JSON content of service account file
}

// NewCredentialsResource creates a new credentials resource
func NewCredentialsResource(ownerID, name string, credType CredentialType) *CredentialsResource {
	now := time.Now()
	return &CredentialsResource{
		BaseResource: BaseResource{
			Type:      ResourceTypeCredentials,
			OwnerID:   ownerID,
			Name:      name,
			Status:    ResourceStatusActive,
			Metadata:  make(map[string]interface{}),
			CreatedAt: now,
			UpdatedAt: now,
		},
		CredentialType: credType,
		EncryptedData:  make(map[string]string),
		UsageCount:     0,
	}
}

// Validate validates the credentials resource
func (c *CredentialsResource) Validate() error {
	if err := c.BaseResource.Validate(); err != nil {
		return err
	}

	if !IsValidCredentialType(c.CredentialType) {
		return &ValidationError{
			Field:   "credential_type",
			Message: fmt.Sprintf("invalid credential type: %s", c.CredentialType),
		}
	}

	// Validate required fields based on credential type
	switch c.CredentialType {
	case CredentialTypeAPIKey:
		if _, ok := c.EncryptedData["api_key"]; !ok {
			return &ValidationError{Field: "encrypted_data.api_key", Message: "API key is required"}
		}
	case CredentialTypeBasicAuth:
		if _, ok := c.EncryptedData["username"]; !ok {
			return &ValidationError{Field: "encrypted_data.username", Message: "username is required"}
		}
		if _, ok := c.EncryptedData["password"]; !ok {
			return &ValidationError{Field: "encrypted_data.password", Message: "password is required"}
		}
	case CredentialTypeOAuth2:
		if _, ok := c.EncryptedData["client_id"]; !ok {
			return &ValidationError{Field: "encrypted_data.client_id", Message: "client_id is required"}
		}
		if _, ok := c.EncryptedData["client_secret"]; !ok {
			return &ValidationError{Field: "encrypted_data.client_secret", Message: "client_secret is required"}
		}
	case CredentialTypeServiceAccount:
		if _, ok := c.EncryptedData["json_key"]; !ok {
			return &ValidationError{Field: "encrypted_data.json_key", Message: "JSON key is required"}
		}
	case CredentialTypeCustom:
		if len(c.EncryptedData) == 0 {
			return &ValidationError{Field: "encrypted_data", Message: "at least one custom field is required"}
		}
	}

	return nil
}

// IsExpired checks if the credential has expired
func (c *CredentialsResource) IsExpired() bool {
	if c.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*c.ExpiresAt)
}

// IncrementUsage increments the usage counter and updates last used time
func (c *CredentialsResource) IncrementUsage() {
	now := time.Now()
	c.UsageCount++
	c.LastUsedAt = &now
	c.UpdatedAt = now
}

// GetAPIKey returns the API key if this is an API key credential
// Returns empty string if not available or wrong type
func (c *CredentialsResource) GetAPIKey() string {
	if c.CredentialType != CredentialTypeAPIKey {
		return ""
	}
	if c.DecryptedData != nil {
		return c.DecryptedData["api_key"]
	}
	return ""
}

// GetBasicAuth returns username and password if this is a basic auth credential
func (c *CredentialsResource) GetBasicAuth() (username, password string) {
	if c.CredentialType != CredentialTypeBasicAuth {
		return "", ""
	}
	if c.DecryptedData != nil {
		return c.DecryptedData["username"], c.DecryptedData["password"]
	}
	return "", ""
}

// GetOAuth2 returns OAuth2 credentials if this is an OAuth2 credential
func (c *CredentialsResource) GetOAuth2() *OAuth2Credential {
	if c.CredentialType != CredentialTypeOAuth2 {
		return nil
	}
	if c.DecryptedData == nil {
		return nil
	}
	return &OAuth2Credential{
		ClientID:     c.DecryptedData["client_id"],
		ClientSecret: c.DecryptedData["client_secret"],
		AccessToken:  c.DecryptedData["access_token"],
		RefreshToken: c.DecryptedData["refresh_token"],
		TokenURL:     c.DecryptedData["token_url"],
		Scopes:       c.DecryptedData["scopes"],
	}
}

// GetServiceAccountJSON returns the service account JSON if this is a service account credential
func (c *CredentialsResource) GetServiceAccountJSON() string {
	if c.CredentialType != CredentialTypeServiceAccount {
		return ""
	}
	if c.DecryptedData != nil {
		return c.DecryptedData["json_key"]
	}
	return ""
}

// GetCustomValue returns a custom field value
func (c *CredentialsResource) GetCustomValue(key string) string {
	if c.DecryptedData != nil {
		return c.DecryptedData[key]
	}
	return ""
}

// SetAPIKey sets the API key credential data (expects encrypted value)
func (c *CredentialsResource) SetAPIKey(encryptedKey string) {
	c.CredentialType = CredentialTypeAPIKey
	c.EncryptedData = map[string]string{"api_key": encryptedKey}
	c.UpdatedAt = time.Now()
}

// SetBasicAuth sets basic auth credential data (expects encrypted values)
func (c *CredentialsResource) SetBasicAuth(encryptedUsername, encryptedPassword string) {
	c.CredentialType = CredentialTypeBasicAuth
	c.EncryptedData = map[string]string{
		"username": encryptedUsername,
		"password": encryptedPassword,
	}
	c.UpdatedAt = time.Now()
}

// SetOAuth2 sets OAuth2 credential data (expects encrypted values)
func (c *CredentialsResource) SetOAuth2(oauth *OAuth2Credential, encryptedData map[string]string) {
	c.CredentialType = CredentialTypeOAuth2
	c.EncryptedData = encryptedData
	c.UpdatedAt = time.Now()
}

// SetServiceAccount sets service account credential data (expects encrypted JSON)
func (c *CredentialsResource) SetServiceAccount(encryptedJSON string) {
	c.CredentialType = CredentialTypeServiceAccount
	c.EncryptedData = map[string]string{"json_key": encryptedJSON}
	c.UpdatedAt = time.Now()
}

// SetCustomData sets custom credential data (expects encrypted values)
func (c *CredentialsResource) SetCustomData(encryptedData map[string]string) {
	c.CredentialType = CredentialTypeCustom
	c.EncryptedData = encryptedData
	c.UpdatedAt = time.Now()
}

// ToJSON returns the credential data as JSON (for templates)
func (c *CredentialsResource) ToJSON() (string, error) {
	if c.DecryptedData == nil {
		return "{}", nil
	}
	data, err := json.Marshal(c.DecryptedData)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ClearDecryptedData clears the decrypted data from memory
func (c *CredentialsResource) ClearDecryptedData() {
	c.DecryptedData = nil
}
