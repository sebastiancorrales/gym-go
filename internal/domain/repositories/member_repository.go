package repositories

import (
	"context"
	"gym-go/internal/domain/entities"
)

// MemberRepository define las operaciones para persistencia de miembros
// Sigue el principio de Inversión de Dependencias (DIP)
type MemberRepository interface {
	Create(ctx context.Context, member *entities.Member) error
	GetByID(ctx context.Context, id string) (*entities.Member, error)
	GetByEmail(ctx context.Context, email string) (*entities.Member, error)
	Update(ctx context.Context, member *entities.Member) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, filters MemberFilters) ([]*entities.Member, error)
	Count(ctx context.Context, filters MemberFilters) (int64, error)
}

// MemberFilters representa los filtros para buscar miembros
type MemberFilters struct {
	Status       *entities.MemberStatus
	MembershipID *string
	SearchTerm   string // para buscar por nombre o email
	Limit        int
	Offset       int
}
