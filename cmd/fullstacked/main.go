package main

import (
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/noor-latif/fulldash/internal/handlers"
	"github.com/noor-latif/fulldash/internal/store"
)

func main() {
	dbPath := getEnv("DB_PATH", "data/fulldash.db")
	port := getEnv("PORT", "8080")

	db, err := store.New(dbPath)
	if err != nil {
		log.Fatalf("DB error: %v", err)
	}
	defer db.Close()

	h := handlers.New(db)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Static files
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Routes
	r.Get("/", h.Dashboard)
	r.Get("/projects/new", h.ProjectForm)
	r.Get("/projects/{id}/edit", h.ProjectForm)
	r.Post("/projects", h.CreateProject)
	r.Put("/projects/{id}", h.UpdateProject)
	r.Delete("/projects/{id}", h.DeleteProject)

	// Stripe webhook
	r.Post("/webhook", h.StripeWebhook)
	r.Get("/payment-link", h.CreatePaymentLink)

	// Health
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	addr := ":" + port
	log.Printf("FullDash on http://localhost%s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func getEnv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
