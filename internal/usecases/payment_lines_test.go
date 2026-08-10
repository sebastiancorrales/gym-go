package usecases_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sebastiancorrales/gym-go/internal/domain/entities"
	"github.com/sebastiancorrales/gym-go/internal/infrastructure/persistence"
	apperrors "github.com/sebastiancorrales/gym-go/pkg/errors"
	"gorm.io/gorm"
)

// newMethod crea un método de pago del tipo pedido.
func newMethod(t *testing.T, db *gorm.DB, name string, kind entities.PaymentMethodType) uuid.UUID {
	t.Helper()
	method := &entities.SalePaymentMethod{
		ID: uuid.New(), Name: name, Type: kind,
		Status: entities.PaymentMethodStatusActive,
	}
	if err := persistence.NewSQLitePaymentMethodRepository(db).Create(context.Background(), method); err != nil {
		t.Fatalf("creando método %q: %v", name, err)
	}
	return method.ID
}

func saleLines(t *testing.T, db *gorm.DB, saleID uuid.UUID) []entities.PaymentLine {
	t.Helper()
	lines, err := persistence.NewSQLitePaymentLineRepository(db).
		GetByPayable(context.Background(), entities.PayableSale, saleID)
	if err != nil {
		t.Fatalf("leyendo líneas: %v", err)
	}
	return lines
}

// Compatibilidad: un cliente que sólo manda payment_method_id sigue funcionando
// y produce una línea por el total. El caso simple es el general con N=1.
func TestCreateSale_MetodoUnicoProduceUnaLinea(t *testing.T) {
	db := newTestDB(t)
	saleUC, product, methodID, sellerID := seedPOS(t, db, 10)

	sale := &entities.Sale{
		UserID:          sellerID,
		PaymentMethodID: methodID,
		Details: []entities.SaleDetail{{
			ProductID: product.ID, UnitPrice: 5000, Quantity: 3,
		}},
	}
	if err := saleUC.CreateSale(context.Background(), sale); err != nil {
		t.Fatalf("CreateSale: %v", err)
	}

	lines := saleLines(t, db, sale.ID)
	if len(lines) != 1 {
		t.Fatalf("líneas = %d, want 1", len(lines))
	}
	if lines[0].Amount != 15000 || lines[0].PaymentMethodID != methodID {
		t.Errorf("línea = %v por %v, want 15000 con el método único", lines[0].Amount, lines[0].PaymentMethodID)
	}
}

// Lo que pidió el negocio: pagar una venta con parte efectivo y parte otro medio.
func TestCreateSale_PagoCombinado(t *testing.T) {
	db := newTestDB(t)
	saleUC, product, efectivo, sellerID := seedPOS(t, db, 10)
	transferencia := newMethod(t, db, "Transferencia", entities.PaymentTypeTransfer)

	sale := &entities.Sale{
		UserID: sellerID,
		PaymentLines: []entities.PaymentLine{
			{PaymentMethodID: efectivo, Amount: 6000},
			{PaymentMethodID: transferencia, Amount: 4000},
		},
		Details: []entities.SaleDetail{{ProductID: product.ID, UnitPrice: 5000, Quantity: 2}},
	}
	if err := saleUC.CreateSale(context.Background(), sale); err != nil {
		t.Fatalf("CreateSale: %v", err)
	}

	lines := saleLines(t, db, sale.ID)
	if len(lines) != 2 {
		t.Fatalf("líneas = %d, want 2", len(lines))
	}

	total := 0.0
	for _, l := range lines {
		total += l.Amount
	}
	if total != 10000 {
		t.Errorf("suma de las líneas = %v, want 10000", total)
	}
	// El método principal (mayor monto) mantiene rellena la columna vieja.
	if sale.PaymentMethodID != efectivo {
		t.Errorf("método principal = %v, want el de mayor monto (efectivo)", sale.PaymentMethodID)
	}
}

// El total lo decide el servidor: si las líneas no lo cubren, no hay venta.
func TestCreateSale_RechazaPagoQueNoCuadra(t *testing.T) {
	db := newTestDB(t)
	saleUC, product, efectivo, sellerID := seedPOS(t, db, 10)
	transferencia := newMethod(t, db, "Transferencia", entities.PaymentTypeTransfer)

	sale := &entities.Sale{
		UserID: sellerID,
		PaymentLines: []entities.PaymentLine{
			{PaymentMethodID: efectivo, Amount: 6000},
			{PaymentMethodID: transferencia, Amount: 3000}, // falta 1000
		},
		Details: []entities.SaleDetail{{ProductID: product.ID, UnitPrice: 5000, Quantity: 2}},
	}
	err := saleUC.CreateSale(context.Background(), sale)
	if !errors.Is(err, apperrors.ErrPaymentLinesMismatch) {
		t.Fatalf("error = %v, want ErrPaymentLinesMismatch", err)
	}

	// Y no puede quedar rastro: ni venta, ni líneas, ni stock movido.
	for table, want := range map[string]int64{"sales": 0, "payment_lines": 0, "stock_movements": 0} {
		var n int64
		db.Raw("SELECT COUNT(*) FROM " + table).Scan(&n)
		if n != want {
			t.Errorf("filas en %s = %d, want %d", table, n, want)
		}
	}
	if got := stockOf(t, db, product.ID); got != 10 {
		t.Errorf("stock = %d, want 10", got)
	}
}

// El caso que pidió el usuario: pagan con 100.000 algo de 45.000.
func TestCreateSale_CalculaElVuelto(t *testing.T) {
	db := newTestDB(t)
	saleUC, product, efectivo, sellerID := seedPOS(t, db, 10)

	sale := &entities.Sale{
		UserID: sellerID,
		PaymentLines: []entities.PaymentLine{
			{PaymentMethodID: efectivo, Amount: 45000, CashReceived: 100000},
		},
		Details: []entities.SaleDetail{{ProductID: product.ID, UnitPrice: 45000, Quantity: 1}},
	}
	if err := saleUC.CreateSale(context.Background(), sale); err != nil {
		t.Fatalf("CreateSale: %v", err)
	}

	lines := saleLines(t, db, sale.ID)
	if len(lines) != 1 {
		t.Fatalf("líneas = %d, want 1", len(lines))
	}
	if lines[0].ChangeGiven != 55000 {
		t.Errorf("vuelto = %v, want 55000", lines[0].ChangeGiven)
	}
	// Lo que hay en el cajón es el monto, no lo recibido: si el desglose sumara
	// CashReceived, la caja saldría inflada en el valor de todos los vueltos.
	if lines[0].Amount != 45000 {
		t.Errorf("monto = %v, want 45000", lines[0].Amount)
	}
	if lines[0].CashReceived != 100000 {
		t.Errorf("recibido = %v, want 100000", lines[0].CashReceived)
	}
}

// El servidor recalcula el vuelto: nunca se confía en un importe monetario que
// venga del navegador.
func TestCreateSale_IgnoraElVueltoDelCliente(t *testing.T) {
	db := newTestDB(t)
	saleUC, product, efectivo, sellerID := seedPOS(t, db, 10)

	sale := &entities.Sale{
		UserID: sellerID,
		PaymentLines: []entities.PaymentLine{
			{PaymentMethodID: efectivo, Amount: 45000, CashReceived: 100000, ChangeGiven: 999999},
		},
		Details: []entities.SaleDetail{{ProductID: product.ID, UnitPrice: 45000, Quantity: 1}},
	}
	if err := saleUC.CreateSale(context.Background(), sale); err != nil {
		t.Fatalf("CreateSale: %v", err)
	}
	if got := saleLines(t, db, sale.ID)[0].ChangeGiven; got != 55000 {
		t.Errorf("vuelto = %v, want 55000 — el del cliente debe ignorarse", got)
	}
}

// Sólo la pata de efectivo admite "recibido": una transferencia no da vuelto.
func TestCreateSale_RechazaVueltoEnMetodoNoEfectivo(t *testing.T) {
	db := newTestDB(t)
	saleUC, product, _, sellerID := seedPOS(t, db, 10)
	tarjeta := newMethod(t, db, "Tarjeta", entities.PaymentTypeCard)

	sale := &entities.Sale{
		UserID: sellerID,
		PaymentLines: []entities.PaymentLine{
			{PaymentMethodID: tarjeta, Amount: 45000, CashReceived: 100000},
		},
		Details: []entities.SaleDetail{{ProductID: product.ID, UnitPrice: 45000, Quantity: 1}},
	}
	if err := saleUC.CreateSale(context.Background(), sale); !errors.Is(err, apperrors.ErrCashChangeOnNonCash) {
		t.Fatalf("error = %v, want ErrCashChangeOnNonCash", err)
	}
}

// La anulación devuelve por las mismas patas por las que se cobró, de modo que
// el desglose por método vuelve a cero sin restar las anulaciones aparte.
func TestVoidSale_EspejaLasLineasNegadas(t *testing.T) {
	db := newTestDB(t)
	saleUC, product, efectivo, sellerID := seedPOS(t, db, 10)
	transferencia := newMethod(t, db, "Transferencia", entities.PaymentTypeTransfer)
	ctx := context.Background()

	sale := &entities.Sale{
		UserID: sellerID,
		PaymentLines: []entities.PaymentLine{
			{PaymentMethodID: efectivo, Amount: 6000, CashReceived: 10000},
			{PaymentMethodID: transferencia, Amount: 4000},
		},
		Details: []entities.SaleDetail{{ProductID: product.ID, UnitPrice: 5000, Quantity: 2}},
	}
	if err := saleUC.CreateSale(ctx, sale); err != nil {
		t.Fatalf("CreateSale: %v", err)
	}

	voided, err := saleUC.VoidSale(ctx, sale.ID, sellerID, time.UTC)
	if err != nil {
		t.Fatalf("VoidSale: %v", err)
	}

	mirror := saleLines(t, db, voided.ID)
	if len(mirror) != 2 {
		t.Fatalf("líneas espejo = %d, want 2 — se devuelve por las dos patas", len(mirror))
	}
	for _, l := range mirror {
		if l.Amount >= 0 {
			t.Errorf("la línea espejo debe ser negativa: %v", l.Amount)
		}
		if l.ReversalOf == nil {
			t.Error("cada línea espejo debe apuntar a la que revierte")
		}
		// Una reversión es un movimiento neto: no se "des-recibe" un billete.
		if l.CashReceived != 0 || l.ChangeGiven != 0 {
			t.Errorf("el espejo no lleva efectivo recibido ni vuelto: %v/%v", l.CashReceived, l.ChangeGiven)
		}
	}

	// La propiedad que hace cuadrar el desglose sin filtrar por estado.
	var net float64
	db.Raw("SELECT COALESCE(SUM(amount), 0) FROM payment_lines").Scan(&net)
	if net != 0 {
		t.Errorf("suma de todas las líneas = %v, want 0", net)
	}
}

// Dos líneas con el mismo método siempre son un error de interfaz, y romperían
// la regla de "una sola línea lleva el efectivo recibido".
func TestCreateSale_RechazaMetodoRepetido(t *testing.T) {
	db := newTestDB(t)
	saleUC, product, efectivo, sellerID := seedPOS(t, db, 10)

	sale := &entities.Sale{
		UserID: sellerID,
		PaymentLines: []entities.PaymentLine{
			{PaymentMethodID: efectivo, Amount: 5000},
			{PaymentMethodID: efectivo, Amount: 5000},
		},
		Details: []entities.SaleDetail{{ProductID: product.ID, UnitPrice: 5000, Quantity: 2}},
	}
	if err := saleUC.CreateSale(context.Background(), sale); !errors.Is(err, apperrors.ErrDuplicatePaymentMethod) {
		t.Fatalf("error = %v, want ErrDuplicatePaymentMethod", err)
	}
}
