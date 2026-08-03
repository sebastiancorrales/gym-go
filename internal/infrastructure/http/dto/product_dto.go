package dto

import (
	"time"

	"github.com/sebastiancorrales/gym-go/internal/domain/entities"
)

// CreateProductRequest representa la solicitud para crear un producto.
//
// Nota sobre `unit_price`: con `required` el precio 0 se rechaza, así que no se
// pueden dar de alta productos gratuitos por la caja. Es intencional para un POS;
// `stock` en cambio sí admite 0 (producto agotado).
//
// Estos tags eran `validate:` — un tag que Gin no interpreta y que nadie había
// registrado —, así que hasta ahora este endpoint aceptaba cuerpos vacíos.
type CreateProductRequest struct {
	Name        string  `json:"name" binding:"required"`
	Description string  `json:"description"`
	UnitPrice   float64 `json:"unit_price" binding:"required,min=0"`
	Stock       int     `json:"stock" binding:"min=0"`
	Status      string  `json:"status,omitempty"`
}

// UpdateProductRequest representa la solicitud para actualizar un producto
type UpdateProductRequest struct {
	Name        string  `json:"name,omitempty"`
	Description string  `json:"description,omitempty"`
	UnitPrice   float64 `json:"unit_price,omitempty" binding:"min=0"`
	Stock       int     `json:"stock,omitempty" binding:"min=0"`
	Status      string  `json:"status,omitempty"`
}

// ProductResponse representa la respuesta de un producto
type ProductResponse struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	UnitPrice   float64   `json:"unit_price"`
	Stock       int       `json:"stock"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// UpdateStockRequest representa la solicitud para fijar el stock de un producto.
//
// El valor es ABSOLUTO, no un incremento: el modal de inventario se precarga con
// el stock actual y el operario escribe la cantidad que hay de verdad. Antes el
// backend lo interpretaba como delta y leía un campo `quantity` que el frontend
// nunca enviaba, así que el ajuste no hacía nada y aun así respondía 200.
//
// Es un puntero para poder distinguir "stock: 0" (agotado, válido) de no enviar
// el campo.
type UpdateStockRequest struct {
	Stock *int `json:"stock" binding:"required,min=0"`
}

// ProductSearchRequest representa la solicitud para buscar productos
type ProductSearchRequest struct {
	SearchTerm string `json:"search_term"`
	Status     string `json:"status,omitempty"`
}

// ToProductEntity convierte CreateProductRequest a Product entity
func (r *CreateProductRequest) ToEntity() *entities.Product {
	status := entities.ProductStatusActive
	if r.Status != "" {
		status = entities.ProductStatus(r.Status)
	}

	return &entities.Product{
		Name:        r.Name,
		Description: r.Description,
		UnitPrice:   r.UnitPrice,
		Stock:       r.Stock,
		Status:      status,
	}
}

// ToProductResponse convierte Product entity a ProductResponse
func ToProductResponse(product *entities.Product) *ProductResponse {
	return &ProductResponse{
		ID:          product.ID.String(),
		Name:        product.Name,
		Description: product.Description,
		UnitPrice:   product.UnitPrice,
		Stock:       product.Stock,
		Status:      string(product.Status),
		CreatedAt:   product.CreatedAt,
		UpdatedAt:   product.UpdatedAt,
	}
}

// ToProductResponseList convierte una lista de Product entities a ProductResponse
func ToProductResponseList(products []entities.Product) []ProductResponse {
	responses := make([]ProductResponse, len(products))
	for i, product := range products {
		responses[i] = *ToProductResponse(&product)
	}
	return responses
}
