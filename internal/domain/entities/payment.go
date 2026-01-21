package entities

import (
	"time"

	"github.com/google/uuid"
)

// PaymentMethod represents payment method
type PaymentMethod string

const (
	PaymentMethodCash        PaymentMethod = "CASH"
	PaymentMethodCard        PaymentMethod = "CARD"
	PaymentMethodTransfer    PaymentMethod = "TRANSFER"
	PaymentMethodOnline      PaymentMethod = "ONLINE"
	PaymentMethodPaypal      PaymentMethod = "PAYPAL"
	PaymentMethodMercadoPago PaymentMethod = "MERCADO_PAGO"
	PaymentMethodCrypto      PaymentMethod = "CRYPTO"
)

// PaymentStatus represents payment status
type PaymentStatus string

const (
	PaymentStatusPending   PaymentStatus = "PENDING"
	PaymentStatusCompleted PaymentStatus = "COMPLETED"
	PaymentStatusFailed    PaymentStatus = "FAILED"
	PaymentStatusRefunded  PaymentStatus = "REFUNDED"
	PaymentStatusCancelled PaymentStatus = "CANCELLED"
)

// PaymentType represents payment type
type PaymentType string

const (
	PaymentTypeSubscription PaymentType = "SUBSCRIPTION"
	PaymentTypeEnrollment   PaymentType = "ENROLLMENT"
	PaymentTypeClass        PaymentType = "CLASS"
	PaymentTypeProduct      PaymentType = "PRODUCT"
	PaymentTypePenalty      PaymentType = "PENALTY"
	PaymentTypeOther        PaymentType = "OTHER"
)

// Payment represents a payment transaction
type Payment struct {
	ID                uuid.UUID     `json:"id"`
	GymID             uuid.UUID     `json:"gym_id"`
	UserID            uuid.UUID     `json:"user_id"`
	SubscriptionID    *uuid.UUID    `json:"subscription_id,omitempty"`
	Amount            float64       `json:"amount"`
	Currency          string        `json:"currency"`
	PaymentMethod     PaymentMethod `json:"payment_method"`
	PaymentType       PaymentType   `json:"payment_type"`
	Status            PaymentStatus `json:"status"`
	TransactionID     string        `json:"transaction_id,omitempty"`
	ExternalReference string        `json:"external_reference,omitempty"`
	Description       string        `json:"description"`
	Notes             string        `json:"notes,omitempty"`
	RefundedAmount    float64       `json:"refunded_amount"`
	RefundedAt        *time.Time    `json:"refunded_at,omitempty"`
	RefundReason      string        `json:"refund_reason,omitempty"`
	ProcessedBy       uuid.UUID     `json:"processed_by"`
	CashRegisterID    *uuid.UUID    `json:"cash_register_id,omitempty"`
	PaymentDate       time.Time     `json:"payment_date"`
	CreatedAt         time.Time     `json:"created_at"`
	UpdatedAt         time.Time     `json:"updated_at"`
}

// NewPayment creates a new payment
func NewPayment(gymID, userID, processedBy uuid.UUID, amount float64, currency string, method PaymentMethod, paymentType PaymentType, description string) *Payment {
	now := time.Now()
	return &Payment{
		ID:            uuid.New(),
		GymID:         gymID,
		UserID:        userID,
		Amount:        amount,
		Currency:      currency,
		PaymentMethod: method,
		PaymentType:   paymentType,
		Status:        PaymentStatusPending,
		Description:   description,
		ProcessedBy:   processedBy,
		PaymentDate:   now,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

// Complete marks payment as completed
func (p *Payment) Complete(transactionID string) {
	p.Status = PaymentStatusCompleted
	p.TransactionID = transactionID
	p.UpdatedAt = time.Now()
}

// Fail marks payment as failed
func (p *Payment) Fail() {
	p.Status = PaymentStatusFailed
	p.UpdatedAt = time.Now()
}

// Refund refunds the payment
func (p *Payment) Refund(amount float64, reason string) error {
	if p.Status != PaymentStatusCompleted {
		return nil
	}
	now := time.Now()
	p.Status = PaymentStatusRefunded
	p.RefundedAmount = amount
	p.RefundReason = reason
	p.RefundedAt = &now
	p.UpdatedAt = now
	return nil
}

// Cancel cancels the payment
func (p *Payment) Cancel() {
	p.Status = PaymentStatusCancelled
	p.UpdatedAt = time.Now()
}

// IsCompleted checks if payment is completed
func (p *Payment) IsCompleted() bool {
	return p.Status == PaymentStatusCompleted
}

// IsRefunded checks if payment is refunded
func (p *Payment) IsRefunded() bool {
	return p.Status == PaymentStatusRefunded
}



