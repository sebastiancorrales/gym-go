package dto

import (
	"time"

	"github.com/google/uuid"
	"github.com/sebastiancorrales/gym-go/internal/domain/entities"
	"github.com/sebastiancorrales/gym-go/internal/domain/repositories"
)

// SaleDetailRequest representa un detalle de venta en una solicitud
type SaleDetailRequest struct {
	ProductID string  `json:"product_id" validate:"required"`
	Quantity  int     `json:"quantity" validate:"required,min=1"`
	UnitPrice float64 `json:"unit_price,omitempty"`
	Discount  float64 `json:"discount,omitempty" validate:"min=0"`
}

// CreateSaleRequest representa la solicitud para crear una venta
type CreateSaleRequest struct {
	PaymentMethodID string              `json:"payment_method_id" validate:"required"`
	Details         []SaleDetailRequest `json:"details" validate:"required,min=1"`
}

// VoidSaleRequest representa la solicitud para anular una venta
type VoidSaleRequest struct {
	Reason string `json:"reason,omitempty"`
}

// SaleDetailResponse representa un detalle de venta en una respuesta
type SaleDetailResponse struct {
	ID          string           `json:"id"`
	ProductID   string           `json:"product_id"`
	ProductName string           `json:"product_name,omitempty"`
	UnitPrice   float64          `json:"unit_price"`
	Quantity    int              `json:"quantity"`
	TotalPrice  float64          `json:"total_price"`
	Discount    float64          `json:"discount"`
	Subtotal    float64          `json:"subtotal"`
	CreatedAt   time.Time        `json:"created_at"`
	Product     *ProductResponse `json:"product,omitempty"`
}

// SaleResponse representa la respuesta de una venta
type SaleResponse struct {
	ID              string               `json:"id"`
	SaleDate        time.Time            `json:"sale_date"`
	Total           float64              `json:"total"`
	TotalDiscount   float64              `json:"total_discount"`
	UserID          string               `json:"user_id"`
	Type            string               `json:"type"`
	Status          string               `json:"status"`
	PaymentMethodID string               `json:"payment_method_id"`
	VoidedSaleID    *string              `json:"voided_sale_id,omitempty"`
	CreatedAt       time.Time            `json:"created_at"`
	UpdatedAt       time.Time            `json:"updated_at"`
	Details         []SaleDetailResponse `json:"details,omitempty"`
}

// SaleReportRequest representa la solicitud para un reporte de ventas
type SaleReportRequest struct {
	StartDate string  `json:"start_date" validate:"required"`
	EndDate   string  `json:"end_date" validate:"required"`
	UserID    *string `json:"user_id,omitempty"`
}

// SaleReportResponse representa el reporte de ventas
type SaleReportResponse struct {
	TotalSales    float64   `json:"total_sales"`
	TotalDiscount float64   `json:"total_discount"`
	NetSales      float64   `json:"net_sales"`
	SalesCount    int       `json:"sales_count"`
	StartDate     time.Time `json:"start_date"`
	EndDate       time.Time `json:"end_date"`
}

// SaleProductReportResponse representa ventas agrupadas por producto
type SaleProductReportResponse struct {
	ProductID     string  `json:"product_id"`
	ProductName   string  `json:"product_name"`
	QuantitySold  int     `json:"quantity_sold"`
	TotalRevenue  float64 `json:"total_revenue"`
	TotalDiscount float64 `json:"total_discount"`
	NetRevenue    float64 `json:"net_revenue"`
}

// ToEntity convierte CreateSaleRequest a Sale entity
func (r *CreateSaleRequest) ToEntity(userID uuid.UUID) (*entities.Sale, error) {
	paymentMethodID, err := uuid.Parse(r.PaymentMethodID)
	if err != nil {
		return nil, err
	}

	details := make([]entities.SaleDetail, len(r.Details))
	for i, d := range r.Details {
		productID, err := uuid.Parse(d.ProductID)
		if err != nil {
			return nil, err
		}
		details[i] = entities.SaleDetail{
			ProductID: productID,
			Quantity:  d.Quantity,
			UnitPrice: d.UnitPrice,
			Discount:  d.Discount,
		}
	}

	return &entities.Sale{
		UserID:          userID,
		PaymentMethodID: paymentMethodID,
		Details:         details,
		Type:            entities.SaleTypeNormal,
		Status:          entities.SaleStatusCompleted,
	}, nil
}

// ToSaleResponse convierte Sale entity a SaleResponse
func ToSaleResponse(sale *entities.Sale) *SaleResponse {
	var voidedSaleID *string
	if sale.VoidedSaleID != nil {
		id := sale.VoidedSaleID.String()
		voidedSaleID = &id
	}

	response := &SaleResponse{
		ID:              sale.ID.String(),
		SaleDate:        sale.SaleDate,
		Total:           sale.Total,
		TotalDiscount:   sale.TotalDiscount,
		UserID:          sale.UserID.String(),
		Type:            string(sale.Type),
		Status:          string(sale.Status),
		PaymentMethodID: sale.PaymentMethodID.String(),
		VoidedSaleID:    voidedSaleID,
		CreatedAt:       sale.CreatedAt,
		UpdatedAt:       sale.UpdatedAt,
	}

	if len(sale.Details) > 0 {
		response.Details = make([]SaleDetailResponse, len(sale.Details))
		for i, detail := range sale.Details {
			response.Details[i] = ToSaleDetailResponse(&detail)
		}
	}

	return response
}

// ToSaleDetailResponse convierte SaleDetail entity a SaleDetailResponse
func ToSaleDetailResponse(detail *entities.SaleDetail) SaleDetailResponse {
	response := SaleDetailResponse{
		ID:         detail.ID.String(),
		ProductID:  detail.ProductID.String(),
		UnitPrice:  detail.UnitPrice,
		Quantity:   detail.Quantity,
		TotalPrice: detail.TotalPrice,
		Discount:   detail.Discount,
		Subtotal:   detail.Subtotal,
		CreatedAt:  detail.CreatedAt,
	}

	if detail.Product != nil {
		response.ProductName = detail.Product.Name
		response.Product = ToProductResponse(detail.Product)
	}

	return response
}

// ToSaleResponseList convierte una lista de Sale entities a SaleResponse
func ToSaleResponseList(sales []entities.Sale) []SaleResponse {
	responses := make([]SaleResponse, len(sales))
	for i, sale := range sales {
		responses[i] = *ToSaleResponse(&sale)
	}
	return responses
}

// ToSaleReportResponse convierte SaleReport a SaleReportResponse
func ToSaleReportResponse(report repositories.SaleReport, startDate, endDate time.Time) SaleReportResponse {
	return SaleReportResponse{
		TotalSales:    report.TotalSales,
		TotalDiscount: report.TotalDiscount,
		NetSales:      report.NetSales,
		SalesCount:    report.SalesCount,
		StartDate:     startDate,
		EndDate:       endDate,
	}
}

// ToSaleProductReportResponseList convierte SaleProductReport a lista de respuestas
func ToSaleProductReportResponseList(reports []repositories.SaleProductReport) []SaleProductReportResponse {
	responses := make([]SaleProductReportResponse, len(reports))
	for i, report := range reports {
		responses[i] = SaleProductReportResponse{
			ProductID:     report.ProductID.String(),
			ProductName:   report.ProductName,
			QuantitySold:  report.QuantitySold,
			TotalRevenue:  report.TotalRevenue,
			TotalDiscount: report.TotalDiscount,
			NetRevenue:    report.NetRevenue,
		}
	}
	return responses
}
