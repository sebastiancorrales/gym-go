package usecases

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/sebastiancorrales/gym-go/internal/domain/entities"
	"github.com/sebastiancorrales/gym-go/internal/domain/repositories"
	"github.com/sebastiancorrales/gym-go/pkg/errors"
)

// SaleUseCase handles business logic for sales
type SaleUseCase struct {
	saleRepo          repositories.SaleRepository
	saleDetailRepo    repositories.SaleDetailRepository
	productRepo       repositories.ProductRepository
	paymentMethodRepo repositories.PaymentMethodRepository
}

// NewSaleUseCase creates a new SaleUseCase
func NewSaleUseCase(
	saleRepo repositories.SaleRepository,
	saleDetailRepo repositories.SaleDetailRepository,
	productRepo repositories.ProductRepository,
	paymentMethodRepo repositories.PaymentMethodRepository,
) *SaleUseCase {
	return &SaleUseCase{
		saleRepo:          saleRepo,
		saleDetailRepo:    saleDetailRepo,
		productRepo:       productRepo,
		paymentMethodRepo: paymentMethodRepo,
	}
}

// CreateSale creates a new sale with its details
// This function handles inventory updates and transaction management
func (uc *SaleUseCase) CreateSale(ctx context.Context, sale *entities.Sale) error {
	// Validate sale
	if sale.UserID == uuid.Nil {
		return errors.ErrInvalidInput
	}
	if sale.PaymentMethodID == uuid.Nil {
		return errors.ErrInvalidInput
	}
	if len(sale.Details) == 0 {
		return errors.ErrInvalidInput
	}

	// Validate payment method
	paymentMethod, err := uc.paymentMethodRepo.GetByID(ctx, sale.PaymentMethodID)
	if err != nil {
		return err
	}
	if paymentMethod == nil {
		return errors.ErrNotFound
	}
	if !paymentMethod.IsActive() {
		return errors.ErrPaymentMethodNotActive
	}

	// Validate all details and check stock
	productMap := make(map[uuid.UUID]*entities.Product)
	for i := range sale.Details {
		detail := &sale.Details[i]

		// Validate detail
		if err := detail.Validate(); err != nil {
			return err
		}

		// Check if product exists and is active
		product, exists := productMap[detail.ProductID]
		if !exists {
			product, err = uc.productRepo.GetByID(ctx, detail.ProductID)
			if err != nil {
				return err
			}
			if product == nil {
				return errors.ErrNotFound
			}
			if !product.IsActive() {
				return errors.ErrProductNotActive
			}
			productMap[detail.ProductID] = product
		}

		// Check stock availability
		if !product.HasStock(detail.Quantity) {
			return errors.ErrInsufficientStock
		}

		// Set unit price from product if not set
		if detail.UnitPrice == 0 {
			detail.UnitPrice = product.UnitPrice
		}

		// Calculate subtotal
		detail.CalculateSubtotal()
	}

	// Set sale defaults
	sale.ID = uuid.New()
	if sale.Type == "" {
		sale.Type = entities.SaleTypeNormal
	}
	if sale.Status == "" {
		sale.Status = entities.SaleStatusCompleted
	}
	if sale.SaleDate.IsZero() {
		sale.SaleDate = time.Now().UTC().Round(0)
	}
	sale.CreatedAt = time.Now().UTC().Round(0)
	sale.UpdatedAt = time.Now().UTC().Round(0)

	// Calculate totals
	sale.CalculateTotal()

	// Create sale
	if err := uc.saleRepo.Create(ctx, sale); err != nil {
		return err
	}

	// Create sale details
	if err := uc.saleDetailRepo.CreateBatch(ctx, sale.ID, sale.Details); err != nil {
		return err
	}

	// Update inventory for normal sales
	if sale.IsNormal() {
		for productID, product := range productMap {
			// Calculate total quantity for this product
			totalQty := 0
			for _, detail := range sale.Details {
				if detail.ProductID == productID {
					totalQty += detail.Quantity
				}
			}

			// Decrease stock
			if err := product.DecreaseStock(totalQty); err != nil {
				return err
			}
			if err := uc.productRepo.UpdateStock(ctx, productID, -totalQty); err != nil {
				return err
			}
		}
	}

	return nil
}

// VoidSale voids an existing sale and creates a void transaction
// This function restores inventory and maintains traceability
func (uc *SaleUseCase) VoidSale(ctx context.Context, saleID uuid.UUID, userID uuid.UUID) (*entities.Sale, error) {
	// Get original sale
	originalSale, err := uc.saleRepo.GetByID(ctx, saleID)
	if err != nil {
		return nil, err
	}
	if originalSale == nil {
		return nil, errors.ErrNotFound
	}

	// Check if sale can be voided
	if !originalSale.CanBeVoided() {
		return nil, errors.ErrSaleCannotBeVoided
	}

	// Get sale details
	details, err := uc.saleDetailRepo.GetBySaleID(ctx, saleID)
	if err != nil {
		return nil, err
	}

	// Update original sale status
	originalSale.Status = entities.SaleStatusVoided
	originalSale.UpdatedAt = time.Now().UTC().Round(0)
	if err := uc.saleRepo.Update(ctx, originalSale); err != nil {
		return nil, err
	}

	// Create void sale
	voidSale := &entities.Sale{
		ID:              uuid.New(),
		SaleDate:        time.Now(),
		Total:           -originalSale.Total,
		TotalDiscount:   -originalSale.TotalDiscount,
		UserID:          userID,
		Type:            entities.SaleTypeVoid,
		Status:          entities.SaleStatusCompleted,
		PaymentMethodID: originalSale.PaymentMethodID,
		VoidedSaleID:    &originalSale.ID,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	// Create void sale record
	if err := uc.saleRepo.Create(ctx, voidSale); err != nil {
		return nil, err
	}

	// Create void sale details (negative quantities)
	voidDetails := make([]entities.SaleDetail, len(details))
	for i, detail := range details {
		voidDetails[i] = entities.SaleDetail{
			SaleID:     voidSale.ID,
			ProductID:  detail.ProductID,
			UnitPrice:  detail.UnitPrice,
			Quantity:   detail.Quantity, // Keep positive for clarity
			TotalPrice: -detail.TotalPrice,
			Discount:   -detail.Discount,
			Subtotal:   -detail.Subtotal,
			CreatedAt:  time.Now(),
		}
	}

	if err := uc.saleDetailRepo.CreateBatch(ctx, voidSale.ID, voidDetails); err != nil {
		return nil, err
	}

	// Restore inventory - Restore ONLY ONCE per product
	restoredProducts := make(map[uuid.UUID]bool)
	for _, detail := range details {
		// Skip if already restored (in case of duplicate details)
		if restoredProducts[detail.ProductID] {
			continue
		}

		// Calculate total quantity for this product across all details
		totalQty := 0
		for _, d := range details {
			if d.ProductID == detail.ProductID {
				totalQty += d.Quantity
			}
		}

		if err := uc.productRepo.UpdateStock(ctx, detail.ProductID, totalQty); err != nil {
			return nil, err
		}
		restoredProducts[detail.ProductID] = true
	}

	return voidSale, nil
}

// GetSaleByID retrieves a sale by ID with its details
func (uc *SaleUseCase) GetSaleByID(ctx context.Context, id uuid.UUID) (*entities.Sale, error) {
	sale, err := uc.saleRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if sale == nil {
		return nil, errors.ErrNotFound
	}

	// Load details
	details, err := uc.saleDetailRepo.GetBySaleID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Load product info for each detail
	for i := range details {
		product, err := uc.productRepo.GetByID(ctx, details[i].ProductID)
		if err == nil && product != nil {
			details[i].Product = product
		}
	}
	sale.Details = details

	// Load payment method
	pm, err := uc.paymentMethodRepo.GetByID(ctx, sale.PaymentMethodID)
	if err == nil && pm != nil {
		sale.PaymentMethod = pm
	}

	return sale, nil
}

// GetAllSales retrieves all sales
func (uc *SaleUseCase) GetAllSales(ctx context.Context) ([]entities.Sale, error) {
	sales, err := uc.saleRepo.GetAll(ctx)
	if err != nil {
		return nil, err
	}
	uc.loadPaymentMethods(ctx, sales)
	return sales, nil
}

// GetSalesByDateRange retrieves sales within a date range
func (uc *SaleUseCase) GetSalesByDateRange(ctx context.Context, startDate, endDate time.Time, userID *uuid.UUID) ([]entities.Sale, error) {
	sales, err := uc.saleRepo.GetByDateRange(ctx, startDate, endDate, userID)
	if err != nil {
		return nil, err
	}
	uc.loadPaymentMethods(ctx, sales)
	return sales, nil
}

// GetSalesReport generates a sales report for a date range
func (uc *SaleUseCase) GetSalesReport(ctx context.Context, startDate, endDate time.Time, userID *uuid.UUID) ([]repositories.SaleReport, error) {
	return uc.saleRepo.GetSalesReport(ctx, startDate, endDate, userID)
}

// loadPaymentMethods batch-loads payment methods for a list of sales
func (uc *SaleUseCase) loadPaymentMethods(ctx context.Context, sales []entities.Sale) {
	pmMap := make(map[uuid.UUID]*entities.SalePaymentMethod)
	for i := range sales {
		pmID := sales[i].PaymentMethodID
		if _, exists := pmMap[pmID]; exists {
			continue
		}
		pm, err := uc.paymentMethodRepo.GetByID(ctx, pmID)
		if err == nil && pm != nil {
			pmMap[pmID] = pm
		}
	}
	for i := range sales {
		if pm, ok := pmMap[sales[i].PaymentMethodID]; ok {
			sales[i].PaymentMethod = pm
		}
	}
}

// GetSalesReportByProduct generates a sales report grouped by product
func (uc *SaleUseCase) GetSalesReportByProduct(ctx context.Context, startDate, endDate time.Time) ([]repositories.SaleProductReport, error) {
	return uc.saleRepo.GetSalesReportByProduct(ctx, startDate, endDate)
}
