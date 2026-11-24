package repositories

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/sebastiancorrales/gym-go/internal/domain/entities"
)

// GiftCardFilter son los filtros del listado de tarjetas regalo.
//
// State admite los estados guardados (pending/redeemed/voided) y el derivado
// "expired"; "pending" excluye las vencidas, igual que en pantalla. Today es
// el día del gimnasio contra el que se decide qué está vencido.
type GiftCardFilter struct {
	StartDate string
	EndDate   string
	State     string
	Query     string
	Today     string
}

// GiftCardOutstanding es el saldo por redimir HOY, sin importar cuándo se
// vendieron las tarjetas: dinero cobrado que todavía es una obligación del
// gimnasio. Las vencidas van aparte porque ya no se pueden redimir salvo que
// un administrador lo autorice.
type GiftCardOutstanding struct {
	Count        int     `json:"count"`
	Value        float64 `json:"value"`
	ExpiredCount int     `json:"expired_count"`
	ExpiredValue float64 `json:"expired_value"`
}

// GiftCardRepository persiste tarjetas regalo.
type GiftCardRepository interface {
	Create(ctx context.Context, card *entities.GiftCard) error
	GetByID(ctx context.Context, id uuid.UUID) (*entities.GiftCard, error)
	GetByCodeKey(ctx context.Context, key string) (*entities.GiftCard, error)
	CodeKeyExists(ctx context.Context, key string) (bool, error)
	List(ctx context.Context, gymID uuid.UUID, filter GiftCardFilter) ([]entities.GiftCard, error)
	Outstanding(ctx context.Context, gymID uuid.UUID, today string) (GiftCardOutstanding, error)

	// MarkRedeemed y MarkVoided son UPDATE condicionales (WHERE status =
	// 'pending') y devuelven ErrGiftCardNotPending si no afectaron ninguna
	// fila. La guarda vive en el WHERE por la misma razón que ApplyStockDelta:
	// leer el estado y luego escribirlo deja que dos recepciones rediman el
	// mismo código a la vez.
	MarkRedeemed(ctx context.Context, id uuid.UUID, at time.Time, byID, userID, subscriptionID uuid.UUID) error
	MarkVoided(ctx context.Context, id uuid.UUID, at time.Time, byID uuid.UUID, reason string) error
}
