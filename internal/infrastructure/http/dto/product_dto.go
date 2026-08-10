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
	MinStock    int     `json:"min_stock" binding:"min=0"`
	Status      string  `json:"status,omitempty"`
}

// UpdateProductRequest representa la solicitud para actualizar un producto.
//
// Todos los campos son punteros para distinguir "no enviado" de "enviado con el
// valor cero". Sin punteros hacía falta un `if req.X > 0` por campo, y ese
// patrón fue exactamente el bug del stock: `if req.Stock >= 0` es cierto para el
// cero de Go, así que un PUT sin el campo ponía el stock a cero en silencio.
//
// `stock` NO está aquí: el stock es propiedad del libro de movimientos y sólo se
// mueve por /stock (conteo) o /stock/adjust (entrada o salida), que dejan asiento.
type UpdateProductRequest struct {
	Name        *string  `json:"name,omitempty"`
	Description *string  `json:"description,omitempty"`
	UnitPrice   *float64 `json:"unit_price,omitempty" binding:"omitempty,min=0"`
	MinStock    *int     `json:"min_stock,omitempty" binding:"omitempty,min=0"`
	Status      *string  `json:"status,omitempty"`
}

// AdjustStockRequest representa una entrada o salida manual de inventario.
//
// A diferencia de UpdateStockRequest, el valor es un DELTA con signo: positivo
// entra mercancía, negativo sale. El motivo es obligatorio porque un movimiento
// sin causa no sirve para auditar nada, y sólo se aceptan los motivos manuales
// (ajuste, merma, conteo).
type AdjustStockRequest struct {
	Delta  int    `json:"delta" binding:"required"`
	Reason string `json:"reason" binding:"required"`
	Note   string `json:"note,omitempty"`
}

// ProductResponse representa la respuesta de un producto
type ProductResponse struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	UnitPrice   float64 `json:"unit_price"`
	Stock       int     `json:"stock"`
	MinStock    int     `json:"min_stock"`
	LowStock    bool    `json:"low_stock"`
	// CostPrice es puntero con omitempty porque revela el margen del negocio:
	// GET /products lo consume el punto de venta, que usan recepción y staff. Sólo
	// se rellena para quien tiene PermInventoryCost. Un float64 plano lo habría
	// filtrado a todo el personal en cuanto se añadió la columna.
	CostPrice *float64  `json:"cost_price,omitempty"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// StockMovementResponse representa un asiento del kardex.
type StockMovementResponse struct {
	ID          string    `json:"id"`
	ProductID   string    `json:"product_id"`
	ProductName string    `json:"product_name,omitempty"`
	Direction   string    `json:"direction"`
	Quantity    int       `json:"quantity"`
	Reason      string    `json:"reason"`
	RefType     string    `json:"ref_type"`
	RefID       string    `json:"ref_id,omitempty"`
	StockAfter  *int      `json:"stock_after,omitempty"`
	// UnitCost filtra el margen igual que Product.CostPrice: recepción puede ver
	// el kardex (PermInventoryMovements) pero no lo que costó la mercancía.
	UnitCost *float64 `json:"unit_cost,omitempty"`
	UserID      string    `json:"user_id,omitempty"`
	Note        string    `json:"note,omitempty"`
	Date        string    `json:"date"`
	Hour        string    `json:"hour"`
	MovedAt     time.Time `json:"moved_at"`
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
	Stock *int   `json:"stock" binding:"required,min=0"`
	Note  string `json:"note,omitempty"`
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
		MinStock:    r.MinStock,
		Status:      status,
	}
}

// ToProductResponse convierte Product entity a ProductResponse
//
// includeCost decide si el costo viaja al cliente. Es un parametro y no un campo
// siempre presente porque el costo revela el margen del negocio, y quien llama
// es quien conoce el permiso del usuario.
func ToProductResponse(product *entities.Product, includeCost bool) *ProductResponse {
	resp := &ProductResponse{
		ID:          product.ID.String(),
		Name:        product.Name,
		Description: product.Description,
		UnitPrice:   product.UnitPrice,
		Stock:       product.Stock,
		MinStock:    product.MinStock,
		LowStock:    product.IsLowStock(),
		Status:      string(product.Status),
		CreatedAt:   product.CreatedAt,
		UpdatedAt:   product.UpdatedAt,
	}
	if includeCost {
		cost := product.CostPrice
		resp.CostPrice = &cost
	}
	return resp
}

// ToStockMovementResponse convierte un asiento del kardex a su respuesta.
func ToStockMovementResponse(m *entities.StockMovement, includeCost bool) *StockMovementResponse {
	resp := &StockMovementResponse{
		ID:         m.ID.String(),
		ProductID:  m.ProductID.String(),
		Direction:  string(m.Direction),
		Quantity:   m.Quantity,
		Reason:     string(m.Reason),
		RefType:    string(m.RefType),
		StockAfter: m.StockAfter,
		Note:       m.Note,
		Date:       m.Date,
		Hour:       m.Hour,
		MovedAt:    m.MovedAt,
	}
	if m.RefID != nil {
		resp.RefID = m.RefID.String()
	}
	if m.UserID != nil {
		resp.UserID = m.UserID.String()
	}
	if m.Product != nil {
		resp.ProductName = m.Product.Name
	}
	if includeCost {
		cost := m.UnitCost
		resp.UnitCost = &cost
	}
	return resp
}

// ToStockMovementResponseList convierte una lista de asientos.
func ToStockMovementResponseList(movements []entities.StockMovement, includeCost bool) []StockMovementResponse {
	responses := make([]StockMovementResponse, len(movements))
	for i := range movements {
		responses[i] = *ToStockMovementResponse(&movements[i], includeCost)
	}
	return responses
}

// ToProductResponseList convierte una lista de Product entities a ProductResponse
func ToProductResponseList(products []entities.Product, includeCost bool) []ProductResponse {
	responses := make([]ProductResponse, len(products))
	for i := range products {
		responses[i] = *ToProductResponse(&products[i], includeCost)
	}
	return responses
}
