package models

import "time"

// Owner represents who secured the project
type Owner string

const (
	OwnerNoor  Owner = "noor"
	OwnerAhmad Owner = "ahmad"
	OwnerBoth  Owner = "both"
)

// ProjectStatus represents the current state
type ProjectStatus string

const (
	StatusNew       ProjectStatus = "new"
	StatusProgress  ProjectStatus = "in_progress"
	StatusDone      ProjectStatus = "done"
	StatusPaid      ProjectStatus = "paid"
)

// Project is the main entity
type Project struct {
	ID              int64         `json:"id" db:"id"`
	Client          string        `json:"client" db:"client"`
	Description     string        `json:"description" db:"description"`
	Revenue         float64       `json:"revenue" db:"revenue"`
	Status          ProjectStatus `json:"status" db:"status"`
	SecuredBy       Owner         `json:"secured_by" db:"secured_by"`
	StripePaymentID string        `json:"stripe_payment_id" db:"stripe_payment_id"`
	CreatedAt       time.Time     `json:"created_at" db:"created_at"`
}

// Contribution tracks work per owner
type Contribution struct {
	ID        int64     `json:"id" db:"id"`
	ProjectID int64     `json:"project_id" db:"project_id"`
	Owner     Owner     `json:"owner" db:"owner"`
	Hours     float64   `json:"hours" db:"hours"`
	Notes     string    `json:"notes" db:"notes"`
}

// Metrics for dashboard
type Metrics struct {
	TotalRevenue   float64 `json:"total_revenue"`
	NoorShare      float64 `json:"noor_share"`
	AhmadShare     float64 `json:"ahmad_share"`
	OpenProjects   int     `json:"open_projects"`
}

// ProjectWithContributions for UI
type ProjectWithContributions struct {
	Project       Project
	Contributions []Contribution
}

// RevenueSplit result
type RevenueSplit struct {
	NoorShare   float64
	AhmadShare  float64
	Method      string // "owner" or "hours"
}
