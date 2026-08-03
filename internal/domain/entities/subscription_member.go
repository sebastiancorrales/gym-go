package entities

import (
	"time"

	"github.com/google/uuid"
)

// SubscriptionMember links additional users to a group subscription
//
// subscription_id se consulta una vez por suscripción al listar (500 veces) y
// user_id es el lado filtrado de los JOIN que resuelven los planes grupales en
// cada check-in.
type SubscriptionMember struct {
	ID             uuid.UUID `json:"id" gorm:"primaryKey"`
	SubscriptionID uuid.UUID `json:"subscription_id" gorm:"index:idx_sub_members_sub"`
	UserID         uuid.UUID `json:"user_id" gorm:"index:idx_sub_members_user"`
	IsPrimary      bool      `json:"is_primary"`
	CreatedAt      time.Time `json:"created_at"`
}
