// handlers/forms.go - Form parsing helpers for DRY
package handlers

import (
	"net/http"
	"strconv"

	"github.com/noor-latif/fulldash/internal/models"
)

// ParsedForm holds all form values for project creation/update
type ParsedForm struct {
	Client      string
	Description string
	SecuredBy   models.Owner
	Status      models.ProjectStatus
	Revenue     float64
	NoorHours   float64
	AhmadHours  float64
}

// parseProjectForm extracts and validates form data
func parseProjectForm(r *http.Request) (*ParsedForm, error) {
	if err := r.ParseForm(); err != nil {
		return nil, err
	}

	revenue, _ := strconv.ParseFloat(r.FormValue("revenue"), 64)
	noorHours, _ := strconv.ParseFloat(r.FormValue("noor_hours"), 64)
	ahmadHours, _ := strconv.ParseFloat(r.FormValue("ahmad_hours"), 64)

	status := models.ProjectStatus(r.FormValue("status"))
	if status == "" {
		status = models.StatusNew
	}

	return &ParsedForm{
		Client:      r.FormValue("client"),
		Description: r.FormValue("description"),
		SecuredBy:   models.Owner(r.FormValue("secured_by")),
		Status:      status,
		Revenue:     revenue,
		NoorHours:   noorHours,
		AhmadHours:  ahmadHours,
	}, nil
}

// toProject converts form data to Project model
func (f *ParsedForm) toProject() *models.Project {
	return &models.Project{
		Client:      f.Client,
		Description: f.Description,
		SecuredBy:   f.SecuredBy,
		Status:      f.Status,
		Revenue:     f.Revenue,
	}
}

// contribution returns contribution for given owner (if hours > 0)
func (f *ParsedForm) contribution(owner models.Owner, projectID int64) *models.Contribution {
	hours := f.NoorHours
	if owner == models.OwnerAhmad {
		hours = f.AhmadHours
	}
	
	if hours <= 0 {
		return nil
	}
	
	return &models.Contribution{
		ProjectID: projectID,
		Owner:     owner,
		Hours:     hours,
	}
}

// applyTo updates a project from form values
func (f *ParsedForm) applyTo(p *models.Project) {
	p.Client = f.Client
	p.Description = f.Description
	p.SecuredBy = f.SecuredBy
	p.Status = f.Status
	p.Revenue = f.Revenue
}

// saveContributions saves both Noor and Ahmad contributions
func (f *ParsedForm) saveContributions(db interface{ SetContribution(c *models.Contribution) error }, projectID int64) error {
	for _, owner := range []models.Owner{models.OwnerNoor, models.OwnerAhmad} {
		if c := f.contribution(owner, projectID); c != nil {
			if err := db.SetContribution(c); err != nil {
				return err
			}
		}
	}
	return nil
}
