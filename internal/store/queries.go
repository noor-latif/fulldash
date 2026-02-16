// store/queries.go - Centralized SQL queries for DRY
package store

// Project columns for SELECT statements
const (
	projectColumns = `id, client, description, revenue, status, secured_by, stripe_payment_id, created_at`
	projectTable   = `projects`
	
	contributionColumns = `id, project_id, owner, hours, notes`
	contributionTable   = `contributions`
)

// SQL query templates
// Metrics queries
const (
	qMetricsTotalRevenue = `SELECT COALESCE(SUM(revenue), 0), COUNT(*) FROM ` + projectTable + ` WHERE status = 'paid'`
	qMetricsOpenProjects = `SELECT COUNT(*) FROM ` + projectTable + ` WHERE status != 'paid'`
)

const (
	qProjectByID = `SELECT ` + projectColumns + ` FROM ` + projectTable + ` WHERE id = ?`
	
	qProjectByStripeID = `SELECT ` + projectColumns + ` FROM ` + projectTable + ` WHERE stripe_payment_id = ?`
	
	qProjectsByStatus = `SELECT ` + projectColumns + ` FROM ` + projectTable + ` WHERE status = ? ORDER BY created_at DESC`
	
	qProjectsAll = `SELECT ` + projectColumns + ` FROM ` + projectTable + ` ORDER BY created_at DESC`
	
	qProjectsSearch = `SELECT ` + projectColumns + ` FROM ` + projectTable + 
		` WHERE client LIKE ? OR description LIKE ? ORDER BY created_at DESC`
	
	qProjectInsert = `INSERT INTO ` + projectTable + 
		` (client, description, revenue, status, secured_by, stripe_payment_id) 
		VALUES (?, ?, ?, ?, ?, ?) RETURNING id, created_at`
	
	qProjectUpdate = `UPDATE ` + projectTable + 
		` SET client=?, description=?, revenue=?, status=?, secured_by=?, stripe_payment_id=? WHERE id=?`
	
	qProjectUpdateStatus = `UPDATE ` + projectTable + 
		` SET status=?, revenue=?, stripe_payment_id=? WHERE id=?`
	
	qProjectDelete = `DELETE FROM ` + projectTable + ` WHERE id = ?`
	
	qContributionByProject = `SELECT ` + contributionColumns + ` FROM ` + contributionTable + ` WHERE project_id = ?`
	
	qContributionUpsert = `INSERT INTO ` + contributionTable + 
		` (project_id, owner, hours, notes) VALUES (?, ?, ?, ?)
		ON CONFLICT(project_id, owner) DO UPDATE SET hours=excluded.hours, notes=excluded.notes`
)
