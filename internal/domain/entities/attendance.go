package entities

import (
	"time"
)

// Attendance representa el registro de asistencia de un miembro
type Attendance struct {
	ID        string
	MemberID  string
	ClassID   *string // Opcional, puede ser asistencia general al gym
	CheckIn   time.Time
	CheckOut  *time.Time
	CreatedAt time.Time
}

// IsCheckedOut verifica si el miembro ya ha hecho checkout
func (a *Attendance) IsCheckedOut() bool {
	return a.CheckOut != nil
}

// GetDuration calcula la duración de la asistencia
func (a *Attendance) GetDuration() *time.Duration {
	if a.CheckOut == nil {
		return nil
	}
	duration := a.CheckOut.Sub(a.CheckIn)
	return &duration
}



