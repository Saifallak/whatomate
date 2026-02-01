package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/shridarpatil/whatomate/internal/models"
	"github.com/shridarpatil/whatomate/pkg/whatsapp"
	"github.com/shridarpatil/whatomate/test/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

// --- ExchangeToken Tests ---

func TestApp_ExchangeToken_Success_AutoRegistration(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := testutil.CreateTestUser(t, app.DB, org.ID)

	// Mock Meta API server
	metaServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/v21.0/oauth/access_token" || r.URL.Path == "/oauth/access_token":
			// Token exchange
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"access_token": "EAABwzLixnjYBO1234567890",
			})
		case r.URL.Path == "/v21.0/123456789" || r.URL.Path == "/123456789":
			// Phone info
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"verified_name":        "Test Business",
				"display_phone_number": "+1234567890",
			})
		case r.URL.Path == "/v21.0/123456789/register" || r.URL.Path == "/123456789/register":
			// Registration
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]bool{"success": true})
		case r.URL.Path == "/v21.0/987654321/subscribed_apps" || r.URL.Path == "/987654321/subscribed_apps":
			// Webhook subscription
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]bool{"success": true})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer metaServer.Close()

	// Override WhatsApp client to use test server
	app.WhatsApp = whatsapp.NewWithBaseURL(app.Log, metaServer.URL)

	req := testutil.NewJSONRequest(t, map[string]interface{}{
		"code":     "test_auth_code_123",
		"phone_id": "123456789",
		"waba_id":  "987654321",
	})
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.ExchangeToken(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Data map[string]interface{} `json:"data"`
	}
	err = json.Unmarshal(testutil.GetResponseBody(req), &resp)
	require.NoError(t, err)
	assert.Equal(t, "active", resp.Data["status"])
	assert.Equal(t, "123456789", resp.Data["phone_id"])
	assert.Equal(t, "987654321", resp.Data["business_id"])
	assert.NotEmpty(t, resp.Data["pin"]) // PIN should be returned

	// Verify account was created in database
	var account models.WhatsAppAccount
	err = app.DB.Where("phone_id = ? AND organization_id = ?", "123456789", org.ID).First(&account).Error
	require.NoError(t, err)
	assert.Equal(t, "active", account.Status)
	assert.NotEmpty(t, account.Pin)
	assert.Equal(t, "EAABwzLixnjYBO1234567890", account.AccessToken)
}

func TestApp_ExchangeToken_Success_PendingRegistration(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := testutil.CreateTestUser(t, app.DB, org.ID)

	// Mock Meta API server - registration fails (PIN already exists)
	metaServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/v21.0/oauth/access_token" || r.URL.Path == "/oauth/access_token":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"access_token": "test_token",
			})
		case r.URL.Path == "/v21.0/123456789" || r.URL.Path == "/123456789":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"verified_name": "Test Business",
			})
		case r.URL.Path == "/v21.0/123456789/register" || r.URL.Path == "/123456789/register":
			// Registration fails - PIN already exists
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(whatsapp.MetaAPIError{
				Error: struct {
					Message      string `json:"message"`
					Type         string `json:"type"`
					Code         int    `json:"code"`
					ErrorSubcode int    `json:"error_subcode"`
					ErrorUserMsg string `json:"error_user_msg"`
					ErrorData    struct {
						Details string `json:"details"`
					} `json:"error_data"`
					FBTraceID string `json:"fbtrace_id"`
				}{
					Message: "Two-step verification is already enabled",
					Code:    33,
				},
			})
		case r.URL.Path == "/v21.0/987654321/subscribed_apps" || r.URL.Path == "/987654321/subscribed_apps":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]bool{"success": true})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer metaServer.Close()

	app.WhatsApp = whatsapp.NewWithBaseURL(app.Log, metaServer.URL)

	req := testutil.NewJSONRequest(t, map[string]interface{}{
		"code":     "test_code",
		"phone_id": "123456789",
		"waba_id":  "987654321",
	})
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.ExchangeToken(req)
	require.NoError(t, err)

	var resp struct {
		Data map[string]interface{} `json:"data"`
	}
	err = json.Unmarshal(testutil.GetResponseBody(req), &resp)
	require.NoError(t, err)
	assert.Equal(t, "pending_registration", resp.Data["status"])
	assert.Nil(t, resp.Data["pin"]) // No PIN when pending
}

func TestApp_ExchangeToken_InvalidCode(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := testutil.CreateTestUser(t, app.DB, org.ID)

	// Mock Meta API server - invalid code
	metaServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(whatsapp.MetaAPIError{
			Error: struct {
				Message      string `json:"message"`
				Type         string `json:"type"`
				Code         int    `json:"code"`
				ErrorSubcode int    `json:"error_subcode"`
				ErrorUserMsg string `json:"error_user_msg"`
				ErrorData    struct {
					Details string `json:"details"`
				} `json:"error_data"`
				FBTraceID string `json:"fbtrace_id"`
			}{
				Message: "Invalid authorization code",
				Code:    100,
			},
		})
	}))
	defer metaServer.Close()

	app.WhatsApp = whatsapp.NewWithBaseURL(app.Log, metaServer.URL)

	req := testutil.NewJSONRequest(t, map[string]interface{}{
		"code":     "invalid_code",
		"phone_id": "123456789",
		"waba_id":  "987654321",
	})
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.ExchangeToken(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusBadRequest, testutil.GetResponseStatusCode(req))

	body := string(testutil.GetResponseBody(req))
	assert.Contains(t, body, "Invalid authorization code")
}

func TestApp_ExchangeToken_MissingFields(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := testutil.CreateTestUser(t, app.DB, org.ID)

	// Mock Meta API server (won't be called but needed for client initialization)
	metaServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer metaServer.Close()

	// Initialize WhatsApp client
	app.WhatsApp = whatsapp.NewWithBaseURL(app.Log, metaServer.URL)

	tests := []struct {
		name string
		body map[string]interface{}
	}{
		{
			name: "missing_code",
			body: map[string]interface{}{
				"phone_id": "123",
				"waba_id":  "456",
			},
		},
		{
			name: "missing_phone_id",
			body: map[string]interface{}{
				"code":    "test",
				"waba_id": "456",
			},
		},
		{
			name: "missing_waba_id",
			body: map[string]interface{}{
				"code":     "test",
				"phone_id": "123",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := testutil.NewJSONRequest(t, tc.body)
			testutil.SetAuthContext(req, org.ID, user.ID)

			err := app.ExchangeToken(req)
			require.NoError(t, err)
			assert.Equal(t, fasthttp.StatusBadRequest, testutil.GetResponseStatusCode(req))
		})
	}
}

func TestApp_ExchangeToken_Unauthorized(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)

	req := testutil.NewJSONRequest(t, map[string]interface{}{
		"code":     "test",
		"phone_id": "123",
		"waba_id":  "456",
	})
	// No auth context set

	err := app.ExchangeToken(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusUnauthorized, testutil.GetResponseStatusCode(req))
}

// --- RegisterPhone Tests ---

func TestApp_RegisterPhone_Success_WithPIN(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := testutil.CreateTestUser(t, app.DB, org.ID)

	// Create account with pending_registration status
	account := &models.WhatsAppAccount{
		OrganizationID: org.ID,
		Name:           "Test Account - RegisterPhone WithPIN",
		PhoneID:        "123456789",
		BusinessID:     "987654321",
		AccessToken:    "test_token",
		APIVersion:     "v21.0",
		Status:         "pending_registration",
	}
	require.NoError(t, app.DB.Create(account).Error)

	// Mock Meta API server
	metaServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v21.0/123456789/register", r.URL.Path)
		assert.Equal(t, "Bearer test_token", r.Header.Get("Authorization"))

		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "654321", body["pin"])

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]bool{"success": true})
	}))
	defer metaServer.Close()

	app.WhatsApp = whatsapp.NewWithBaseURL(app.Log, metaServer.URL)

	req := testutil.NewJSONRequest(t, map[string]interface{}{
		"pin": "654321",
	})
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", account.ID.String())

	err := app.RegisterPhone(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Data map[string]interface{} `json:"data"`
	}
	err = json.Unmarshal(testutil.GetResponseBody(req), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Data["success"].(bool))
	assert.Equal(t, "654321", resp.Data["pin"])

	// Verify account status updated
	var updated models.WhatsAppAccount
	require.NoError(t, app.DB.Where("id = ?", account.ID).First(&updated).Error)
	assert.Equal(t, "active", updated.Status)
	assert.Equal(t, "654321", updated.Pin)
}

func TestApp_RegisterPhone_Success_GeneratedPIN(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := testutil.CreateTestUser(t, app.DB, org.ID)

	account := &models.WhatsAppAccount{
		OrganizationID: org.ID,
		Name:           "Test Account - GeneratedPIN",
		PhoneID:        "123456789",
		BusinessID:     "987654321",
		AccessToken:    "test_token",
		APIVersion:     "v21.0",
		Status:         "pending_registration",
	}
	require.NoError(t, app.DB.Create(account).Error)

	metaServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		assert.Len(t, body["pin"], 6) // Generated PIN should be 6 digits

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]bool{"success": true})
	}))
	defer metaServer.Close()

	app.WhatsApp = whatsapp.NewWithBaseURL(app.Log, metaServer.URL)

	req := testutil.NewJSONRequest(t, map[string]interface{}{
		// No PIN provided - should generate one
	})
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", account.ID.String())

	err := app.RegisterPhone(req)
	require.NoError(t, err)

	var resp struct {
		Data map[string]interface{} `json:"data"`
	}
	err = json.Unmarshal(testutil.GetResponseBody(req), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Data["success"].(bool))
	assert.NotEmpty(t, resp.Data["pin"])
	assert.Len(t, resp.Data["pin"].(string), 6)
}

func TestApp_RegisterPhone_RegistrationFailed(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := testutil.CreateTestUser(t, app.DB, org.ID)

	account := &models.WhatsAppAccount{
		OrganizationID: org.ID,
		Name:           "Test Account - RegFailed",
		PhoneID:        "123456789",
		BusinessID:     "987654321",
		AccessToken:    "test_token",
		APIVersion:     "v21.0",
		Status:         "pending_registration",
	}
	require.NoError(t, app.DB.Create(account).Error)

	// Mock Meta API server - registration fails
	metaServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(whatsapp.MetaAPIError{
			Error: struct {
				Message      string `json:"message"`
				Type         string `json:"type"`
				Code         int    `json:"code"`
				ErrorSubcode int    `json:"error_subcode"`
				ErrorUserMsg string `json:"error_user_msg"`
				ErrorData    struct {
					Details string `json:"details"`
				} `json:"error_data"`
				FBTraceID string `json:"fbtrace_id"`
			}{
				Message: "Phone number must be verified before registration",
				Code:    368,
			},
		})
	}))
	defer metaServer.Close()

	app.WhatsApp = whatsapp.NewWithBaseURL(app.Log, metaServer.URL)

	req := testutil.NewJSONRequest(t, map[string]interface{}{
		"pin": "123456",
	})
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", account.ID.String())

	err := app.RegisterPhone(req)
	require.NoError(t, err)

	var resp struct {
		Data map[string]interface{} `json:"data"`
	}
	err = json.Unmarshal(testutil.GetResponseBody(req), &resp)
	require.NoError(t, err)
	assert.False(t, resp.Data["success"].(bool))
	assert.Contains(t, resp.Data["error"], "Phone number must be verified")

	// Verify account status NOT updated
	var updated models.WhatsAppAccount
	require.NoError(t, app.DB.Where("id = ?", account.ID).First(&updated).Error)
	assert.Equal(t, "pending_registration", updated.Status)
}

func TestApp_RegisterPhone_AccountNotFound(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := testutil.CreateTestUser(t, app.DB, org.ID)

	req := testutil.NewJSONRequest(t, map[string]interface{}{
		"pin": "123456",
	})
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", uuid.New().String())

	err := app.RegisterPhone(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusNotFound, testutil.GetResponseStatusCode(req))
}

func TestApp_RegisterPhone_InvalidID(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := testutil.CreateTestUser(t, app.DB, org.ID)

	req := testutil.NewJSONRequest(t, map[string]interface{}{
		"pin": "123456",
	})
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", "not-a-uuid")

	err := app.RegisterPhone(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusBadRequest, testutil.GetResponseStatusCode(req))
}

func TestApp_RegisterPhone_CrossOrgIsolation(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org1 := testutil.CreateTestOrganization(t, app.DB)
	org2 := testutil.CreateTestOrganization(t, app.DB)
	user2 := testutil.CreateTestUser(t, app.DB, org2.ID)

	// Create account in org1
	account := &models.WhatsAppAccount{
		OrganizationID: org1.ID,
		Name:           "Test Account - CrossOrg Isolation",
		PhoneID:        "123456789",
		BusinessID:     "987654321",
		AccessToken:    "test_token",
		APIVersion:     "v21.0",
		Status:         "pending_registration",
	}
	require.NoError(t, app.DB.Create(account).Error)

	// User from org2 tries to register org1's account
	req := testutil.NewJSONRequest(t, map[string]interface{}{
		"pin": "123456",
	})
	testutil.SetAuthContext(req, org2.ID, user2.ID)
	testutil.SetPathParam(req, "id", account.ID.String())

	err := app.RegisterPhone(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusNotFound, testutil.GetResponseStatusCode(req))
}
