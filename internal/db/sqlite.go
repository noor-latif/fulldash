// db/sqlite.go - Database operations
package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/noor-latif/fulldash/internal/models"
	_ "modernc.org/sqlite"
)

// DB wraps sql.DB with our methods
type DB struct {
	*sql.DB
}

// New creates/opens database and runs migrations
func New(dbPath string) (*DB, error) {
	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create db dir: %w", err)
	}

	sqlDB, err := sql.Open("sqlite", dbPath+"?_pragma=foreign_keys(1)")
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	db := &DB{sqlDB}
	if err := db.migrate(); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return db, nil
}

// migrate creates tables
func (db *DB) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS projects (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		description TEXT,
		client TEXT,
		secured_by TEXT CHECK(secured_by IN ('noor', 'ahmad', 'both')) DEFAULT 'both',
		amount_cents INTEGER DEFAULT 0,
		revenue REAL DEFAULT 0,
		status TEXT CHECK(status IN ('pending', 'paid', 'done')) DEFAULT 'pending',
		stripe_payment_id TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS contributions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		project_id INTEGER NOT NULL,
		person TEXT CHECK(person IN ('noor', 'ahmad')) NOT NULL,
		hours REAL DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_projects_status ON projects(status);
	CREATE INDEX IF NOT EXISTS idx_contributions_project ON contributions(project_id);
	`

	if _, err := db.Exec(schema); err != nil {
		return err
	}
	return nil
}

// CreateProject inserts a new project
func (db *DB) CreateProject(p *models.Project) error {
	query := `
		INSERT INTO projects (name, description, client, secured_by, amount_cents, revenue, status)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		RETURNING id, created_at, updated_at
	`
	return db.QueryRow(query, p.Name, p.Description, p.Client, p.SecuredBy, 
		p.AmountCents, p.Revenue, p.Status).Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt)
}

// GetProjectByID fetches a single project
func (db *DB) GetProjectByID(id int64) (*models.Project, error) {
	p := &models.Project{}
	query := `SELECT id, name, description, client, secured_by, amount_cents, revenue, 
		status, stripe_payment_id, created_at, updated_at FROM projects WHERE id = ?`
	
	err := db.QueryRow(query, id).Scan(&p.ID, &p.Name, &p.Description, &p.Client,
		&p.SecuredBy, &p.AmountCents, &p.Revenue, &p.Status, &p.StripePaymentID,
		&p.CreatedAt, &p.UpdatedAt)
	
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return p, err
}

// UpdateProject updates project fields
func (db *DB) UpdateProject(p *models.Project) error {
	query := `
		UPDATE projects SET name=?, description=?, client=?, secured_by=?, 
		amount_cents=?, revenue=?, status=?, stripe_payment_id=?, updated_at=CURRENT_TIMESTAMP
		WHERE id=?
	`
	_, err := db.Exec(query, p.Name, p.Description, p.Client, p.SecuredBy,
		p.AmountCents, p.Revenue, p.Status, p.StripePaymentID, p.ID)
	return err
}

// UpdateProjectStatus updates only status and payment info (for webhooks)
func (db *DB) UpdateProjectStatus(id int64, status models.ProjectStatus, revenue float64, stripeID string) error {
	query := `UPDATE projects SET status=?, revenue=?, stripe_payment_id=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`
	_, err := db.Exec(query, status, revenue, stripeID, id)
	return err
}

// DeleteProject removes a project (cascades to contributions)
func (db *DB) DeleteProject(id int64) error {
	_, err := db.Exec(`DELETE FROM projects WHERE id = ?`, id)
	return err
}

// ListProjectsByStatus returns projects filtered by status
func (db *DB) ListProjectsByStatus(status models.ProjectStatus) ([]models.Project, error) {
	query := `SELECT id, name, description, client, secured_by, amount_cents, revenue, 
		status, stripe_payment_id, created_at, updated_at FROM projects WHERE status = ? ORDER BY created_at DESC`
	
	rows, err := db.Query(query, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanProjects(rows)
}

// ListAllProjects returns all projects
func (db *DB) ListAllProjects() ([]models.Project, error) {
	query := `SELECT id, name, description, client, secured_by, amount_cents, revenue, 
		status, stripe_payment_id, created_at, updated_at FROM projects ORDER BY created_at DESC`
	
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanProjects(rows)
}

// GetContributionsByProject returns all contributions for a project
func (db *DB) GetContributionsByProject(projectID int64) ([]models.Contribution, error) {
	query := `SELECT id, project_id, person, hours, created_at FROM contributions WHERE project_id = ?`
	rows, err := db.Query(query, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var contribs []models.Contribution
	for rows.Next() {
		var c models.Contribution
		if err := rows.Scan(&c.ID, &c.ProjectID, &c.Person, &c.Hours, &c.CreatedAt); err != nil {
			return nil, err
		}
		contribs = append(contribs, c)
	}
	return contribs, rows.Err()
}

// SetContribution inserts or updates contribution for a person on a project
func (db *DB) SetContribution(projectID int64, person models.Owner, hours float64) error {
	// Delete existing
	_, err := db.Exec(`DELETE FROM contributions WHERE project_id = ? AND person = ?`, projectID, person)
	if err != nil {
		return err
	}
	
	// Insert new if hours > 0
	if hours > 0 {
		_, err = db.Exec(`INSERT INTO contributions (project_id, person, hours) VALUES (?, ?, ?)`,
			projectID, person, hours)
	}
	return err
}

// DeleteContributions removes all contributions for a project
func (db *DB) DeleteContributions(projectID int64) error {
	_, err := db.Exec(`DELETE FROM contributions WHERE project_id = ?`, projectID)
	return err
}

// GetDashboardStats returns aggregated stats
func (db *DB) GetDashboardStats() (*models.DashboardStats, error) {
	stats := &models.DashboardStats{}
	
	// Count projects
	err := db.QueryRow(`SELECT COUNT(*), 
		SUM(CASE WHEN status = 'pending' THEN 1 ELSE 0 END),
		SUM(CASE WHEN status = 'paid' THEN 1 ELSE 0 END),
		SUM(revenue)
		FROM projects`).Scan(
		&stats.TotalProjects, &stats.PendingProjects, &stats.PaidProjects, &stats.TotalRevenue)
	
	if err != nil {
		return nil, err
	}

	// Calculate shares based on paid projects
	projects, err := db.ListProjectsByStatus(models.StatusPaid)
	if err != nil {
		return nil, err
	}

	for _, p := range projects {
		split := calculateSplit(&p, nil) // nil contributions = use owner split
		stats.NoorShare += split.NoorShare
		stats.AhmadShare += split.AhmadShare
	}

	return stats, nil
}

// scanProjects helper
func scanProjects(rows *sql.Rows) ([]models.Project, error) {
	var projects []models.Project
	for rows.Next() {
		var p models.Project
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.Client, &p.SecuredBy,
			&p.AmountCents, &p.Revenue, &p.Status, &p.StripePaymentID, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}
	return projects, rows.Err()
}

// calculateSplit determines revenue sharing
func calculateSplit(p *models.Project, contribs []models.Contribution) models.RevenueSplit {
	if p.Revenue <= 0 {
		return models.RevenueSplit{SplitMethod: "none"}
	}

	// If contributions exist, use hours-based split
	if len(contribs) > 0 {
		var noorHours, ahmadHours float64
		for _, c := range contribs {
			if c.Person == models.OwnerNoor {
				noorHours += c.Hours
			} else {
				ahmadHours += c.Hours
			}
		}
		totalHours := noorHours + ahmadHours
		if totalHours > 0 {
			return models.RevenueSplit{
				NoorShare:   p.Revenue * (noorHours / totalHours),
				AhmadShare:  p.Revenue * (ahmadHours / totalHours),
				SplitMethod: "hours",
			}
		}
	}

	// Fallback: split by owner
	switch p.SecuredBy {
	case models.OwnerNoor:
		return models.RevenueSplit{NoorShare: p.Revenue, AhmadShare: 0, SplitMethod: "owner"}
	case models.OwnerAhmad:
		return models.RevenueSplit{NoorShare: 0, AhmadShare: p.Revenue, SplitMethod: "owner"}
	default: // both
		half := p.Revenue / 2
		return models.RevenueSplit{NoorShare: half, AhmadShare: half, SplitMethod: "owner"}
	}
}

// Helper for handlers to get full project with contributions
func (db *DB) GetProjectFull(id int64) (*models.ProjectWithContributions, error) {
	p, err := db.GetProjectByID(id)
	if err != nil || p == nil {
		return nil, err
	}

	contribs, err := db.GetContributionsByProject(id)
	if err != nil {
		return nil, err
	}

	split := calculateSplit(p, contribs)

	return &models.ProjectWithContributions{
		Project:       *p,
		Contributions: contribs,
		Split:         split,
	}, nil
}

// LogError is a simple error logger
func LogError(msg string, err error) {
	if err != nil {
		log.Printf("[ERROR] %s: %v", msg, err)
	}
}
