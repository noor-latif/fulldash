// store/metrics.go - Metrics calculation and business logic
package store

import (
	"github.com/noor-latif/fulldash/internal/models"
)

// GetMetrics calculates dashboard metrics including revenue splits
func (db *DB) GetMetrics() (*models.Metrics, error) {
	m := &models.Metrics{}
	
	// Total revenue from paid projects
	var paidCount int
	err := db.QueryRow(qMetricsTotalRevenue).Scan(&m.TotalRevenue, &paidCount)
	if err != nil {
		return nil, err
	}

	// Open projects (not paid)
	err = db.QueryRow(qMetricsOpenProjects).Scan(&m.OpenProjects)
	if err != nil {
		return nil, err
	}

	// Calculate shares from paid projects
	if err := db.calcRevenueShares(m); err != nil {
		return nil, err
	}

	return m, nil
}

// calcRevenueShares calculates Noor/Ahmad shares from paid projects
func (db *DB) calcRevenueShares(m *models.Metrics) error {
	paid, err := db.ListProjectsByStatus(models.StatusPaid)
	if err != nil {
		return err
	}

	for _, p := range paid {
		contribs, _ := db.GetContributions(p.ID)
		split := CalcRevenueSplit(&p, contribs)
		m.NoorShare += split.NoorShare
		m.AhmadShare += split.AhmadShare
	}
	return nil
}

// CalcRevenueSplit determines revenue sharing based on hours or ownership
func CalcRevenueSplit(p *models.Project, contribs []models.Contribution) *models.RevenueSplit {
	if p.Revenue <= 0 {
		return &models.RevenueSplit{Method: "none"}
	}

	// Extract hours
	var noorHours, ahmadHours float64
	for _, c := range contribs {
		switch c.Owner {
		case models.OwnerNoor:
			noorHours = c.Hours
		case models.OwnerAhmad:
			ahmadHours = c.Hours
		}
	}

	// If both logged hours, use hours-based split
	if noorHours > 0 && ahmadHours > 0 {
		total := noorHours + ahmadHours
		return &models.RevenueSplit{
			NoorShare:  p.Revenue * (noorHours / total),
			AhmadShare: p.Revenue * (ahmadHours / total),
			Method:     "hours",
		}
	}

	// Fall back to ownership-based split
	return splitByOwner(p)
}

// splitByOwner calculates revenue based on who secured the project
func splitByOwner(p *models.Project) *models.RevenueSplit {
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
