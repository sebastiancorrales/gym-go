package usecases

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/sebastiancorrales/gym-go/internal/domain/entities"
	"github.com/sebastiancorrales/gym-go/internal/domain/repositories"
	"github.com/sebastiancorrales/gym-go/pkg/errors"
)

// PaymentMethodUseCase handles business logic for payment methods
type PaymentMethodUseCase struct {
	paymentMethodRepo repositories.PaymentMethodRepository
}

// NewPaymentMethodUseCase creates a new PaymentMethodUseCase
func NewPaymentMethodUseCase(paymentMethodRepo repositories.PaymentMethodRepository) *PaymentMethodUseCase {
	return &PaymentMethodUseCase{
		paymentMethodRepo: paymentMethodRepo,
	}
}

// CreatePaymentMethod creates a new payment method
func (uc *PaymentMethodUseCase) CreatePaymentMethod(ctx context.Context, method *entities.SalePaymentMethod) error {
	if method.Name == "" {
		return errors.ErrInvalidInput
	}

	// Set default values
	method.ID = uuid.New()
	if method.Status == "" {
		method.Status = entities.PaymentMethodStatusActive
	}
	method.CreatedAt = time.Now()
	method.UpdatedAt = time.Now()

	return uc.paymentMethodRepo.Create(ctx, method)
}

// GetPaymentMethodByID retrieves a payment method by ID
func (uc *PaymentMethodUseCase) GetPaymentMethodByID(ctx context.Context, id uuid.UUID) (*entities.SalePaymentMethod, error) {
	return uc.paymentMethodRepo.GetByID(ctx, id)
}

// GetAllPaymentMethods retrieves all payment methods, optionally filtered by status
func (uc *PaymentMethodUseCase) GetAllPaymentMethods(ctx context.Context, status *entities.PaymentMethodStatus) ([]entities.SalePaymentMethod, error) {
	return uc.paymentMethodRepo.GetAll(ctx, status)
}

// UpdatePaymentMethod updates an existing payment method
func (uc *PaymentMethodUseCase) UpdatePaymentMethod(ctx context.Context, method *entities.SalePaymentMethod) error {
	if method.ID == uuid.Nil {
		return errors.ErrInvalidInput
	}
	if method.Name == "" {
		return errors.ErrInvalidInput
	}

	// Check if payment method exists
	existing, err := uc.paymentMethodRepo.GetByID(ctx, method.ID)
	if err != nil {
		return err
	}
	if existing == nil {
		return errors.ErrNotFound
	}

	method.UpdatedAt = time.Now()
	method.CreatedAt = existing.CreatedAt // Preserve creation date

	return uc.paymentMethodRepo.Update(ctx, method)
}

// DeletePaymentMethod deletes a payment method
func (uc *PaymentMethodUseCase) DeletePaymentMethod(ctx context.Context, id uuid.UUID) error {
	// Check if payment method exists
	existing, err := uc.paymentMethodRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if existing == nil {
		return errors.ErrNotFound
	}

	return uc.paymentMethodRepo.Delete(ctx, id)
}
