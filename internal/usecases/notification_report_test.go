package usecases

// Test interno (mismo paquete, no usecases_test) porque ejercita
// buildDailyCloseReport, que es intencionalmente privado: nadie fuera del
// propio caso de uso debería poder construir un reporte a medias sin pasar
// por SendDailyClose.

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sebastiancorrales/gym-go/internal/config"
	"github.com/sebastiancorrales/gym-go/internal/domain/entities"
	"github.com/sebastiancorrales/gym-go/internal/infrastructure/persistence"
	"github.com/sebastiancorrales/gym-go/internal/infrastructure/persistence/migrations"
)

// TestBuildDailyCloseReport_SumaMetodosGastosYNeto es el aserto que motiva la
// fase 7: un día con venta + anulación + suscripción + gasto en efectivo debe
// (a) tener el desglose por método unificado (SumByMethod, sin reconciliar
// por etiqueta) y (b) descontar el gasto del ingreso bruto en NetRevenue.
func TestBuildDailyCloseReport_SumaMetodosGastosYNeto(t *testing.T) {
	ctx := context.Background()

	database, err := config.NewDatabase(&config.DatabaseConfig{
		DatabasePath: filepath.Join(t.TempDir(), "notif.db"),
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
	if err := migrations.Seed(database.DB); err != nil {
		t.Fatalf("Seed: %v", err)
	}
	db := database.DB

	gymRepo := persistence.NewSQLiteGymRepository(db)
	gym := &entities.Gym{ID: uuid.New(), Name: "Gimnasio de Prueba", Country: "CO", Currency: "COP", Status: "active"}
	if err := gymRepo.Create(gym); err != nil {
		t.Fatalf("creando gimnasio: %v", err)
	}

	productRepo := persistence.NewSQLiteProductRepository(db)
	product := &entities.Product{ID: uuid.New(), Name: "Proteina", UnitPrice: 45000, Stock: 10, Status: entities.ProductStatusActive}
	if err := productRepo.Create(ctx, product); err != nil {
		t.Fatalf("creando producto: %v", err)
	}

	paymentMethodRepo := persistence.NewSQLitePaymentMethodRepository(db)
	efectivo := &entities.SalePaymentMethod{ID: uuid.New(), Name: "Efectivo", Type: entities.PaymentTypeCash, Status: entities.PaymentMethodStatusActive}
	transferencia := &entities.SalePaymentMethod{ID: uuid.New(), Name: "Transferencia", Type: entities.PaymentTypeTransfer, Status: entities.PaymentMethodStatusActive}
	if err := paymentMethodRepo.Create(ctx, efectivo); err != nil {
		t.Fatalf("creando método efectivo: %v", err)
	}
	if err := paymentMethodRepo.Create(ctx, transferencia); err != nil {
		t.Fatalf("creando método transferencia: %v", err)
	}

	saleRepo := persistence.NewSQLiteSaleRepository(db)
	saleDetailRepo := persistence.NewSQLiteSaleDetailRepository(db)
	uow := persistence.NewUnitOfWork(db)
	saleUC := NewSaleUseCase(saleRepo, saleDetailRepo, productRepo, paymentMethodRepo,
		persistence.NewSQLitePaymentLineRepository(db),
		persistence.NewSQLiteUserRepository(db), persistence.NewSQLiteDebtRepository(db), uow)

	sellerID := uuid.New()
	today := time.Now().UTC().Format("2006-01-02")

	// Venta en transferencia (45.000, sin anular).
	sale1 := &entities.Sale{
		UserID: sellerID, PaymentMethodID: transferencia.ID, Date: today, Hour: "09:00",
		Details: []entities.SaleDetail{{ProductID: product.ID, UnitPrice: 45000, Quantity: 1}},
	}
	if err := saleUC.CreateSale(ctx, sale1); err != nil {
		t.Fatalf("CreateSale sale1: %v", err)
	}

	// Venta en efectivo (100.000) que luego se anula: debe netear a cero en el
	// desglose y no ensuciar el total.
	sale2 := &entities.Sale{
		UserID: sellerID, PaymentMethodID: efectivo.ID, Date: today, Hour: "10:00",
		Details: []entities.SaleDetail{{ProductID: product.ID, UnitPrice: 50000, Quantity: 2}},
	}
	if err := saleUC.CreateSale(ctx, sale2); err != nil {
		t.Fatalf("CreateSale sale2: %v", err)
	}
	if _, err := saleUC.VoidSale(ctx, sale2.ID, sellerID, time.UTC); err != nil {
		t.Fatalf("VoidSale: %v", err)
	}

	// Suscripción pagada en efectivo (80.000), sembrada directo como lo haría
	// CreateSubscription (con su línea de pago).
	planRepo := persistence.NewSQLitePlanRepository(db)
	userRepo := persistence.NewSQLiteUserRepository(db)
	subscriptionRepo := persistence.NewSQLiteSubscriptionRepository(db)
	lineRepo := persistence.NewSQLitePaymentLineRepository(db)

	plan := &entities.Plan{ID: uuid.New(), GymID: gym.ID, Name: "Mensual", Price: 80000, DurationDays: 30, Status: "ACTIVE"}
	if err := planRepo.Create(plan); err != nil {
		t.Fatalf("creando plan: %v", err)
	}
	member := &entities.User{ID: uuid.New(), GymID: gym.ID, Email: "member@test.com", FirstName: "Juan", LastName: "Perez", Role: entities.RoleMember, Status: entities.UserStatusActive}
	if err := userRepo.Create(member); err != nil {
		t.Fatalf("creando socio: %v", err)
	}
	sub := &entities.Subscription{
		ID: uuid.New(), UserID: member.ID, PlanID: plan.ID, GymID: gym.ID,
		StartDate: time.Now(), EndDate: time.Now().AddDate(0, 1, 0),
		PricePaid: 80000, TotalPaid: 80000, PaymentMethod: "Efectivo",
		Status: "ACTIVE", Date: today, Hour: "11:00",
	}
	if err := subscriptionRepo.Create(sub); err != nil {
		t.Fatalf("creando suscripción: %v", err)
	}
	if err := lineRepo.CreateBatch(ctx, []entities.PaymentLine{
		{PaymentMethodID: efectivo.ID, Amount: 80000, PayableKind: entities.PayableSubscription, PayableID: sub.ID, Date: today},
	}); err != nil {
		t.Fatalf("sembrando línea de suscripción: %v", err)
	}

	// Gasto operativo en efectivo (15.000): debe descontarse del ingreso bruto
	// en NetRevenue, sin aparecer en el desglose por método (ese sólo cubre
	// ingresos).
	expenseRepo := persistence.NewSQLiteExpenseRepository(db)
	var category entities.ExpenseCategory
	if err := db.Where("kind = ?", "operating").First(&category).Error; err != nil {
		t.Fatalf("buscando categoría sembrada: %v", err)
	}
	expense := &entities.Expense{
		ID: uuid.New(), GymID: gym.ID, CategoryID: category.ID, PaymentMethodID: efectivo.ID,
		Description: "Agua", Subtotal: 15000, Total: 15000,
		Status: entities.ExpenseStatusRegistered, UserID: sellerID, Date: today, Hour: "12:00",
	}
	if err := expenseRepo.Create(ctx, expense); err != nil {
		t.Fatalf("creando gasto: %v", err)
	}

	// Construir el use case de notificaciones con los mismos repos, y llamar
	// directamente a buildDailyCloseReport (sin pasar por SMTP).
	notifUC := NewNotificationUseCase(
		persistence.NewSQLiteNotificationRecipientRepository(db),
		saleRepo, saleDetailRepo, subscriptionRepo, planRepo, gymRepo, userRepo,
		paymentMethodRepo, lineRepo, expenseRepo, productRepo, nil,
	)

	startDate, _ := time.Parse("2006-01-02", today)
	report, err := notifUC.buildDailyCloseReport(gym.ID, gym, startDate, startDate, time.UTC)
	if err != nil {
		t.Fatalf("buildDailyCloseReport: %v", err)
	}

	var sumMethods float64
	for _, m := range report.PaymentMethods {
		sumMethods += m.Total
	}
	if sumMethods != report.TotalRevenue {
		t.Errorf("sum(métodos) = %v, TotalRevenue = %v — deben coincidir siempre", sumMethods, report.TotalRevenue)
	}
	if report.TotalRevenue != 125000 {
		t.Errorf("TotalRevenue = %v, want 125000 (45000 transferencia + 80000 efectivo; la venta anulada neta a cero)", report.TotalRevenue)
	}
	if report.TotalExpensesAmount != 15000 {
		t.Errorf("TotalExpensesAmount = %v, want 15000", report.TotalExpensesAmount)
	}
	if report.TotalExpensesCount != 1 {
		t.Errorf("TotalExpensesCount = %v, want 1", report.TotalExpensesCount)
	}
	wantNet := 125000.0 - 15000.0
	if report.NetRevenue != wantNet {
		t.Errorf("NetRevenue = %v, want %v (ingresos brutos - egresos)", report.NetRevenue, wantNet)
	}
}
