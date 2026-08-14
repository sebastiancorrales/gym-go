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

// fiarVenta crea una venta 100% fiada por 2.000 con el deudor indicado.
func fiarVenta(t *testing.T, ctx context.Context, saleUC *usecases.SaleUseCase,
	productID, fiadoMethodID, sellerID uuid.UUID, date string, debtor *entities.DebtorInfo) *entities.Sale {
	t.Helper()
	sale := &entities.Sale{
		UserID: sellerID, Date: date, Hour: "09:00",
		Details:      []entities.SaleDetail{{ProductID: productID, UnitPrice: 2000, Quantity: 1}},
		PaymentLines: []entities.PaymentLine{{PaymentMethodID: fiadoMethodID, Amount: 2000}},
		Debtor:       debtor,
	}
	if err := saleUC.CreateSale(ctx, sale); err != nil {
		t.Fatalf("CreateSale (fiado): %v", err)
	}
	return sale
}

func TestCreateSale_ConFiadoDeNombreLibre_CreaDeudaPendiente(t *testing.T) {
	db := newTestDB(t)
	saleUC, product, _, sellerID := seedPOS(t, db, 10)
	fiado := newMethod(t, db, "Fiado", entities.PaymentTypeCredit)
	ctx := context.Background()

	sale := fiarVenta(t, ctx, saleUC, product.ID, fiado, sellerID, today(t),
		&entities.DebtorInfo{Name: "Juan Pérez"})

	debt, err := persistence.NewSQLiteDebtRepository(db).GetBySaleID(ctx, sale.ID)
	if err != nil {
		t.Fatalf("GetBySaleID: %v", err)
	}
	if debt == nil {
		t.Fatal("no se creó ninguna deuda para la venta fiada")
	}
	if debt.Status != entities.DebtStatusPending {
		t.Errorf("Status = %v, want pending", debt.Status)
	}
	if debt.Amount != 2000 {
		t.Errorf("Amount = %v, want 2000", debt.Amount)
	}
	if debt.DebtorName != "Juan Pérez" {
		t.Errorf("DebtorName = %q, want %q", debt.DebtorName, "Juan Pérez")
	}
	if debt.DebtorUserID != nil {
		t.Errorf("DebtorUserID = %v, want nil (nombre libre, sin socio)", debt.DebtorUserID)
	}
}

func TestCreateSale_ConFiadoVinculadoASocio_GuardaElUserID(t *testing.T) {
	db := newTestDB(t)
	saleUC, product, _, sellerID := seedPOS(t, db, 10)
	fiado := newMethod(t, db, "Fiado", entities.PaymentTypeCredit)
	ctx := context.Background()

	member := &entities.User{
		ID: uuid.New(), Email: "socio@test.local", FirstName: "María", LastName: "Gómez",
		Role: entities.RoleMember, Status: entities.UserStatusActive,
	}
	if err := persistence.NewSQLiteUserRepository(db).Create(member); err != nil {
		t.Fatalf("creando socio: %v", err)
	}

	sale := fiarVenta(t, ctx, saleUC, product.ID, fiado, sellerID, today(t),
		&entities.DebtorInfo{UserID: &member.ID})

	debt, err := persistence.NewSQLiteDebtRepository(db).GetBySaleID(ctx, sale.ID)
	if err != nil {
		t.Fatalf("GetBySaleID: %v", err)
	}
	if debt.DebtorUserID == nil || *debt.DebtorUserID != member.ID {
		t.Errorf("DebtorUserID = %v, want %v", debt.DebtorUserID, member.ID)
	}
	// Sin nombre libre tecleado, se usa el del socio — así la pantalla de
	// deudores no muestra una fila en blanco.
	if debt.DebtorName != "María Gómez" {
		t.Errorf("DebtorName = %q, want %q (del socio, a falta de uno tecleado)", debt.DebtorName, "María Gómez")
	}
}

func TestCreateSale_ConFiado_RequiereNombreOSocio(t *testing.T) {
	db := newTestDB(t)
	saleUC, product, _, sellerID := seedPOS(t, db, 10)
	fiado := newMethod(t, db, "Fiado", entities.PaymentTypeCredit)
	ctx := context.Background()

	sale := &entities.Sale{
		UserID: sellerID, Date: today(t), Hour: "09:00",
		Details:      []entities.SaleDetail{{ProductID: product.ID, UnitPrice: 2000, Quantity: 1}},
		PaymentLines: []entities.PaymentLine{{PaymentMethodID: fiado, Amount: 2000}},
		// Debtor deliberadamente nil.
	}
	err := saleUC.CreateSale(ctx, sale)
	if !errors.Is(err, apperrors.ErrDebtorRequired) {
		t.Errorf("err = %v, want ErrDebtorRequired", err)
	}
}

// newDebtUC arma un DebtUseCase con los mismos repos que newTestDB ya migró.
func newDebtUC(db *gorm.DB) *usecases.DebtUseCase {
	return usecases.NewDebtUseCase(
		persistence.NewSQLiteDebtRepository(db),
		persistence.NewSQLitePaymentMethodRepository(db),
		persistence.NewSQLitePaymentLineRepository(db),
		persistence.NewSQLiteUserRepository(db),
		persistence.NewSQLiteSaleDetailRepository(db),
		persistence.NewSQLiteProductRepository(db),
		persistence.NewUnitOfWork(db),
	)
}

func TestCollectDebt_MarcaLaDeudaComoPagada(t *testing.T) {
	db := newTestDB(t)
	saleUC, product, efectivoID, sellerID := seedPOS(t, db, 10)
	fiado := newMethod(t, db, "Fiado", entities.PaymentTypeCredit)
	ctx := context.Background()

	sale := fiarVenta(t, ctx, saleUC, product.ID, fiado, sellerID, today(t), &entities.DebtorInfo{Name: "Juan"})
	debtRepo := persistence.NewSQLiteDebtRepository(db)
	debt, _ := debtRepo.GetBySaleID(ctx, sale.ID)

	debtUC := newDebtUC(db)
	collected, err := debtUC.CollectDebt(ctx, debt.ID, usecases.CollectDebtInput{
		PaymentMethodID: efectivoID,
		CollectedByID:   sellerID,
	}, time.UTC)
	if err != nil {
		t.Fatalf("CollectDebt: %v", err)
	}
	if collected.Status != entities.DebtStatusCollected {
		t.Errorf("Status = %v, want collected", collected.Status)
	}
	if collected.CollectedPaymentLineID == nil {
		t.Error("CollectedPaymentLineID no quedó enlazado a la línea de pago real")
	}

	line, err := persistence.NewSQLitePaymentLineRepository(db).GetByPayable(ctx, entities.PayableDebt, debt.ID)
	if err != nil || len(line) != 1 {
		t.Fatalf("esperaba exactamente una línea PayableDebt, lines=%v err=%v", line, err)
	}
	if line[0].Amount != 2000 {
		t.Errorf("Amount de la línea de cobro = %v, want 2000", line[0].Amount)
	}
}

func TestCollectDebt_FallaSiYaFueCobrada(t *testing.T) {
	db := newTestDB(t)
	saleUC, product, efectivoID, sellerID := seedPOS(t, db, 10)
	fiado := newMethod(t, db, "Fiado", entities.PaymentTypeCredit)
	ctx := context.Background()

	sale := fiarVenta(t, ctx, saleUC, product.ID, fiado, sellerID, today(t), &entities.DebtorInfo{Name: "Juan"})
	debt, _ := persistence.NewSQLiteDebtRepository(db).GetBySaleID(ctx, sale.ID)

	debtUC := newDebtUC(db)
	if _, err := debtUC.CollectDebt(ctx, debt.ID, usecases.CollectDebtInput{PaymentMethodID: efectivoID, CollectedByID: sellerID}, time.UTC); err != nil {
		t.Fatalf("primer cobro: %v", err)
	}
	if _, err := debtUC.CollectDebt(ctx, debt.ID, usecases.CollectDebtInput{PaymentMethodID: efectivoID, CollectedByID: sellerID}, time.UTC); !errors.Is(err, apperrors.ErrDebtNotPending) {
		t.Errorf("segundo cobro: err = %v, want ErrDebtNotPending", err)
	}
}

func TestCollectDebt_FallaSiSeIntentaCobrarConOtroFiado(t *testing.T) {
	db := newTestDB(t)
	saleUC, product, _, sellerID := seedPOS(t, db, 10)
	fiado := newMethod(t, db, "Fiado", entities.PaymentTypeCredit)
	ctx := context.Background()

	sale := fiarVenta(t, ctx, saleUC, product.ID, fiado, sellerID, today(t), &entities.DebtorInfo{Name: "Juan"})
	debt, _ := persistence.NewSQLiteDebtRepository(db).GetBySaleID(ctx, sale.ID)

	debtUC := newDebtUC(db)
	_, err := debtUC.CollectDebt(ctx, debt.ID, usecases.CollectDebtInput{PaymentMethodID: fiado, CollectedByID: sellerID}, time.UTC)
	if !errors.Is(err, apperrors.ErrCannotPayDebtWithCredit) {
		t.Errorf("err = %v, want ErrCannotPayDebtWithCredit", err)
	}
}

func TestUncollectDebt_RevierteYQuedaPendienteOtraVez(t *testing.T) {
	db := newTestDB(t)
	saleUC, product, efectivoID, sellerID := seedPOS(t, db, 10)
	fiado := newMethod(t, db, "Fiado", entities.PaymentTypeCredit)
	ctx := context.Background()

	sale := fiarVenta(t, ctx, saleUC, product.ID, fiado, sellerID, today(t), &entities.DebtorInfo{Name: "Juan"})
	debtRepo := persistence.NewSQLiteDebtRepository(db)
	debt, _ := debtRepo.GetBySaleID(ctx, sale.ID)

	debtUC := newDebtUC(db)
	if _, err := debtUC.CollectDebt(ctx, debt.ID, usecases.CollectDebtInput{PaymentMethodID: efectivoID, CollectedByID: sellerID}, time.UTC); err != nil {
		t.Fatalf("CollectDebt: %v", err)
	}

	reverted, err := debtUC.UncollectDebt(ctx, debt.ID, time.UTC)
	if err != nil {
		t.Fatalf("UncollectDebt: %v", err)
	}
	if reverted.Status != entities.DebtStatusPending {
		t.Errorf("Status = %v, want pending otra vez", reverted.Status)
	}
	if reverted.CollectedPaymentLineID != nil {
		t.Error("CollectedPaymentLineID debería limpiarse al revertir")
	}

	// El neto de las dos líneas de payment_lines (cobro + su espejo) debe ser
	// cero: el dinero "salió" tan real como "entró".
	lines, err := persistence.NewSQLitePaymentLineRepository(db).GetByPayable(ctx, entities.PayableDebt, debt.ID)
	if err != nil {
		t.Fatalf("GetByPayable: %v", err)
	}
	var net float64
	for _, l := range lines {
		net += l.Amount
	}
	if net != 0 {
		t.Errorf("neto de las líneas de cobro/reversión = %v, want 0", net)
	}
}

func TestVoidSale_ConFiadoPendiente_AnulaLaDeudaTambien(t *testing.T) {
	db := newTestDB(t)
	saleUC, product, _, sellerID := seedPOS(t, db, 10)
	fiado := newMethod(t, db, "Fiado", entities.PaymentTypeCredit)
	ctx := context.Background()

	sale := fiarVenta(t, ctx, saleUC, product.ID, fiado, sellerID, today(t), &entities.DebtorInfo{Name: "Juan"})
	if _, err := saleUC.VoidSale(ctx, sale.ID, sellerID, time.UTC); err != nil {
		t.Fatalf("VoidSale: %v", err)
	}

	debt, err := persistence.NewSQLiteDebtRepository(db).GetBySaleID(ctx, sale.ID)
	if err != nil {
		t.Fatalf("GetBySaleID: %v", err)
	}
	if debt.Status != entities.DebtStatusVoided {
		t.Errorf("Status = %v, want voided", debt.Status)
	}
}

func TestVoidSale_ConFiadoYaCobrado_Falla(t *testing.T) {
	db := newTestDB(t)
	saleUC, product, efectivoID, sellerID := seedPOS(t, db, 10)
	fiado := newMethod(t, db, "Fiado", entities.PaymentTypeCredit)
	ctx := context.Background()

	sale := fiarVenta(t, ctx, saleUC, product.ID, fiado, sellerID, today(t), &entities.DebtorInfo{Name: "Juan"})
	debtRepo := persistence.NewSQLiteDebtRepository(db)
	debt, _ := debtRepo.GetBySaleID(ctx, sale.ID)

	debtUC := newDebtUC(db)
	if _, err := debtUC.CollectDebt(ctx, debt.ID, usecases.CollectDebtInput{PaymentMethodID: efectivoID, CollectedByID: sellerID}, time.UTC); err != nil {
		t.Fatalf("CollectDebt: %v", err)
	}

	_, err := saleUC.VoidSale(ctx, sale.ID, sellerID, time.UTC)
	if !errors.Is(err, apperrors.ErrDebtAlreadyCollected) {
		t.Errorf("err = %v, want ErrDebtAlreadyCollected", err)
	}
}

func TestAmendPayments_RechazaVentaConFiado(t *testing.T) {
	db := newTestDB(t)
	saleUC, product, _, sellerID := seedPOS(t, db, 10)
	fiado := newMethod(t, db, "Fiado", entities.PaymentTypeCredit)
	transferencia := newMethod(t, db, "Transferencia", entities.PaymentTypeTransfer)
	ctx := context.Background()

	sale := fiarVenta(t, ctx, saleUC, product.ID, fiado, sellerID, today(t), &entities.DebtorInfo{Name: "Juan"})

	paymentUC := usecases.NewPaymentUseCase(
		persistence.NewSQLitePaymentLineRepository(db),
		persistence.NewSQLitePaymentMethodRepository(db),
		persistence.NewSQLiteChangeLogRepository(db),
		persistence.NewSQLiteSaleRepository(db),
		persistence.NewSQLiteSubscriptionRepository(db),
		persistence.NewSQLiteExpenseRepository(db),
		persistence.NewSQLiteDebtRepository(db),
		persistence.NewUnitOfWork(db),
	)

	_, err := paymentUC.AmendPayments(ctx, usecases.AmendPaymentsInput{
		Kind:        entities.PayableSale,
		PayableID:   sale.ID,
		Lines:       []entities.PaymentLine{{PaymentMethodID: transferencia, Amount: 2000}},
		Reason:      "cambio de método",
		ChangedByID: sellerID,
	})
	if !errors.Is(err, apperrors.ErrCannotAmendCreditSale) {
		t.Errorf("err = %v, want ErrCannotAmendCreditSale", err)
	}
}

func TestDebtsSummary_AgrupaPorDeudorVinculado(t *testing.T) {
	db := newTestDB(t)
	saleUC, product, _, sellerID := seedPOS(t, db, 10)
	fiado := newMethod(t, db, "Fiado", entities.PaymentTypeCredit)
	ctx := context.Background()

	member := &entities.User{
		ID: uuid.New(), Email: "socio2@test.local", FirstName: "Pedro", LastName: "López",
		Role: entities.RoleMember, Status: entities.UserStatusActive,
	}
	if err := persistence.NewSQLiteUserRepository(db).Create(member); err != nil {
		t.Fatalf("creando socio: %v", err)
	}

	// Dos fiados distintos, mismo socio vinculado: deben sumar en una sola fila.
	fiarVenta(t, ctx, saleUC, product.ID, fiado, sellerID, today(t), &entities.DebtorInfo{UserID: &member.ID})
	fiarVenta(t, ctx, saleUC, product.ID, fiado, sellerID, today(t), &entities.DebtorInfo{UserID: &member.ID})

	summaries, err := persistence.NewSQLiteDebtRepository(db).SummaryByDebtor(ctx, uuid.Nil)
	if err != nil {
		t.Fatalf("SummaryByDebtor: %v", err)
	}
	if len(summaries) != 1 {
		t.Fatalf("summaries = %d filas, want 1 (un solo deudor)", len(summaries))
	}
	if summaries[0].TotalOwed != 4000 {
		t.Errorf("TotalOwed = %v, want 4000", summaries[0].TotalOwed)
	}
	if summaries[0].DebtCount != 2 {
		t.Errorf("DebtCount = %d, want 2", summaries[0].DebtCount)
	}
	if summaries[0].DebtorUserID == nil || *summaries[0].DebtorUserID != member.ID {
		t.Errorf("DebtorUserID = %v, want %v", summaries[0].DebtorUserID, member.ID)
	}
}

func TestDebtsSummary_AgrupaPorNombreLibreCuandoNoHaySocio(t *testing.T) {
	db := newTestDB(t)
	saleUC, product, _, sellerID := seedPOS(t, db, 10)
	fiado := newMethod(t, db, "Fiado", entities.PaymentTypeCredit)
	ctx := context.Background()

	// Mismo nombre, con diferencias de mayúsculas/espacios que un cajero
	// tecleando rápido produciría — deben agruparse igual.
	fiarVenta(t, ctx, saleUC, product.ID, fiado, sellerID, today(t), &entities.DebtorInfo{Name: "Pedro Gómez"})
	fiarVenta(t, ctx, saleUC, product.ID, fiado, sellerID, today(t), &entities.DebtorInfo{Name: " pedro gómez "})

	summaries, err := persistence.NewSQLiteDebtRepository(db).SummaryByDebtor(ctx, uuid.Nil)
	if err != nil {
		t.Fatalf("SummaryByDebtor: %v", err)
	}
	if len(summaries) != 1 {
		t.Fatalf("summaries = %d filas, want 1 (mismo nombre normalizado)", len(summaries))
	}
	if summaries[0].TotalOwed != 4000 {
		t.Errorf("TotalOwed = %v, want 4000", summaries[0].TotalOwed)
	}
	if summaries[0].DebtorUserID != nil {
		t.Errorf("DebtorUserID = %v, want nil (nombre libre)", summaries[0].DebtorUserID)
	}
}

// TestDebtUseCase_GetByIDYList_IncluyenLosProductosDeLaVenta prueba que
// "quién debe" no se quede en un monto: el mostrador necesita saber QUÉ se
// llevó cada deudor para poder confirmarlo al cobrar. Cubre también la
// regresión de un bug real: GetByID llamaba attachExtras sobre una copia
// ([]entities.Debt{*debt}) y nunca copiaba el resultado de vuelta al puntero
// que devolvía, así que PaymentMethod/SaleItems se perdían en silencio.
func TestDebtUseCase_GetByIDYList_IncluyenLosProductosDeLaVenta(t *testing.T) {
	db := newTestDB(t)
	saleUC, product, _, sellerID := seedPOS(t, db, 10)
	fiado := newMethod(t, db, "Fiado", entities.PaymentTypeCredit)
	ctx := context.Background()

	sale := fiarVenta(t, ctx, saleUC, product.ID, fiado, sellerID, today(t), &entities.DebtorInfo{Name: "Juan"})
	debt, err := persistence.NewSQLiteDebtRepository(db).GetBySaleID(ctx, sale.ID)
	if err != nil {
		t.Fatalf("GetBySaleID: %v", err)
	}

	debtUC := newDebtUC(db)

	got, err := debtUC.GetByID(ctx, debt.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if len(got.SaleItems) != 1 {
		t.Fatalf("GetByID: SaleItems = %d, want 1", len(got.SaleItems))
	}
	if got.SaleItems[0].Product == nil || got.SaleItems[0].Product.Name != product.Name {
		t.Errorf("GetByID: producto = %+v, want %q", got.SaleItems[0].Product, product.Name)
	}
	if got.SaleItems[0].Quantity != 1 {
		t.Errorf("GetByID: Quantity = %d, want 1", got.SaleItems[0].Quantity)
	}

	list, err := debtUC.List(ctx, uuid.Nil, nil, "", nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 1 || len(list[0].SaleItems) != 1 {
		t.Fatalf("List: esperaba 1 deuda con 1 producto, got %+v", list)
	}
	if list[0].SaleItems[0].Product == nil || list[0].SaleItems[0].Product.Name != product.Name {
		t.Errorf("List: producto = %+v, want %q", list[0].SaleItems[0].Product, product.Name)
	}
}

// TestCollectDebt_AparaceEnElCuadreDelDiaDeCobroNoDelDiaDeLaVenta es el test
// central del módulo: el fiado se vende un día y se cobra otro, y cada día
// debe cuadrar con lo que de verdad pasó ESE día — ni el desglose del día de
// la venta debe mostrar el método real de cobro, ni el del día del cobro debe
// mostrar Fiado.
func TestCollectDebt_AparaceEnElCuadreDelDiaDeCobroNoDelDiaDeLaVenta(t *testing.T) {
	db := newTestDB(t)
	saleUC, product, efectivoID, sellerID := seedPOS(t, db, 10)
	fiado := newMethod(t, db, "Fiado", entities.PaymentTypeCredit)
	ctx := context.Background()

	// Bien en el pasado, para no coincidir nunca con "hoy" (el día real en que
	// corre el test, que es cuando CollectDebt fecha su línea).
	saleDate := "2020-01-15"
	sale := fiarVenta(t, ctx, saleUC, product.ID, fiado, sellerID, saleDate, &entities.DebtorInfo{Name: "Juan"})

	debtRepo := persistence.NewSQLiteDebtRepository(db)
	debt, _ := debtRepo.GetBySaleID(ctx, sale.ID)

	debtUC := newDebtUC(db)
	if _, err := debtUC.CollectDebt(ctx, debt.ID, usecases.CollectDebtInput{PaymentMethodID: efectivoID, CollectedByID: sellerID}, time.UTC); err != nil {
		t.Fatalf("CollectDebt: %v", err)
	}

	paymentLineRepo := persistence.NewSQLitePaymentLineRepository(db)
	paymentMethodRepo := persistence.NewSQLitePaymentMethodRepository(db)
	paymentUC := usecases.NewPaymentUseCase(
		paymentLineRepo, paymentMethodRepo,
		persistence.NewSQLiteChangeLogRepository(db),
		persistence.NewSQLiteSaleRepository(db),
		persistence.NewSQLiteSubscriptionRepository(db),
		persistence.NewSQLiteExpenseRepository(db),
		debtRepo,
		persistence.NewUnitOfWork(db),
	)

	saleDaySummary, err := paymentUC.Summary(ctx, saleDate, saleDate)
	if err != nil {
		t.Fatalf("Summary(día de la venta): %v", err)
	}
	assertSumEqualsTotal(t, saleDaySummary)
	if saleDaySummary.Total != 2000 {
		t.Errorf("día de la venta: Total = %v, want 2000", saleDaySummary.Total)
	}
	for _, m := range saleDaySummary.Methods {
		if m.Type == "cash" {
			t.Errorf("día de la venta: no debería aparecer nada en Efectivo todavía, encontré %v", m)
		}
	}

	todayStr := time.Now().UTC().Format("2006-01-02")
	todaySummary, err := paymentUC.Summary(ctx, todayStr, todayStr)
	if err != nil {
		t.Fatalf("Summary(día del cobro): %v", err)
	}
	assertSumEqualsTotal(t, todaySummary)
	if todaySummary.Total != 2000 {
		t.Errorf("día del cobro: Total = %v, want 2000", todaySummary.Total)
	}
	for _, m := range todaySummary.Methods {
		if m.Type == "credit" {
			t.Errorf("día del cobro: Fiado no debería aparecer, el dinero entró en efectivo, encontré %v", m)
		}
	}
}

func assertSumEqualsTotal(t *testing.T, summary usecases.PaymentSummary) {
	t.Helper()
	var sum float64
	for _, m := range summary.Methods {
		sum += m.Total
	}
	if sum != summary.Total {
		t.Errorf("sum(métodos) = %v, Total = %v — deben coincidir siempre", sum, summary.Total)
	}
}
