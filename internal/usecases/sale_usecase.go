package usecases

import (
	"context"
	"log"
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

	log.Printf("🔍 DEBUG CreateSale - Received %d details", len(sale.Details))
	for i, d := range sale.Details {
		log.Printf("🔍 DEBUG CreateSale - Detail %d: ProductID=%s, Quantity=%d", i, d.ProductID, d.Quantity)
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
		sale.SaleDate = time.Now()
	}
	sale.CreatedAt = time.Now()
	sale.UpdatedAt = time.Now()

	// Calculate totals
	sale.CalculateTotal()

	// Create sale
	if err := uc.saleRepo.Create(ctx, sale); err != nil {
		return err
	}

	// Create sale details
	log.Printf("📝 DEBUG CreateSale - Creating %d detail records in database", len(sale.Details))
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
			log.Printf("➖ DEBUG CreateSale - Decreasing stock for product %s: -%d", productID, totalQty)
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

	// DEBUG: Log details to understand the issue
	log.Printf("🔍 DEBUG VoidSale - Original sale ID: %s", originalSale.ID)
	log.Printf("🔍 DEBUG VoidSale - Details count: %d", len(details))
	for i, detail := range details {
		log.Printf("🔍 DEBUG VoidSale - Detail %d: ProductID=%s, Quantity=%d", i, detail.ProductID, detail.Quantity)
	}

	// Update original sale status
	originalSale.Status = entities.SaleStatusVoided
	originalSale.UpdatedAt = time.Now()
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
			log.Printf("⚠️ DEBUG VoidSale - Skipping duplicate product: %s", detail.ProductID)
			continue
		}

		// Calculate total quantity for this product across all details
		totalQty := 0
		for _, d := range details {
			if d.ProductID == detail.ProductID {
				totalQty += d.Quantity
			}
		}

		log.Printf("✅ DEBUG VoidSale - Restoring stock for product %s: +%d", detail.ProductID, totalQty)
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
	sale.Details = details

	return sale, nil
}

// GetAllSales retrieves all sales
func (uc *SaleUseCase) GetAllSales(ctx context.Context) ([]entities.Sale, error) {
	return uc.saleRepo.GetAll(ctx)
}

// GetSalesByDateRange retrieves sales within a date range
func (uc *SaleUseCase) GetSalesByDateRange(ctx context.Context, startDate, endDate time.Time, userID *uuid.UUID) ([]entities.Sale, error) {
	return uc.saleRepo.GetByDateRange(ctx, startDate, endDate, userID)
}

// GetSalesReport generates a sales report for a date range
func (uc *SaleUseCase) GetSalesReport(ctx context.Context, startDate, endDate time.Time, userID *uuid.UUID) ([]repositories.SaleReport, error) {
	return uc.saleRepo.GetSalesReport(ctx, startDate, endDate, userID)
}

// GetSalesReportByProduct generates a sales report grouped by product
func (uc *SaleUseCase) GetSalesReportByProduct(ctx context.Context, startDate, endDate time.Time) ([]repositories.SaleProductReport, error) {
	return uc.saleRepo.GetSalesReportByProduct(ctx, startDate, endDate)
}
