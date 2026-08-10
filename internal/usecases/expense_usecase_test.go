package usecases_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sebastiancorrales/gym-go/internal/domain/entities"
	"github.com/sebastiancorrales/gym-go/internal/infrastructure/persistence"
	"github.com/sebastiancorrales/gym-go/internal/usecases"
	apperrors "github.com/sebastiancorrales/gym-go/pkg/errors"
	"gorm.io/gorm"
)

type expenseFixture struct {
	uc         *usecases.ExpenseUseCase
	saleUC     *usecases.SaleUseCase
	gymID      uuid.UUID
	userID     uuid.UUID
	categoryID uuid.UUID
	methodID   uuid.UUID
	product    *entities.Product
}

// seedExpenses monta el módulo de gastos sobre una base real, reutilizando el
// producto y el método de pago que ya siembra seedPOS.
func seedExpenses(t *testing.T, db *gorm.DB, stock int) expenseFixture {
	t.Helper()
	ctx := context.Background()

	saleUC, product, methodID, userID := seedPOS(t, db, stock)

	categoryRepo := persistence.NewSQLiteExpenseCategoryRepository(db)
	category := &entities.ExpenseCategory{
		ID: uuid.New(), GymID: uuid.Nil, Name: "Mercancía",
		Kind: entities.ExpenseKindInventory, Status: "active",
	}
	if err := categoryRepo.Create(ctx, category); err != nil {
		t.Fatalf("creando categoría: %v", err)
	}

	return expenseFixture{
		uc: usecases.NewExpenseUseCase(
			persistence.NewSQLiteExpenseRepository(db),
			persistence.NewSQLiteExpenseLineRepository(db),
			categoryRepo,
			persistence.NewSQLiteSupplierRepository(db),
			persistence.NewSQLiteProductRepository(db),
			persistence.NewSQLitePaymentMethodRepository(db),
			persistence.NewSQLiteStockMovementRepository(db),
			persistence.NewUnitOfWork(db),
		),
		saleUC:     saleUC,
		gymID:      uuid.Nil, // el vendedor que siembra seedPOS pertenece al gym nulo
		userID:     userID,
		categoryID: category.ID,
		methodID:   methodID,
		product:    product,
	}
}

func (f expenseFixture) newExpense(lines ...entities.ExpenseLine) *entities.Expense {
	return &entities.Expense{
		GymID:           f.gymID,
		CategoryID:      f.categoryID,
		PaymentMethodID: f.methodID,
		UserID:          f.userID,
		Description:     "Compra de prueba",
		Lines:           lines,
	}
}

func productLine(productID uuid.UUID, qty float64, cost float64) entities.ExpenseLine {
	return entities.ExpenseLine{ProductID: &productID, Quantity: qty, UnitCost: cost}
}

func costOf(t *testing.T, db *gorm.DB, productID uuid.UUID) float64 {
	t.Helper()
	var cost float64
	if err := db.Raw("SELECT cost_price FROM products WHERE id = ?", productID).Scan(&cost).Error; err != nil {
		t.Fatalf("leyendo cost_price: %v", err)
	}
	return cost
}

// Lo que pidió el negocio: una compra tiene que mover el inventario, y las dos
// cosas van juntas o no van.
func TestCreateExpense_MueveInventarioAtomicamente(t *testing.T) {
	db := newTestDB(t)
	f := seedExpenses(t, db, 10)

	expense := f.newExpense(productLine(f.product.ID, 24, 3000))
	if err := f.uc.CreateExpense(context.Background(), expense, time.UTC); err != nil {
		t.Fatalf("CreateExpense: %v", err)
	}

	if got := stockOf(t, db, f.product.ID); got != 34 {
		t.Errorf("stock = %d, want 34", got)
	}
	if expense.Total != 72000 {
		t.Errorf("total = %v, want 72000 (24 × 3000)", expense.Total)
	}
	if !expense.AffectsInventory {
		t.Error("un gasto con línea de producto debe marcarse como que afecta inventario")
	}

	var m struct {
		Reason     string
		Direction  string
		Quantity   int
		UnitCost   float64
		StockAfter *int
	}
	if err := db.Raw(
		`SELECT reason, direction, quantity, unit_cost, stock_after FROM stock_movements
		  WHERE product_id = ? AND reason = 'purchase'`, f.product.ID).Scan(&m).Error; err != nil {
		t.Fatalf("leyendo el asiento de compra: %v", err)
	}
	if m.Direction != "in" || m.Quantity != 24 {
		t.Errorf("asiento = %s %d, want in 24", m.Direction, m.Quantity)
	}
	if m.UnitCost != 3000 {
		t.Errorf("unit_cost del asiento = %v, want 3000 — el costo histórico debe quedar congelado", m.UnitCost)
	}
	if m.StockAfter == nil || *m.StockAfter != 34 {
		t.Errorf("stock_after = %v, want 34", m.StockAfter)
	}
}

// El arriendo: un gasto sin líneas de producto se registra igual y no toca
// inventario. Es el mismo camino de código, sin ramas paralelas que mantener.
func TestCreateExpense_GastoGeneralNoTocaInventario(t *testing.T) {
	db := newTestDB(t)
	f := seedExpenses(t, db, 10)

	expense := f.newExpense(entities.ExpenseLine{
		Description: "Arriendo agosto", Quantity: 1, UnitCost: 2500000,
	})
	if err := f.uc.CreateExpense(context.Background(), expense, time.UTC); err != nil {
		t.Fatalf("CreateExpense: %v", err)
	}

	if expense.Total != 2500000 {
		t.Errorf("total = %v, want 2500000", expense.Total)
	}
	if expense.AffectsInventory {
		t.Error("un gasto sin líneas de producto no afecta inventario")
	}
	if got := stockOf(t, db, f.product.ID); got != 10 {
		t.Errorf("stock = %d, want 10 — el arriendo no mueve mercancía", got)
	}
	if n := movementCount(t, db, f.product.ID); n != 0 {
		t.Errorf("asientos = %d, want 0", n)
	}
}

// Espejo de TestCreateSale_RollsBackOnStockFailure: si algo falla, no puede
// quedar ni el gasto, ni sus líneas, ni el movimiento.
func TestCreateExpense_RevierteEnteroSiUnProductoNoExiste(t *testing.T) {
	db := newTestDB(t)
	f := seedExpenses(t, db, 10)

	expense := f.newExpense(
		productLine(f.product.ID, 5, 3000),
		productLine(uuid.New(), 2, 1000), // no existe
	)
	if err := f.uc.CreateExpense(context.Background(), expense, time.UTC); !errors.Is(err, apperrors.ErrNotFound) {
		t.Fatalf("error = %v, want ErrNotFound", err)
	}

	if got := stockOf(t, db, f.product.ID); got != 10 {
		t.Errorf("stock = %d, want 10", got)
	}
	for table, want := range map[string]int64{"expenses": 0, "expense_lines": 0, "stock_movements": 0} {
		var n int64
		db.Raw("SELECT COUNT(*) FROM " + table).Scan(&n)
		if n != want {
			t.Errorf("filas en %s = %d, want %d", table, n, want)
		}
	}
}

// Media unidad de un producto no se puede representar en el kardex (Stock es
// int), así que se rechaza en vez de truncar en silencio. Un servicio sí puede
// llevar decimales.
func TestCreateExpense_RechazaCantidadFraccionariaDeProducto(t *testing.T) {
	db := newTestDB(t)
	f := seedExpenses(t, db, 10)

	err := f.uc.CreateExpense(context.Background(), f.newExpense(productLine(f.product.ID, 2.5, 3000)), time.UTC)
	if !errors.Is(err, apperrors.ErrInvalidQuantity) {
		t.Fatalf("error = %v, want ErrInvalidQuantity", err)
	}

	// Pero un servicio con horas fraccionarias sí pasa.
	ok := f.newExpense(entities.ExpenseLine{Description: "Mantenimiento", Quantity: 3.5, UnitCost: 40000})
	if err := f.uc.CreateExpense(context.Background(), ok, time.UTC); err != nil {
		t.Fatalf("un servicio con horas fraccionarias debería aceptarse: %v", err)
	}
	if ok.Total != 140000 {
		t.Errorf("total = %v, want 140000", ok.Total)
	}
}

// Promedio ponderado y no último costo: una compra cara no puede reescribir el
// margen de todo el stock que ya estaba en bodega.
func TestCreateExpense_CostoPromedioPonderado(t *testing.T) {
	db := newTestDB(t)
	f := seedExpenses(t, db, 0) // empieza sin stock para que el promedio sea limpio
	ctx := context.Background()

	if err := f.uc.CreateExpense(ctx, f.newExpense(productLine(f.product.ID, 10, 1000)), time.UTC); err != nil {
		t.Fatalf("primera compra: %v", err)
	}
	if got := costOf(t, db, f.product.ID); got != 1000 {
		t.Fatalf("costo tras la primera compra = %v, want 1000", got)
	}

	if err := f.uc.CreateExpense(ctx, f.newExpense(productLine(f.product.ID, 10, 2000)), time.UTC); err != nil {
		t.Fatalf("segunda compra: %v", err)
	}

	// (10×1000 + 10×2000) / 20 = 1500. Con "último costo" saldría 2000 y el
	// margen de las 10 primeras unidades quedaría falseado.
	if got := costOf(t, db, f.product.ID); got != 1500 {
		t.Errorf("costo promedio = %v, want 1500", got)
	}
	if got := stockOf(t, db, f.product.ID); got != 20 {
		t.Errorf("stock = %d, want 20", got)
	}
}

func TestVoidExpense_DevuelveLaMercancia(t *testing.T) {
	db := newTestDB(t)
	f := seedExpenses(t, db, 10)
	ctx := context.Background()

	expense := f.newExpense(productLine(f.product.ID, 6, 3000))
	if err := f.uc.CreateExpense(ctx, expense, time.UTC); err != nil {
		t.Fatalf("CreateExpense: %v", err)
	}
	if got := stockOf(t, db, f.product.ID); got != 16 {
		t.Fatalf("stock tras la compra = %d, want 16", got)
	}

	voided, err := f.uc.VoidExpense(ctx, expense.ID, f.userID, time.UTC)
	if err != nil {
		t.Fatalf("VoidExpense: %v", err)
	}

	if got := stockOf(t, db, f.product.ID); got != 10 {
		t.Errorf("stock tras anular = %d, want 10", got)
	}
	if voided.Total != -expense.Total {
		t.Errorf("total del espejo = %v, want %v", voided.Total, -expense.Total)
	}

	// La propiedad que hace que los reportes cuadren sin filtrar por estado:
	// sumar todas las filas del rango ya sale neteado.
	var netTotal float64
	db.Raw("SELECT COALESCE(SUM(total), 0) FROM expenses").Scan(&netTotal)
	if netTotal != 0 {
		t.Errorf("suma de gastos = %v, want 0 (compra y anulación se cancelan)", netTotal)
	}

	// Y el original queda marcado, no borrado.
	var status string
	db.Raw("SELECT status FROM expenses WHERE id = ?", expense.ID).Scan(&status)
	if status != string(entities.ExpenseStatusVoided) {
		t.Errorf("estado del original = %q, want voided", status)
	}
}

// El caso genuinamente difícil: si la mercancía comprada ya se vendió, devolverla
// dejaría el stock negativo. Se prefiere fallar antes que romper la guarda que
// impide sobrevender en el punto de venta.
func TestVoidExpense_FallaSiLaMercanciaYaSeVendio(t *testing.T) {
	db := newTestDB(t)
	f := seedExpenses(t, db, 0)
	ctx := context.Background()

	expense := f.newExpense(productLine(f.product.ID, 5, 3000))
	if err := f.uc.CreateExpense(ctx, expense, time.UTC); err != nil {
		t.Fatalf("CreateExpense: %v", err)
	}

	sale := &entities.Sale{
		UserID:          f.userID,
		PaymentMethodID: f.methodID,
		Details: []entities.SaleDetail{{
			ProductID: f.product.ID, UnitPrice: f.product.UnitPrice, Quantity: 4,
		}},
	}
	if err := f.saleUC.CreateSale(ctx, sale); err != nil {
		t.Fatalf("CreateSale: %v", err)
	}
	if got := stockOf(t, db, f.product.ID); got != 1 {
		t.Fatalf("stock tras la venta = %d, want 1", got)
	}

	_, err := f.uc.VoidExpense(ctx, expense.ID, f.userID, time.UTC)
	if !errors.Is(err, apperrors.ErrInsufficientStock) {
		t.Fatalf("error = %v, want ErrInsufficientStock", err)
	}

	// Y la anulación fallida no puede haber dejado nada a medias.
	if got := stockOf(t, db, f.product.ID); got != 1 {
		t.Errorf("stock = %d, want 1 — una anulación rechazada no mueve nada", got)
	}
	var status string
	db.Raw("SELECT status FROM expenses WHERE id = ?", expense.ID).Scan(&status)
	if status != string(entities.ExpenseStatusRegistered) {
		t.Errorf("estado del original = %q, want registered — la anulación falló", status)
	}
	var count int64
	db.Raw("SELECT COUNT(*) FROM expenses").Scan(&count)
	if count != 1 {
		t.Errorf("filas en expenses = %d, want 1 — no puede quedar el espejo de una anulación fallida", count)
	}
}

// Un gasto se paga con un método activo, igual que una venta.
func TestCreateExpense_RechazaMetodoDePagoInactivo(t *testing.T) {
	db := newTestDB(t)
	f := seedExpenses(t, db, 10)

	if err := db.Exec("UPDATE sale_payment_methods SET status = 'inactive' WHERE id = ?", f.methodID).Error; err != nil {
		t.Fatalf("desactivando el método: %v", err)
	}

	err := f.uc.CreateExpense(context.Background(), f.newExpense(productLine(f.product.ID, 1, 100)), time.UTC)
	if !errors.Is(err, apperrors.ErrPaymentMethodNotActive) {
		t.Fatalf("error = %v, want ErrPaymentMethodNotActive", err)
	}
}

// Dos líneas del mismo producto se funden en un solo movimiento, igual que
// qtyByProduct en CreateSale: si no, la guarda de stock se evaluaría dos veces
// por separado.
func TestCreateExpense_AgregaLineasDelMismoProducto(t *testing.T) {
	db := newTestDB(t)
	f := seedExpenses(t, db, 0)

	expense := f.newExpense(
		productLine(f.product.ID, 10, 1000),
		productLine(f.product.ID, 10, 2000),
	)
	if err := f.uc.CreateExpense(context.Background(), expense, time.UTC); err != nil {
		t.Fatalf("CreateExpense: %v", err)
	}

	var movements int64
	db.Raw("SELECT COUNT(*) FROM stock_movements WHERE product_id = ? AND reason = 'purchase'", f.product.ID).Scan(&movements)
	if movements != 1 {
		t.Errorf("asientos de compra = %d, want 1 (dos líneas del mismo producto se funden)", movements)
	}
	if got := stockOf(t, db, f.product.ID); got != 20 {
		t.Errorf("stock = %d, want 20", got)
	}
	if got := costOf(t, db, f.product.ID); got != 1500 {
		t.Errorf("costo = %v, want 1500 (promedio de las dos líneas)", got)
	}
}
