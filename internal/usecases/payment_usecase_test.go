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

func newPaymentUC(db *gorm.DB) *usecases.PaymentUseCase {
	return usecases.NewPaymentUseCase(
		persistence.NewSQLitePaymentLineRepository(db),
		persistence.NewSQLitePaymentMethodRepository(db),
		persistence.NewSQLiteChangeLogRepository(db),
		persistence.NewSQLiteSaleRepository(db),
		persistence.NewSQLiteSubscriptionRepository(db),
		persistence.NewSQLiteExpenseRepository(db),
		persistence.NewSQLiteDebtRepository(db),
		persistence.NewUnitOfWork(db),
	)
}

func amend(methodID uuid.UUID, amount float64) entities.PaymentLine {
	return entities.PaymentLine{PaymentMethodID: methodID, Amount: amount}
}

func auditCount(t *testing.T, db *gorm.DB) int64 {
	t.Helper()
	var n int64
	db.Raw("SELECT COUNT(*) FROM change_logs").Scan(&n)
	return n
}

// netByMethod devuelve el desglose neto por método, que es lo que ve el cuadre.
func netByMethod(t *testing.T, db *gorm.DB) map[string]float64 {
	t.Helper()
	rows, err := db.Raw(`SELECT m.name, COALESCE(SUM(p.amount), 0) FROM payment_lines p
	                     LEFT JOIN sale_payment_methods m ON m.id = p.payment_method_id
	                     GROUP BY m.name`).Rows()
	if err != nil {
		t.Fatalf("desglose: %v", err)
	}
	defer rows.Close()

	out := map[string]float64{}
	for rows.Next() {
		var name string
		var total float64
		if err := rows.Scan(&name, &total); err != nil {
			t.Fatalf("scan: %v", err)
		}
		out[name] = total
	}
	return out
}

func TestAmendPayments_CorrigeElMetodoDeUnaVenta(t *testing.T) {
	db := newTestDB(t)
	saleUC, product, efectivo, sellerID := seedPOS(t, db, 10)
	transferencia := newMethod(t, db, "Transferencia", entities.PaymentTypeTransfer)
	paymentUC := newPaymentUC(db)
	ctx := context.Background()

	sale := &entities.Sale{
		UserID:          sellerID,
		PaymentMethodID: efectivo,
		Details:         []entities.SaleDetail{{ProductID: product.ID, UnitPrice: 45000, Quantity: 1}},
	}
	if err := saleUC.CreateSale(ctx, sale); err != nil {
		t.Fatalf("CreateSale: %v", err)
	}

	_, err := paymentUC.AmendPayments(ctx, usecases.AmendPaymentsInput{
		Kind:          entities.PayableSale,
		PayableID:     sale.ID,
		Lines:         []entities.PaymentLine{amend(transferencia, 45000)},
		Reason:        "El cajero marco efectivo por error",
		ChangedByID:   sellerID,
		ChangedByName: "Admin Uno",
	})
	if err != nil {
		t.Fatalf("AmendPayments: %v", err)
	}

	lines := saleLines(t, db, sale.ID)
	if len(lines) != 1 || lines[0].PaymentMethodID != transferencia {
		t.Fatalf("líneas = %+v, se esperaba una de transferencia", lines)
	}

	// El denormalizado que leen la API y cinco sitios de producción.
	var primary uuid.UUID
	var raw string
	db.Raw("SELECT payment_method_id FROM sales WHERE id = ?", sale.ID).Scan(&raw)
	primary, _ = uuid.Parse(raw)
	if primary != transferencia {
		t.Errorf("método principal de la venta = %v, want transferencia", primary)
	}

	// Y queda constancia de quién y por qué.
	var entry entities.ChangeLog
	if err := db.Where("entity_id = ?", sale.ID).First(&entry).Error; err != nil {
		t.Fatalf("leyendo la bitácora: %v", err)
	}
	if entry.OldValue != "Efectivo $45.000" || entry.NewValue != "Transferencia $45.000" {
		t.Errorf("bitácora: %q -> %q", entry.OldValue, entry.NewValue)
	}
	if entry.ChangedByName != "Admin Uno" || entry.Reason == "" {
		t.Errorf("la bitácora debe registrar quién y por qué: %+v", entry)
	}
}

// El caso que el plan marcaba como crítico.
//
// Venta de 100.000 en Efectivo, anulada. El admin corrige: "en realidad fue
// Transferencia". Si el espejo no se reescribe, el desglose muestra
// Transferencia +100.000 y Efectivo −100.000: el total del día sigue en cero,
// pero aparece un fantasma en efectivo que no corresponde a nada.
func TestAmendPayments_CascadeaAlEspejoDeUnaAnulacion(t *testing.T) {
	db := newTestDB(t)
	saleUC, product, efectivo, sellerID := seedPOS(t, db, 10)
	transferencia := newMethod(t, db, "Transferencia", entities.PaymentTypeTransfer)
	paymentUC := newPaymentUC(db)
	ctx := context.Background()

	sale := &entities.Sale{
		UserID:          sellerID,
		PaymentMethodID: efectivo,
		Details:         []entities.SaleDetail{{ProductID: product.ID, UnitPrice: 50000, Quantity: 2}},
	}
	if err := saleUC.CreateSale(ctx, sale); err != nil {
		t.Fatalf("CreateSale: %v", err)
	}
	voided, err := saleUC.VoidSale(ctx, sale.ID, sellerID, time.UTC)
	if err != nil {
		t.Fatalf("VoidSale: %v", err)
	}

	if _, err := paymentUC.AmendPayments(ctx, usecases.AmendPaymentsInput{
		Kind:          entities.PayableSale,
		PayableID:     sale.ID,
		Lines:         []entities.PaymentLine{amend(transferencia, 100000)},
		ChangedByID:   sellerID,
		ChangedByName: "Admin",
	}); err != nil {
		t.Fatalf("AmendPayments: %v", err)
	}

	mirror := saleLines(t, db, voided.ID)
	if len(mirror) != 1 {
		t.Fatalf("líneas del espejo = %d, want 1", len(mirror))
	}
	if mirror[0].PaymentMethodID != transferencia {
		t.Error("el espejo debe seguir al método corregido")
	}
	if mirror[0].Amount != -100000 {
		t.Errorf("importe del espejo = %v, want -100000", mirror[0].Amount)
	}

	// Lo que de verdad importa: no queda ningún fantasma en el desglose.
	net := netByMethod(t, db)
	for name, total := range net {
		if total != 0 {
			t.Errorf("%s = %v, want 0 — venta y anulación deben cancelarse por el mismo método", name, total)
		}
	}
}

// El espejo se corrige a través del original. Editarlo por separado dejaría las
// dos mitades diciendo cosas distintas.
func TestAmendPayments_RechazaCorregirElEspejo(t *testing.T) {
	db := newTestDB(t)
	saleUC, product, efectivo, sellerID := seedPOS(t, db, 10)
	transferencia := newMethod(t, db, "Transferencia", entities.PaymentTypeTransfer)
	paymentUC := newPaymentUC(db)
	ctx := context.Background()

	sale := &entities.Sale{
		UserID:          sellerID,
		PaymentMethodID: efectivo,
		Details:         []entities.SaleDetail{{ProductID: product.ID, UnitPrice: 45000, Quantity: 1}},
	}
	if err := saleUC.CreateSale(ctx, sale); err != nil {
		t.Fatalf("CreateSale: %v", err)
	}
	voided, err := saleUC.VoidSale(ctx, sale.ID, sellerID, time.UTC)
	if err != nil {
		t.Fatalf("VoidSale: %v", err)
	}

	_, err = paymentUC.AmendPayments(ctx, usecases.AmendPaymentsInput{
		Kind:      entities.PayableSale,
		PayableID: voided.ID,
		Lines:     []entities.PaymentLine{amend(transferencia, -45000)},
	})
	if !errors.Is(err, apperrors.ErrCannotAmendVoidSale) {
		t.Fatalf("error = %v, want ErrCannotAmendVoidSale", err)
	}
}

// Cada clic en Guardar no puede ensuciar la bitácora con una entrada que no
// cambió nada.
func TestAmendPayments_NoRegistraCuandoNoHayCambio(t *testing.T) {
	db := newTestDB(t)
	saleUC, product, efectivo, sellerID := seedPOS(t, db, 10)
	paymentUC := newPaymentUC(db)
	ctx := context.Background()

	sale := &entities.Sale{
		UserID:          sellerID,
		PaymentMethodID: efectivo,
		Details:         []entities.SaleDetail{{ProductID: product.ID, UnitPrice: 45000, Quantity: 1}},
	}
	if err := saleUC.CreateSale(ctx, sale); err != nil {
		t.Fatalf("CreateSale: %v", err)
	}

	for i := 0; i < 3; i++ {
		if _, err := paymentUC.AmendPayments(ctx, usecases.AmendPaymentsInput{
			Kind:      entities.PayableSale,
			PayableID: sale.ID,
			Lines:     []entities.PaymentLine{amend(efectivo, 45000)},
		}); err != nil {
			t.Fatalf("AmendPayments: %v", err)
		}
	}

	if n := auditCount(t, db); n != 0 {
		t.Errorf("entradas de bitácora = %d, want 0 — guardar lo mismo no es un cambio", n)
	}
}

// La corrección no puede cambiar el importe: el total se lee del documento.
func TestAmendPayments_RechazaCambiarElTotal(t *testing.T) {
	db := newTestDB(t)
	saleUC, product, efectivo, sellerID := seedPOS(t, db, 10)
	transferencia := newMethod(t, db, "Transferencia", entities.PaymentTypeTransfer)
	paymentUC := newPaymentUC(db)
	ctx := context.Background()

	sale := &entities.Sale{
		UserID:          sellerID,
		PaymentMethodID: efectivo,
		Details:         []entities.SaleDetail{{ProductID: product.ID, UnitPrice: 45000, Quantity: 1}},
	}
	if err := saleUC.CreateSale(ctx, sale); err != nil {
		t.Fatalf("CreateSale: %v", err)
	}

	_, err := paymentUC.AmendPayments(ctx, usecases.AmendPaymentsInput{
		Kind:      entities.PayableSale,
		PayableID: sale.ID,
		Lines:     []entities.PaymentLine{amend(transferencia, 999000)},
	})
	if !errors.Is(err, apperrors.ErrPaymentLinesMismatch) {
		t.Fatalf("error = %v, want ErrPaymentLinesMismatch", err)
	}

	if lines := saleLines(t, db, sale.ID); lines[0].Amount != 45000 {
		t.Errorf("el importe no puede haber cambiado: %v", lines[0].Amount)
	}
}

// Se puede corregir hacia un pago combinado, no sólo cambiar de método.
func TestAmendPayments_PermiteDividirUnCobroYaHecho(t *testing.T) {
	db := newTestDB(t)
	saleUC, product, efectivo, sellerID := seedPOS(t, db, 10)
	transferencia := newMethod(t, db, "Transferencia", entities.PaymentTypeTransfer)
	paymentUC := newPaymentUC(db)
	ctx := context.Background()

	sale := &entities.Sale{
		UserID:          sellerID,
		PaymentMethodID: efectivo,
		Details:         []entities.SaleDetail{{ProductID: product.ID, UnitPrice: 50000, Quantity: 2}},
	}
	if err := saleUC.CreateSale(ctx, sale); err != nil {
		t.Fatalf("CreateSale: %v", err)
	}

	if _, err := paymentUC.AmendPayments(ctx, usecases.AmendPaymentsInput{
		Kind:      entities.PayableSale,
		PayableID: sale.ID,
		Lines: []entities.PaymentLine{
			amend(efectivo, 30000),
			amend(transferencia, 70000),
		},
		ChangedByName: "Admin",
	}); err != nil {
		t.Fatalf("AmendPayments: %v", err)
	}

	lines := saleLines(t, db, sale.ID)
	if len(lines) != 2 {
		t.Fatalf("líneas = %d, want 2", len(lines))
	}
	net := netByMethod(t, db)
	if net["Efectivo"] != 30000 || net["Transferencia"] != 70000 {
		t.Errorf("desglose = %+v", net)
	}
}

// Un método desactivado después del cobro no puede impedir corregir la OTRA
// pata: si no, el admin se quedaría sin poder arreglar nada.
func TestAmendPayments_ToleraMetodoInactivoQueYaEstaba(t *testing.T) {
	db := newTestDB(t)
	saleUC, product, efectivo, sellerID := seedPOS(t, db, 10)
	transferencia := newMethod(t, db, "Transferencia", entities.PaymentTypeTransfer)
	tarjeta := newMethod(t, db, "Tarjeta", entities.PaymentTypeCard)
	paymentUC := newPaymentUC(db)
	ctx := context.Background()

	sale := &entities.Sale{
		UserID: sellerID,
		PaymentLines: []entities.PaymentLine{
			amend(efectivo, 40000),
			amend(transferencia, 60000),
		},
		Details: []entities.SaleDetail{{ProductID: product.ID, UnitPrice: 50000, Quantity: 2}},
	}
	if err := saleUC.CreateSale(ctx, sale); err != nil {
		t.Fatalf("CreateSale: %v", err)
	}

	// Se desactiva efectivo DESPUÉS del cobro.
	if err := db.Exec("UPDATE sale_payment_methods SET status = 'inactive' WHERE id = ?", efectivo).Error; err != nil {
		t.Fatalf("desactivando: %v", err)
	}

	// Corregir la otra pata debe funcionar aunque efectivo siga en el conjunto.
	if _, err := paymentUC.AmendPayments(ctx, usecases.AmendPaymentsInput{
		Kind:      entities.PayableSale,
		PayableID: sale.ID,
		Lines: []entities.PaymentLine{
			amend(efectivo, 40000),
			amend(tarjeta, 60000),
		},
		ChangedByName: "Admin",
	}); err != nil {
		t.Fatalf("AmendPayments: %v — un método ya presente no debe bloquear la corrección", err)
	}

	// Pero introducir uno inactivo que NO estaba sí se rechaza.
	inactivo := newMethod(t, db, "Bono", entities.PaymentMethodTypeOther)
	if err := db.Exec("UPDATE sale_payment_methods SET status = 'inactive' WHERE id = ?", inactivo).Error; err != nil {
		t.Fatalf("desactivando: %v", err)
	}
	_, err := paymentUC.AmendPayments(ctx, usecases.AmendPaymentsInput{
		Kind:      entities.PayableSale,
		PayableID: sale.ID,
		Lines:     []entities.PaymentLine{amend(inactivo, 100000)},
	})
	if !errors.Is(err, apperrors.ErrPaymentMethodNotActive) {
		t.Fatalf("error = %v, want ErrPaymentMethodNotActive", err)
	}
}

// Corregir no puede mover el cobro de día: eso lo sacaría del cierre diario en
// el que ya se contó y descuadraría dos días a la vez.
func TestAmendPayments_ConservaLaFechaDelDocumento(t *testing.T) {
	db := newTestDB(t)
	saleUC, product, efectivo, sellerID := seedPOS(t, db, 10)
	transferencia := newMethod(t, db, "Transferencia", entities.PaymentTypeTransfer)
	paymentUC := newPaymentUC(db)
	ctx := context.Background()

	sale := &entities.Sale{
		UserID:          sellerID,
		PaymentMethodID: efectivo,
		Date:            "2026-01-15",
		Hour:            "14:30",
		Details:         []entities.SaleDetail{{ProductID: product.ID, UnitPrice: 45000, Quantity: 1}},
	}
	if err := saleUC.CreateSale(ctx, sale); err != nil {
		t.Fatalf("CreateSale: %v", err)
	}

	if _, err := paymentUC.AmendPayments(ctx, usecases.AmendPaymentsInput{
		Kind:      entities.PayableSale,
		PayableID: sale.ID,
		Lines:     []entities.PaymentLine{amend(transferencia, 45000)},
	}); err != nil {
		t.Fatalf("AmendPayments: %v", err)
	}

	lines := saleLines(t, db, sale.ID)
	if lines[0].Date != "2026-01-15" || lines[0].Hour != "14:30" {
		t.Errorf("fecha = %s %s, want 2026-01-15 14:30", lines[0].Date, lines[0].Hour)
	}
}

// Regresión de un bug real: al cascadear se pasaba a Sales.Update un
// entities.Sale PARCIAL (sólo ID y método). SaleRepository.Update usa Save(),
// que escribe TODAS las columnas, así que borraba voided_sale_id, total, type y
// date del espejo y lo dejaba huérfano — el desglose quedaba descuadrado y una
// segunda corrección ya no encontraba la contrapartida.
//
// Es el mismo bug de Save() que se cerró en productos, por otra puerta.
func TestAmendPayments_NoDestruyeLaVentaEspejo(t *testing.T) {
	db := newTestDB(t)
	saleUC, product, efectivo, sellerID := seedPOS(t, db, 10)
	transferencia := newMethod(t, db, "Transferencia", entities.PaymentTypeTransfer)
	paymentUC := newPaymentUC(db)
	ctx := context.Background()

	sale := &entities.Sale{
		UserID:          sellerID,
		PaymentMethodID: efectivo,
		Details:         []entities.SaleDetail{{ProductID: product.ID, UnitPrice: 50000, Quantity: 2}},
	}
	if err := saleUC.CreateSale(ctx, sale); err != nil {
		t.Fatalf("CreateSale: %v", err)
	}
	voided, err := saleUC.VoidSale(ctx, sale.ID, sellerID, time.UTC)
	if err != nil {
		t.Fatalf("VoidSale: %v", err)
	}
	beforeDate, beforeTotal := voided.Date, voided.Total

	// Dos correcciones seguidas: la segunda sólo funciona si la primera dejó el
	// espejo intacto.
	for i, lines := range [][]entities.PaymentLine{
		{amend(transferencia, 100000)},
		{amend(efectivo, 60000), amend(transferencia, 40000)},
	} {
		if _, err := paymentUC.AmendPayments(ctx, usecases.AmendPaymentsInput{
			Kind: entities.PayableSale, PayableID: sale.ID, Lines: lines,
			ChangedByName: "Admin",
		}); err != nil {
			t.Fatalf("corrección %d: %v", i+1, err)
		}
	}

	var mirror struct {
		VoidedSaleID *string
		Total        float64
		Type         string
		Date         string
	}
	if err := db.Raw("SELECT voided_sale_id, total, type, date FROM sales WHERE id = ?", voided.ID).
		Scan(&mirror).Error; err != nil {
		t.Fatalf("leyendo el espejo: %v", err)
	}

	if mirror.VoidedSaleID == nil || *mirror.VoidedSaleID != sale.ID.String() {
		t.Errorf("voided_sale_id = %v, want %s — la cascada no puede dejar el espejo huérfano", mirror.VoidedSaleID, sale.ID)
	}
	if mirror.Total != beforeTotal {
		t.Errorf("total del espejo = %v, want %v", mirror.Total, beforeTotal)
	}
	if mirror.Type != string(entities.SaleTypeVoid) {
		t.Errorf("type del espejo = %q, want void", mirror.Type)
	}
	if mirror.Date != beforeDate {
		t.Errorf("date del espejo = %q, want %q", mirror.Date, beforeDate)
	}

	// Y el desglose vuelve a cero pata por pata.
	for name, total := range netByMethod(t, db) {
		if total != 0 {
			t.Errorf("%s = %v, want 0", name, total)
		}
	}
}
