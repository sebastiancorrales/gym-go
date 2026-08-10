package entities

import (
	"time"

	"github.com/google/uuid"
)

// StockMovementDirection es el sentido del movimiento.
//
// Se guarda el sentido y una cantidad siempre positiva, en lugar de una sola
// columna con signo. Una columna con signo haría el SUM trivial, pero "entradas
// y salidas" es el lenguaje del dominio y es lo que se filtra en pantalla. No se
// guardan las dos cosas: dos columnas que codifican el mismo dato acaban
// discrepando.
type StockMovementDirection string

const (
	StockDirectionIn  StockMovementDirection = "in"
	StockDirectionOut StockMovementDirection = "out"
)

// StockMovementReason es el PORQUÉ del movimiento.
//
// Es un enum en Go y no una tabla administrable a propósito: cada motivo tiene
// un camino de código que lo genera, así que añadir uno desde una pantalla sin
// escribir ese código no serviría de nada.
type StockMovementReason string

const (
	StockReasonInitial      StockMovementReason = "initial"       // alta de producto con stock, o backfill
	StockReasonSale         StockMovementReason = "sale"          // venta en el punto de venta
	StockReasonSaleVoid     StockMovementReason = "sale_void"     // anulación de venta
	StockReasonPurchase     StockMovementReason = "purchase"      // compra a proveedor
	StockReasonPurchaseVoid StockMovementReason = "purchase_void" // anulación de compra
	StockReasonCount        StockMovementReason = "count"         // conteo físico (ajuste a un valor absoluto)
	StockReasonAdjustment   StockMovementReason = "adjustment"    // entrada o salida manual
	StockReasonShrinkage    StockMovementReason = "shrinkage"     // merma: vencido, roto, robo
)

// IsManualStockReason indica si un motivo puede llegar desde un endpoint manual.
//
// Sin esta lista, un cliente podría mandar reason="sale" a /stock/adjust y
// ensuciar el reporte de ventas del kardex con movimientos que no corresponden a
// ninguna venta.
func IsManualStockReason(r StockMovementReason) bool {
	switch r {
	case StockReasonAdjustment, StockReasonShrinkage, StockReasonCount:
		return true
	}
	return false
}

// StockMovementRefType identifica contra qué documento se hizo el movimiento.
type StockMovementRefType string

const (
	StockRefSale    StockMovementRefType = "sale"
	StockRefExpense StockMovementRefType = "expense"
	StockRefProduct StockMovementRefType = "product" // alta inicial del producto
	StockRefManual  StockMovementRefType = "manual"  // ajuste sin documento
)

// StockMovement es el libro mayor del inventario: cada fila es un cambio de
// stock, y products.stock es su saldo acumulado.
//
// Antes de esto el stock era un contador mutable sin historia: un ajuste manual
// perdía el valor anterior para siempre y no quedaba rastro de quién lo hizo.
//
// Índices, cada uno con su consumidor real:
//   - idx_stock_movements_product_date → el kardex de UN producto en un rango
//   - idx_stock_movements_date         → el listado general por fecha
//   - idx_stock_movements_ref          → "¿qué movió esta venta?"
//
// Se filtra por la columna `date` (string local del gimnasio) y no por MovedAt,
// igual que en Sale: un movimiento de las 7 PM en Colombia es 00:xx UTC del día
// siguiente y caería en el cuadre equivocado.
type StockMovement struct {
	ID        uuid.UUID `json:"id" db:"id"`
	ProductID uuid.UUID `json:"product_id" db:"product_id" gorm:"index:idx_stock_movements_product_date,priority:1"`

	Direction StockMovementDirection `json:"direction" db:"direction"`
	Quantity  int                    `json:"quantity" db:"quantity"` // siempre positiva; el signo lo da Direction

	Reason StockMovementReason `json:"reason" db:"reason" gorm:"index:idx_stock_movements_reason"`

	RefType StockMovementRefType `json:"ref_type" db:"ref_type"`
	RefID   *uuid.UUID           `json:"ref_id,omitempty" db:"ref_id" gorm:"index:idx_stock_movements_ref"`

	// UnitCost es el costo unitario con el que entró la mercancía (compras) o el
	// costo vigente al salir (ventas, mermas). Congela el costo histórico: sin
	// esto, el margen de una venta de enero se reescribiría solo cada vez que se
	// vuelve a comprar el producto a otro precio.
	UnitCost float64 `json:"unit_cost" db:"unit_cost"`

	UserID *uuid.UUID `json:"user_id,omitempty" db:"user_id" gorm:"index:idx_stock_movements_user"`
	Note   string     `json:"note" db:"note"`

	// StockAfter es el saldo del producto DESPUÉS de aplicar este movimiento,
	// leído dentro de la misma transacción que lo escribió.
	//
	// Es lo que hace el libro autoverificable: ordenando los movimientos de un
	// producto, debe cumplirse stock_after[n] + signed[n+1] == stock_after[n+1].
	// Si no se cumple, alguien escribió products.stock sin pasar por el libro —
	// que es justo lo que una auditoría necesita poder detectar.
	//
	// Es nullable a propósito: el backfill histórico y cualquier movimiento
	// generado fuera de transacción lo dejan vacío, y eso significa "no
	// verificable", no cero.
	StockAfter *int `json:"stock_after,omitempty" db:"stock_after"`

	MovedAt   time.Time `json:"moved_at" db:"moved_at"`
	Date      string    `json:"date" db:"date" gorm:"index:idx_stock_movements_product_date,priority:2;index:idx_stock_movements_date"`
	Hour      string    `json:"hour" db:"hour"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`

	// Relación, no persistida.
	Product *Product `json:"product,omitempty" gorm:"-" db:"-"`
}

// SignedQuantity devuelve la cantidad con signo, para acumular saldos.
func (m *StockMovement) SignedQuantity() int {
	if m.Direction == StockDirectionOut {
		return -m.Quantity
	}
	return m.Quantity
}

// NewStockMovement construye un movimiento con la dirección deducida del signo
// del delta y la cantidad en positivo.
//
// Deja StockAfter vacío a propósito: sólo se conoce dentro de la transacción que
// aplica el cambio, y quien la ejecuta lo rellena. El resto de campos se fija
// aquí, ANTES de entrar a la transacción, porque el Unit of Work reintenta ante
// contención y exige que los UUID y los timestamps ya estén decididos.
func NewStockMovement(productID uuid.UUID, delta int, reason StockMovementReason, refType StockMovementRefType, refID *uuid.UUID, userID *uuid.UUID, movedAt time.Time, date, hour string) StockMovement {
	direction := StockDirectionIn
	quantity := delta
	if delta < 0 {
		direction = StockDirectionOut
		quantity = -delta
	}

	return StockMovement{
		ID:        uuid.New(),
		ProductID: productID,
		Direction: direction,
		Quantity:  quantity,
		Reason:    reason,
		RefType:   refType,
		RefID:     refID,
		UserID:    userID,
		MovedAt:   movedAt,
		Date:      date,
		Hour:      hour,
		CreatedAt: movedAt,
	}
}
