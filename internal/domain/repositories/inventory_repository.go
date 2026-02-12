package repositories

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/sebastiancorrales/gym-go/internal/domain/entities"
)

// ProductRepository defines the interface for product data operations
type ProductRepository interface {
	Create(ctx context.Context, product *entities.Product) error
	GetByID(ctx context.Context, id uuid.UUID) (*entities.Product, error)
	GetAll(ctx context.Context, status *entities.ProductStatus) ([]entities.Product, error)
	Update(ctx context.Context, product *entities.Product) error
	Delete(ctx context.Context, id uuid.UUID) error
	UpdateStock(ctx context.Context, productID uuid.UUID, quantity int) error
	Search(ctx context.Context, searchTerm string) ([]entities.Product, error)
}

// SaleRepository defines the interface for sale data operations
type SaleRepository interface {
	Create(ctx context.Context, sale *entities.Sale) error
	GetByID(ctx context.Context, id uuid.UUID) (*entities.Sale, error)
	GetAll(ctx context.Context) ([]entities.Sale, error)
	Update(ctx context.Context, sale *entities.Sale) error
	GetByDateRange(ctx context.Context, startDate, endDate time.Time, userID *uuid.UUID) ([]entities.Sale, error)
	GetSalesReport(ctx context.Context, startDate, endDate time.Time, userID *uuid.UUID) ([]SaleReport, error)
	GetSalesReportByProduct(ctx context.Context, startDate, endDate time.Time) ([]SaleProductReport, error)
}

// SaleDetailRepository defines the interface for sale detail data operations
type SaleDetailRepository interface {
	Create(ctx context.Context, detail *entities.SaleDetail) error
	CreateBatch(ctx context.Context, saleID uuid.UUID, details []entities.SaleDetail) error
	GetBySaleID(ctx context.Context, saleID uuid.UUID) ([]entities.SaleDetail, error)
	GetByID(ctx context.Context, id uuid.UUID) (*entities.SaleDetail, error)
}

// PaymentMethodRepository defines the interface for payment method data operations
type PaymentMethodRepository interface {
	Create(ctx context.Context, method *entities.SalePaymentMethod) error
	GetByID(ctx context.Context, id uuid.UUID) (*entities.SalePaymentMethod, error)
	GetAll(ctx context.Context, status *entities.PaymentMethodStatus) ([]entities.SalePaymentMethod, error)
	Update(ctx context.Context, method *entities.SalePaymentMethod) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// SaleReport represents aggregated sale data
type SaleReport struct {
	TotalSales    float64 `json:"total_sales"`
	TotalDiscount float64 `json:"total_discount"`
	NetSales      float64 `json:"net_sales"`
	SalesCount    int     `json:"sales_count"`
}

// SaleProductReport represents sales grouped by product
type SaleProductReport struct {
	ProductID     uuid.UUID `json:"product_id"`
	ProductName   string    `json:"product_name"`
	QuantitySold  int       `json:"quantity_sold"`
	TotalRevenue  float64   `json:"total_revenue"`
	TotalDiscount float64   `json:"total_discount"`
	NetRevenue    float64   `json:"net_revenue"`
}
