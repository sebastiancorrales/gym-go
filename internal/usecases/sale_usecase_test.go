package usecases_test

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/sebastiancorrales/gym-go/internal/config"
	"github.com/sebastiancorrales/gym-go/internal/domain/entities"
	"github.com/sebastiancorrales/gym-go/internal/infrastructure/persistence"
	"github.com/sebastiancorrales/gym-go/internal/infrastructure/persistence/migrations"
	"github.com/sebastiancorrales/gym-go/internal/usecases"
	apperrors "github.com/sebastiancorrales/gym-go/pkg/errors"
	"gorm.io/gorm"
)

// TestConcurrentSales_NoOversell is the test the whole database phase hinges on.
// A single run exercises, together:
//
//   - the DSN (WAL + busy_timeout + _txlock=immediate),
//   - the unit of work wrapping a sale in one transaction,
//   - RetryOnBusy,
//   - and the conditional DecrementStock.
//
// 20 tills try to sell the last 10 units at the same time. Exactly 10 must
// succeed, the other 10 must fail with "insufficient stock" — not with a database
// error — stock must never go negative, and no sale may be recorded without its
// line items.
func TestConcurrentSales_NoOversell(t *testing.T) {
	const initialStock = 10
	const tills = 20

	db := newTestDB(t)
	saleUC, product, paymentMethodID, sellerID := seedPOS(t, db, initialStock)

	var wg sync.WaitGroup
	results := make(chan error, tills)

	for i := 0; i < tills; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sale := &entities.Sale{
				UserID:          sellerID,
				PaymentMethodID: paymentMethodID,
				Details: []entities.SaleDetail{{
					ProductID: product.ID,
					UnitPrice: product.UnitPrice,
					Quantity:  1,
				}},
			}
			results <- saleUC.CreateSale(context.Background(), sale)
		}()
	}

	wg.Wait()
	close(results)

	var ok, outOfStock int
	for err := range results {
		switch {
		case err == nil:
			ok++
		case errors.Is(err, apperrors.ErrInsufficientStock):
			outOfStock++
		case persistence.IsBusy(err) || errors.Is(err, apperrors.ErrDatabaseBusy):
			t.Errorf("una venta falló por contención de base de datos: %v", err)
		default:
			t.Errorf("una venta falló por un motivo inesperado: %v", err)
		}
	}

	if ok != initialStock {
		t.Errorf("ventas exitosas = %d, want %d", ok, initialStock)
	}
	if outOfStock != tills-initialStock {
		t.Errorf("ventas rechazadas por stock = %d, want %d", outOfStock, tills-initialStock)
	}

	// El invariante que importa: el stock nunca puede quedar negativo.
	var finalStock int
	if err := db.Raw("SELECT stock FROM products WHERE id = ?", product.ID).Scan(&finalStock).Error; err != nil {
		t.Fatalf("leyendo stock final: %v", err)
	}
	if finalStock != 0 {
		t.Errorf("stock final = %d, want 0 (negativo significaría sobreventa)", finalStock)
	}

	// Atomicidad: tantas ventas como líneas de detalle, ninguna a medias.
	var sales, details int64
	db.Raw("SELECT COUNT(*) FROM sales").Scan(&sales)
	db.Raw("SELECT COUNT(*) FROM sale_details").Scan(&details)
	if sales != int64(initialStock) {
		t.Errorf("filas en sales = %d, want %d", sales, initialStock)
	}
	if details != int64(initialStock) {
		t.Errorf("filas en sale_details = %d, want %d — una venta sin detalles es una venta a medias", details, initialStock)
	}

	assertLedgerReconciles(t, db, product.ID, initialStock, finalStock)
}

// assertLedgerReconciles comprueba que el kardex es de fiar bajo concurrencia
// real, que es la única forma de saberlo: stock_after se lee DENTRO de la
// transacción que aplica el cambio, y toda su utilidad depende de que ese valor
// sea el definitivo y no uno intermedio.
//
// Las tres comprobaciones, de menos a más exigente:
//  1. hay un asiento por venta exitosa
//  2. la suma con signo del libro coincide con products.stock
//  3. los saldos, ordenados en el tiempo, bajan de uno en uno sin repetir ni
//     saltar — si dos cajas leyeran el mismo saldo, aquí saldría un duplicado
func assertLedgerReconciles(t *testing.T, db *gorm.DB, productID uuid.UUID, initialStock, finalStock int) {
	t.Helper()

	var movements []struct {
		Direction  string
		Quantity   int
		StockAfter *int
	}
	if err := db.Raw(
		`SELECT direction, quantity, stock_after FROM stock_movements
		  WHERE product_id = ? AND reason = 'sale' ORDER BY stock_after DESC`,
		productID,
	).Scan(&movements).Error; err != nil {
		t.Fatalf("leyendo el kardex: %v", err)
	}

	if len(movements) != initialStock {
		t.Fatalf("asientos de venta en el kardex = %d, want %d", len(movements), initialStock)
	}

	balance := 0
	for _, m := range movements {
		if m.Direction == "out" {
			balance -= m.Quantity
		} else {
			balance += m.Quantity
		}
	}
	if want := finalStock - initialStock; balance != want {
		t.Errorf("suma del kardex = %d, want %d — el libro no cuadra con products.stock", balance, want)
	}

	// initialStock-1, initialStock-2, ..., 0
	for i, m := range movements {
		if m.StockAfter == nil {
			t.Fatalf("asiento %d sin stock_after: no se pudo leer el saldo dentro de la transacción", i)
		}
		if want := initialStock - 1 - i; *m.StockAfter != want {
			t.Errorf("stock_after[%d] = %d, want %d — dos cajas vieron el mismo saldo", i, *m.StockAfter, want)
		}
	}
}

// TestCreateSale_RollsBackOnStockFailure checks that when the stock update fails
// the sale does not stay behind. Before the transaction the sale and its line
// items were already committed by then.
func TestCreateSale_RollsBackOnStockFailure(t *testing.T) {
	db := newTestDB(t)
	saleUC, product, paymentMethodID, sellerID := seedPOS(t, db, 1)

	// Pide 2 unidades habiendo 1: la validación previa lo rechaza y no debe quedar
	// rastro de la venta.
	sale := &entities.Sale{
		UserID:          sellerID,
		PaymentMethodID: paymentMethodID,
		Details: []entities.SaleDetail{{
			ProductID: product.ID,
			UnitPrice: product.UnitPrice,
			Quantity:  2,
		}},
	}

	err := saleUC.CreateSale(context.Background(), sale)
	if !errors.Is(err, apperrors.ErrInsufficientStock) {
		t.Fatalf("err = %v, want ErrInsufficientStock", err)
	}

	var sales, details int64
	db.Raw("SELECT COUNT(*) FROM sales").Scan(&sales)
	db.Raw("SELECT COUNT(*) FROM sale_details").Scan(&details)
	if sales != 0 || details != 0 {
		t.Errorf("quedaron restos de la venta fallida: sales=%d sale_details=%d, want 0 y 0", sales, details)
	}

	var stock int
	db.Raw("SELECT stock FROM products WHERE id = ?", product.ID).Scan(&stock)
	if stock != 1 {
		t.Errorf("stock = %d, want 1 sin tocar", stock)
	}
}

// newTestDB opens a temporary database with the production DSN and schema.
func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	database, err := config.NewDatabase(&config.DatabaseConfig{
		DatabasePath: filepath.Join(t.TempDir(), "pos.db"),
		MaxIdleConns: 8,
		MaxOpenConns: 8,
	})
	if err != nil {
		t.Fatalf("NewDatabase: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	if err := migrations.Migrate(database.DB); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	return database.DB
}

// seedPOS creates a product, a payment method and a seller, and returns a wired
// SaleUseCase.
func seedPOS(t *testing.T, db *gorm.DB, stock int) (*usecases.SaleUseCase, *entities.Product, uuid.UUID, uuid.UUID) {
	t.Helper()

	ctx := context.Background()
	productRepo := persistence.NewSQLiteProductRepository(db)
	paymentMethodRepo := persistence.NewSQLitePaymentMethodRepository(db)
	saleRepo := persistence.NewSQLiteSaleRepository(db)
	saleDetailRepo := persistence.NewSQLiteSaleDetailRepository(db)
	uow := persistence.NewUnitOfWork(db)

	product := &entities.Product{
		ID:        uuid.New(),
		Name:      "Botella de agua",
		UnitPrice: 2000,
		Stock:     stock,
		Status:    entities.ProductStatusActive,
	}
	if err := productRepo.Create(ctx, product); err != nil {
		t.Fatalf("creando producto: %v", err)
	}

	method := &entities.SalePaymentMethod{
		ID:     uuid.New(),
		Name:   "Efectivo",
		Type:   entities.PaymentTypeCash,
		Status: entities.PaymentMethodStatusActive,
	}
	if err := paymentMethodRepo.Create(ctx, method); err != nil {
		t.Fatalf("creando método de pago: %v", err)
	}

	sellerID := uuid.New()
	if err := db.Exec(`INSERT INTO users (id, gym_id, email, first_name, last_name, role, status)
	                   VALUES (?, ?, 'cajero@test.local', 'Caja', 'Uno', 'RECEPCIONISTA', 'ACTIVE')`,
		sellerID, uuid.Nil).Error; err != nil {
		t.Fatalf("creando vendedor: %v", err)
	}

	saleUC := usecases.NewSaleUseCase(saleRepo, saleDetailRepo, productRepo, paymentMethodRepo,
		persistence.NewSQLitePaymentLineRepository(db), uow)
	return saleUC, product, method.ID, sellerID
}
