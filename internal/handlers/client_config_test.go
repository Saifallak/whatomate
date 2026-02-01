package handlers

import (
	"testing"

	"github.com/shridarpatil/whatomate/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

func TestGetClientConfig(t *testing.T) {
	// Setup test app with mock config
	app := &App{
		Config: &config.Config{
			WhatsApp: config.WhatsAppConfig{
				AppID:      "test-app-id-123",
				ConfigID:   "test-config-id-456",
				APIVersion: "v21.0",
			},
		},
	}

	// Create test request
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.SetRequestURI("/api/config")
	ctx.Request.Header.SetMethod("GET")

	req := fastglue.NewRequest(ctx)

	// Call handler
	err := app.GetClientConfig(req)
	assert.NoError(t, err)

	// Verify response
	assert.Equal(t, fasthttp.StatusOK, ctx.Response.StatusCode())

	// Parse response body
	body := ctx.Response.Body()
	assert.Contains(t, string(body), "test-app-id-123")
	assert.Contains(t, string(body), "test-config-id-456")
	assert.Contains(t, string(body), "v21.0")
	assert.Contains(t, string(body), "whatsapp_app_id")
	assert.Contains(t, string(body), "whatsapp_config_id")
	assert.Contains(t, string(body), "whatsapp_api_version")
}

func TestGetClientConfig_EmptyValues(t *testing.T) {
	// Setup test app with empty config
	app := &App{
		Config: &config.Config{
			WhatsApp: config.WhatsAppConfig{
				AppID:    "",
				ConfigID: "",
			},
		},
	}

	// Create test request
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.SetRequestURI("/api/config")
	ctx.Request.Header.SetMethod("GET")

	req := fastglue.NewRequest(ctx)

	// Call handler
	err := app.GetClientConfig(req)
	assert.NoError(t, err)

	// Verify response
	assert.Equal(t, fasthttp.StatusOK, ctx.Response.StatusCode())

	// Parse response body - should still have structure
	body := ctx.Response.Body()
	assert.Contains(t, string(body), "whatsapp_app_id")
	assert.Contains(t, string(body), "whatsapp_config_id")
	assert.Contains(t, string(body), "whatsapp_api_version")
}
