package dto

import "time"

// CreateMemberRequest representa la solicitud para crear un miembro
type CreateMemberRequest struct {
	FirstName   string    `json:"first_name" validate:"required"`
	LastName    string    `json:"last_name" validate:"required"`
	Email       string    `json:"email" validate:"required,email"`
	Phone       string    `json:"phone" validate:"required"`
	DateOfBirth time.Time `json:"date_of_birth" validate:"required"`
}

// UpdateMemberRequest representa la solicitud para actualizar un miembro
type UpdateMemberRequest struct {
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
	Phone     string `json:"phone,omitempty"`
}

// MemberResponse representa la respuesta de un miembro
type MemberResponse struct {
	ID           string    `json:"id"`
	FirstName    string    `json:"first_name"`
	LastName     string    `json:"last_name"`
	Email        string    `json:"email"`
	Phone        string    `json:"phone"`
	DateOfBirth  time.Time `json:"date_of_birth"`
	JoinDate     time.Time `json:"join_date"`
	Status       string    `json:"status"`
	MembershipID *string   `json:"membership_id,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// AssignMembershipRequest representa la solicitud para asignar una membresía
type AssignMembershipRequest struct {
	MembershipID string `json:"membership_id" validate:"required"`
}



