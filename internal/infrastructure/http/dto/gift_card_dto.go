package dto

import (
	"github.com/sebastiancorrales/gym-go/internal/domain/entities"
	"github.com/sebastiancorrales/gym-go/internal/domain/repositories"
)

// SellGiftCardRequest vende una tarjeta regalo. El valor NO viaja: lo fija el
// precio del plan en el servidor, y las líneas de pago se validan contra él.
type SellGiftCardRequest struct {
	PlanID         string               `json:"plan_id" binding:"required,uuid"`
	RecipientName  string               `json:"recipient_name" binding:"required"`
	RecipientPhone string               `json:"recipient_phone"`
	BuyerName      string               `json:"buyer_name" binding:"required"`
	BuyerPhone     string               `json:"buyer_phone"`
	Dedication     string               `json:"dedication"`
	Payments       []PaymentLineRequest `json:"payments" binding:"required,min=1,dive"`
}

// RedeemGiftCardRequest redime una tarjeta creando la suscripción de UserID.
type RedeemGiftCardRequest struct {
	UserID              string   `json:"user_id" binding:"required,uuid"`
	AdditionalMemberIDs []string `json:"additional_member_ids"`
	AllowExpired        bool     `json:"allow_expired"`
}

// VoidGiftCardRequest anula una tarjeta pendiente.
type VoidGiftCardRequest struct {
	Reason string `json:"reason"`
}

// GiftCardResponse es la tarjeta más su estado visto HOY en la zona del
// gimnasio ("expired" incluido), que es lo que la pantalla necesita mostrar.
type GiftCardResponse struct {
	entities.GiftCard
	State string `json:"state"`
}

func ToGiftCardResponse(card *entities.GiftCard, today string) GiftCardResponse {
	return GiftCardResponse{GiftCard: *card, State: card.StateOn(today)}
}

// GiftCardListResponse es el listado con sus cifras.
type GiftCardListResponse struct {
	Cards       []GiftCardResponse               `json:"cards"`
	SoldCount   int                              `json:"sold_count"`
	SoldValue   float64                          `json:"sold_value"`
	Redeemed    int                              `json:"redeemed_count"`
	Voided      int                              `json:"voided_count"`
	Outstanding repositories.GiftCardOutstanding `json:"outstanding"`
}
