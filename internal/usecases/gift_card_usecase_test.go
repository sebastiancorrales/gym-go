package usecases_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sebastiancorrales/gym-go/internal/domain/entities"
	"github.com/sebastiancorrales/gym-go/internal/domain/repositories"
	"github.com/sebastiancorrales/gym-go/internal/infrastructure/persistence"
	"github.com/sebastiancorrales/gym-go/internal/infrastructure/persistence/migrations"
	"github.com/sebastiancorrales/gym-go/internal/usecases"
	apperrors "github.com/sebastiancorrales/gym-go/pkg/errors"
	"gorm.io/gorm"
)

type giftFixture struct {
	db     *gorm.DB
	uc     *usecases.GiftCardUseCase
	gymID  uuid.UUID
	plan   *entities.Plan
	member *entities.User
	cash   uuid.UUID
	card   uuid.UUID
}

func newGiftFixture(t *testing.T) *giftFixture {
	t.Helper()
	db := newTestDB(t)
	gymID := uuid.New()

	plan := entities.NewPlan(gymID, "1 mes de entrenamiento", 30, 70000)
	plan.GiftEnabled = true
	if err := persistence.NewSQLitePlanRepository(db).Create(plan); err != nil {
		t.Fatalf("creando plan: %v", err)
	}
	member := &entities.User{
		ID: uuid.New(), GymID: gymID, Email: "regalo@test.local", FirstName: "Ana", LastName: "Ríos",
		Role: entities.RoleMember, Status: entities.UserStatusActive,
	}
	if err := persistence.NewSQLiteUserRepository(db).Create(member); err != nil {
		t.Fatalf("creando socio: %v", err)
	}

	uc := usecases.NewGiftCardUseCase(
		persistence.NewSQLiteGiftCardRepository(db),
		persistence.NewSQLitePlanRepository(db),
		persistence.NewSQLiteGymRepository(db),
		persistence.NewSQLiteUserRepository(db),
		persistence.NewSQLiteSubscriptionRepository(db),
		persistence.NewSQLitePaymentMethodRepository(db),
		persistence.NewSQLitePaymentLineRepository(db),
		persistence.NewUnitOfWork(db),
		nil,
	)
	return &giftFixture{
		db: db, uc: uc, gymID: gymID, plan: plan, member: member,
		cash: newMethod(t, db, "Caja", entities.PaymentTypeCash),
		card: newMethod(t, db, "Datáfono", entities.PaymentTypeCard),
	}
}

func (f *giftFixture) sell(t *testing.T, lines ...entities.PaymentLine) *entities.GiftCard {
	t.Helper()
	if len(lines) == 0 {
		lines = []entities.PaymentLine{{PaymentMethodID: f.cash, Amount: 70000}}
	}
	card, err := f.uc.Sell(context.Background(), f.gymID, usecases.SellGiftCardInput{
		PlanID: f.plan.ID, RecipientName: "Ana Ríos", RecipientPhone: "300 123 4567",
		BuyerName: "Luis", Payment: usecases.PaymentInput{Lines: lines},
	}, time.UTC)
	if err != nil {
		t.Fatalf("Sell: %v", err)
	}
	return card
}

func (f *giftFixture) summaryToday(t *testing.T) usecases.PaymentSummary {
	t.Helper()
	rows, err := persistence.NewSQLitePaymentLineRepository(f.db).SumByMethod(context.Background(), today(t), today(t))
	if err != nil {
		t.Fatalf("SumByMethod: %v", err)
	}
	return usecases.AggregatePaymentSummary(rows, today(t), today(t))
}

// El dinero entra el día de la venta, en el cuadre, como tarjeta regalo.
func TestGiftCard_Venta_EntraAlCuadreDelDia(t *testing.T) {
	f := newGiftFixture(t)
	card := f.sell(t,
		entities.PaymentLine{PaymentMethodID: f.cash, Amount: 50000, CashReceived: 50000},
		entities.PaymentLine{PaymentMethodID: f.card, Amount: 20000},
	)

	if card.Value != 70000 || card.Status != entities.GiftCardPending {
		t.Fatalf("tarjeta = %+v", card)
	}
	if card.RecipientPhone != "3001234567" {
		t.Errorf("RecipientPhone = %q, se guardan sólo dígitos", card.RecipientPhone)
	}
	wantExpiry := time.Now().UTC().AddDate(0, 0, entities.DefaultGiftCardValidityDays).Format("2006-01-02")
	if card.ExpiresOn != wantExpiry {
		t.Errorf("ExpiresOn = %s, want %s (60 días)", card.ExpiresOn, wantExpiry)
	}

	s := f.summaryToday(t)
	var gift float64
	var sumMethods float64
	for _, m := range s.Methods {
		gift += m.GiftCardsTotal
		sumMethods += m.Total
	}
	if gift != 70000 || s.Total != 70000 {
		t.Errorf("GiftCardsTotal = %v, Total = %v, want 70000", gift, s.Total)
	}
	if sumMethods != s.Total {
		t.Errorf("sum(methods) = %v ≠ total %v", sumMethods, s.Total)
	}
}

func TestGiftCard_Venta_PlanNoRegalableSeRechaza(t *testing.T) {
	f := newGiftFixture(t)
	f.plan.GiftEnabled = false
	if err := persistence.NewSQLitePlanRepository(f.db).Update(f.plan); err != nil {
		t.Fatal(err)
	}
	_, err := f.uc.Sell(context.Background(), f.gymID, usecases.SellGiftCardInput{
		PlanID: f.plan.ID, RecipientName: "Ana", BuyerName: "Luis",
		Payment: usecases.PaymentInput{Lines: []entities.PaymentLine{{PaymentMethodID: f.cash, Amount: 70000}}},
	}, time.UTC)
	if !errors.Is(err, apperrors.ErrPlanNotGiftable) {
		t.Errorf("err = %v, want ErrPlanNotGiftable", err)
	}
}

func TestGiftCard_Venta_NoSePuedeFiar(t *testing.T) {
	f := newGiftFixture(t)
	fiado := newMethod(t, f.db, "Fiado test", entities.PaymentTypeCredit)
	_, err := f.uc.Sell(context.Background(), f.gymID, usecases.SellGiftCardInput{
		PlanID: f.plan.ID, RecipientName: "Ana", BuyerName: "Luis",
		Payment: usecases.PaymentInput{Lines: []entities.PaymentLine{{PaymentMethodID: fiado, Amount: 70000}}},
	}, time.UTC)
	if !errors.Is(err, apperrors.ErrCannotPayGiftCardWithCredit) {
		t.Errorf("err = %v, want ErrCannotPayGiftCardWithCredit", err)
	}
}

func TestGiftCard_Venta_ElValorLoFijaElPlanNoElCliente(t *testing.T) {
	f := newGiftFixture(t)
	_, err := f.uc.Sell(context.Background(), f.gymID, usecases.SellGiftCardInput{
		PlanID: f.plan.ID, RecipientName: "Ana", BuyerName: "Luis",
		Payment: usecases.PaymentInput{Lines: []entities.PaymentLine{{PaymentMethodID: f.cash, Amount: 1000}}},
	}, time.UTC)
	if !errors.Is(err, apperrors.ErrPaymentLinesMismatch) {
		t.Errorf("err = %v, want ErrPaymentLinesMismatch", err)
	}
}

// Redimir crea la suscripción sin cobro: el dinero ya está en el día de la venta.
func TestGiftCard_Redimir_CreaSuscripcionSinCobro(t *testing.T) {
	f := newGiftFixture(t)
	card := f.sell(t)

	redeemed, sub, err := f.uc.Redeem(context.Background(), card.ID,
		usecases.RedeemGiftCardInput{UserID: f.member.ID}, time.UTC)
	if err != nil {
		t.Fatalf("Redeem: %v", err)
	}
	if redeemed.Status != entities.GiftCardRedeemed || *redeemed.RedeemedSubscriptionID != sub.ID {
		t.Errorf("tarjeta = %+v", redeemed)
	}
	if sub.TotalPaid != 0 || sub.EnrollmentFeePaid != 0 {
		t.Errorf("TotalPaid = %v, matrícula = %v: el regalo ya se cobró al venderlo", sub.TotalPaid, sub.EnrollmentFeePaid)
	}
	if sub.GiftCardID == nil || *sub.GiftCardID != card.ID {
		t.Errorf("GiftCardID = %v, want %v", sub.GiftCardID, card.ID)
	}
	if sub.Status != entities.SubscriptionStatusActive {
		t.Errorf("Status = %v, want ACTIVE", sub.Status)
	}

	lines, _ := persistence.NewSQLitePaymentLineRepository(f.db).
		GetByPayable(context.Background(), entities.PayableSubscription, sub.ID)
	if len(lines) != 0 {
		t.Errorf("la suscripción regalada tiene %d líneas de pago, want 0", len(lines))
	}
	// El cuadre del día sigue mostrando sólo la venta de la tarjeta.
	if s := f.summaryToday(t); s.Total != 70000 {
		t.Errorf("Total del día = %v, want 70000 (redimir no mueve dinero)", s.Total)
	}

	// El backfill de pagos corre en cada arranque: no debe inventarle una
	// línea a la suscripción regalada.
	if err := migrations.Migrate(f.db); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	lines, _ = persistence.NewSQLitePaymentLineRepository(f.db).
		GetByPayable(context.Background(), entities.PayableSubscription, sub.ID)
	if len(lines) != 0 {
		t.Errorf("tras reiniciar, el backfill creó %d líneas para la suscripción regalada", len(lines))
	}
}

// Con una suscripción vigente, la regalada arranca cuando termina la actual.
func TestGiftCard_Redimir_EncadenaTrasLaSuscripcionVigente(t *testing.T) {
	f := newGiftFixture(t)
	currentEnd := time.Now().AddDate(0, 0, 12).UTC().Round(time.Second)
	current := entities.NewSubscription(f.member.ID, f.plan.ID, f.gymID, time.Now().AddDate(0, 0, -18), 30, "30_DAYS", 70000, 0, 0)
	current.EndDate = currentEnd
	current.Activate()
	if err := persistence.NewSQLiteSubscriptionRepository(f.db).Create(current); err != nil {
		t.Fatal(err)
	}

	card := f.sell(t)
	_, sub, err := f.uc.Redeem(context.Background(), card.ID, usecases.RedeemGiftCardInput{UserID: f.member.ID}, time.UTC)
	if err != nil {
		t.Fatalf("Redeem: %v", err)
	}
	if !sub.StartDate.Equal(currentEnd) {
		t.Errorf("StartDate = %v, want %v (fin de la suscripción vigente)", sub.StartDate, currentEnd)
	}
}

// Dos recepciones redimen el mismo código a la vez: sólo una puede ganar.
func TestGiftCard_Redimir_DobleRedencionConcurrente(t *testing.T) {
	f := newGiftFixture(t)
	card := f.sell(t)

	other := &entities.User{
		ID: uuid.New(), GymID: f.gymID, Email: "otro@test.local", FirstName: "Otro", LastName: "Socio",
		Role: entities.RoleMember, Status: entities.UserStatusActive,
	}
	if err := persistence.NewSQLiteUserRepository(f.db).Create(other); err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	errs := make([]error, 2)
	for i, uid := range []uuid.UUID{f.member.ID, other.ID} {
		wg.Add(1)
		go func(i int, uid uuid.UUID) {
			defer wg.Done()
			_, _, errs[i] = f.uc.Redeem(context.Background(), card.ID, usecases.RedeemGiftCardInput{UserID: uid}, time.UTC)
		}(i, uid)
	}
	wg.Wait()

	ok, conflicts := 0, 0
	for _, err := range errs {
		switch {
		case err == nil:
			ok++
		case errors.Is(err, apperrors.ErrGiftCardNotPending):
			conflicts++
		default:
			t.Errorf("error inesperado: %v", err)
		}
	}
	if ok != 1 || conflicts != 1 {
		t.Errorf("éxitos = %d, conflictos = %d; want 1 y 1", ok, conflicts)
	}

	var subs int64
	f.db.Model(&entities.Subscription{}).Where("gift_card_id = ?", card.ID).Count(&subs)
	if subs != 1 {
		t.Errorf("se crearon %d suscripciones con la misma tarjeta, want 1", subs)
	}
}

func TestGiftCard_Redimir_VencidaSoloConAutorizacion(t *testing.T) {
	f := newGiftFixture(t)
	card := f.sell(t)
	yesterday := time.Now().UTC().AddDate(0, 0, -1).Format("2006-01-02")
	f.db.Model(&entities.GiftCard{}).Where("id = ?", card.ID).Update("expires_on", yesterday)

	_, _, err := f.uc.Redeem(context.Background(), card.ID, usecases.RedeemGiftCardInput{UserID: f.member.ID}, time.UTC)
	if !errors.Is(err, apperrors.ErrGiftCardExpired) {
		t.Fatalf("err = %v, want ErrGiftCardExpired", err)
	}
	if _, _, err := f.uc.Redeem(context.Background(), card.ID,
		usecases.RedeemGiftCardInput{UserID: f.member.ID, AllowExpired: true}, time.UTC); err != nil {
		t.Errorf("con autorización debería redimirse: %v", err)
	}
}

// Anular devuelve el dinero con líneas espejo de hoy; el día neto queda en 0.
func TestGiftCard_Anular_EspejoNegativoYNoTrasRedimir(t *testing.T) {
	f := newGiftFixture(t)
	card := f.sell(t)

	if _, err := f.uc.Void(context.Background(), card.ID, "se arrepintió", uuid.New(), time.UTC); err != nil {
		t.Fatalf("Void: %v", err)
	}
	if s := f.summaryToday(t); s.Total != 0 {
		t.Errorf("Total del día tras anular = %v, want 0", s.Total)
	}
	if _, _, err := f.uc.Redeem(context.Background(), card.ID, usecases.RedeemGiftCardInput{UserID: f.member.ID}, time.UTC); !errors.Is(err, apperrors.ErrGiftCardNotPending) {
		t.Errorf("redimir una anulada: err = %v, want ErrGiftCardNotPending", err)
	}

	redeemed := f.sell(t)
	if _, _, err := f.uc.Redeem(context.Background(), redeemed.ID, usecases.RedeemGiftCardInput{UserID: f.member.ID}, time.UTC); err != nil {
		t.Fatal(err)
	}
	if _, err := f.uc.Void(context.Background(), redeemed.ID, "", uuid.New(), time.UTC); !errors.Is(err, apperrors.ErrGiftCardNotPending) {
		t.Errorf("anular una redimida: err = %v, want ErrGiftCardNotPending", err)
	}
}

func TestGiftCard_BuscarPorCodigo_ConOSinPrefijo(t *testing.T) {
	f := newGiftFixture(t)
	card := f.sell(t)
	suffix := card.Code[len(card.Code)-6:]

	for _, raw := range []string{card.Code, " " + card.CodeKey + " ", suffix, "reg-" + suffix} {
		got, err := f.uc.FindByCode(context.Background(), f.gymID, raw)
		if err != nil || got.ID != card.ID {
			t.Errorf("FindByCode(%q) = %v, %v", raw, got, err)
		}
	}
	if _, err := f.uc.FindByCode(context.Background(), uuid.New(), card.Code); !errors.Is(err, apperrors.ErrNotFound) {
		t.Errorf("otro gimnasio no debe encontrarla: err = %v", err)
	}
}

func TestGiftCard_SaldoPorRedimir(t *testing.T) {
	f := newGiftFixture(t)
	f.sell(t)
	expired := f.sell(t)
	redeemed := f.sell(t)
	voided := f.sell(t)
	f.db.Model(&entities.GiftCard{}).Where("id = ?", expired.ID).
		Update("expires_on", time.Now().UTC().AddDate(0, 0, -1).Format("2006-01-02"))
	if _, _, err := f.uc.Redeem(context.Background(), redeemed.ID, usecases.RedeemGiftCardInput{UserID: f.member.ID}, time.UTC); err != nil {
		t.Fatal(err)
	}
	if _, err := f.uc.Void(context.Background(), voided.ID, "", uuid.New(), time.UTC); err != nil {
		t.Fatal(err)
	}

	out, err := f.uc.Outstanding(context.Background(), f.gymID, time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	if out.Count != 1 || out.Value != 70000 || out.ExpiredCount != 1 || out.ExpiredValue != 70000 {
		t.Errorf("saldo = %+v; want 1 vigente y 1 vencida de 70000", out)
	}

	list, err := f.uc.List(context.Background(), f.gymID, repositories.GiftCardFilter{StartDate: today(t), EndDate: today(t), State: "pending"}, time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Cards) != 1 {
		t.Errorf("filtro pending devolvió %d tarjetas, want 1 (sin la vencida)", len(list.Cards))
	}
	if list.SoldCount != 3 || list.Voided != 1 || list.Redeemed != 1 {
		t.Errorf("cifras = vendidas %d, anuladas %d, redimidas %d; want 3, 1, 1 (no dependen del filtro)",
			list.SoldCount, list.Voided, list.Redeemed)
	}
}
