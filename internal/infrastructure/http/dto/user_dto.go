package dto

// UserResponse represents user response
type UserResponse struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Role      string `json:"role"`
	GymID     string `json:"gym_id,omitempty"`
	// Permissions solo se rellena en /auth/me y en el login: es lo que usa el
	// frontend para decidir qué secciones del menú mostrar, en vez de repetir la
	// matriz de roles en JavaScript.
	Permissions []string `json:"permissions,omitempty"`
}
