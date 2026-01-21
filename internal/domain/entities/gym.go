package entities

import (
	"time"

	"github.com/google/uuid"
)

// Gym represents a gym/facility
type Gym struct {
	ID         uuid.UUID `json:"id"`
	Name       string    `json:"name"`
	LegalName  string    `json:"legal_name,omitempty"`
	TaxID      string    `json:"tax_id,omitempty"`
	Address    string    `json:"address,omitempty"`
	City       string    `json:"city,omitempty"`
	State      string    `json:"state,omitempty"`
	Country    string    `json:"country"`
	PostalCode string    `json:"postal_code,omitempty"`
	Phone      string    `json:"phone,omitempty"`
	Email      string    `json:"email,omitempty"`
	LogoURL    string    `json:"logo_url,omitempty"`
	Timezone   string    `json:"timezone"`
	Currency   string    `json:"currency"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// NewGym creates a new gym
func NewGym(name, email, phone string) *Gym {
	now := time.Now()
	return &Gym{
		ID:        uuid.New(),
		Name:      name,
		Email:     email,
		Phone:     phone,
		Country:   "CO",
		Timezone:  "America/Bogota",
		Currency:  "COP",
		Status:    "ACTIVE",
		CreatedAt: now,
		UpdatedAt: now,
	}
}



