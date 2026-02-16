// cmd/fullstacked/main.go - Entry point
package main

import (
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/noor-latif/fulldash/internal/db"
	"github.com/noor-latif/fulldash/internal/handlers"
)

func main() {
	// Config
	dbPath := getEnv("DB_PATH", "data/fulldash.db")
	port := getEnv("PORT", "8080")

	// Init database
	database, err := db.New(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()
	log.Printf("Database initialized: %s", dbPath)

	// Init handlers
	handler, err := handlers.NewHandler(database)
	if err != nil {
		log.Fatalf("Failed to load templates: %v", err)
	}

	// Setup router
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RealIP)

	// Static files
	fs := http.FileServer(http.Dir("web/static"))
	r.Handle("/static/*", http.StripPrefix("/static/", fs))

	// Routes
	r.Get("/", handler.Dashboard)
	r.Get("/dashboard", handler.Dashboard)
	
	// Project routes
	r.Get("/projects/new", handler.ProjectForm)
	r.Get("/projects/{id}/edit", handler.ProjectForm)
	r.Get("/projects/{id}/card", handler.ProjectCard)
	r.Get("/projects/{id}/revenue", handler.RevenueDetails)
	r.Post("/projects", handler.CreateProject)
	r.Put("/projects/{id}", handler.UpdateProject)
	r.Delete("/projects/{id}", handler.DeleteProject)
	r.Post("/projects/{id}/move/{status}", handler.MoveProject)

	// Stripe webhook
	r.Post("/webhook", handler.StripeWebhook)
	r.Get("/payment-link", handler.CreatePaymentLink)

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	// Start server
	addr := ":" + port
	log.Printf("🚀 FullDash starting on http://localhost%s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
