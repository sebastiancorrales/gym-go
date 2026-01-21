package entities

import (
	"time"
)

// Member representa un miembro del gimnasio (Entidad del dominio)
type Member struct {
	ID           string
	FirstName    string
	LastName     string
	Email        string
	Phone        string
	DateOfBirth  time.Time
	JoinDate     time.Time
	Status       MemberStatus
	MembershipID *string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// MemberStatus representa el estado de un miembro
type MemberStatus string

const (
	MemberStatusActive   MemberStatus = "active"
	MemberStatusInactive MemberStatus = "inactive"
	MemberStatusSuspended MemberStatus = "suspended"
)

// IsActive verifica si el miembro está activo
func (m *Member) IsActive() bool {
	return m.Status == MemberStatusActive
}

// Activate activa la membresía del miembro
func (m *Member) Activate() {
	m.Status = MemberStatusActive
	m.UpdatedAt = time.Now()
}

// Suspend suspende la membresía del miembro
func (m *Member) Suspend() {
	m.Status = MemberStatusSuspended
	m.UpdatedAt = time.Now()
}

// Deactivate desactiva la membresía del miembro
func (m *Member) Deactivate() {
	m.Status = MemberStatusInactive
	m.UpdatedAt = time.Now()
}

// FullName retorna el nombre completo del miembro
func (m *Member) FullName() string {
	return m.FirstName + " " + m.LastName
}



