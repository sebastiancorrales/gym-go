package dto

// UserResponse represents user response
type UserResponse struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Role      string `json:"role"`
	GymID     string `json:"gym_id,omitempty"`
}
