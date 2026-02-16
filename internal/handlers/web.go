package handlers

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/noor-latif/fulldash/internal/models"
	"github.com/noor-latif/fulldash/internal/store"
	"github.com/noor-latif/fulldash/internal/templates"
)

type Handler struct {
	DB *store.DB
}

func New(db *store.DB) *Handler {
	return &Handler{DB: db}
}

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

	// Split by status
	var new, progress, done, paid []models.Project
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

	// Check if HTMX request
	if r.Header.Get("HX-Request") == "true" {
		templates.KanbanBoard(new, progress, done, paid).Render(r.Context(), w)
	} else {
		templates.Layout("FullDash", templates.Dashboard(metrics, new, progress, done, paid, search)).Render(r.Context(), w)
	}
}

func (h *Handler) ProjectForm(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	
	var p *models.Project
	var noorHours, ahmadHours float64
	isEdit := idStr != ""
	
	if isEdit {
		id, _ := strconv.ParseInt(idStr, 10, 64)
		p, _ = h.DB.GetProject(id)
		if p != nil {
			contribs, _ := h.DB.GetContributions(p.ID)
			for _, c := range contribs {
				if c.Owner == models.OwnerNoor {
					noorHours = c.Hours
				} else {
					ahmadHours = c.Hours
				}
			}
		}
	}
	
	if p == nil {
		p = &models.Project{Status: models.StatusNew, SecuredBy: models.OwnerBoth}
	}
	
	templates.ProjectForm(p, isEdit, noorHours, ahmadHours).Render(r.Context(), w)
}

func (h *Handler) CreateProject(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	revenue, _ := strconv.ParseFloat(r.FormValue("revenue"), 64)
	
	p := &models.Project{
		Client:    r.FormValue("client"),
		Description: r.FormValue("description"),
		SecuredBy: models.Owner(r.FormValue("secured_by")),
		Status:    models.ProjectStatus(r.FormValue("status")),
		Revenue:   revenue,
	}
	
	if p.Status == "" {
		p.Status = models.StatusNew
	}

	if err := h.DB.CreateProject(p); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Save contributions
	noorHours, _ := strconv.ParseFloat(r.FormValue("noor_hours"), 64)
	ahmadHours, _ := strconv.ParseFloat(r.FormValue("ahmad_hours"), 64)
	
	if noorHours > 0 {
		h.DB.SetContribution(&models.Contribution{ProjectID: p.ID, Owner: models.OwnerNoor, Hours: noorHours})
	}
	if ahmadHours > 0 {
		h.DB.SetContribution(&models.Contribution{ProjectID: p.ID, Owner: models.OwnerAhmad, Hours: ahmadHours})
	}

	h.Dashboard(w, r)
}

func (h *Handler) UpdateProject(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	p, err := h.DB.GetProject(id)
	if err != nil || p == nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	revenue, _ := strconv.ParseFloat(r.FormValue("revenue"), 64)
	
	p.Client = r.FormValue("client")
	p.Description = r.FormValue("description")
	p.SecuredBy = models.Owner(r.FormValue("secured_by"))
	p.Status = models.ProjectStatus(r.FormValue("status"))
	p.Revenue = revenue

	if err := h.DB.UpdateProject(p); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Update contributions
	noorHours, _ := strconv.ParseFloat(r.FormValue("noor_hours"), 64)
	ahmadHours, _ := strconv.ParseFloat(r.FormValue("ahmad_hours"), 64)
	
	h.DB.SetContribution(&models.Contribution{ProjectID: p.ID, Owner: models.OwnerNoor, Hours: noorHours})
	h.DB.SetContribution(&models.Contribution{ProjectID: p.ID, Owner: models.OwnerAhmad, Hours: ahmadHours})

	h.Dashboard(w, r)
}

func (h *Handler) DeleteProject(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	
	if err := h.DB.DeleteProject(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.Dashboard(w, r)
}
