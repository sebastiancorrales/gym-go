package dto

import "time"

// CheckInRequest representa la solicitud para hacer check-in
type CheckInRequest struct {
	MemberID string  `json:"member_id" validate:"required"`
	ClassID  *string `json:"class_id,omitempty"`
}

// AttendanceResponse representa la respuesta de una asistencia
type AttendanceResponse struct {
	ID        string     `json:"id"`
	MemberID  string     `json:"member_id"`
	ClassID   *string    `json:"class_id,omitempty"`
	CheckIn   time.Time  `json:"check_in"`
	CheckOut  *time.Time `json:"check_out,omitempty"`
	Duration  *string    `json:"duration,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}
