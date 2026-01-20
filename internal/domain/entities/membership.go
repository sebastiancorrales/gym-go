package entities

import (
	"time"
)

// Membership representa un plan de membresía
type Membership struct {
	ID          string
	Name        string
	Description string
	DurationDays int
	Price       float64
	Benefits    []string
	IsActive    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// IsExpired verifica si la membresía ha expirado
func (m *Membership) IsExpired(startDate time.Time) bool {
	expirationDate := startDate.AddDate(0, 0, m.DurationDays)
	return time.Now().After(expirationDate)
}

// GetExpirationDate calcula la fecha de expiración basándose en la fecha de inicio
func (m *Membership) GetExpirationDate(startDate time.Time) time.Time {
	return startDate.AddDate(0, 0, m.DurationDays)
}
