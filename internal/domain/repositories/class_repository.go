package repositories

import (
	"context"
	"github.com/yourusername/gym-go/internal/domain/entities"
	"time"
)

// ClassRepository define las operaciones para persistencia de clases
type ClassRepository interface {
	Create(ctx context.Context, class *entities.Class) error
	GetByID(ctx context.Context, id string) (*entities.Class, error)
	Update(ctx context.Context, class *entities.Class) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, filters ClassFilters) ([]*entities.Class, error)
	GetEnrollmentCount(ctx context.Context, classID string) (int, error)
}

// ClassFilters representa los filtros para buscar clases
type ClassFilters struct {
	InstructorID *string
	Status       *entities.ClassStatus
	StartDate    *time.Time
	EndDate      *time.Time
	Limit        int
	Offset       int
}



