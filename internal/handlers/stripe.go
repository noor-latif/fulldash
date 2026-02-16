// handlers/stripe.go - Stripe webhook handler
package handlers

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/noor-latif/fulldash/internal/models"
)

// StripeWebhook handles Stripe payment events
func (h *Handler) StripeWebhook(w http.ResponseWriter, r *http.Request) {
	// Always return 200 to Stripe to prevent retries
	defer func() {
		w.WriteHeader(http.StatusOK)
	}()

	// Read and verify webhook (simplified - production should verify signature)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("[STRIPE] Error reading body: %v", err)
		return
	}

	var event map[string]interface{}
	if err := json.Unmarshal(body, &event); err != nil {
		log.Printf("[STRIPE] Error parsing JSON: %v", err)
		return
	}

	eventType, _ := event["type"].(string)
	log.Printf("[STRIPE] Received event: %s", eventType)

	// Only process payment_intent.succeeded
	if eventType != "payment_intent.succeeded" {
		log.Printf("[STRIPE] Ignoring event type: %s", eventType)
		return
	}

	// Extract data
	data, ok := event["data"].(map[string]interface{})
	if !ok {
		log.Printf("[STRIPE] Missing data field")
		return
	}

	obj, ok := data["object"].(map[string]interface{})
	if !ok {
		log.Printf("[STRIPE] Missing object field")
		return
	}

	// Get payment intent ID
	paymentID, _ := obj["id"].(string)
	
	// Get amount (in cents)
	var amount int64
	if amt, ok := obj["amount"].(float64); ok {
		amount = int64(amt)
	}
	if amt, ok := obj["amount_received"].(float64); ok {
		amount = int64(amt)
	}

	// Get metadata for project_id
	metadata, _ := obj["metadata"].(map[string]interface{})
	projectIDStr, _ := metadata["project_id"].(string)
	
	if projectIDStr == "" {
		log.Printf("[STRIPE] No project_id in metadata, skipping")
		return
	}

	projectID, err := strconv.ParseInt(projectIDStr, 10, 64)
	if err != nil {
		log.Printf("[STRIPE] Invalid project_id: %s", projectIDStr)
		return
	}

	// Update database
	revenue := float64(amount) / 100.0
	err = h.DB.UpdateProjectStatus(projectID, models.StatusPaid, revenue, paymentID)
	if err != nil {
		log.Printf("[STRIPE] Failed to update project %d: %v", projectID, err)
		return
	}

	log.Printf("[STRIPE] ✅ Project %d marked as paid (%.2f USD, payment: %s)", 
		projectID, revenue, paymentID)
}

// VerifyStripeSignature would verify webhook signature in production
func VerifyStripeSignature(payload []byte, sigHeader string) bool {
	// For now, skip verification - add STRIPE_WEBHOOK_SECRET check in production
	// See: https://stripe.com/docs/webhooks/signatures
	webhookSecret := os.Getenv("STRIPE_WEBHOOK_SECRET")
	if webhookSecret == "" {
		log.Printf("[STRIPE] Warning: No STRIPE_WEBHOOK_SECRET set, skipping signature verification")
		return true
	}
	// TODO: Implement actual signature verification
	return true
}

// CreatePaymentLink creates a Stripe payment link for a project
// This is a placeholder for the future feature
func (h *Handler) CreatePaymentLink(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("project_id")
	if idStr == "" {
		http.Error(w, "Missing project_id", http.StatusBadRequest)
		return
	}

	projectID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid project_id", http.StatusBadRequest)
		return
	}

	project, err := h.DB.GetProjectByID(projectID)
	if err != nil || project == nil {
		http.Error(w, "Project not found", http.StatusNotFound)
		return
	}

	// Return a simple payment link structure
	// In production, this would call Stripe API to create a real payment link
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"project_id":   projectID,
		"amount_cents": project.AmountCents,
		"payment_link": "https://dashboard.stripe.com/test/payments", // Placeholder
		"note":         "Use Stripe Dashboard to create payment links manually for now",
	})
}

// LogStripeEvent for debugging
func LogStripeEvent(eventType string, data map[string]interface{}) {
	if os.Getenv("DEBUG") == "true" {
		log.Printf("[STRIPE DEBUG] Event: %s", eventType)
		log.Printf("[STRIPE DEBUG] Data: %+v", data)
	}
}

// WebhookEvent represents a Stripe webhook event
type WebhookEvent struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Data      WebhookData            `json:"data"`
	CreatedAt int64                  `json:"created"`
	Metadata  map[string]interface{} `json:"metadata"`
}

type WebhookData struct {
	Object map[string]interface{} `json:"object"`
}

// ParseStripeEvent parses webhook body into structured event
func ParseStripeEvent(body []byte) (*WebhookEvent, error) {
	var event WebhookEvent
	if err := json.Unmarshal(body, &event); err != nil {
		return nil, err
	}
	return &event, nil
}
