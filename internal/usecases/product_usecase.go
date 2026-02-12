package usecases

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/sebastiancorrales/gym-go/internal/domain/entities"
	"github.com/sebastiancorrales/gym-go/internal/domain/repositories"
	"github.com/sebastiancorrales/gym-go/pkg/errors"
)

// ProductUseCase handles business logic for products
type ProductUseCase struct {
	productRepo repositories.ProductRepository
}

// NewProductUseCase creates a new ProductUseCase
func NewProductUseCase(productRepo repositories.ProductRepository) *ProductUseCase {
	return &ProductUseCase{
		productRepo: productRepo,
	}
}

// CreateProduct creates a new product
func (uc *ProductUseCase) CreateProduct(ctx context.Context, product *entities.Product) error {
	if product.Name == "" {
		return errors.ErrInvalidInput
	}
	if product.UnitPrice < 0 {
		return errors.ErrInvalidPrice
	}
	if product.Stock < 0 {
		return errors.ErrInvalidQuantity
	}

	// Set default values
	product.ID = uuid.New()
	if product.Status == "" {
		product.Status = entities.ProductStatusActive
	}
	product.CreatedAt = time.Now()
	product.UpdatedAt = time.Now()

	return uc.productRepo.Create(ctx, product)
}

// GetProductByID retrieves a product by ID
func (uc *ProductUseCase) GetProductByID(ctx context.Context, id uuid.UUID) (*entities.Product, error) {
	return uc.productRepo.GetByID(ctx, id)
}

// GetAllProducts retrieves all products, optionally filtered by status
func (uc *ProductUseCase) GetAllProducts(ctx context.Context, status *entities.ProductStatus) ([]entities.Product, error) {
	return uc.productRepo.GetAll(ctx, status)
}

// UpdateProduct updates an existing product
func (uc *ProductUseCase) UpdateProduct(ctx context.Context, product *entities.Product) error {
	if product.ID == uuid.Nil {
		return errors.ErrInvalidInput
	}
	if product.Name == "" {
		return errors.ErrInvalidInput
	}
	if product.UnitPrice < 0 {
		return errors.ErrInvalidPrice
	}
	if product.Stock < 0 {
		return errors.ErrInvalidQuantity
	}

	// Check if product exists
	existing, err := uc.productRepo.GetByID(ctx, product.ID)
	if err != nil {
		return err
	}
	if existing == nil {
		return errors.ErrNotFound
	}

	product.UpdatedAt = time.Now()
	product.CreatedAt = existing.CreatedAt // Preserve creation date

	return uc.productRepo.Update(ctx, product)
}

// DeleteProduct deletes a product
func (uc *ProductUseCase) DeleteProduct(ctx context.Context, id uuid.UUID) error {
	// Check if product exists
	existing, err := uc.productRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if existing == nil {
		return errors.ErrNotFound
	}

	return uc.productRepo.Delete(ctx, id)
}

// UpdateProductStock updates the stock of a product
func (uc *ProductUseCase) UpdateProductStock(ctx context.Context, productID uuid.UUID, quantity int) error {
	product, err := uc.productRepo.GetByID(ctx, productID)
	if err != nil {
		return err
	}
	if product == nil {
		return errors.ErrNotFound
	}

	newStock := product.Stock + quantity
	if newStock < 0 {
		return errors.ErrInsufficientStock
	}

	return uc.productRepo.UpdateStock(ctx, productID, quantity)
}

// SearchProducts searches for products by name or description
func (uc *ProductUseCase) SearchProducts(ctx context.Context, searchTerm string) ([]entities.Product, error) {
	return uc.productRepo.Search(ctx, searchTerm)
}
