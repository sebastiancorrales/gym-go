package entities

import (
	"time"
)

// Instructor representa un instructor del gimnasio
type Instructor struct {
	ID            string
	FirstName     string
	LastName      string
	Email         string
	Phone         string
	Specialties   []string
	Certifications []string
	HireDate      time.Time
	IsActive      bool
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// FullName retorna el nombre completo del instructor
func (i *Instructor) FullName() string {
	return i.FirstName + " " + i.LastName
}

// HasSpecialty verifica si el instructor tiene una especialidad específica
func (i *Instructor) HasSpecialty(specialty string) bool {
	for _, s := range i.Specialties {
		if s == specialty {
			return true
		}
	}
	return false
}
