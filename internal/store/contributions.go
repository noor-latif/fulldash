// store/contributions.go - Contribution-related database operations
package store

import (
	"database/sql"

	"github.com/noor-latif/fulldash/internal/models"
)

// contributionScanner for DRY row scanning
type contributionScanner struct {
	dest *models.Contribution
}

func (s contributionScanner) Scan(rows *sql.Rows) error {
	return rows.Scan(&s.dest.ID, &s.dest.ProjectID, &s.dest.Owner, &s.dest.Hours, &s.dest.Notes)
}

// GetContributions retrieves all contributions for a project
func (db *DB) GetContributions(projectID int64) ([]models.Contribution, error) {
	rows, err := db.Query(qContributionByProject, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanAll(rows, 
		func() *models.Contribution { return &models.Contribution{} },
		func(c *models.Contribution) scanner { return contributionScanner{c} })
}

// SetContribution creates or updates a contribution (upsert)
func (db *DB) SetContribution(c *models.Contribution) error {
	res, err := db.Exec(qContributionUpsert, c.ProjectID, c.Owner, c.Hours, c.Notes)
	if err != nil {
		return err
	}
	if c.ID == 0 {
		id, _ := res.LastInsertId()
		c.ID = id
	}
	return nil
}
