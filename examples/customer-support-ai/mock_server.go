package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

// MockServer provides test endpoints for customer support workflow
type MockServer struct {
	port string
}

// NewMockServer creates a new mock server instance
func NewMockServer(port string) *MockServer {
	return &MockServer{port: port}
}

// Start starts the mock server
func (s *MockServer) Start() error {
	mux := http.NewServeMux()

	// Account status endpoint
	mux.HandleFunc("/accounts/", s.handleGetAccount)

	// Escalation endpoint
	mux.HandleFunc("/support/escalate", s.handleEscalate)

	// Send response endpoint
	mux.HandleFunc("/support/send", s.handleSendResponse)

	// Analytics logging endpoint
	mux.HandleFunc("/analytics/log", s.handleLogInteraction)

	server := &http.Server{
		Addr:         ":" + s.port,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	log.Printf("Mock server starting on port %s...\n", s.port)
	return server.ListenAndServe()
}

// handleGetAccount handles GET /accounts/{id}
func (s *MockServer) handleGetAccount(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract account ID from path
	accountID := strings.TrimPrefix(r.URL.Path, "/accounts/")
	log.Printf("📋 GET /accounts/%s - Fetching account status\n", accountID)

	// Mock account data based on ID
	var response map[string]any
	switch accountID {
	case "refund_case":
		response = map[string]any{
			"account_id":        accountID,
			"status":            "refund_eligible",
			"subscription_tier": "premium",
			"balance":           -50.00,
			"last_payment":      "2025-11-15",
			"outstanding_issues": []string{"payment_failed"},
		}
	case "suspended":
		response = map[string]any{
			"account_id":        accountID,
			"status":            "suspended",
			"subscription_tier": "free",
			"balance":           0.00,
			"suspension_reason": "terms_violation",
		}
	default:
		response = map[string]any{
			"account_id":        accountID,
			"status":            "active",
			"subscription_tier": "basic",
			"balance":           120.50,
			"last_payment":      "2025-11-20",
			"next_billing_date": "2025-12-20",
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)

	log.Printf("✅ Account status returned: %s\n", response["status"])
}

// handleEscalate handles POST /support/escalate
func (s *MockServer) handleEscalate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req map[string]any
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	log.Printf("🚨 POST /support/escalate - Escalating ticket\n")
	log.Printf("   Customer: %v\n", req["customer_info"])
	log.Printf("   Type: %v\n", req["inquiry_type"])
	log.Printf("   Sentiment: %v\n", req["sentiment"])

	// Generate escalation ticket
	ticketID := fmt.Sprintf("ESC-%d", time.Now().Unix())
	response := map[string]any{
		"ticket_id":    ticketID,
		"status":       "escalated",
		"assigned_to":  "senior_support_team",
		"priority":     req["priority"],
		"created_at":   time.Now().Format(time.RFC3339),
		"estimated_response": "2 hours",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)

	log.Printf("✅ Ticket escalated: %s\n", ticketID)
}

// handleSendResponse handles POST /support/send
func (s *MockServer) handleSendResponse(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req map[string]any
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	log.Printf("📧 POST /support/send - Sending response to customer\n")
	log.Printf("   Email: %v\n", req["customer_email"])
	log.Printf("   Subject: %v\n", req["subject"])
	log.Printf("   Ticket: %v\n", req["ticket_id"])

	// Mock email sending
	messageID := fmt.Sprintf("MSG-%d", time.Now().Unix())
	response := map[string]any{
		"message_id":  messageID,
		"status":      "sent",
		"sent_at":     time.Now().Format(time.RFC3339),
		"delivery_status": "delivered",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)

	log.Printf("✅ Response sent: %s\n", messageID)
}

// handleLogInteraction handles POST /analytics/log
func (s *MockServer) handleLogInteraction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req map[string]any
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	log.Printf("📊 POST /analytics/log - Logging interaction\n")
	log.Printf("   Type: %v\n", req["inquiry_type"])
	log.Printf("   Sentiment: %v\n", req["sentiment"])
	log.Printf("   Escalated: %v\n", req["escalated"])
	log.Printf("   Quality Score: %v\n", req["quality_score"])

	// Mock analytics logging
	response := map[string]any{
		"log_id":     fmt.Sprintf("LOG-%d", time.Now().Unix()),
		"status":     "logged",
		"logged_at":  time.Now().Format(time.RFC3339),
		"indexed":    true,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)

	log.Printf("✅ Interaction logged\n")
}
