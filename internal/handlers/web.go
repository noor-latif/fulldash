// handlers/web.go - HTTP handlers for web UI
package handlers

import (
	"html/template"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/noor-latif/fulldash/internal/db"
	"github.com/noor-latif/fulldash/internal/models"
)

// Handler holds dependencies
type Handler struct {
	DB        *db.DB
	Templates *template.Template
}

// NewHandler creates handler with loaded templates
func NewHandler(database *db.DB) (*Handler, error) {
	tmpl, err := template.New("").Funcs(TemplateFuncs()).ParseGlob("web/templates/*.html")
	if err != nil {
		return nil, err
	}

	return &Handler{
		DB:        database,
		Templates: tmpl,
	}, nil
}

// Dashboard shows the main page with kanban and metrics
func (h *Handler) Dashboard(w http.ResponseWriter, r *http.Request) {
	stats, err := h.DB.GetDashboardStats()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	pending, _ := h.DB.ListProjectsByStatus(models.StatusPending)
	paid, _ := h.DB.ListProjectsByStatus(models.StatusPaid)
	done, _ := h.DB.ListProjectsByStatus(models.StatusDone)

	data := struct {
		Stats   *models.DashboardStats
		Pending []models.Project
		Paid    []models.Project
		Done    []models.Project
	}{
		Stats:   stats,
		Pending: pending,
		Paid:    paid,
		Done:    done,
	}

	if r.Header.Get("HX-Request") == "true" {
		h.Templates.ExecuteTemplate(w, "kanban", data)
	} else {
		h.Templates.ExecuteTemplate(w, "dashboard", data)
	}
}

// ProjectCard renders a single project card (for HTMX swaps)
func (h *Handler) ProjectCard(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	full, err := h.DB.GetProjectFull(id)
	if err != nil || full == nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	h.Templates.ExecuteTemplate(w, "project-card", full)
}

// ProjectForm shows add/edit form
func (h *Handler) ProjectForm(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	
	var project *models.ProjectWithContributions
	if idStr != "" {
		id, _ := strconv.ParseInt(idStr, 10, 64)
		project, _ = h.DB.GetProjectFull(id)
	}

	data := struct {
		Project *models.ProjectWithContributions
		IsEdit  bool
	}{
		Project: project,
		IsEdit:  idStr != "",
	}

	h.Templates.ExecuteTemplate(w, "project-form", data)
}

// CreateProject handles POST /projects
func (h *Handler) CreateProject(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	amount, _ := strconv.ParseInt(r.FormValue("amount_cents"), 10, 64)
	initialRevenue, _ := strconv.ParseFloat(r.FormValue("initial_revenue"), 64)

	project := &models.Project{
		Name:        r.FormValue("name"),
		Description: r.FormValue("description"),
		Client:      r.FormValue("client"),
		SecuredBy:   models.Owner(r.FormValue("secured_by")),
		AmountCents: amount,
		Revenue:     initialRevenue,
		Status:      models.StatusPending,
	}

	if err := h.DB.CreateProject(project); err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	// Handle contributions
	noorHours, _ := strconv.ParseFloat(r.FormValue("noor_hours"), 64)
	ahmadHours, _ := strconv.ParseFloat(r.FormValue("ahmad_hours"), 64)
	
	if noorHours > 0 {
		h.DB.SetContribution(project.ID, models.OwnerNoor, noorHours)
	}
	if ahmadHours > 0 {
		h.DB.SetContribution(project.ID, models.OwnerAhmad, ahmadHours)
	}

	// Return updated kanban
	h.Dashboard(w, r)
}

// UpdateProject handles PUT /projects/:id
func (h *Handler) UpdateProject(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	project, err := h.DB.GetProjectByID(id)
	if err != nil || project == nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	amount, _ := strconv.ParseInt(r.FormValue("amount_cents"), 10, 64)
	revenue, _ := strconv.ParseFloat(r.FormValue("revenue"), 64)

	project.Name = r.FormValue("name")
	project.Description = r.FormValue("description")
	project.Client = r.FormValue("client")
	project.SecuredBy = models.Owner(r.FormValue("secured_by"))
	project.AmountCents = amount
	project.Revenue = revenue
	project.Status = models.ProjectStatus(r.FormValue("status"))

	if err := h.DB.UpdateProject(project); err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	// Update contributions
	noorHours, _ := strconv.ParseFloat(r.FormValue("noor_hours"), 64)
	ahmadHours, _ := strconv.ParseFloat(r.FormValue("ahmad_hours"), 64)
	
	h.DB.SetContribution(project.ID, models.OwnerNoor, noorHours)
	h.DB.SetContribution(project.ID, models.OwnerAhmad, ahmadHours)

	h.Dashboard(w, r)
}

// DeleteProject handles DELETE /projects/:id
func (h *Handler) DeleteProject(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	
	if err := h.DB.DeleteProject(id); err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	h.Dashboard(w, r)
}

// MoveProject changes status (drag & drop)
func (h *Handler) MoveProject(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	status := models.ProjectStatus(chi.URLParam(r, "status"))

	project, err := h.DB.GetProjectByID(id)
	if err != nil || project == nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	project.Status = status
	project.UpdatedAt = time.Now()
	
	if err := h.DB.UpdateProject(project); err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// RevenueDetails returns split calculation for a project
func (h *Handler) RevenueDetails(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	full, err := h.DB.GetProjectFull(id)
	if err != nil || full == nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	h.Templates.ExecuteTemplate(w, "revenue-details", full)
}
