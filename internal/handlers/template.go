// handlers/template.go - Template helpers
package handlers

import (
	"html/template"
	"math"

	"github.com/noor-latif/fulldash/internal/models"
)

// TemplateFuncs returns map of helper functions for templates
func TemplateFuncs() template.FuncMap {
	return template.FuncMap{
		"percentage": func(part, total float64) float64 {
			if total == 0 {
				return 0
			}
			return math.Min(100, math.Max(0, (part/total)*100))
		},
		"div": func(a, b int64) float64 {
			if b == 0 {
				return 0
			}
			return float64(a) / float64(b)
		},
		"noorHours": func(contribs []models.Contribution) float64 {
			for _, c := range contribs {
				if c.Person == models.OwnerNoor {
					return c.Hours
				}
			}
			return 0
		},
		"ahmadHours": func(contribs []models.Contribution) float64 {
			for _, c := range contribs {
				if c.Person == models.OwnerAhmad {
					return c.Hours
				}
			}
			return 0
		},
	}
}
