// store/db.go - Database operations (DRY refactored)
package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"github.com/noor-latif/fulldash/internal/models"
	_ "modernc.org/sqlite"
)

// Compile-time check that DB implements Store
var _ Store = (*DB)(nil)

type DB struct {
	*sql.DB
}

// New creates/opens database and runs migrations
func New(dbPath string) (*DB, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create dir: %w", err)
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

func (db *DB) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS projects (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		client TEXT NOT NULL,
		description TEXT,
		revenue REAL NOT NULL DEFAULT 0.0,
		status TEXT NOT NULL DEFAULT 'new' CHECK(status IN ('new', 'in_progress', 'done', 'paid')),
		secured_by TEXT NOT NULL CHECK(secured_by IN ('noor', 'ahmad', 'both')),
		stripe_payment_id TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS contributions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		project_id INTEGER REFERENCES projects(id) ON DELETE CASCADE,
		owner TEXT NOT NULL CHECK(owner IN ('noor', 'ahmad')),
		hours REAL DEFAULT 0.0,
		notes TEXT,
		UNIQUE(project_id, owner)
	);

	CREATE INDEX IF NOT EXISTS idx_projects_status ON projects(status);
	CREATE INDEX IF NOT EXISTS idx_projects_stripe ON projects(stripe_payment_id);
	`
	_, err := db.Exec(schema)
	return err
}

// Project Scanner - DRY scan helper
type projectScanner struct {
	dest *models.Project
}

func (s projectScanner) Scan(rows *sql.Rows) error {
	return rows.Scan(&s.dest.ID, &s.dest.Client, &s.dest.Description, &s.dest.Revenue, 
		&s.dest.Status, &s.dest.SecuredBy, &s.dest.StripePaymentID, &s.dest.CreatedAt)
}

func (s projectScanner) ScanRow(row *sql.Row) error {
	return row.Scan(&s.dest.ID, &s.dest.Client, &s.dest.Description, &s.dest.Revenue, 
		&s.dest.Status, &s.dest.SecuredBy, &s.dest.StripePaymentID, &s.dest.CreatedAt)
}

// CreateProject inserts a new project
func (db *DB) CreateProject(p *models.Project) error {
	return db.QueryRow(qProjectInsert, p.Client, p.Description, p.Revenue, p.Status, 
		p.SecuredBy, p.StripePaymentID).Scan(&p.ID, &p.CreatedAt)
}

// GetProject fetches a project by ID
func (db *DB) GetProject(id int64) (*models.Project, error) {
	p := &models.Project{}
	err := projectScanner{p}.ScanRow(db.QueryRow(qProjectByID, id))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return p, err
}

// GetProjectByStripeID fetches a project by Stripe payment ID
func (db *DB) GetProjectByStripeID(stripeID string) (*models.Project, error) {
	p := &models.Project{}
	err := projectScanner{p}.ScanRow(db.QueryRow(qProjectByStripeID, stripeID))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return p, err
}

// UpdateProject updates all project fields
func (db *DB) UpdateProject(p *models.Project) error {
	_, err := db.Exec(qProjectUpdate, p.Client, p.Description, p.Revenue, p.Status, 
		p.SecuredBy, p.StripePaymentID, p.ID)
	return err
}

// UpdateProjectStatus updates status and payment info (used by webhooks)
func (db *DB) UpdateProjectStatus(id int64, status models.ProjectStatus, revenue float64, stripeID string) error {
	_, err := db.Exec(qProjectUpdateStatus, status, revenue, stripeID, id)
	return err
}

// DeleteProject removes a project (cascades to contributions)
func (db *DB) DeleteProject(id int64) error {
	_, err := db.Exec(qProjectDelete, id)
	return err
}

// ListProjects returns all projects, optionally filtered by search
func (db *DB) ListProjects(search string) ([]models.Project, error) {
	var rows *sql.Rows
	var err error
	
	if search != "" {
		like := "%" + search + "%"
		rows, err = db.Query(qProjectsSearch, like, like)
	} else {
		rows, err = db.Query(qProjectsAll)
	}
	
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanAll(rows, func() *models.Project { return &models.Project{} }, 
		func(p *models.Project) scanner { return projectScanner{p} })
}

// ListProjectsByStatus returns projects filtered by status
func (db *DB) ListProjectsByStatus(status models.ProjectStatus) ([]models.Project, error) {
	rows, err := db.Query(qProjectsByStatus, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	return scanAll(rows, func() *models.Project { return &models.Project{} }, 
		func(p *models.Project) scanner { return projectScanner{p} })
}

// Generic scanner interface
type scanner interface {
	Scan(rows *sql.Rows) error
}

// Generic scanAll helper - DRY for scanning rows into slices
func scanAll[T any](rows *sql.Rows, newFn func() *T, scannerFn func(*T) scanner) ([]T, error) {
	var results []T
	for rows.Next() {
		item := newFn()
		if err := scannerFn(item).Scan(rows); err != nil {
			return nil, err
		}
		results = append(results, *item)
	}
	return results, rows.Err()
}
