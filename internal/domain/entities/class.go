package entities

import (
	"time"
)

// Class representa una clase o sesión de entrenamiento
type Class struct {
	ID           string
	Name         string
	Description  string
	InstructorID string
	Capacity     int
	Duration     int // en minutos
	Schedule     time.Time
	Status       ClassStatus
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// ClassStatus representa el estado de una clase
type ClassStatus string

const (
	ClassStatusScheduled ClassStatus = "scheduled"
	ClassStatusOngoing   ClassStatus = "ongoing"
	ClassStatusCompleted ClassStatus = "completed"
	ClassStatusCancelled ClassStatus = "cancelled"
)

// IsFull verifica si la clase está llena
func (c *Class) IsFull(currentEnrollments int) bool {
	return currentEnrollments >= c.Capacity
}

// CanEnroll verifica si se puede inscribir a la clase
func (c *Class) CanEnroll(currentEnrollments int) bool {
	return !c.IsFull(currentEnrollments) && c.Status == ClassStatusScheduled && c.Schedule.After(time.Now())
}

// Cancel cancela la clase
func (c *Class) Cancel() {
	c.Status = ClassStatusCancelled
	c.UpdatedAt = time.Now()
}

// Start inicia la clase
func (c *Class) Start() {
	c.Status = ClassStatusOngoing
	c.UpdatedAt = time.Now()
}

// Complete completa la clase
func (c *Class) Complete() {
	c.Status = ClassStatusCompleted
	c.UpdatedAt = time.Now()
}
