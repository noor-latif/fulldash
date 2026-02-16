package handlers

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/stripe/stripe-go/v84"
	"github.com/stripe/stripe-go/v84/webhook"
)

func (h *Handler) StripeWebhook(w http.ResponseWriter, r *http.Request) {
	// Always return 200 to Stripe
	w.WriteHeader(http.StatusOK)

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("[STRIPE] Read error: %v", err)
		return
	}

	sigHeader := r.Header.Get("Stripe-Signature")
	webhookSecret := os.Getenv("STRIPE_WEBHOOK_SECRET")

	var event stripe.Event
	
	if webhookSecret != "" {
		event, err = webhook.ConstructEvent(body, sigHeader, webhookSecret)
		if err != nil {
			log.Printf("[STRIPE] Signature verify failed: %v", err)
			return
		}
	} else {
		// Dev mode: parse without verification
		if err := json.Unmarshal(body, &event); err != nil {
			log.Printf("[STRIPE] Parse error: %v", err)
			return
		}
		log.Printf("[STRIPE] Warning: No WEBHOOK_SECRET, skipping signature verify")
	}

	log.Printf("[STRIPE] Event: %s", event.Type)

	switch event.Type {
	case "payment_intent.succeeded":
		h.handlePaymentIntentSucceeded(event)
	case "charge.succeeded":
		h.handleChargeSucceeded(event)
	case "invoice.paid":
		h.handleInvoicePaid(event)
	}
}

func (h *Handler) handlePaymentIntentSucceeded(event stripe.Event) {
	var pi stripe.PaymentIntent
	if err := json.Unmarshal(event.Data.Raw, &pi); err != nil {
		log.Printf("[STRIPE] Unmarshal error: %v", err)
		return
	}

	projectID := pi.Metadata["project_id"]
	if projectID == "" {
		log.Printf("[STRIPE] No project_id in metadata")
		return
	}

	// Find project by stripe_payment_id or metadata
	// For now, we use metadata to link
	log.Printf("[STRIPE] Payment succeeded for project %s: %.2f %s", 
		projectID, float64(pi.AmountReceived)/100, pi.Currency)
	
	// TODO: Link to project and update status
	// This requires the project to have been created with the stripe_payment_id
	// or we look up by client name match
}

func (h *Handler) handleChargeSucceeded(event stripe.Event) {
	var charge stripe.Charge
	if err := json.Unmarshal(event.Data.Raw, &charge); err != nil {
		return
	}
	
	// Try to find project by payment intent in metadata
	if charge.PaymentIntent != nil {
		// Look up project
		log.Printf("[STRIPE] Charge succeeded: %s", charge.ID)
	}
}

func (h *Handler) handleInvoicePaid(event stripe.Event) {
	var invoice stripe.Invoice
	if err := json.Unmarshal(event.Data.Raw, &invoice); err != nil {
		return
	}
	
	projectID := invoice.Metadata["project_id"]
	if projectID == "" {
		return
	}

	// Find and update project
	// For now, log it
	log.Printf("[STRIPE] Invoice paid for project %s: %.2f", 
		projectID, float64(invoice.AmountPaid)/100)
}

// CreatePaymentLink placeholder for future Stripe integration
func (h *Handler) CreatePaymentLink(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"note": "Stripe payment links not yet implemented",
		"action": "Use Stripe Dashboard to create payment links",
	})
}
