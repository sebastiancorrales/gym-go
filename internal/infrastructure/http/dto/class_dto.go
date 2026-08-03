package dto

import "time"

// CreateClassRequest representa la solicitud para crear una clase
type CreateClassRequest struct {
	Name         string    `json:"name" binding:"required"`
	Description  string    `json:"description"`
	InstructorID string    `json:"instructor_id" binding:"required"`
	Capacity     int       `json:"capacity" binding:"required,gt=0"`
	Duration     int       `json:"duration" binding:"required,gt=0"`
	Schedule     time.Time `json:"schedule" binding:"required"`
}

// UpdateClassRequest representa la solicitud para actualizar una clase
type UpdateClassRequest struct {
	Name        string     `json:"name,omitempty"`
	Description string     `json:"description,omitempty"`
	Capacity    *int       `json:"capacity,omitempty"`
	Duration    *int       `json:"duration,omitempty"`
	Schedule    *time.Time `json:"schedule,omitempty"`
}

// ClassResponse representa la respuesta de una clase
type ClassResponse struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	InstructorID string    `json:"instructor_id"`
	Capacity     int       `json:"capacity"`
	Duration     int       `json:"duration"`
	Schedule     time.Time `json:"schedule"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}



