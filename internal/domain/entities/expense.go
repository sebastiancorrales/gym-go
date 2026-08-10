package entities

import (
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/sebastiancorrales/gym-go/pkg/errors"
)

// ExpenseCategoryKind clasifica la categoría para los reportes.
//
// El nombre lo pone el usuario; el kind es lo que permite separar la compra de
// mercancía del gasto operativo sin adivinar por el texto de la etiqueta.
type ExpenseCategoryKind string

const (
	ExpenseKindInventory ExpenseCategoryKind = "inventory" // mercancía para revender
	ExpenseKindOperating ExpenseCategoryKind = "operating" // arriendo, servicios, mantenimiento
	ExpenseKindPayroll   ExpenseCategoryKind = "payroll"
	ExpenseKindOther     ExpenseCategoryKind = "other"
)

// ExpenseCategory es una tabla y no un enum de Go.
//
// Un enum obligaría a recompilar y redesplegar el binario para añadir
// "Mantenimiento de máquinas", que es inviable para un gimnasio. Un string libre
// fragmentaría el reporte en "Arriendo" / "arriendo" / "Arriendo local". Hay
// precedente exacto en el proyecto: SalePaymentMethod es justo esto.
//
// GymID nulo (uuid.Nil) marca las categorías del sistema, visibles para todos
// los gimnasios; es el mismo compromiso que ya se acepta con el super admin.
type ExpenseCategory struct {
	ID    uuid.UUID `json:"id" db:"id"`
	GymID uuid.UUID `json:"gym_id" db:"gym_id" gorm:"index:idx_expense_categories_gym"`

	Name   string              `json:"name" db:"name"`
	Kind   ExpenseCategoryKind `json:"kind" db:"kind"`
	Status string              `json:"status" db:"status"`

	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// ExpenseStatus represents the status of an expense.
type ExpenseStatus string

const (
	ExpenseStatusRegistered ExpenseStatus = "registered"
	ExpenseStatusVoided     ExpenseStatus = "voided"
)

// Expense es un egreso: dinero que sale del gimnasio.
//
// Cubre tanto el gasto general (el arriendo) como la compra de mercancía; lo
// único que las distingue es si sus líneas referencian productos. No hay dos
// entidades porque no hay dos flujos: los dos salen de la misma caja el mismo
// día y tienen que aparecer juntos en el cuadre.
//
// Date/Hour (string local del gimnasio) no son opcionales: todos los reportes
// del proyecto filtran por esa columna y RestrictReportRange compara contra el
// "hoy" local. Un gasto de las 7 PM en Colombia guardado sólo con timestamp UTC
// caería en el cuadre del día siguiente.
type Expense struct {
	ID    uuid.UUID `json:"id" db:"id"`
	GymID uuid.UUID `json:"gym_id" db:"gym_id" gorm:"index:idx_expenses_gym_date,priority:1"`

	CategoryID uuid.UUID  `json:"category_id" db:"category_id" gorm:"index:idx_expenses_category"`
	SupplierID *uuid.UUID `json:"supplier_id,omitempty" db:"supplier_id" gorm:"index:idx_expenses_supplier"`
	// PaymentMethodID apunta a SalePaymentMethod, el mismo catálogo que usa el
	// punto de venta. Un gasto lleva UN método: el pago combinado se implementa
	// aparte y sólo para ventas y suscripciones.
	PaymentMethodID uuid.UUID `json:"payment_method_id" db:"payment_method_id" gorm:"index:idx_expenses_payment_method"`

	Description    string  `json:"description" db:"description"`
	DocumentNumber string  `json:"document_number" db:"document_number"` // nº de la factura del proveedor
	Subtotal       float64 `json:"subtotal" db:"subtotal"`
	TaxAmount      float64 `json:"tax_amount" db:"tax_amount"`
	Total          float64 `json:"total" db:"total"`

	Status ExpenseStatus `json:"status" db:"status" gorm:"index:idx_expenses_status"`
	// AffectsInventory se calcula al crear y evita que cada reporte tenga que
	// mirar las líneas para saber si el gasto movió stock.
	AffectsInventory bool       `json:"affects_inventory" db:"affects_inventory"`
	VoidedExpenseID  *uuid.UUID `json:"voided_expense_id,omitempty" db:"voided_expense_id" gorm:"index:idx_expenses_voided"`

	UserID      uuid.UUID `json:"user_id" db:"user_id" gorm:"index:idx_expenses_user"`
	ExpenseDate time.Time `json:"expense_date" db:"expense_date"`
	Date        string    `json:"date" db:"date" gorm:"index:idx_expenses_gym_date,priority:2;index:idx_expenses_date"`
	Hour        string    `json:"hour" db:"hour"`

	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`

	// Relaciones, no persistidas.
	Lines         []ExpenseLine      `json:"lines,omitempty" gorm:"-" db:"-"`
	Supplier      *Supplier          `json:"supplier,omitempty" gorm:"-" db:"-"`
	Category      *ExpenseCategory   `json:"category,omitempty" gorm:"-" db:"-"`
	PaymentMethod *SalePaymentMethod `json:"payment_method,omitempty" gorm:"-" db:"-"`
}

// ExpenseLine es un concepto del gasto.
//
// Si ProductID != nil genera una entrada de inventario al UnitCost de la línea;
// si es nil es un concepto suelto ("Arriendo agosto") que sólo aporta importe.
//
// Un gasto SIEMPRE tiene al menos una línea: el modo simple sintetiza una desde
// la descripción y el total. Así Subtotal == SUM(lines.total) se cumple siempre
// y ningún reporte necesita dos caminos según si el gasto tiene líneas.
type ExpenseLine struct {
	ID        uuid.UUID  `json:"id" db:"id"`
	ExpenseID uuid.UUID  `json:"expense_id" db:"expense_id" gorm:"index:idx_expense_lines_expense"`
	ProductID *uuid.UUID `json:"product_id,omitempty" db:"product_id" gorm:"index:idx_expense_lines_product"`

	Description string `json:"description" db:"description"`
	// Quantity es float para admitir servicios ("3.5 horas de mantenimiento").
	// Cuando la línea referencia un producto el use case exige que sea un entero
	// positivo, porque Product.Stock es int.
	Quantity float64 `json:"quantity" db:"quantity"`
	UnitCost float64 `json:"unit_cost" db:"unit_cost"`
	Total    float64 `json:"total" db:"total"`

	CreatedAt time.Time `json:"created_at" db:"created_at"`

	Product *Product `json:"product,omitempty" gorm:"-" db:"-"`
}

// Validate comprueba una línea de gasto.
func (l *ExpenseLine) Validate() error {
	if l.UnitCost < 0 {
		return errors.ErrInvalidPrice
	}
	if l.Quantity <= 0 {
		return errors.ErrInvalidQuantity
	}

	if l.ProductID != nil {
		// Product.Stock es int: media unidad de un producto no se puede
		// representar en el kardex, así que se rechaza aquí en vez de truncar en
		// silencio.
		if l.Quantity != math.Trunc(l.Quantity) {
			return errors.ErrInvalidQuantity
		}
	} else if l.Description == "" {
		// Una línea sin producto y sin descripción no dice qué se pagó.
		return errors.ErrInvalidInput
	}
	return nil
}

// CalculateTotal recalcula el importe de la línea. El total lo decide siempre el
// servidor; lo que mande el cliente se ignora.
func (l *ExpenseLine) CalculateTotal() {
	l.Total = l.Quantity * l.UnitCost
}

// CalculateTotals recalcula subtotal y total del gasto a partir de sus líneas.
func (e *Expense) CalculateTotals() {
	subtotal := 0.0
	affectsInventory := false
	for i := range e.Lines {
		e.Lines[i].CalculateTotal()
		subtotal += e.Lines[i].Total
		if e.Lines[i].ProductID != nil {
			affectsInventory = true
		}
	}
	e.Subtotal = subtotal
	e.Total = subtotal + e.TaxAmount
	e.AffectsInventory = affectsInventory
}

// CanBeVoided indica si el gasto se puede anular. Espejo de Sale.CanBeVoided.
func (e *Expense) CanBeVoided() bool {
	return e.Status == ExpenseStatusRegistered && e.VoidedExpenseID == nil
}

// WeightedAverageCost calcula el nuevo costo promedio ponderado tras una compra.
//
// Promedio ponderado y no "último costo": con el último costo, un pedido urgente
// pagado un 40% más caro reescribiría el margen de todo el stock que ya estaba
// en bodega. El promedio ponderado es el estándar de retail.
//
// Casos borde: si el stock resultante no es positivo se conserva el costo
// anterior (no hay sobre qué promediar), y si el stock previo era negativo o el
// costo previo era cero se toma el costo de la compra como referencia.
func WeightedAverageCost(oldStock int, oldCost float64, qty int, unitCost float64) float64 {
	newQty := oldStock + qty
	if newQty <= 0 {
		return oldCost
	}
	if oldStock <= 0 || oldCost <= 0 {
		return unitCost
	}
	return (float64(oldStock)*oldCost + float64(qty)*unitCost) / float64(newQty)
}
