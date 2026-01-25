package handlers

import (
	"github.com/google/uuid"
	"github.com/shridarpatil/whatomate/internal/models"
	"github.com/shridarpatil/whatomate/pkg/whatsapp"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// GetBusinessProfile returns the business profile for a WhatsApp account
func (a *App) GetBusinessProfile(r *fastglue.Request) error {
	orgID, err := getOrganizationID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	idStr := r.RequestCtx.UserValue("id").(string)
	id, err := uuid.Parse(idStr)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid account ID", nil, "")
	}

	var account models.WhatsAppAccount
	if err := a.DB.Where("id = ? AND organization_id = ?", id, orgID).First(&account).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Account not found", nil, "")
	}

	// Create a context for the request
	ctx := r.RequestCtx

	// Call the WhatsApp client
	profile, err := a.WhatsApp.GetBusinessProfile(ctx, &whatsapp.Account{
		PhoneID:     account.PhoneID,
		BusinessID:  account.BusinessID,
		AppID:       account.AppID,
		APIVersion:  account.APIVersion,
		AccessToken: account.AccessToken,
	})
	if err != nil {
		a.Log.Error("Failed to get business profile", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to get business profile: "+err.Error(), nil, "")
	}

	return r.SendEnvelope(profile)
}

// UpdateBusinessProfile updates the business profile for a WhatsApp account
func (a *App) UpdateBusinessProfile(r *fastglue.Request) error {
	orgID, err := getOrganizationID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	idStr := r.RequestCtx.UserValue("id").(string)
	id, err := uuid.Parse(idStr)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid account ID", nil, "")
	}

	var account models.WhatsAppAccount
	if err := a.DB.Where("id = ? AND organization_id = ?", id, orgID).First(&account).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Account not found", nil, "")
	}

	var input whatsapp.BusinessProfileInput
	if err := r.Decode(&input, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid request body", nil, "")
	}

	// Create a context for the request
	ctx := r.RequestCtx

	// Call the WhatsApp client
	err = a.WhatsApp.UpdateBusinessProfile(ctx, &whatsapp.Account{
		PhoneID:     account.PhoneID,
		BusinessID:  account.BusinessID,
		AppID:       account.AppID,
		APIVersion:  account.APIVersion,
		AccessToken: account.AccessToken,
	}, input)

	if err != nil {
		a.Log.Error("Failed to update business profile", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to update business profile: "+err.Error(), nil, "")
	}

	// Return updated profile to confirm
	// We re-fetch it to ensure we have the latest state (including any server-side processing)
	profile, err := a.WhatsApp.GetBusinessProfile(ctx, &whatsapp.Account{
		PhoneID:     account.PhoneID,
		BusinessID:  account.BusinessID,
		AppID:       account.AppID,
		APIVersion:  account.APIVersion,
		AccessToken: account.AccessToken,
	})
	if err != nil {
		// If re-fetch fails, just return success message
		return r.SendEnvelope(map[string]string{"message": "Profile updated successfully"})
	}

	return r.SendEnvelope(profile)
}
