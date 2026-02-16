// handlers/web.go - HTTP handlers (DRY refactored)
package handlers

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/noor-latif/fulldash/internal/models"
	"github.com/noor-latif/fulldash/internal/templates"
)

// Store defines the interface for data operations (enables mocking)
type Store interface {
	CreateProject(p *models.Project) error
	GetProject(id int64) (*models.Project, error)
	GetProjectByStripeID(stripeID string) (*models.Project, error)
	UpdateProject(p *models.Project) error
	UpdateProjectStatus(id int64, status models.ProjectStatus, revenue float64, stripeID string) error
	DeleteProject(id int64) error
	ListProjects(search string) ([]models.Project, error)
	GetMetrics() (*models.Metrics, error)
	GetContributions(projectID int64) ([]models.Contribution, error)
	SetContribution(c *models.Contribution) error
}

// Handler holds dependencies
type Handler struct {
	DB Store
}

// New creates a new Handler
func New(db Store) *Handler {
	return &Handler{DB: db}
}

// Dashboard renders the main dashboard with kanban
func (h *Handler) Dashboard(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("search")
	
	projects, err := h.DB.ListProjects(search)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	metrics, err := h.DB.GetMetrics()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	new, progress, done, paid := splitByStatus(projects)

	if r.Header.Get("HX-Request") == "true" {
		templates.KanbanBoard(new, progress, done, paid).Render(r.Context(), w)
	} else {
		templates.Layout("FullDash", 
			templates.Dashboard(metrics, new, progress, done, paid, search)).Render(r.Context(), w)
	}
}

// splitByStatus groups projects by their status
func splitByStatus(projects []models.Project) (new, progress, done, paid []models.Project) {
	for _, p := range projects {
		switch p.Status {
		case models.StatusNew:
			new = append(new, p)
		case models.StatusProgress:
			progress = append(progress, p)
		case models.StatusDone:
			done = append(done, p)
		case models.StatusPaid:
			paid = append(paid, p)
		}
	}
	return
}

// ProjectForm renders the add/edit form
func (h *Handler) ProjectForm(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	
	var p *models.Project
	var noorHours, ahmadHours float64
	isEdit := idStr != ""
	
	if isEdit {
		if id, err := strconv.ParseInt(idStr, 10, 64); err == nil {
			p, _ = h.DB.GetProject(id)
			if p != nil {
				noorHours, ahmadHours = h.getHours(p.ID)
			}
		}
	}
	
	if p == nil {
		p = &models.Project{Status: models.StatusNew, SecuredBy: models.OwnerBoth}
	}
	
	templates.ProjectForm(p, isEdit, noorHours, ahmadHours).Render(r.Context(), w)
}

// getHours retrieves contribution hours for both owners
func (h *Handler) getHours(projectID int64) (noorHours, ahmadHours float64) {
	contribs, _ := h.DB.GetContributions(projectID)
	for _, c := range contribs {
		switch c.Owner {
		case models.OwnerNoor:
			noorHours = c.Hours
		case models.OwnerAhmad:
			ahmadHours = c.Hours
		}
	}
	return
}

// CreateProject handles new project creation
func (h *Handler) CreateProject(w http.ResponseWriter, r *http.Request) {
	form, err := parseProjectForm(r)
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	p := form.toProject()
	if err := h.DB.CreateProject(p); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Save contributions (even zero hours, for consistency)
	if err := form.saveContributions(h.DB, p.ID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.Dashboard(w, r)
}

// UpdateProject handles project updates
func (h *Handler) UpdateProject(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	p, err := h.DB.GetProject(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if p == nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	form, err := parseProjectForm(r)
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	form.applyTo(p)
	if err := h.DB.UpdateProject(p); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Update contributions (even zero hours, to clear old values)
	if err := form.saveContributions(h.DB, p.ID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.Dashboard(w, r)
}

// DeleteProject handles project deletion
func (h *Handler) DeleteProject(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	
	if err := h.DB.DeleteProject(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.Dashboard(w, r)
}
