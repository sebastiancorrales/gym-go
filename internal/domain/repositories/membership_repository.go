package repositories

import (
	"context"
	"github.com/sebastiancorrales/gym-go/internal/domain/entities"
)

// MembershipRepository define las operaciones para persistencia de membresías
type MembershipRepository interface {
	Create(ctx context.Context, membership *entities.Membership) error
	GetByID(ctx context.Context, id string) (*entities.Membership, error)
	Update(ctx context.Context, membership *entities.Membership) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, filters MembershipFilters) ([]*entities.Membership, error)
	GetActive(ctx context.Context) ([]*entities.Membership, error)
}

// MembershipFilters representa los filtros para buscar membresías
type MembershipFilters struct {
	IsActive *bool
	MinPrice *float64
	MaxPrice *float64
	Limit    int
	Offset   int
}



