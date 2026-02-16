package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"github.com/noor-latif/fulldash/internal/models"
	_ "modernc.org/sqlite"
)

type DB struct {
	*sql.DB
}

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

func (db *DB) CreateProject(p *models.Project) error {
	query := `INSERT INTO projects (client, description, revenue, status, secured_by, stripe_payment_id) 
		VALUES (?, ?, ?, ?, ?, ?) RETURNING id, created_at`
	return db.QueryRow(query, p.Client, p.Description, p.Revenue, p.Status, p.SecuredBy, p.StripePaymentID).Scan(&p.ID, &p.CreatedAt)
}

func (db *DB) GetProject(id int64) (*models.Project, error) {
	p := &models.Project{}
	query := `SELECT id, client, description, revenue, status, secured_by, stripe_payment_id, created_at FROM projects WHERE id = ?`
	err := db.QueryRow(query, id).Scan(&p.ID, &p.Client, &p.Description, &p.Revenue, &p.Status, &p.SecuredBy, &p.StripePaymentID, &p.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return p, err
}

func (db *DB) GetProjectByStripeID(stripeID string) (*models.Project, error) {
	p := &models.Project{}
	query := `SELECT id, client, description, revenue, status, secured_by, stripe_payment_id, created_at FROM projects WHERE stripe_payment_id = ?`
	err := db.QueryRow(query, stripeID).Scan(&p.ID, &p.Client, &p.Description, &p.Revenue, &p.Status, &p.SecuredBy, &p.StripePaymentID, &p.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return p, err
}

func (db *DB) UpdateProject(p *models.Project) error {
	query := `UPDATE projects SET client=?, description=?, revenue=?, status=?, secured_by=?, stripe_payment_id=? WHERE id=?`
	_, err := db.Exec(query, p.Client, p.Description, p.Revenue, p.Status, p.SecuredBy, p.StripePaymentID, p.ID)
	return err
}

func (db *DB) UpdateProjectStatus(id int64, status models.ProjectStatus, revenue float64, stripeID string) error {
	query := `UPDATE projects SET status=?, revenue=?, stripe_payment_id=? WHERE id=?`
	_, err := db.Exec(query, status, revenue, stripeID, id)
	return err
}

func (db *DB) DeleteProject(id int64) error {
	_, err := db.Exec(`DELETE FROM projects WHERE id = ?`, id)
	return err
}

func (db *DB) ListProjects(search string) ([]models.Project, error) {
	var query string
	var args []interface{}
	
	if search != "" {
		query = `SELECT id, client, description, revenue, status, secured_by, stripe_payment_id, created_at 
			FROM projects WHERE client LIKE ? OR description LIKE ? ORDER BY created_at DESC`
		args = []interface{}{"%" + search + "%", "%" + search + "%"}
	} else {
		query = `SELECT id, client, description, revenue, status, secured_by, stripe_payment_id, created_at 
			FROM projects ORDER BY created_at DESC`
	}
	
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanProjects(rows)
}

func (db *DB) ListProjectsByStatus(status models.ProjectStatus) ([]models.Project, error) {
	query := `SELECT id, client, description, revenue, status, secured_by, stripe_payment_id, created_at 
		FROM projects WHERE status = ? ORDER BY created_at DESC`
	rows, err := db.Query(query, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanProjects(rows)
}

func scanProjects(rows *sql.Rows) ([]models.Project, error) {
	var projects []models.Project
	for rows.Next() {
		var p models.Project
		if err := rows.Scan(&p.ID, &p.Client, &p.Description, &p.Revenue, &p.Status, &p.SecuredBy, &p.StripePaymentID, &p.CreatedAt); err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}
	return projects, rows.Err()
}

func (db *DB) GetContributions(projectID int64) ([]models.Contribution, error) {
	query := `SELECT id, project_id, owner, hours, notes FROM contributions WHERE project_id = ?`
	rows, err := db.Query(query, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var contribs []models.Contribution
	for rows.Next() {
		var c models.Contribution
		if err := rows.Scan(&c.ID, &c.ProjectID, &c.Owner, &c.Hours, &c.Notes); err != nil {
			return nil, err
		}
		contribs = append(contribs, c)
	}
	return contribs, rows.Err()
}

func (db *DB) SetContribution(c *models.Contribution) error {
	// Upsert
	query := `INSERT INTO contributions (project_id, owner, hours, notes) VALUES (?, ?, ?, ?)
		ON CONFLICT(project_id, owner) DO UPDATE SET hours=excluded.hours, notes=excluded.notes`
	res, err := db.Exec(query, c.ProjectID, c.Owner, c.Hours, c.Notes)
	if err != nil {
		return err
	}
	if c.ID == 0 {
		id, _ := res.LastInsertId()
		c.ID = id
	}
	return nil
}

func (db *DB) GetMetrics() (*models.Metrics, error) {
	m := &models.Metrics{}
	
	// Total revenue and paid count
	var paidCount int
	err := db.QueryRow(`SELECT COALESCE(SUM(revenue), 0), COUNT(*) FROM projects WHERE status = 'paid'`).Scan(&m.TotalRevenue, &paidCount)
	if err != nil {
		return nil, err
	}

	// Open projects (not paid)
	err = db.QueryRow(`SELECT COUNT(*) FROM projects WHERE status != 'paid'`).Scan(&m.OpenProjects)
	if err != nil {
		return nil, err
	}

	// Calculate shares from paid projects
	paid, err := db.ListProjectsByStatus(models.StatusPaid)
	if err != nil {
		return nil, err
	}

	for _, p := range paid {
		contribs, _ := db.GetContributions(p.ID)
		split := CalcRevenueSplit(&p, contribs)
		m.NoorShare += split.NoorShare
		m.AhmadShare += split.AhmadShare
	}

	return m, nil
}

func CalcRevenueSplit(p *models.Project, contribs []models.Contribution) *models.RevenueSplit {
	if p.Revenue <= 0 {
		return &models.RevenueSplit{Method: "none"}
	}

	// Check if we have contribution hours
	var noorHours, ahmadHours float64
	for _, c := range contribs {
		if c.Owner == models.OwnerNoor {
			noorHours = c.Hours
		} else {
			ahmadHours = c.Hours
		}
	}

	// If both have hours, use hours-based split
	if noorHours > 0 && ahmadHours > 0 {
		total := noorHours + ahmadHours
		return &models.RevenueSplit{
			NoorShare:  p.Revenue * (noorHours / total),
			AhmadShare: p.Revenue * (ahmadHours / total),
			Method:     "hours",
		}
	}

	// Default: owner-based split
	switch p.SecuredBy {
	case models.OwnerNoor:
		return &models.RevenueSplit{NoorShare: p.Revenue, AhmadShare: 0, Method: "owner"}
	case models.OwnerAhmad:
		return &models.RevenueSplit{NoorShare: 0, AhmadShare: p.Revenue, Method: "owner"}
	default: // both
		half := p.Revenue / 2
		return &models.RevenueSplit{NoorShare: half, AhmadShare: half, Method: "owner"}
	}
}
