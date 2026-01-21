package repositories

import (
	"context"
	"github.com/yourusername/gym-go/internal/domain/entities"
)

// InstructorRepository define las operaciones para persistencia de instructores
type InstructorRepository interface {
	Create(ctx context.Context, instructor *entities.Instructor) error
	GetByID(ctx context.Context, id string) (*entities.Instructor, error)
	GetByEmail(ctx context.Context, email string) (*entities.Instructor, error)
	Update(ctx context.Context, instructor *entities.Instructor) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, filters InstructorFilters) ([]*entities.Instructor, error)
	GetActive(ctx context.Context) ([]*entities.Instructor, error)
}

// InstructorFilters representa los filtros para buscar instructores
type InstructorFilters struct {
	IsActive   *bool
	Specialty  string
	SearchTerm string
	Limit      int
	Offset     int
}



