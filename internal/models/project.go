// models/project.go - Data models for FullDash
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
	StatusPending ProjectStatus = "pending"
	StatusPaid    ProjectStatus = "paid"
	StatusDone    ProjectStatus = "done"
)

// Project is the main entity
type Project struct {
	ID              int64         `json:"id"`
	Name            string        `json:"name"`
	Description     string        `json:"description"`
	Client          string        `json:"client"`
	SecuredBy       Owner         `json:"secured_by"` // noor, ahmad, both
	AmountCents     int64         `json:"amount_cents"` // Stripe amount (cents)
	Revenue         float64       `json:"revenue"`      // actual received (dollars)
	Status          ProjectStatus `json:"status"`
	StripePaymentID string        `json:"stripe_payment_id"`
	CreatedAt       time.Time     `json:"created_at"`
	UpdatedAt       time.Time     `json:"updated_at"`
}

// Contribution tracks hours worked per person
type Contribution struct {
	ID        int64     `json:"id"`
	ProjectID int64     `json:"project_id"`
	Person    Owner     `json:"person"` // noor or ahmad
	Hours     float64   `json:"hours"`
	CreatedAt time.Time `json:"created_at"`
}

// RevenueSplit represents calculated payouts
type RevenueSplit struct {
	NoorShare   float64 `json:"noor_share"`
	AhmadShare  float64 `json:"ahmad_share"`
	SplitMethod string  `json:"split_method"` // "owner" or "hours"
}

// DashboardStats for the overview
type DashboardStats struct {
	TotalProjects   int     `json:"total_projects"`
	PendingProjects int     `json:"pending_projects"`
	PaidProjects    int     `json:"paid_projects"`
	TotalRevenue    float64 `json:"total_revenue"`
	NoorShare       float64 `json:"noor_share"`
	AhmadShare      float64 `json:"ahmad_share"`
}

// ProjectWithContributions for UI display
type ProjectWithContributions struct {
	Project       Project        `json:"project"`
	Contributions []Contribution `json:"contributions"`
	Split         RevenueSplit   `json:"split"`
}
