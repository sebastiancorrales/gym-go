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

// seedInventory monta un ProductUseCase con su kardex sobre una base real.
func seedInventory(t *testing.T, db *gorm.DB, stock int) (*usecases.ProductUseCase, *entities.Product, uuid.UUID) {
	t.Helper()

	ctx := context.Background()
	productRepo := persistence.NewSQLiteProductRepository(db)
	movementRepo := persistence.NewSQLiteStockMovementRepository(db)
	uow := persistence.NewUnitOfWork(db)

	product := &entities.Product{
		ID:        uuid.New(),
		Name:      "Barra de proteína",
		UnitPrice: 8000,
		Stock:     stock,
		MinStock:  5,
		Status:    entities.ProductStatusActive,
	}
	if err := productRepo.Create(ctx, product); err != nil {
		t.Fatalf("creando producto: %v", err)
	}

	userID := uuid.New()
	return usecases.NewProductUseCase(productRepo, movementRepo, uow), product, userID
}

func stockOf(t *testing.T, db *gorm.DB, productID uuid.UUID) int {
	t.Helper()
	var stock int
	if err := db.Raw("SELECT stock FROM products WHERE id = ?", productID).Scan(&stock).Error; err != nil {
		t.Fatalf("leyendo stock: %v", err)
	}
	return stock
}

func movementCount(t *testing.T, db *gorm.DB, productID uuid.UUID) int64 {
	t.Helper()
	var n int64
	db.Raw("SELECT COUNT(*) FROM stock_movements WHERE product_id = ?", productID).Scan(&n)
	return n
}

// El bug que motiva cerrar Update(): Stock es un int no puntero, así que un PUT
// sin el campo llegaba con Stock=0 —indistinguible de "ponlo a cero"— y borraba
// el stock en silencio. Ahora el repositorio escribe una lista explícita de
// columnas que no lo incluye.
func TestUpdateProduct_NoTocaElStock(t *testing.T) {
	db := newTestDB(t)
	productUC, product, _ := seedInventory(t, db, 42)

	// Igual que hace el handler cuando el cuerpo no trae `stock`: la entidad se
	// lee, se cambia el nombre y se guarda con Stock a su valor por defecto.
	edited := &entities.Product{
		ID:        product.ID,
		Name:      "Barra de proteína (nuevo nombre)",
		UnitPrice: product.UnitPrice,
		Stock:     0, // el cero de Go, no una intención del usuario
		Status:    entities.ProductStatusActive,
	}
	if err := productUC.UpdateProduct(context.Background(), edited); err != nil {
		t.Fatalf("UpdateProduct: %v", err)
	}

	if got := stockOf(t, db, product.ID); got != 42 {
		t.Errorf("stock = %d, want 42 — editar un producto no puede vaciar su inventario", got)
	}
}

func TestAdjustStock_EscribeAsientoYMueveSaldo(t *testing.T) {
	db := newTestDB(t)
	productUC, product, userID := seedInventory(t, db, 10)

	change := usecases.StockChange{
		ProductID: product.ID,
		Reason:    entities.StockReasonShrinkage,
		Note:      "Tres unidades vencidas",
		UserID:    userID,
		Loc:       time.UTC,
	}

	movement, err := productUC.AdjustProductStock(context.Background(), -3, change)
	if err != nil {
		t.Fatalf("AdjustProductStock: %v", err)
	}

	if got := stockOf(t, db, product.ID); got != 7 {
		t.Errorf("stock = %d, want 7", got)
	}
	if movement.Direction != entities.StockDirectionOut || movement.Quantity != 3 {
		t.Errorf("asiento = %s %d, want out 3", movement.Direction, movement.Quantity)
	}
	if movement.StockAfter == nil || *movement.StockAfter != 7 {
		t.Errorf("stock_after = %v, want 7", movement.StockAfter)
	}
	if movement.Note != "Tres unidades vencidas" {
		t.Errorf("la nota no se guardó: %q", movement.Note)
	}
	if movement.UserID == nil || *movement.UserID != userID {
		t.Error("el asiento debe registrar quién lo hizo")
	}
}

// La guarda vive en el WHERE del UPDATE, no en una comprobación en Go: leer el
// stock, decidir y luego escribir es la carrera que dejaba el saldo negativo.
func TestAdjustStock_RechazaDejarElStockNegativo(t *testing.T) {
	db := newTestDB(t)
	productUC, product, userID := seedInventory(t, db, 2)

	_, err := productUC.AdjustProductStock(context.Background(), -5, usecases.StockChange{
		ProductID: product.ID,
		Reason:    entities.StockReasonAdjustment,
		UserID:    userID,
		Loc:       time.UTC,
	})
	if !errors.Is(err, apperrors.ErrInsufficientStock) {
		t.Fatalf("error = %v, want ErrInsufficientStock", err)
	}

	if got := stockOf(t, db, product.ID); got != 2 {
		t.Errorf("stock = %d, want 2 — un ajuste rechazado no puede mover nada", got)
	}
	if n := movementCount(t, db, product.ID); n != 0 {
		t.Errorf("asientos = %d, want 0 — un ajuste rechazado no deja rastro", n)
	}
}

// Sin esta guarda, un cliente podría mandar reason="sale" a /stock/adjust y
// ensuciar el reporte de ventas con movimientos que no corresponden a ninguna.
func TestAdjustStock_SoloAceptaMotivosManuales(t *testing.T) {
	db := newTestDB(t)
	productUC, product, userID := seedInventory(t, db, 10)

	for _, reason := range []entities.StockMovementReason{
		entities.StockReasonSale,
		entities.StockReasonSaleVoid,
		entities.StockReasonPurchase,
		entities.StockReasonInitial,
	} {
		_, err := productUC.AdjustProductStock(context.Background(), 1, usecases.StockChange{
			ProductID: product.ID,
			Reason:    reason,
			UserID:    userID,
			Loc:       time.UTC,
		})
		if !errors.Is(err, apperrors.ErrInvalidInput) {
			t.Errorf("reason=%s: error = %v, want ErrInvalidInput", reason, err)
		}
	}

	if got := stockOf(t, db, product.ID); got != 10 {
		t.Errorf("stock = %d, want 10", got)
	}
}

// Un conteo físico se expresa como el delta que realmente ocurrió, no como el
// valor absoluto: es lo que permite contar la diferencia en el libro.
func TestCountStock_RegistraLaDiferencia(t *testing.T) {
	db := newTestDB(t)
	productUC, product, userID := seedInventory(t, db, 10)

	change := usecases.StockChange{ProductID: product.ID, UserID: userID, Loc: time.UTC}

	movement, err := productUC.CountProductStock(context.Background(), 7, change)
	if err != nil {
		t.Fatalf("CountProductStock: %v", err)
	}
	if movement.Direction != entities.StockDirectionOut || movement.Quantity != 3 {
		t.Errorf("asiento = %s %d, want out 3 (10 contados como 7)", movement.Direction, movement.Quantity)
	}
	if got := stockOf(t, db, product.ID); got != 7 {
		t.Errorf("stock = %d, want 7", got)
	}

	// Contar lo mismo que ya había no es un movimiento: un asiento de cero sólo
	// ensucia el libro.
	before := movementCount(t, db, product.ID)
	noop, err := productUC.CountProductStock(context.Background(), 7, change)
	if err != nil {
		t.Fatalf("CountProductStock (sin cambio): %v", err)
	}
	if noop != nil {
		t.Error("contar el mismo valor no debería generar asiento")
	}
	if after := movementCount(t, db, product.ID); after != before {
		t.Errorf("asientos = %d, want %d", after, before)
	}
}

// Un producto que nace con stock necesita su asiento inicial, o el libro
// arrancaría con un saldo que no viene de ningún movimiento.
func TestCreateProduct_SiembraElSaldoInicial(t *testing.T) {
	db := newTestDB(t)
	productUC, _, userID := seedInventory(t, db, 0)

	product := &entities.Product{
		Name:      "Bebida isotónica",
		UnitPrice: 5000,
		Stock:     24,
		Status:    entities.ProductStatusActive,
	}
	change := usecases.StockChange{UserID: userID, Loc: time.UTC}
	if err := productUC.CreateProduct(context.Background(), product, change); err != nil {
		t.Fatalf("CreateProduct: %v", err)
	}

	movements, err := productUC.ListProductMovements(context.Background(), product.ID, "", "")
	if err != nil {
		t.Fatalf("ListProductMovements: %v", err)
	}
	if len(movements) != 1 {
		t.Fatalf("asientos = %d, want 1", len(movements))
	}
	m := movements[0]
	if m.Reason != entities.StockReasonInitial || m.Direction != entities.StockDirectionIn || m.Quantity != 24 {
		t.Errorf("asiento = %s/%s/%d, want initial/in/24", m.Reason, m.Direction, m.Quantity)
	}
	if m.StockAfter == nil || *m.StockAfter != 24 {
		t.Errorf("stock_after = %v, want 24", m.StockAfter)
	}

	// Un producto sin stock no genera asiento: no hay nada que registrar.
	empty := &entities.Product{Name: "Toalla", UnitPrice: 15000, Status: entities.ProductStatusActive}
	if err := productUC.CreateProduct(context.Background(), empty, change); err != nil {
		t.Fatalf("CreateProduct (sin stock): %v", err)
	}
	if n := movementCount(t, db, empty.ID); n != 0 {
		t.Errorf("asientos de un producto sin stock = %d, want 0", n)
	}
}

// Anular una venta devuelve la mercancía, y eso también es un movimiento.
func TestVoidSale_EscribeAsientosDeDevolucion(t *testing.T) {
	db := newTestDB(t)
	saleUC, product, paymentMethodID, sellerID := seedPOS(t, db, 10)
	ctx := context.Background()

	sale := &entities.Sale{
		UserID:          sellerID,
		PaymentMethodID: paymentMethodID,
		Details: []entities.SaleDetail{{
			ProductID: product.ID,
			UnitPrice: product.UnitPrice,
			Quantity:  3,
		}},
	}
	if err := saleUC.CreateSale(ctx, sale); err != nil {
		t.Fatalf("CreateSale: %v", err)
	}
	if got := stockOf(t, db, product.ID); got != 7 {
		t.Fatalf("stock tras la venta = %d, want 7", got)
	}

	if _, err := saleUC.VoidSale(ctx, sale.ID, sellerID, time.UTC); err != nil {
		t.Fatalf("VoidSale: %v", err)
	}

	if got := stockOf(t, db, product.ID); got != 10 {
		t.Errorf("stock tras anular = %d, want 10", got)
	}

	var out struct {
		Direction  string
		Quantity   int
		StockAfter *int
	}
	if err := db.Raw(
		`SELECT direction, quantity, stock_after FROM stock_movements
		  WHERE product_id = ? AND reason = 'sale_void'`, product.ID,
	).Scan(&out).Error; err != nil {
		t.Fatalf("leyendo el asiento de anulación: %v", err)
	}
	if out.Direction != "in" || out.Quantity != 3 {
		t.Errorf("asiento de anulación = %s %d, want in 3", out.Direction, out.Quantity)
	}
	if out.StockAfter == nil || *out.StockAfter != 10 {
		t.Errorf("stock_after = %v, want 10", out.StockAfter)
	}

	// La propiedad que hace útil el libro: su saldo acumulado ES products.stock.
	var balance int
	db.Raw(`SELECT COALESCE(SUM(CASE WHEN direction = 'out' THEN -quantity ELSE quantity END), 0)
	          FROM stock_movements WHERE product_id = ?`, product.ID).Scan(&balance)
	if balance != 0 {
		t.Errorf("saldo del kardex = %d, want 0 (venta y anulación se cancelan)", balance)
	}
}
