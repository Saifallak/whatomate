package handlers

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/shridarpatil/whatomate/internal/models"
	"github.com/shridarpatil/whatomate/internal/websocket"
	"github.com/shridarpatil/whatomate/pkg/whatsapp"
)

// SLAProcessor handles periodic SLA checks and escalations
type SLAProcessor struct {
	app      *App
	interval time.Duration
	stopCh   chan struct{}
}

// NewSLAProcessor creates a new SLA processor
func NewSLAProcessor(app *App, interval time.Duration) *SLAProcessor {
	return &SLAProcessor{
		app:      app,
		interval: interval,
		stopCh:   make(chan struct{}),
	}
}

// Start begins the SLA processing loop
func (p *SLAProcessor) Start(ctx context.Context) {
	p.app.Log.Info("SLA processor started", "interval", p.interval)

	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			p.app.Log.Info("SLA processor stopped by context")
			return
		case <-p.stopCh:
			p.app.Log.Info("SLA processor stopped")
			return
		case <-ticker.C:
			p.processStaleTransfers()
		}
	}
}

// Stop stops the SLA processor
func (p *SLAProcessor) Stop() {
	close(p.stopCh)
}

// processStaleTransfers checks for transfers that need escalation or auto-close
func (p *SLAProcessor) processStaleTransfers() {
	now := time.Now()

	// Get all organizations with SLA enabled
	var settings []models.ChatbotSettings
	if err := p.app.DB.Where("sla_enabled = ?", true).Find(&settings).Error; err != nil {
		p.app.Log.Error("Failed to load SLA settings", "error", err)
		return
	}

	for _, s := range settings {
		p.processOrganizationSLA(s, now)
	}
}

// processOrganizationSLA processes SLA for a single organization
func (p *SLAProcessor) processOrganizationSLA(settings models.ChatbotSettings, now time.Time) {
	orgID := settings.OrganizationID

	// 1. Auto-close expired transfers
	if settings.SLAAutoCloseHours > 0 {
		p.autoCloseExpiredTransfers(orgID, settings, now)
	}

	// 2. Escalate transfers past escalation deadline
	if settings.SLAEscalationMinutes > 0 {
		p.escalateTransfers(orgID, settings, now)
	}

	// 3. Mark SLA breached for transfers past response deadline
	if settings.SLAResponseMinutes > 0 {
		p.markSLABreached(orgID, settings, now)
	}
}

// autoCloseExpiredTransfers closes transfers that have exceeded their expiry time
func (p *SLAProcessor) autoCloseExpiredTransfers(orgID uuid.UUID, settings models.ChatbotSettings, now time.Time) {
	var transfers []models.AgentTransfer
	if err := p.app.DB.Where(
		"organization_id = ? AND status = ? AND expires_at IS NOT NULL AND expires_at < ?",
		orgID, "active", now,
	).Find(&transfers).Error; err != nil {
		p.app.Log.Error("Failed to find expired transfers", "error", err, "org_id", orgID)
		return
	}

	for _, transfer := range transfers {
		// Send auto-close message to customer if configured
		if settings.SLAAutoCloseMessage != "" {
			p.sendSLAAutoCloseToCustomer(transfer, settings.SLAAutoCloseMessage)
		}

		// Update transfer status
		if err := p.app.DB.Model(&transfer).Updates(map[string]interface{}{
			"status":     "expired",
			"resumed_at": now,
			"notes":      transfer.Notes + "\n[Auto-closed: No agent response within SLA]",
		}).Error; err != nil {
			p.app.Log.Error("Failed to expire transfer", "error", err, "transfer_id", transfer.ID)
			continue
		}

		p.app.Log.Info("Transfer auto-closed due to expiry",
			"transfer_id", transfer.ID,
			"contact_id", transfer.ContactID,
			"expires_at", transfer.ExpiresAt,
		)

		// Broadcast update
		p.broadcastTransferUpdate(transfer, "expired")
	}

	if len(transfers) > 0 {
		p.app.Log.Info("Auto-closed expired transfers", "count", len(transfers), "org_id", orgID)
	}
}

// escalateTransfers escalates transfers past their escalation deadline
func (p *SLAProcessor) escalateTransfers(orgID uuid.UUID, settings models.ChatbotSettings, now time.Time) {
	var transfers []models.AgentTransfer
	if err := p.app.DB.Where(
		"organization_id = ? AND status = ? AND sla_escalation_at IS NOT NULL AND sla_escalation_at < ? AND escalation_level < 2",
		orgID, "active", now,
	).Find(&transfers).Error; err != nil {
		p.app.Log.Error("Failed to find transfers for escalation", "error", err, "org_id", orgID)
		return
	}

	for _, transfer := range transfers {
		newLevel := transfer.EscalationLevel + 1

		// Update transfer
		updates := map[string]interface{}{
			"escalation_level": newLevel,
			"escalated_at":     now,
		}

		// If not yet breached and past response deadline, mark as breached
		if !transfer.SLABreached && transfer.SLAResponseDeadline != nil && now.After(*transfer.SLAResponseDeadline) {
			updates["sla_breached"] = true
			updates["sla_breached_at"] = now
		}

		if err := p.app.DB.Model(&transfer).Updates(updates).Error; err != nil {
			p.app.Log.Error("Failed to escalate transfer", "error", err, "transfer_id", transfer.ID)
			continue
		}

		p.app.Log.Warn("Transfer escalated",
			"transfer_id", transfer.ID,
			"contact_id", transfer.ContactID,
			"new_level", newLevel,
			"escalation_at", transfer.SLAEscalationAt,
		)

		// Send notification to escalation contacts
		p.notifyEscalation(transfer, settings, newLevel)

		// Broadcast update
		p.broadcastTransferUpdate(transfer, "escalated")

		// Send warning message to customer if configured
		if newLevel == 1 && settings.SLAWarningMessage != "" {
			p.sendSLAWarningToCustomer(transfer, settings.SLAWarningMessage)
		}
	}

	if len(transfers) > 0 {
		p.app.Log.Info("Escalated transfers", "count", len(transfers), "org_id", orgID)
	}
}

// markSLABreached marks transfers as SLA breached when past response deadline
func (p *SLAProcessor) markSLABreached(orgID uuid.UUID, settings models.ChatbotSettings, now time.Time) {
	result := p.app.DB.Model(&models.AgentTransfer{}).Where(
		"organization_id = ? AND status = ? AND sla_breached = ? AND sla_response_deadline IS NOT NULL AND sla_response_deadline < ? AND agent_id IS NULL",
		orgID, "active", false, now,
	).Updates(map[string]interface{}{
		"sla_breached":    true,
		"sla_breached_at": now,
	})

	if result.Error != nil {
		p.app.Log.Error("Failed to mark SLA breached", "error", result.Error, "org_id", orgID)
		return
	}

	if result.RowsAffected > 0 {
		p.app.Log.Warn("Marked transfers as SLA breached", "count", result.RowsAffected, "org_id", orgID)
	}
}

// notifyEscalation sends notifications to escalation contacts via WebSocket broadcast
func (p *SLAProcessor) notifyEscalation(transfer models.AgentTransfer, settings models.ChatbotSettings, level int) {
	if len(settings.SLAEscalationNotifyIDs) == 0 {
		return
	}

	// Get contact info for the notification
	var contact models.Contact
	if err := p.app.DB.Where("id = ?", transfer.ContactID).First(&contact).Error; err != nil {
		p.app.Log.Error("Failed to load contact for escalation notification", "error", err)
		return
	}

	// Prepare notification payload
	levelName := "warning"
	if level >= 2 {
		levelName = "critical"
	}

	// Broadcast escalation notification to the organization
	// Escalation contacts will receive this via the org-wide broadcast
	p.app.WSHub.BroadcastToOrg(transfer.OrganizationID, websocket.WSMessage{
		Type: "transfer_escalation",
		Payload: map[string]interface{}{
			"transfer_id":           transfer.ID.String(),
			"contact_id":            transfer.ContactID.String(),
			"contact_name":          contact.ProfileName,
			"phone_number":          contact.PhoneNumber,
			"escalation_level":      level,
			"level_name":            levelName,
			"waiting_since":         transfer.TransferredAt.Format(time.RFC3339),
			"team_id":               transfer.TeamID,
			"escalation_notify_ids": settings.SLAEscalationNotifyIDs,
		},
	})

	p.app.Log.Info("Escalation notification sent",
		"transfer_id", transfer.ID,
		"level", level,
		"notify_count", len(settings.SLAEscalationNotifyIDs),
	)
}

// sendSLAWarningToCustomer sends a warning message to the customer
func (p *SLAProcessor) sendSLAWarningToCustomer(transfer models.AgentTransfer, message string) {
	// Get WhatsApp account
	var account models.WhatsAppAccount
	if err := p.app.DB.Where("name = ?", transfer.WhatsAppAccount).First(&account).Error; err != nil {
		p.app.Log.Error("Failed to load WhatsApp account for SLA warning", "error", err)
		return
	}

	waAccount := &whatsapp.Account{
		PhoneID:     account.PhoneID,
		BusinessID:  account.BusinessID,
		APIVersion:  account.APIVersion,
		AccessToken: account.AccessToken,
	}

	// Send message
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := p.app.WhatsApp.SendTextMessage(ctx, waAccount, transfer.PhoneNumber, message)
	if err != nil {
		p.app.Log.Error("Failed to send SLA warning message", "error", err, "phone", transfer.PhoneNumber)
		return
	}

	p.app.Log.Info("SLA warning message sent to customer", "phone", transfer.PhoneNumber, "transfer_id", transfer.ID)
}

// sendSLAAutoCloseToCustomer sends an auto-close notification message to the customer
func (p *SLAProcessor) sendSLAAutoCloseToCustomer(transfer models.AgentTransfer, message string) {
	// Get WhatsApp account
	var account models.WhatsAppAccount
	if err := p.app.DB.Where("name = ?", transfer.WhatsAppAccount).First(&account).Error; err != nil {
		p.app.Log.Error("Failed to load WhatsApp account for SLA auto-close message", "error", err)
		return
	}

	waAccount := &whatsapp.Account{
		PhoneID:     account.PhoneID,
		BusinessID:  account.BusinessID,
		APIVersion:  account.APIVersion,
		AccessToken: account.AccessToken,
	}

	// Send message
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := p.app.WhatsApp.SendTextMessage(ctx, waAccount, transfer.PhoneNumber, message)
	if err != nil {
		p.app.Log.Error("Failed to send SLA auto-close message", "error", err, "phone", transfer.PhoneNumber)
		return
	}

	p.app.Log.Info("SLA auto-close message sent to customer", "phone", transfer.PhoneNumber, "transfer_id", transfer.ID)
}

// broadcastTransferUpdate broadcasts transfer update via WebSocket
func (p *SLAProcessor) broadcastTransferUpdate(transfer models.AgentTransfer, eventType string) {
	// Get contact info
	var contact models.Contact
	p.app.DB.Where("id = ?", transfer.ContactID).First(&contact)

	p.app.WSHub.BroadcastToOrg(transfer.OrganizationID, websocket.WSMessage{
		Type: "transfer_" + eventType,
		Payload: map[string]interface{}{
			"id":               transfer.ID.String(),
			"contact_id":       transfer.ContactID.String(),
			"contact_name":     contact.ProfileName,
			"phone_number":     contact.PhoneNumber,
			"status":           transfer.Status,
			"escalation_level": transfer.EscalationLevel,
			"sla_breached":     transfer.SLABreached,
		},
	})
}

// SetSLADeadlines sets SLA deadlines on a new transfer based on settings
func (a *App) SetSLADeadlines(transfer *models.AgentTransfer, settings *models.ChatbotSettings) {
	if !settings.SLAEnabled {
		return
	}

	now := time.Now()

	// Response deadline (time to pick up)
	if settings.SLAResponseMinutes > 0 {
		deadline := now.Add(time.Duration(settings.SLAResponseMinutes) * time.Minute)
		transfer.SLAResponseDeadline = &deadline
	}

	// Resolution deadline
	if settings.SLAResolutionMinutes > 0 {
		deadline := now.Add(time.Duration(settings.SLAResolutionMinutes) * time.Minute)
		transfer.SLAResolutionDeadline = &deadline
	}

	// Escalation deadline
	if settings.SLAEscalationMinutes > 0 {
		deadline := now.Add(time.Duration(settings.SLAEscalationMinutes) * time.Minute)
		transfer.SLAEscalationAt = &deadline
	}

	// Expiry deadline (auto-close)
	if settings.SLAAutoCloseHours > 0 {
		deadline := now.Add(time.Duration(settings.SLAAutoCloseHours) * time.Hour)
		transfer.ExpiresAt = &deadline
	}

	a.Log.Debug("SLA deadlines set",
		"transfer_id", transfer.ID,
		"response_deadline", transfer.SLAResponseDeadline,
		"escalation_at", transfer.SLAEscalationAt,
		"expires_at", transfer.ExpiresAt,
	)
}

// UpdateSLAOnPickup updates SLA tracking when a transfer is picked up
func (a *App) UpdateSLAOnPickup(transfer *models.AgentTransfer) {
	now := time.Now()
	transfer.PickedUpAt = &now

	// Check if SLA was breached (picked up after response deadline)
	if transfer.SLAResponseDeadline != nil && now.After(*transfer.SLAResponseDeadline) {
		transfer.SLABreached = true
		transfer.SLABreachedAt = &now
	}
}

// UpdateSLAOnFirstResponse updates SLA tracking when agent sends first response
func (a *App) UpdateSLAOnFirstResponse(transfer *models.AgentTransfer) {
	if transfer.FirstResponseAt != nil {
		return // Already responded
	}

	now := time.Now()
	transfer.FirstResponseAt = &now
}
