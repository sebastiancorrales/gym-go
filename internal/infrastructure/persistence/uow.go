package persistence

import (
	"context"

	"github.com/sebastiancorrales/gym-go/internal/domain/repositories"
	"gorm.io/gorm"
)

// GormUnitOfWork implementa repositories.UnitOfWork sobre GORM.
//
// El truco que lo hace poco invasivo: todos los constructores de repositorio ya
// aceptan un *gorm.DB, y una transacción de GORM *es* un *gorm.DB. Así que se
// pueden reconstruir los repositorios sobre la transacción sin cambiar ni las
// interfaces del dominio, ni el cuerpo de los repositorios, ni los llamadores que
// no necesitan transacción.
type GormUnitOfWork struct {
	db *gorm.DB
}

func NewUnitOfWork(db *gorm.DB) repositories.UnitOfWork {
	return &GormUnitOfWork{db: db}
}

func (u *GormUnitOfWork) Do(ctx context.Context, fn func(repositories.Repos) error) error {
	return RetryOnBusy(ctx, func() error {
		return u.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			return fn(reposOn(tx))
		})
	})
}

// reposOn builds the repository set bound to a given handle (a transaction).
func reposOn(tx *gorm.DB) repositories.Repos {
	return repositories.Repos{
		Users:         NewSQLiteUserRepository(tx),
		Gyms:          NewSQLiteGymRepository(tx),
		Subscriptions: NewSQLiteSubscriptionRepository(tx),
		Members:       NewSQLiteSubscriptionMemberRepository(tx),
		Audit:         NewSQLiteSubscriptionAuditLogRepository(tx),
		Products:      NewSQLiteProductRepository(tx),
		Sales:         NewSQLiteSaleRepository(tx),
		SaleDetails:   NewSQLiteSaleDetailRepository(tx),
	}
}
