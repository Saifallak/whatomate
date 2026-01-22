package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/google/uuid"
	"github.com/shridarpatil/whatomate/internal/models"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// AccountRequest represents the request body for creating/updating an account
type AccountRequest struct {
	Name               string `json:"name" validate:"required"`
	AppID              string `json:"app_id"`
	PhoneID            string `json:"phone_id" validate:"required"`
	BusinessID         string `json:"business_id" validate:"required"`
	AccessToken        string `json:"access_token" validate:"required"`
	AppSecret          string `json:"app_secret"` // Meta App Secret for webhook signature verification
	WebhookVerifyToken string `json:"webhook_verify_token"`
	APIVersion         string `json:"api_version"`
	IsDefaultIncoming  bool   `json:"is_default_incoming"`
	IsDefaultOutgoing  bool   `json:"is_default_outgoing"`
	AutoReadReceipt    bool   `json:"auto_read_receipt"`
}

// AccountResponse represents the response for an account (without sensitive data)
type AccountResponse struct {
	ID                 uuid.UUID `json:"id"`
	Name               string    `json:"name"`
	AppID              string    `json:"app_id"`
	PhoneID            string    `json:"phone_id"`
	BusinessID         string    `json:"business_id"`
	WebhookVerifyToken string    `json:"webhook_verify_token"`
	APIVersion         string    `json:"api_version"`
	IsDefaultIncoming  bool      `json:"is_default_incoming"`
	IsDefaultOutgoing  bool      `json:"is_default_outgoing"`
	AutoReadReceipt    bool      `json:"auto_read_receipt"`
	Status             string    `json:"status"`
	HasAccessToken     bool      `json:"has_access_token"`
	HasAppSecret       bool      `json:"has_app_secret"`
	PhoneNumber        string    `json:"phone_number,omitempty"`
	DisplayName        string    `json:"display_name,omitempty"`
	CreatedAt          string    `json:"created_at"`
	UpdatedAt          string    `json:"updated_at"`
}

// ListAccounts returns all WhatsApp accounts for the organization
func (a *App) ListAccounts(r *fastglue.Request) error {
	orgID, err := a.getOrgID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	var accounts []models.WhatsAppAccount
	if err := a.DB.Where("organization_id = ?", orgID).Order("created_at DESC").Find(&accounts).Error; err != nil {
		a.Log.Error("Failed to list accounts", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to list accounts", nil, "")
	}

	// Convert to response format (hide sensitive data)
	response := make([]AccountResponse, len(accounts))
	for i, acc := range accounts {
		response[i] = accountToResponse(acc)
	}

	return r.SendEnvelope(map[string]interface{}{
		"accounts": response,
	})
}

// CreateAccount creates a new WhatsApp account
func (a *App) CreateAccount(r *fastglue.Request) error {
	orgID, err := a.getOrgID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	var req AccountRequest
	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid request body", nil, "")
	}

	// Validate required fields
	if req.Name == "" || req.PhoneID == "" || req.BusinessID == "" || req.AccessToken == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Name, phone_id, business_id, and access_token are required", nil, "")
	}

	// Generate webhook verify token if not provided
	webhookVerifyToken := req.WebhookVerifyToken
	if webhookVerifyToken == "" {
		webhookVerifyToken = generateVerifyToken()
	}

	// Set default API version
	apiVersion := req.APIVersion
	if apiVersion == "" {
		apiVersion = "v21.0"
	}

	account := models.WhatsAppAccount{
		OrganizationID:     orgID,
		Name:               req.Name,
		AppID:              req.AppID,
		PhoneID:            req.PhoneID,
		BusinessID:         req.BusinessID,
		AccessToken:        req.AccessToken, // TODO: encrypt before storing
		AppSecret:          req.AppSecret,   // Meta App Secret for webhook signature verification
		WebhookVerifyToken: webhookVerifyToken,
		APIVersion:         apiVersion,
		IsDefaultIncoming:  req.IsDefaultIncoming,
		IsDefaultOutgoing:  req.IsDefaultOutgoing,
		AutoReadReceipt:    req.AutoReadReceipt,
		Status:             "active",
	}

	// If this is set as default, unset other defaults
	if req.IsDefaultIncoming {
		a.DB.Model(&models.WhatsAppAccount{}).
			Where("organization_id = ? AND is_default_incoming = ?", orgID, true).
			Update("is_default_incoming", false)
	}
	if req.IsDefaultOutgoing {
		a.DB.Model(&models.WhatsAppAccount{}).
			Where("organization_id = ? AND is_default_outgoing = ?", orgID, true).
			Update("is_default_outgoing", false)
	}

	if err := a.DB.Create(&account).Error; err != nil {
		a.Log.Error("Failed to create account", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to create account", nil, "")
	}

	return r.SendEnvelope(accountToResponse(account))
}

// GetAccount returns a single WhatsApp account
func (a *App) GetAccount(r *fastglue.Request) error {
	orgID, err := a.getOrgID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	id, err := parsePathUUID(r, "id", "account")
	if err != nil {
		return nil
	}

	account, err := findByIDAndOrg[models.WhatsAppAccount](a.DB, r, id, orgID, "Account")
	if err != nil {
		return nil
	}

	return r.SendEnvelope(accountToResponse(*account))
}

// UpdateAccount updates a WhatsApp account
func (a *App) UpdateAccount(r *fastglue.Request) error {
	orgID, err := a.getOrgID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	id, err := parsePathUUID(r, "id", "account")
	if err != nil {
		return nil
	}

	account, err := findByIDAndOrg[models.WhatsAppAccount](a.DB, r, id, orgID, "Account")
	if err != nil {
		return nil
	}

	var req AccountRequest
	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid request body", nil, "")
	}

	// Update fields if provided
	if req.Name != "" {
		account.Name = req.Name
	}
	if req.AppID != "" {
		account.AppID = req.AppID
	}
	if req.PhoneID != "" {
		account.PhoneID = req.PhoneID
	}
	if req.BusinessID != "" {
		account.BusinessID = req.BusinessID
	}
	if req.AccessToken != "" {
		account.AccessToken = req.AccessToken // TODO: encrypt
	}
	if req.AppSecret != "" {
		account.AppSecret = req.AppSecret
	}
	if req.WebhookVerifyToken != "" {
		account.WebhookVerifyToken = req.WebhookVerifyToken
	}
	if req.APIVersion != "" {
		account.APIVersion = req.APIVersion
	}
	account.AutoReadReceipt = req.AutoReadReceipt

	// Handle default flags
	if req.IsDefaultIncoming && !account.IsDefaultIncoming {
		a.DB.Model(&models.WhatsAppAccount{}).
			Where("organization_id = ? AND is_default_incoming = ?", orgID, true).
			Update("is_default_incoming", false)
	}
	if req.IsDefaultOutgoing && !account.IsDefaultOutgoing {
		a.DB.Model(&models.WhatsAppAccount{}).
			Where("organization_id = ? AND is_default_outgoing = ?", orgID, true).
			Update("is_default_outgoing", false)
	}
	account.IsDefaultIncoming = req.IsDefaultIncoming
	account.IsDefaultOutgoing = req.IsDefaultOutgoing

	if err := a.DB.Save(account).Error; err != nil {
		a.Log.Error("Failed to update account", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to update account", nil, "")
	}

	// Invalidate cache
	a.InvalidateWhatsAppAccountCache(account.PhoneID)

	return r.SendEnvelope(accountToResponse(*account))
}

// DeleteAccount deletes a WhatsApp account
func (a *App) DeleteAccount(r *fastglue.Request) error {
	orgID, err := a.getOrgID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	id, err := parsePathUUID(r, "id", "account")
	if err != nil {
		return nil
	}

	// Get account first for cache invalidation
	account, err := findByIDAndOrg[models.WhatsAppAccount](a.DB, r, id, orgID, "Account")
	if err != nil {
		return nil
	}

	if err := a.DB.Delete(account).Error; err != nil {
		a.Log.Error("Failed to delete account", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to delete account", nil, "")
	}

	// Invalidate cache
	a.InvalidateWhatsAppAccountCache(account.PhoneID)

	return r.SendEnvelope(map[string]string{"message": "Account deleted successfully"})
}

// TestAccountConnection tests the WhatsApp API connection
// This validates both PhoneID and BusinessID to ensure all credentials are correct
func (a *App) TestAccountConnection(r *fastglue.Request) error {
	orgID, err := a.getOrgID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	id, err := parsePathUUID(r, "id", "account")
	if err != nil {
		return nil
	}

	account, err := findByIDAndOrg[models.WhatsAppAccount](a.DB, r, id, orgID, "Account")
	if err != nil {
		return nil
	}

	// Use the comprehensive validation function
	if err := a.validateAccountCredentials(account.PhoneID, account.BusinessID, account.AccessToken, account.APIVersion); err != nil {
		a.Log.Error("Account test failed", "error", err, "account", account.Name)
		return r.SendEnvelope(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
	}

	// Fetch additional details for display
	url := fmt.Sprintf("%s/%s/%s?fields=display_phone_number,verified_name,quality_rating,messaging_limit_tier",
		a.Config.WhatsApp.BaseURL, account.APIVersion, account.PhoneID)

	req, _ := http.NewRequest(http.MethodGet, url, nil)
	req.Header.Set("Authorization", "Bearer "+account.AccessToken)

	resp, err := a.HTTPClient.Do(req)
	if err != nil {
		return r.SendEnvelope(map[string]interface{}{
			"success": false,
			"error":   "Failed to connect to WhatsApp API: " + err.Error(),
		})
	}
	defer func() { _ = resp.Body.Close() }()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		var errorResp map[string]interface{}
		_ = json.Unmarshal(body, &errorResp)
		return r.SendEnvelope(map[string]interface{}{
			"success": false,
			"error":   "API error",
			"details": errorResp,
		})
	}

	var result map[string]interface{}
	_ = json.Unmarshal(body, &result)

	return r.SendEnvelope(map[string]interface{}{
		"success":              true,
		"display_phone_number": result["display_phone_number"],
		"verified_name":        result["verified_name"],
		"quality_rating":       result["quality_rating"],
		"messaging_limit_tier": result["messaging_limit_tier"],
	})
}

// Helper functions

func accountToResponse(acc models.WhatsAppAccount) AccountResponse {
	return AccountResponse{
		ID:                 acc.ID,
		Name:               acc.Name,
		AppID:              acc.AppID,
		PhoneID:            acc.PhoneID,
		BusinessID:         acc.BusinessID,
		WebhookVerifyToken: acc.WebhookVerifyToken,
		APIVersion:         acc.APIVersion,
		IsDefaultIncoming:  acc.IsDefaultIncoming,
		IsDefaultOutgoing:  acc.IsDefaultOutgoing,
		AutoReadReceipt:    acc.AutoReadReceipt,
		Status:             acc.Status,
		HasAccessToken:     acc.AccessToken != "",
		HasAppSecret:       acc.AppSecret != "",
		CreatedAt:          acc.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:          acc.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

func generateVerifyToken() string {
	bytes := make([]byte, 32)
	_, _ = rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// validateAccountCredentials validates WhatsApp account credentials with Meta API
// It checks both the phone number endpoint and business account endpoint
// and verifies that the phone number actually belongs to the specified business account
func (a *App) validateAccountCredentials(phoneID, businessID, accessToken, apiVersion string) error {
	client := &http.Client{}

	// 1. Validate PhoneID and get its WABA ID
	phoneURL := fmt.Sprintf("%s/%s/%s?fields=display_phone_number,verified_name,account_mode,name_status,quality_rating,messaging_limit_tier",
		a.Config.WhatsApp.BaseURL, apiVersion, phoneID)

	phoneReq, err := http.NewRequest("GET", phoneURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create phone validation request: %w", err)
	}
	phoneReq.Header.Set("Authorization", "Bearer "+accessToken)

	phoneResp, err := client.Do(phoneReq)
	if err != nil {
		return fmt.Errorf("failed to validate phone_id: %w", err)
	}
	defer func() { _ = phoneResp.Body.Close() }()

	phoneBody, _ := io.ReadAll(phoneResp.Body)

	if phoneResp.StatusCode != 200 {
		var errorResp map[string]interface{}
		_ = json.Unmarshal(phoneBody, &errorResp)
		if errData, ok := errorResp["error"].(map[string]interface{}); ok {
			if msg, ok := errData["message"].(string); ok {
				return fmt.Errorf("invalid phone_id or access_token: %s", msg)
			}
		}
		return fmt.Errorf("invalid phone_id or access_token (status %d)", phoneResp.StatusCode)
	}

	// 2. Validate BusinessID
	businessURL := fmt.Sprintf("%s/%s/%s?fields=id,name",
		a.Config.WhatsApp.BaseURL, apiVersion, businessID)

	businessReq, err := http.NewRequest("GET", businessURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create business validation request: %w", err)
	}
	businessReq.Header.Set("Authorization", "Bearer "+accessToken)

	businessResp, err := client.Do(businessReq)
	if err != nil {
		return fmt.Errorf("failed to validate business_id: %w", err)
	}
	defer func() { _ = businessResp.Body.Close() }()

	businessBody, _ := io.ReadAll(businessResp.Body)

	if businessResp.StatusCode != 200 {
		var errorResp map[string]interface{}
		_ = json.Unmarshal(businessBody, &errorResp)
		if errData, ok := errorResp["error"].(map[string]interface{}); ok {
			if msg, ok := errData["message"].(string); ok {
				return fmt.Errorf("invalid business_id: %s", msg)
			}
		}
		return fmt.Errorf("invalid business_id (status %d)", businessResp.StatusCode)
	}

	// 3. Verify that the phone number belongs to this business account
	// Get the list of phone numbers for this business account
	phonesURL := fmt.Sprintf("%s/%s/%s/phone_numbers",
		a.Config.WhatsApp.BaseURL, apiVersion, businessID)

	phonesReq, err := http.NewRequest("GET", phonesURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create phones list request: %w", err)
	}
	phonesReq.Header.Set("Authorization", "Bearer "+accessToken)

	phonesResp, err := client.Do(phonesReq)
	if err != nil {
		return fmt.Errorf("failed to fetch business phone numbers: %w", err)
	}
	defer func() { _ = phonesResp.Body.Close() }()

	phonesBody, _ := io.ReadAll(phonesResp.Body)

	if phonesResp.StatusCode != 200 {
		var errorResp map[string]interface{}
		_ = json.Unmarshal(phonesBody, &errorResp)
		if errData, ok := errorResp["error"].(map[string]interface{}); ok {
			if msg, ok := errData["message"].(string); ok {
				return fmt.Errorf("failed to verify phone-business relationship: %s", msg)
			}
		}
		return fmt.Errorf("failed to verify phone-business relationship (status %d)", phonesResp.StatusCode)
	}

	// Parse the phone numbers list
	var phonesResult struct {
		Data []struct {
			ID                 string `json:"id"`
			DisplayPhoneNumber string `json:"display_phone_number"`
			VerifiedName       string `json:"verified_name"`
		} `json:"data"`
	}
	if err := json.Unmarshal(phonesBody, &phonesResult); err != nil {
		return fmt.Errorf("failed to parse phone numbers list: %w", err)
	}

	// Check if our phoneID is in the list
	phoneFound := false
	for _, phone := range phonesResult.Data {
		if phone.ID == phoneID {
			phoneFound = true
			break
		}
	}

	if !phoneFound {
		return fmt.Errorf("phone_id '%s' does not belong to business_id '%s'. Please verify your configuration", phoneID, businessID)
	}

	a.Log.Info("Account credentials validated successfully", "phone_id", phoneID, "business_id", businessID)
	return nil
}
