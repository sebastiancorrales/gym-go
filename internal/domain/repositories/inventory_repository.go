package repositories

import (
	"context"

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

	// ApplyStockDelta suma delta al stock en un ÚNICO UPDATE condicional y
	// devuelve el stock resultante.
	//
	// Es la única forma soportada de mover stock. Sustituye a UpdateStock (que no
	// tenía guarda y podía dejar el stock negativo) y a SetStock (que pisaba el
	// valor sin mirar). La guarda va en el WHERE y no en Go: leer el stock,
	// comprobarlo y luego escribirlo es la carrera que dejaba el stock negativo
	// cuando dos cajas vendían la última unidad a la vez.
	//
	// Devuelve el stock nuevo para que quien registre el movimiento pueda anotar
	// el saldo resultante sin una segunda lectura. Errores: ErrInsufficientStock
	// si el delta es negativo y no hay saldo, ErrNotFound si el producto no
	// existe.
	ApplyStockDelta(ctx context.Context, productID uuid.UUID, delta int) (newStock int, err error)

	// ApplyPurchase suma qty al stock y recalcula el costo promedio ponderado.
	//
	// Es una operación distinta de ApplyStockDelta porque el promedio necesita el
	// stock y el costo PREVIOS: calcularlo fuera de la transacción lo dejaría
	// obsoleto en cuanto otra compra del mismo producto se cuele en medio.
	// Debe llamarse dentro de una transacción.
	ApplyPurchase(ctx context.Context, productID uuid.UUID, qty int, unitCost float64) (newStock int, newAvgCost float64, err error)

	// DecrementStock subtracts qty atomically, failing with ErrInsufficientStock
	// if there is not enough.
	DecrementStock(ctx context.Context, productID uuid.UUID, qty int) error
	// IncrementStock returns qty to stock (voided sales).
	IncrementStock(ctx context.Context, productID uuid.UUID, qty int) error
	Search(ctx context.Context, searchTerm string) ([]entities.Product, error)
}

// StockMovementRepository es el libro mayor del inventario.
type StockMovementRepository interface {
	CreateBatch(ctx context.Context, movements []entities.StockMovement) error
	// ListByProduct devuelve el kardex de un producto en un rango de fechas
	// locales (YYYY-MM-DD), más reciente primero.
	ListByProduct(ctx context.Context, productID uuid.UUID, startDate, endDate string) ([]entities.StockMovement, error)
	// List devuelve los movimientos de un rango, con filtros opcionales.
	List(ctx context.Context, startDate, endDate string, productID *uuid.UUID, reason *entities.StockMovementReason) ([]entities.StockMovement, error)
	// SumSignedByProduct devuelve el saldo acumulado del libro por producto. Es
	// lo que permite comprobar que el kardex cuadra con products.stock.
	SumSignedByProduct(ctx context.Context) (map[uuid.UUID]int, error)
}

// SaleRepository defines the interface for sale data operations
type SaleRepository interface {
	Create(ctx context.Context, sale *entities.Sale) error
	GetByID(ctx context.Context, id uuid.UUID) (*entities.Sale, error)
	GetAll(ctx context.Context) ([]entities.Sale, error)
	Update(ctx context.Context, sale *entities.Sale) error
	GetByDateRange(ctx context.Context, startDate, endDate string, userID *uuid.UUID) ([]entities.Sale, error)
	GetSalesReport(ctx context.Context, startDate, endDate string, userID *uuid.UUID) ([]SaleReport, error)
	GetSalesReportByProduct(ctx context.Context, startDate, endDate string) ([]SaleProductReport, error)
	// FindVoidBySaleID localiza la venta espejo de una anulación. Devuelve nil si
	// la venta no está anulada. Requiere el índice idx_sales_voided: sin él sería
	// un escaneo completo de una tabla append-only bajo lock de escritura.
	FindVoidBySaleID(ctx context.Context, originalSaleID uuid.UUID) (*entities.Sale, error)
}

// SaleDetailRepository defines the interface for sale detail data operations
type SaleDetailRepository interface {
	Create(ctx context.Context, detail *entities.SaleDetail) error
	CreateBatch(ctx context.Context, saleID uuid.UUID, details []entities.SaleDetail) error
	GetBySaleID(ctx context.Context, saleID uuid.UUID) ([]entities.SaleDetail, error)
	// GetBySaleIDs resolves the line items of many sales in one query, grouped by
	// sale ID. Use it instead of calling GetBySaleID per sale in a report.
	GetBySaleIDs(ctx context.Context, saleIDs []uuid.UUID) (map[uuid.UUID][]entities.SaleDetail, error)
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

// PaymentLineRepository es el libro de pagos: las patas (método, monto) de cada
// venta, suscripción o gasto.
type PaymentLineRepository interface {
	CreateBatch(ctx context.Context, lines []entities.PaymentLine) error
	GetByPayable(ctx context.Context, kind entities.PayableKind, payableID uuid.UUID) ([]entities.PaymentLine, error)
	// GetByPayables resuelve las líneas de muchos documentos en una consulta.
	GetByPayables(ctx context.Context, kind entities.PayableKind, payableIDs []uuid.UUID) (map[uuid.UUID][]entities.PaymentLine, error)
	// SumByMethod agrega EN LA BASE: es la única fuente del desglose por método
	// de pago, y sustituye a los seis sitios que hoy lo reconcilian por etiqueta.
	SumByMethod(ctx context.Context, startDate, endDate string) ([]PaymentMethodTotal, error)
	// ReplaceForPayable borra e inserta las líneas de un documento, para poder
	// corregir el método de pago sin dejar duplicados ante un reintento.
	ReplaceForPayable(ctx context.Context, kind entities.PayableKind, payableID uuid.UUID, lines []entities.PaymentLine) error
}

// PaymentMethodTotal es una fila del desglose por método de pago.
//
// La forma refleja a propósito la de email.PaymentMethodSummary, para que la
// pantalla del cuadre y el correo del cierre diario produzcan números idénticos.
type PaymentMethodTotal struct {
	PaymentMethodID   uuid.UUID `json:"payment_method_id"`
	PaymentMethodName string    `json:"payment_method_name"`
	PaymentMethodType string    `json:"payment_method_type"`
	PayableKind       string    `json:"payable_kind"`
	Total             float64   `json:"total"`
	ChangeGiven       float64   `json:"change_given"`
	Count             int       `json:"count"`
}

// ChangeLogRepository es la bitácora de correcciones manuales.
type ChangeLogRepository interface {
	Create(ctx context.Context, entry *entities.ChangeLog) error
	FindByEntity(ctx context.Context, entityType string, entityID uuid.UUID) ([]entities.ChangeLog, error)
}
