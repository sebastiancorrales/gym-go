package usecases_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sebastiancorrales/gym-go/internal/domain/entities"
	"github.com/sebastiancorrales/gym-go/internal/infrastructure/persistence"
	"github.com/sebastiancorrales/gym-go/internal/usecases"
	"gorm.io/gorm"
)

// insertBareSubscription mete una fila de suscripción sin pasar por
// CreateSubscription: aquí sólo hace falta que exista el documento al que la
// línea de pago sembrada a mano pueda apuntar.
func insertBareSubscription(t *testing.T, db *gorm.DB, gymID uuid.UUID, totalPaid float64, date string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	now := time.Now().UTC().Round(0)
	err := db.Exec(`INSERT INTO subscriptions (id, user_id, plan_id, gym_id, start_date, end_date,
	                                           price_paid, enrollment_fee_paid, discount_applied, total_paid,
	                                           payment_method, status, date, hour, created_at, updated_at)
	                VALUES (?, ?, ?, ?, ?, ?, ?, 0, 0, ?, 'Efectivo', 'ACTIVE', ?, '11:00', ?, ?)`,
		id, uuid.New(), uuid.New(), gymID, now, now, totalPaid, totalPaid, date, now, now).Error
	if err != nil {
		t.Fatalf("insert subscription: %v", err)
	}
	return id
}

// El aserto que antes fallaba: un día con una venta, su anulación y una
// suscripción tiene que sumar exactamente el total — sin importar cuántas
// anulaciones haya. Antes de este endpoint, el Cuadre del Día excluía las
// anulaciones del desglose por método pero su total sí las restaba, así que
// sum(métodos) ≠ total en cualquier día con una anulación.
func TestPaymentSummary_CuadraConVentaAnulacionYSuscripcion(t *testing.T) {
	db := newTestDB(t)
	saleUC, product, efectivo, sellerID := seedPOS(t, db, 10)
	transferencia := newMethod(t, db, "Transferencia", entities.PaymentTypeTransfer)
	ctx := context.Background()

	// Venta normal en transferencia. Date/Hour se fijan a mano porque en
	// producción los pone el handler HTTP desde la zona horaria del gimnasio;
	// estos tests llaman al use case directamente, sin pasar por él.
	sale1 := &entities.Sale{
		UserID: sellerID, PaymentMethodID: transferencia, Date: today(t), Hour: "09:00",
		Details: []entities.SaleDetail{{ProductID: product.ID, UnitPrice: 45000, Quantity: 1}},
	}
	if err := saleUC.CreateSale(ctx, sale1); err != nil {
		t.Fatalf("CreateSale sale1: %v", err)
	}

	// Venta que luego se anula.
	sale2 := &entities.Sale{
		UserID: sellerID, PaymentMethodID: efectivo, Date: today(t), Hour: "10:00",
		Details: []entities.SaleDetail{{ProductID: product.ID, UnitPrice: 50000, Quantity: 2}},
	}
	if err := saleUC.CreateSale(ctx, sale2); err != nil {
		t.Fatalf("CreateSale sale2: %v", err)
	}
	if _, err := saleUC.VoidSale(ctx, sale2.ID, sellerID, time.UTC); err != nil {
		t.Fatalf("VoidSale: %v", err)
	}

	// Una suscripción pagada en efectivo, escrita directo a payment_lines como
	// lo haría CreateSubscription.
	lineRepo := persistence.NewSQLitePaymentLineRepository(db)
	subID := insertBareSubscription(t, db, sale1.GymID, 80000, today(t))
	if err := lineRepo.CreateBatch(ctx, []entities.PaymentLine{
		{PaymentMethodID: efectivo, Amount: 80000, PayableKind: entities.PayableSubscription, PayableID: subID, Date: today(t)},
	}); err != nil {
		t.Fatalf("sembrando línea de suscripción: %v", err)
	}

	paymentUC := usecases.NewPaymentUseCase(
		lineRepo,
		persistence.NewSQLitePaymentMethodRepository(db),
		persistence.NewSQLiteChangeLogRepository(db),
		persistence.NewSQLiteSaleRepository(db),
		persistence.NewSQLiteSubscriptionRepository(db),
		persistence.NewSQLiteExpenseRepository(db),
		persistence.NewUnitOfWork(db),
	)

	summary, err := paymentUC.Summary(ctx, today(t), today(t))
	if err != nil {
		t.Fatalf("Summary: %v", err)
	}

	// El aserto que de verdad importa: la suma de los métodos ES el total.
	var sumMethods float64
	for _, m := range summary.Methods {
		sumMethods += m.Total
	}
	if sumMethods != summary.Total {
		t.Errorf("sum(métodos) = %v, total = %v — deben coincidir siempre", sumMethods, summary.Total)
	}

	// La venta anulada neta cero: no debe sumar ni restar del total esperado.
	want := 45000.0 + 80000.0 // sale1 (transferencia) + suscripción (efectivo); sale2 se anula a sí misma
	if summary.Total != want {
		t.Errorf("total = %v, want %v (la venta anulada debe netear a cero)", summary.Total, want)
	}

	byName := map[string]float64{}
	for _, m := range summary.Methods {
		byName[m.PaymentMethodName] = m.Total
	}
	if byName["Transferencia"] != 45000 {
		t.Errorf("Transferencia = %v, want 45000", byName["Transferencia"])
	}
	if byName["Efectivo"] != 80000 {
		t.Errorf("Efectivo = %v, want 80000 (100000 de la venta anulada - 100000 del espejo + 80000 de la suscripción)", byName["Efectivo"])
	}
}

func today(t *testing.T) string {
	t.Helper()
	return time.Now().UTC().Format("2006-01-02")
}
