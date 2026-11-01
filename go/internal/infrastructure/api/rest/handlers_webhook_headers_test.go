package rest

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWebhookHeaderValue_MasksCredentials(t *testing.T) {
	for _, name := range []string{"Authorization", "Proxy-Authorization", "Cookie", "Set-Cookie",
		"X-Api-Key", "X-Auth-Token", "X-Webhook-Secret", "X-Password", "Apikey"} {
		assert.Equal(t, "***", webhookHeaderValue(name, "s3cr3t"), name)
	}
	for _, name := range []string{"Content-Type", "User-Agent", "X-Webhook-Signature", "X-Request-Id"} {
		assert.Equal(t, "v", webhookHeaderValue(name, "v"), name)
	}
}
