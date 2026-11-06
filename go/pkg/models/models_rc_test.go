package models

import (
	"errors"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func rcValidationField(t *testing.T, err error) string {
	t.Helper()
	var ve *ValidationError
	require.ErrorAs(t, err, &ve)
	return ve.Field
}

func TestRCTrigger_CronValidation(t *testing.T) {
	valid := []string{
		"0 0 * * *",         // standard 5 field
		"*/5 * * * *",       // every 5 minutes
		"0 */5 * * * *",     // 6 field with seconds
		"*/30 * * * * *",    // every 30 seconds
		"0 0 9 * * MON-FRI", // named weekdays
		"@hourly",
		"@every 90s",
		"CRON_TZ=Europe/Berlin 0 9 * * *",
	}
	for _, expr := range valid {
		tr := &Trigger{WorkflowID: "w", Name: "n", Type: TriggerTypeCron, Config: map[string]any{"schedule": expr}}
		assert.NoError(t, tr.Validate(), expr)
		assert.NoError(t, ValidateCronExpression(expr), expr)
	}

	invalid := []string{
		"every minute",
		"* * *",
		"61 * * * *",
		"0 0 0 0 0 0 0",
		"@sometimes",
		"* * 32 * *",
	}
	for _, expr := range invalid {
		tr := &Trigger{WorkflowID: "w", Name: "n", Type: TriggerTypeCron, Config: map[string]any{"schedule": expr}}
		err := tr.Validate()
		require.Error(t, err, expr)
		assert.Equal(t, "config.schedule", rcValidationField(t, err), expr)
		assert.Contains(t, err.Error(), "invalid cron expression", expr)
		assert.Error(t, ValidateCronExpression(expr), expr)
	}
}

func TestRCTrigger_CronTimezone(t *testing.T) {
	mk := func(tz any) *Trigger {
		return &Trigger{WorkflowID: "w", Name: "n", Type: TriggerTypeCron, Config: map[string]any{"schedule": "0 9 * * *", "timezone": tz}}
	}
	assert.NoError(t, mk("Europe/Moscow").Validate())
	assert.NoError(t, mk("").Validate())
	assert.NoError(t, mk(nil).Validate())

	err := mk("Mars/Base").Validate()
	assert.Equal(t, "config.timezone", rcValidationField(t, err))
	err = mk(3).Validate()
	assert.Equal(t, "config.timezone", rcValidationField(t, err))
}

func TestRCTrigger_OtherTypes(t *testing.T) {
	tests := []struct {
		name    string
		tr      Trigger
		field   string
		wantErr bool
	}{
		{"missing workflow", Trigger{Name: "n", Type: TriggerTypeManual}, "workflow_id", true},
		{"missing name", Trigger{WorkflowID: "w", Type: TriggerTypeManual}, "name", true},
		{"missing type", Trigger{WorkflowID: "w", Name: "n"}, "type", true},
		{"unknown type", Trigger{WorkflowID: "w", Name: "n", Type: "bogus"}, "type", true},
		{"manual", Trigger{WorkflowID: "w", Name: "n", Type: TriggerTypeManual}, "", false},
		{"webhook", Trigger{WorkflowID: "w", Name: "n", Type: TriggerTypeWebhook}, "", false},
		{"event ok", Trigger{WorkflowID: "w", Name: "n", Type: TriggerTypeEvent, Config: map[string]any{"event_type": "x"}}, "", false},
		{"event missing", Trigger{WorkflowID: "w", Name: "n", Type: TriggerTypeEvent, Config: map[string]any{}}, "config.event_type", true},
		{"interval number", Trigger{WorkflowID: "w", Name: "n", Type: TriggerTypeInterval, Config: map[string]any{"interval": 5.0}}, "", false},
		{"interval string", Trigger{WorkflowID: "w", Name: "n", Type: TriggerTypeInterval, Config: map[string]any{"interval": "5m"}}, "", false},
		{"interval missing", Trigger{WorkflowID: "w", Name: "n", Type: TriggerTypeInterval, Config: map[string]any{}}, "config.interval", true},
		{"interval negative", Trigger{WorkflowID: "w", Name: "n", Type: TriggerTypeInterval, Config: map[string]any{"interval": -1.0}}, "config.interval", true},
		{"interval bad string", Trigger{WorkflowID: "w", Name: "n", Type: TriggerTypeInterval, Config: map[string]any{"interval": "soon"}}, "config.interval", true},
		{"interval bad type", Trigger{WorkflowID: "w", Name: "n", Type: TriggerTypeInterval, Config: map[string]any{"interval": true}}, "config.interval", true},
		{"cron missing schedule", Trigger{WorkflowID: "w", Name: "n", Type: TriggerTypeCron, Config: map[string]any{}}, "config.schedule", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.tr.Validate()
			if !tt.wantErr {
				assert.NoError(t, err)
				return
			}
			assert.Equal(t, tt.field, rcValidationField(t, err))
		})
	}
}

func TestRCAPIClient(t *testing.T) {
	c := &APIClient{
		OwnerID: "o", Name: "api", BaseURL: "https://api.example.com",
		DefaultHeaders: map[string]string{"X-A": "1"},
		Operations:     []*APIOperation{{ID: "op1"}, {ID: "op2"}},
	}
	assert.NoError(t, c.Validate())
	assert.Equal(t, "op2", c.FindOperation("op2").ID)
	assert.Nil(t, c.FindOperation("missing"))

	mutate := []struct {
		name  string
		f     func(*APIClient)
		field string
	}{
		{"owner", func(c *APIClient) { c.OwnerID = "" }, "owner_id"},
		{"name blank", func(c *APIClient) { c.Name = "  " }, "name"},
		{"name long", func(c *APIClient) { c.Name = strings.Repeat("a", 256) }, "name"},
		{"base url scheme", func(c *APIClient) { c.BaseURL = "ftp://x" }, "base_url"},
		{"base url host", func(c *APIClient) { c.BaseURL = "http://" }, "base_url"},
		{"base url parse", func(c *APIClient) { c.BaseURL = "http://[::1" }, "base_url"},
		{"header", func(c *APIClient) { c.DefaultHeaders = map[string]string{" ": "v"} }, "default_headers"},
		{"auth", func(c *APIClient) { c.Auth.Type = "magic" }, "auth.type"},
	}
	for _, m := range mutate {
		t.Run(m.name, func(t *testing.T) {
			cp := *c
			m.f(&cp)
			assert.Equal(t, m.field, rcValidationField(t, cp.Validate()))
		})
	}
}

func TestRCAPIClientAuth(t *testing.T) {
	assert.False(t, APIClientAuth{}.IsEnabled())
	assert.False(t, APIClientAuth{Type: APIClientAuthNone}.IsEnabled())
	assert.True(t, APIClientAuth{Type: APIClientAuthBearer}.IsEnabled())

	for _, a := range []APIClientAuth{
		{}, {Type: APIClientAuthNone}, {Type: APIClientAuthBearer}, {Type: APIClientAuthBasic},
		{Type: APIClientAuthAPIKey, In: APIKeyInHeader, Name: "X-Key"},
		{Type: APIClientAuthAPIKey, In: APIKeyInQuery, Name: "key"},
		{Type: APIClientAuthAPIKey, In: APIKeyInCookie, Name: "c"},
		{Type: APIClientAuthOAuth2ClientCredentials},
		{Type: APIClientAuthOAuth2ClientCredentials, TokenURL: "https://auth.example.com/token"},
	} {
		assert.NoError(t, a.Validate(), a)
	}

	assert.Equal(t, "auth.in", rcValidationField(t, APIClientAuth{Type: APIClientAuthAPIKey, In: "body", Name: "k"}.Validate()))
	assert.Equal(t, "auth.name", rcValidationField(t, APIClientAuth{Type: APIClientAuthAPIKey, In: APIKeyInHeader, Name: " "}.Validate()))
	assert.Equal(t, "auth.token_url", rcValidationField(t, APIClientAuth{Type: APIClientAuthOAuth2ClientCredentials, TokenURL: "/relative"}.Validate()))
}

func TestRCEvent(t *testing.T) {
	cases := map[string][3]bool{ // execution, node, wave
		EventTypeExecutionStarted:   {true, false, false},
		EventTypeExecutionResumed:   {true, false, false},
		EventTypeExecutionTimeout:   {true, false, false},
		EventTypeNodeRetrying:       {false, true, false},
		EventTypeNodeSkipped:        {false, true, false},
		EventTypeWaveStarted:        {false, false, true},
		EventTypeWaveCompleted:      {false, false, true},
		EventTypeConditionEvaluated: {false, false, false},
	}
	for typ, want := range cases {
		e := &Event{EventType: typ}
		assert.Equal(t, want[0], e.IsExecutionEvent(), typ)
		assert.Equal(t, want[1], e.IsNodeEvent(), typ)
		assert.Equal(t, want[2], e.IsWaveEvent(), typ)
	}

	e := &Event{}
	assert.Equal(t, "execution_id", rcValidationField(t, e.Validate()))
	e.ExecutionID = "x"
	assert.Equal(t, "event_type", rcValidationField(t, e.Validate()))
	e.EventType = EventTypeNodeFailed
	assert.NoError(t, e.Validate())

	assert.Empty(t, e.GetNodeID())
	assert.Empty(t, e.GetNodeName())
	assert.Empty(t, e.GetError())
	e.Payload = map[string]any{"node_id": 1, "node_name": 2, "error": 3}
	assert.Empty(t, e.GetNodeID())
	assert.Empty(t, e.GetNodeName())
	assert.Empty(t, e.GetError())
	e.Payload = map[string]any{"node_id": "n1", "node_name": "Node", "error": "boom"}
	assert.Equal(t, "n1", e.GetNodeID())
	assert.Equal(t, "Node", e.GetNodeName())
	assert.Equal(t, "boom", e.GetError())
}

func TestRCSystemKey(t *testing.T) {
	k := NewSystemKey("name", "desc", "svc", "admin")
	assert.NotEmpty(t, k.ID)
	assert.Equal(t, SystemKeyStatusActive, k.Status)
	assert.NoError(t, k.Validate())
	assert.True(t, k.IsActive())
	assert.False(t, k.IsExpired())
	assert.NoError(t, k.CanUse())

	k.IncrementUsage()
	k.IncrementUsage()
	assert.Equal(t, int64(2), k.UsageCount)
	assert.NotNil(t, k.LastUsedAt)

	assert.Error(t, k.SetExpiration(time.Now().Add(-time.Hour)))
	require.NoError(t, k.SetExpiration(time.Now().Add(time.Hour)))
	assert.True(t, k.IsActive())

	past := time.Now().Add(-time.Minute)
	k.ExpiresAt = &past
	assert.True(t, k.IsExpired())
	assert.False(t, k.IsActive())
	assert.ErrorIs(t, k.CanUse(), ErrSystemKeyExpired)

	k.Revoke()
	assert.Equal(t, SystemKeyStatusRevoked, k.Status)
	assert.NotNil(t, k.RevokedAt)
	assert.False(t, k.IsActive())
	assert.ErrorIs(t, k.CanUse(), ErrSystemKeyRevoked)

	for _, m := range []struct {
		f     func(*SystemKey)
		field string
	}{
		{func(k *SystemKey) { k.Name = " " }, "name"},
		{func(k *SystemKey) { k.Name = strings.Repeat("n", 256) }, "name"},
		{func(k *SystemKey) { k.ServiceName = "" }, "service_name"},
		{func(k *SystemKey) { k.ServiceName = strings.Repeat("s", 101) }, "service_name"},
		{func(k *SystemKey) { k.CreatedBy = "" }, "created_by"},
	} {
		key := NewSystemKey("name", "", "svc", "admin")
		m.f(key)
		assert.Equal(t, m.field, rcValidationField(t, key.Validate()))
	}
}

func TestRCServiceAuditLog(t *testing.T) {
	l := NewServiceAuditLog("k1", "svc", "workflow.create", "workflow", "POST", "/api/v1/service/workflows", "1.2.3.4", 201)
	assert.NotEmpty(t, l.ID)
	assert.Equal(t, "k1", l.SystemKeyID)
	assert.Equal(t, 201, l.ResponseStatus)
	assert.Equal(t, "1.2.3.4", l.IPAddress)
	assert.WithinDuration(t, time.Now(), l.CreatedAt, time.Minute)
}

func TestRCUserAndRoles(t *testing.T) {
	u := &User{ID: "u1"}
	assert.False(t, u.HasRole("viewer"))
	u.Roles = []string{"viewer", "ghost"}
	assert.True(t, u.HasRole("viewer"))
	assert.False(t, u.HasRole("editor"))

	assert.True(t, u.HasPermission(PermissionWorkflowRead))
	assert.False(t, u.HasPermission(PermissionWorkflowDelete))

	u.Roles = []string{"admin"}
	// admin role carries system:admin, which grants everything.
	assert.True(t, u.HasPermission("anything:else"))

	u.Roles = nil
	u.IsAdmin = true
	assert.True(t, u.HasPermission(PermissionUserManage))
	assert.True(t, u.CanAccessResource("someone-else", PermissionUserManage))

	u.IsAdmin = false
	u.Roles = []string{"executor"}
	assert.True(t, u.IsOwner("u1"))
	assert.False(t, u.IsOwner("u2"))
	assert.True(t, u.CanAccessResource("u1", PermissionRoleManage))
	assert.True(t, u.CanAccessResource("u2", PermissionWorkflowExecute))
	assert.False(t, u.CanAccessResource("u2", PermissionWorkflowDelete))

	assert.Equal(t, "email", rcValidationField(t, (&User{}).Validate()))
	assert.Equal(t, "username", rcValidationField(t, (&User{Email: "e"}).Validate()))
	assert.Equal(t, "password_hash", rcValidationField(t, (&User{Email: "e", Username: "u"}).Validate()))
	assert.NoError(t, (&User{Email: "e", Username: "u", PasswordHash: "h"}).Validate())

	s := &Session{}
	assert.Equal(t, "user_id", rcValidationField(t, s.Validate()))
	s.UserID = "u"
	assert.Equal(t, "token", rcValidationField(t, s.Validate()))
	s.Token = "t"
	assert.Equal(t, "expires_at", rcValidationField(t, s.Validate()))
	s.ExpiresAt = time.Now().Add(time.Hour)
	assert.NoError(t, s.Validate())
	assert.False(t, s.IsExpired())
	s.ExpiresAt = time.Now().Add(-time.Hour)
	assert.True(t, s.IsExpired())

	r := &Role{}
	assert.Equal(t, "name", rcValidationField(t, r.Validate()))
	r.Name = "custom"
	assert.Equal(t, "permissions", rcValidationField(t, r.Validate()))
	r.Permissions = []string{PermissionTriggerRead}
	assert.NoError(t, r.Validate())
	assert.True(t, r.HasPermission(PermissionTriggerRead))
	assert.False(t, r.HasPermission(PermissionTriggerDelete))

	assert.Equal(t, "action", rcValidationField(t, (&AuditLog{}).Validate()))
	assert.NoError(t, (&AuditLog{Action: "login"}).Validate())
}

func TestRCAuthError(t *testing.T) {
	inner := errors.New("denied")
	e := &AuthError{Err: inner}
	assert.Equal(t, "auth error: denied", e.Error())
	e = &AuthError{UserID: "u1", Action: "delete", Err: inner}
	assert.Equal(t, "auth error for user u1 during delete: denied", e.Error())
	assert.ErrorIs(t, e, inner)
}

func TestRCCredentialsResource(t *testing.T) {
	c := NewCredentialsResource("o", "cred", CredentialTypeAPIKey)

	c.SetAPIKey("enc-key")
	assert.Equal(t, CredentialTypeAPIKey, c.CredentialType)
	assert.NoError(t, c.Validate())
	assert.Empty(t, c.GetAPIKey())
	c.DecryptedData = map[string]string{"api_key": "plain"}
	assert.Equal(t, "plain", c.GetAPIKey())
	assert.Equal(t, "plain", c.GetCustomValue("api_key"))

	c.SetBasicAuth("u", "p")
	assert.NoError(t, c.Validate())
	assert.Empty(t, c.GetAPIKey())
	c.DecryptedData = nil
	user, pass := c.GetBasicAuth()
	assert.Empty(t, user+pass)
	c.DecryptedData = map[string]string{"username": "bob", "password": "pw"}
	user, pass = c.GetBasicAuth()
	assert.Equal(t, "bob", user)
	assert.Equal(t, "pw", pass)

	c.SetOAuth2(&OAuth2Credential{ClientID: "ignored"}, map[string]string{"client_id": "x", "client_secret": "y"})
	assert.NoError(t, c.Validate())
	c.DecryptedData = nil
	assert.Nil(t, c.GetOAuth2())
	c.DecryptedData = map[string]string{"client_id": "id", "client_secret": "sec", "token_url": "https://t", "scopes": "a b"}
	o := c.GetOAuth2()
	require.NotNil(t, o)
	assert.Equal(t, "id", o.ClientID)
	assert.Equal(t, "a b", o.Scopes)
	user, _ = c.GetBasicAuth()
	assert.Empty(t, user)

	c.SetServiceAccount("{json}")
	assert.NoError(t, c.Validate())
	assert.Nil(t, c.GetOAuth2())
	c.DecryptedData = nil
	assert.Empty(t, c.GetServiceAccountJSON())
	c.DecryptedData = map[string]string{"json_key": "{}"}
	assert.Equal(t, "{}", c.GetServiceAccountJSON())

	c.SetCustomData(map[string]string{"token": "t"})
	assert.NoError(t, c.Validate())
	assert.Empty(t, c.GetServiceAccountJSON())
	c.DecryptedData = nil
	assert.Empty(t, c.GetCustomValue("token"))

	js, err := c.ToJSON()
	require.NoError(t, err)
	assert.Equal(t, "{}", js)
	c.DecryptedData = map[string]string{"a": "b"}
	js, err = c.ToJSON()
	require.NoError(t, err)
	assert.JSONEq(t, `{"a":"b"}`, js)

	for _, m := range []struct {
		typ   CredentialType
		data  map[string]string
		field string
	}{
		{"weird", map[string]string{}, "credential_type"},
		{CredentialTypeAPIKey, map[string]string{}, "encrypted_data.api_key"},
		{CredentialTypeBasicAuth, map[string]string{}, "encrypted_data.username"},
		{CredentialTypeBasicAuth, map[string]string{"username": "u"}, "encrypted_data.password"},
		{CredentialTypeOAuth2, map[string]string{}, "encrypted_data.client_id"},
		{CredentialTypeOAuth2, map[string]string{"client_id": "c"}, "encrypted_data.client_secret"},
		{CredentialTypeServiceAccount, map[string]string{}, "encrypted_data.json_key"},
		{CredentialTypeCustom, map[string]string{}, "encrypted_data"},
	} {
		cr := NewCredentialsResource("o", "n", m.typ)
		cr.EncryptedData = m.data
		assert.Equal(t, m.field, rcValidationField(t, cr.Validate()), m.field)
	}
	cr := NewCredentialsResource("", "n", CredentialTypeCustom)
	assert.Equal(t, "owner_id", rcValidationField(t, cr.Validate()))
}

func TestRCRentalKey(t *testing.T) {
	u := MultimodalUsage{PromptTokens: 1, CompletionTokens: 2, ImageInputTokens: 3, ImageOutputTokens: 4, AudioInputTokens: 5, AudioOutputTokens: 6, VideoInputTokens: 7, VideoOutputTokens: 8}
	assert.Equal(t, int64(1+3+5+7), u.TotalInputTokens())
	assert.Equal(t, int64(2+4+6+8), u.TotalOutputTokens())
	assert.False(t, u.IsEmpty())
	assert.True(t, (&MultimodalUsage{}).IsEmpty())

	r := NewRentalKeyResource("o", "k", LLMProviderTypeOpenAI)
	assert.NoError(t, r.Validate())
	assert.Nil(t, r.GetRemainingDailyRequests())
	assert.Nil(t, r.GetRemainingMonthlyTokens())
	assert.Zero(t, r.GetDailyUsagePercent())
	assert.Zero(t, r.GetMonthlyUsagePercent())

	daily, monthly := 4, int64(100)
	r.DailyRequestLimit = &daily
	r.MonthlyTokenLimit = &monthly
	r.RecordUsage(MultimodalUsage{PromptTokens: 30}, 0.1)
	assert.Equal(t, 3, *r.GetRemainingDailyRequests())
	assert.Equal(t, int64(70), *r.GetRemainingMonthlyTokens())
	assert.InDelta(t, 25.0, r.GetDailyUsagePercent(), 1e-9)
	assert.InDelta(t, 30.0, r.GetMonthlyUsagePercent(), 1e-9)

	r.RequestsToday = 10
	r.TokensThisMonth = 500
	assert.Equal(t, 0, *r.GetRemainingDailyRequests())
	assert.Equal(t, int64(0), *r.GetRemainingMonthlyTokens())

	zero := 0
	r.DailyRequestLimit = &zero
	assert.Zero(t, r.GetDailyUsagePercent())
	assert.False(t, math.IsNaN(r.GetDailyUsagePercent()))

	r.Suspend()
	assert.Equal(t, ResourceStatusSuspended, r.Status)
	assert.ErrorIs(t, r.CanMakeRequest(), ErrRentalKeySuspended)
	r.Activate()
	assert.Equal(t, ResourceStatusActive, r.Status)

	bad := NewRentalKeyResource("o", "k", "unknown")
	assert.Equal(t, "provider", rcValidationField(t, bad.Validate()))
	bad = NewRentalKeyResource("o", "k", LLMProviderTypeAnthropic)
	bad.ProvisionerType = "nope"
	assert.Equal(t, "provisioner_type", rcValidationField(t, bad.Validate()))
	bad = NewRentalKeyResource("o", "", LLMProviderTypeAnthropic)
	assert.Equal(t, "name", rcValidationField(t, bad.Validate()))

	rec := NewRentalKeyUsageRecord("rk", "gpt", u)
	assert.Equal(t, "success", rec.Status)
	rec.SetFailed(nil)
	assert.Equal(t, "failed", rec.Status)
	assert.Empty(t, rec.ErrorMessage)
	rec.SetFailed(errors.New("boom"))
	assert.Equal(t, "boom", rec.ErrorMessage)
	rec.SetRateLimited(nil)
	assert.Equal(t, "rate_limited", rec.Status)
	rec.SetRateLimited(errors.New("slow down"))
	assert.Equal(t, "slow down", rec.ErrorMessage)
}

func TestRCBaseResourceAccessors(t *testing.T) {
	r := &BaseResource{Description: "d", Metadata: map[string]any{"k": 1}}
	assert.Equal(t, "d", r.GetDescription())
	assert.Equal(t, map[string]any{"k": 1}, r.GetMetadata())

	f := NewFileStorageResource("", "files")
	assert.Equal(t, "owner_id", rcValidationField(t, f.Validate()))
}

func TestRCWorkflowResourcesAndEdges(t *testing.T) {
	wr := &WorkflowResource{}
	assert.Equal(t, "resource_id", rcValidationField(t, wr.Validate()))
	wr.ResourceID = "r"
	assert.Equal(t, "alias", rcValidationField(t, wr.Validate()))
	for _, bad := range []string{"1abc", "_x", "has-dash", "sp ace", strings.Repeat("a", 101)} {
		wr.Alias = bad
		assert.Equal(t, "alias", rcValidationField(t, wr.Validate()), bad)
	}
	wr.Alias = "My_alias1"
	wr.AccessType = "bogus"
	assert.NoError(t, wr.Validate())
	assert.Equal(t, "read", wr.AccessType, "unknown access types default to read")
	wr.AccessType = "admin"
	assert.NoError(t, wr.Validate())
	assert.Equal(t, "admin", wr.AccessType)

	edges := []struct {
		e     Edge
		field string
	}{
		{Edge{}, "id"},
		{Edge{ID: "e"}, "from"},
		{Edge{ID: "e", From: "a"}, "to"},
		{Edge{ID: "e", From: "a", To: "a"}, "edge"},
		{Edge{ID: "e", From: "a", To: "b", Loop: &LoopConfig{}}, "loop.max_iterations"},
		{Edge{ID: "e", From: "a", To: "b", Loop: &LoopConfig{MaxIterations: 2}, Condition: "x"}, "loop"},
	}
	for _, tt := range edges {
		assert.Equal(t, tt.field, rcValidationField(t, tt.e.Validate()), tt.field)
	}
	loop := Edge{ID: "e", From: "b", To: "a", Loop: &LoopConfig{MaxIterations: 2}}
	assert.NoError(t, loop.Validate())
	assert.True(t, loop.IsLoop())
	assert.False(t, (&Edge{}).IsLoop())

	wf := &Workflow{Name: "w", Nodes: []*Node{{ID: "a", Name: "A", Type: "t"}, {ID: "b", Name: "B", Type: "t"}}}
	assert.NoError(t, wf.Validate())

	assert.Equal(t, "nodes", rcValidationField(t, (&Workflow{Name: "w"}).Validate()))
	dup := &Workflow{Name: "w", Nodes: []*Node{{ID: "a", Name: "A", Type: "t"}, {ID: "a", Name: "A", Type: "t"}}}
	assert.Equal(t, "nodes", rcValidationField(t, dup.Validate()))
	badNode := &Workflow{Name: "w", Nodes: []*Node{{ID: "a"}}}
	assert.Equal(t, "name", rcValidationField(t, badNode.Validate()))

	wf.Edges = []*Edge{{ID: "e"}}
	assert.Equal(t, "from", rcValidationField(t, wf.Validate()))
	wf.Edges = []*Edge{{ID: "e", From: "x", To: "b"}}
	assert.Contains(t, wf.Validate().Error(), "non-existent source")
	wf.Edges = []*Edge{{ID: "e", From: "a", To: "x"}}
	assert.Contains(t, wf.Validate().Error(), "non-existent target")
	wf.Edges = nil

	wf.Resources = []WorkflowResource{{ResourceID: "r1", Alias: "a1"}, {ResourceID: "r2", Alias: "a1"}}
	assert.Contains(t, wf.Validate().Error(), "duplicate resource alias")
	wf.Resources = []WorkflowResource{{ResourceID: "", Alias: "a1"}}
	assert.Equal(t, "resource_id", rcValidationField(t, wf.Validate()))
	wf.Resources = nil

	// AddEdge error branches.
	assert.Error(t, wf.AddEdge(&Edge{}))
	assert.Equal(t, "from", rcValidationField(t, wf.AddEdge(&Edge{ID: "e", From: "x", To: "b"})))
	assert.Equal(t, "to", rcValidationField(t, wf.AddEdge(&Edge{ID: "e", From: "a", To: "x"})))
	require.NoError(t, wf.AddEdge(&Edge{ID: "e", From: "a", To: "b"}))
	assert.Equal(t, "id", rcValidationField(t, wf.AddEdge(&Edge{ID: "e", From: "b", To: "a"})))

	// RemoveNode drops attached edges; unknown node errors.
	assert.ErrorIs(t, wf.RemoveNode("zzz"), ErrNodeNotFound)
	require.NoError(t, wf.RemoveNode("a"))
	assert.Empty(t, wf.Edges)

	// Clone failure on unmarshalable data.
	wf.Variables = map[string]any{"ch": make(chan int)}
	_, err := wf.Clone()
	assert.Error(t, err)
}
