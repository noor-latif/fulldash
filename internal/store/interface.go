// store/interface.go - Store interface for testability
package store

import "github.com/noor-latif/fulldash/internal/models"

type Store interface {
	// Projects
	CreateProject(p *models.Project) error
	GetProject(id int64) (*models.Project, error)
	GetProjectByStripeID(stripeID string) (*models.Project, error)
	UpdateProject(p *models.Project) error
	UpdateProjectStatus(id int64, status models.ProjectStatus, revenue float64, stripeID string) error
	DeleteProject(id int64) error
	ListProjects(search string) ([]models.Project, error)
	ListProjectsByStatus(status models.ProjectStatus) ([]models.Project, error)
	
	// Contributions
	GetContributions(projectID int64) ([]models.Contribution, error)
	SetContribution(c *models.Contribution) error
	
	// Metrics
	GetMetrics() (*models.Metrics, error)
}
