package usecases

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sebastiancorrales/gym-go/internal/domain/entities"
	"github.com/sebastiancorrales/gym-go/internal/domain/repositories"
	"github.com/sebastiancorrales/gym-go/pkg/errors"
	"github.com/sebastiancorrales/gym-go/pkg/money"
)

// GiftCardUseCase orquesta las tarjetas regalo: venderlas, redimirlas (lo que
// crea la suscripción) y anularlas.
//
// La regla contable que atraviesa todo el módulo: el dinero entra el día de la
// VENTA, con sus propias payment_lines (PayableGiftCard). Redimir no mueve
// dinero — crea una suscripción con TotalPaid = 0 enlazada a la tarjeta. Así el
// cuadre del día de la venta cuadra, y el reporte de suscripciones no cuenta el
// mismo dinero una segunda vez.
type GiftCardUseCase struct {
	repo              repositories.GiftCardRepository
	planRepo          repositories.PlanRepository
	gymRepo           repositories.GymRepository
	userRepo          repositories.UserRepository
	subscriptionRepo  repositories.SubscriptionRepository
	paymentMethodRepo repositories.PaymentMethodRepository
	paymentLineRepo   repositories.PaymentLineRepository
	uow               repositories.UnitOfWork
	// subscriptions se usa sólo para notifyAccessChanged: quien redime una
	// tarjeta camina a la puerta en segundos, igual que quien acaba de pagar.
	subscriptions *SubscriptionUseCase
}

func NewGiftCardUseCase(
	repo repositories.GiftCardRepository,
	planRepo repositories.PlanRepository,
	gymRepo repositories.GymRepository,
	userRepo repositories.UserRepository,
	subscriptionRepo repositories.SubscriptionRepository,
	paymentMethodRepo repositories.PaymentMethodRepository,
	paymentLineRepo repositories.PaymentLineRepository,
	uow repositories.UnitOfWork,
	subscriptions *SubscriptionUseCase,
) *GiftCardUseCase {
	return &GiftCardUseCase{
		repo:              repo,
		planRepo:          planRepo,
		gymRepo:           gymRepo,
		userRepo:          userRepo,
		subscriptionRepo:  subscriptionRepo,
		paymentMethodRepo: paymentMethodRepo,
		paymentLineRepo:   paymentLineRepo,
		uow:               uow,
		subscriptions:     subscriptions,
	}
}

// GiftCardConfigView es lo que necesita la pantalla para vender y dibujar la
// tarjeta: los datos de contacto del gimnasio, la configuración y los planes
// que se pueden regalar. Existe aparte de GET /gym porque ese endpoint pide
// PermGymSettings, que recepción no tiene.
type GiftCardConfigView struct {
	GymName   string                    `json:"gym_name"`
	LogoURL   string                    `json:"logo_url"`
	Phone     string                    `json:"phone"`
	Address   string                    `json:"address"`
	Instagram string                    `json:"instagram"`
	Settings  entities.GiftCardSettings `json:"settings"`
	Plans     []*entities.Plan          `json:"plans"`
}

func (uc *GiftCardUseCase) loadGym(gymID uuid.UUID) *entities.Gym {
	gym, err := uc.gymRepo.FindByID(gymID)
	if err != nil {
		return nil
	}
	return gym
}

// Config devuelve la configuración efectiva y los planes regalables activos.
func (uc *GiftCardUseCase) Config(gymID uuid.UUID) (*GiftCardConfigView, error) {
	gym := uc.loadGym(gymID)
	view := &GiftCardConfigView{Settings: entities.GiftCardConfig(gym), Plans: []*entities.Plan{}}
	if gym != nil {
		view.GymName, view.LogoURL, view.Phone = gym.Name, gym.LogoURL, gym.Phone
		view.Address, view.Instagram = gym.Address, gym.Instagram
	}

	plans, err := uc.planRepo.FindActiveByGymID(gymID)
	if err != nil {
		return nil, err
	}
	for _, p := range plans {
		if p.GiftEnabled && p.Price > 0 {
			view.Plans = append(view.Plans, p)
		}
	}
	return view, nil
}

// SellGiftCardInput es lo que pide vender una tarjeta.
type SellGiftCardInput struct {
	PlanID         uuid.UUID
	RecipientName  string
	RecipientPhone string
	BuyerName      string
	BuyerPhone     string
	Dedication     string
	Payment        PaymentInput
	CreatedByID    uuid.UUID
}

// Sell registra una tarjeta ya pagada: la tarjeta y sus líneas de pago entran
// en la misma transacción, fechadas hoy en la zona del gimnasio.
func (uc *GiftCardUseCase) Sell(ctx context.Context, gymID uuid.UUID, in SellGiftCardInput, loc *time.Location) (*entities.GiftCard, error) {
	recipient := truncate(strings.TrimSpace(in.RecipientName), 40)
	buyer := truncate(strings.TrimSpace(in.BuyerName), 40)
	if recipient == "" || buyer == "" {
		return nil, errors.ErrGiftCardRecipientRequired
	}

	plan, err := uc.planRepo.FindByID(in.PlanID)
	if err != nil {
		return nil, err
	}
	if plan == nil {
		return nil, errors.ErrNotFound
	}
	if !plan.IsActive() || !plan.GiftEnabled || (gymID != uuid.Nil && plan.GymID != gymID) {
		return nil, errors.ErrPlanNotGiftable
	}
	if plan.Price <= 0 {
		return nil, errors.ErrInvalidPrice
	}

	cfg := entities.GiftCardConfig(uc.loadGym(gymID))
	code, key, err := uc.uniqueCode(ctx, cfg.CodePrefix)
	if err != nil {
		return nil, err
	}

	if loc == nil {
		loc = time.UTC
	}
	now := time.Now().UTC().Round(0)
	localNow := now.In(loc)
	date := localNow.Format("2006-01-02")

	card := &entities.GiftCard{
		ID:             uuid.New(),
		GymID:          gymID,
		Code:           code,
		CodeKey:        key,
		PlanID:         plan.ID,
		PlanName:       plan.Name,
		Value:          money.Round(plan.Price),
		RecipientName:  recipient,
		RecipientPhone: digitsOnly(in.RecipientPhone, 15),
		BuyerName:      buyer,
		BuyerPhone:     digitsOnly(in.BuyerPhone, 15),
		Dedication:     truncate(strings.TrimSpace(in.Dedication), 90),
		Status:         entities.GiftCardPending,
		Date:           date,
		Hour:           localNow.Format("15:04"),
		ExpiresOn:      localNow.AddDate(0, 0, cfg.ValidityDays).Format("2006-01-02"),
		CreatedByID:    in.CreatedByID,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	resolver := paymentResolver{methodRepo: uc.paymentMethodRepo}
	lines, _, err := resolver.resolve(ctx, in.Payment, card.Value,
		entities.PayableGiftCard, card.ID, gymID, card.Date, card.Hour, now)
	if err != nil {
		return nil, err
	}
	// Se paga de contado: una línea de fiado abriría una Debt, y Debt cuelga
	// de una venta (SaleID), no de una tarjeta.
	for _, l := range lines {
		if l.PaymentMethod != nil && l.PaymentMethod.Type.IsCredit() {
			return nil, errors.ErrCannotPayGiftCardWithCredit
		}
	}
	card.PaymentMethodID = entities.PrimaryPaymentMethod(lines)

	if err := uc.uow.Do(ctx, func(r repositories.Repos) error {
		if err := r.GiftCards.Create(ctx, card); err != nil {
			return err
		}
		// Copia por intento: uow.Do reintenta ante SQLITE_BUSY.
		attempt := make([]entities.PaymentLine, len(lines))
		copy(attempt, lines)
		return r.PaymentLines.CreateBatch(ctx, attempt)
	}); err != nil {
		return nil, err
	}

	card.PaymentLines = lines
	return card, nil
}

// uniqueCode genera códigos hasta dar con uno libre. Con ~887 millones de
// combinaciones el reintento casi nunca ocurre; el índice único sobre CodeKey
// es la garantía de verdad.
func (uc *GiftCardUseCase) uniqueCode(ctx context.Context, prefix string) (string, string, error) {
	for i := 0; i < 20; i++ {
		code, key, err := entities.NewGiftCardCode(prefix)
		if err != nil {
			return "", "", err
		}
		exists, err := uc.repo.CodeKeyExists(ctx, key)
		if err != nil {
			return "", "", err
		}
		if !exists {
			return code, key, nil
		}
	}
	return "", "", fmt.Errorf("no se pudo generar un código de tarjeta libre")
}

// FindByCode busca una tarjeta por lo que se haya tecleado o escaneado. Acepta
// el código con o sin prefijo: si "7K3QXM" no existe tal cual, prueba con el
// prefijo actual del gimnasio delante.
func (uc *GiftCardUseCase) FindByCode(ctx context.Context, gymID uuid.UUID, raw string) (*entities.GiftCard, error) {
	key := entities.GiftCardCodeKey(raw)
	if key == "" {
		return nil, errors.ErrNotFound
	}
	card, err := uc.repo.GetByCodeKey(ctx, key)
	if err != nil {
		return nil, err
	}
	if card == nil {
		prefix := entities.GiftCardConfig(uc.loadGym(gymID)).CodePrefix
		if !strings.HasPrefix(key, prefix) {
			card, err = uc.repo.GetByCodeKey(ctx, prefix+key)
			if err != nil {
				return nil, err
			}
		}
	}
	if card == nil || (gymID != uuid.Nil && card.GymID != gymID) {
		return nil, errors.ErrNotFound
	}
	return uc.withDetails(ctx, card)
}

// GetByID devuelve una tarjeta con sus líneas de pago y quién la redimió.
func (uc *GiftCardUseCase) GetByID(ctx context.Context, id uuid.UUID) (*entities.GiftCard, error) {
	card, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if card == nil {
		return nil, errors.ErrNotFound
	}
	return uc.withDetails(ctx, card)
}

func (uc *GiftCardUseCase) withDetails(ctx context.Context, card *entities.GiftCard) (*entities.GiftCard, error) {
	lines, err := uc.paymentLineRepo.GetByPayable(ctx, entities.PayableGiftCard, card.ID)
	if err != nil {
		return nil, err
	}
	uc.attachMethods(ctx, lines)
	card.PaymentLines = lines
	if card.RedeemedUserID != nil {
		if u, err := uc.userRepo.FindByID(*card.RedeemedUserID); err == nil {
			card.RedeemedUser = u
		}
	}
	return card, nil
}

func (uc *GiftCardUseCase) attachMethods(ctx context.Context, lines []entities.PaymentLine) {
	cache := map[uuid.UUID]*entities.SalePaymentMethod{}
	for i := range lines {
		id := lines[i].PaymentMethodID
		if _, ok := cache[id]; !ok {
			m, err := uc.paymentMethodRepo.GetByID(ctx, id)
			if err == nil {
				cache[id] = m
			}
		}
		lines[i].PaymentMethod = cache[id]
	}
}

// GiftCardList es el listado más sus cifras: las del rango (vendidas, valor,
// redimidas) y el saldo por redimir de HOY, que no depende del rango.
type GiftCardList struct {
	Cards       []entities.GiftCard              `json:"cards"`
	SoldCount   int                              `json:"sold_count"`
	SoldValue   float64                          `json:"sold_value"`
	Redeemed    int                              `json:"redeemed_count"`
	Voided      int                              `json:"voided_count"`
	Outstanding repositories.GiftCardOutstanding `json:"outstanding"`
}

// List devuelve las tarjetas vendidas en el rango con sus cifras.
func (uc *GiftCardUseCase) List(ctx context.Context, gymID uuid.UUID, filter repositories.GiftCardFilter, loc *time.Location) (*GiftCardList, error) {
	filter.Today = todayIn(loc)

	cards, err := uc.repo.List(ctx, gymID, filter)
	if err != nil {
		return nil, err
	}
	out := &GiftCardList{Cards: cards}

	// Las cifras del rango salen de TODAS las tarjetas del rango, no de las
	// que deja pasar el filtro de estado o el buscador: "vendidas este mes" no
	// debe cambiar porque alguien escribió un nombre en la caja de búsqueda.
	all := cards
	if filter.State != "" || strings.TrimSpace(filter.Query) != "" {
		all, err = uc.repo.List(ctx, gymID, repositories.GiftCardFilter{
			StartDate: filter.StartDate, EndDate: filter.EndDate, Today: filter.Today,
		})
		if err != nil {
			return nil, err
		}
	}
	for _, c := range all {
		switch c.Status {
		case entities.GiftCardVoided:
			out.Voided++
			continue
		case entities.GiftCardRedeemed:
			out.Redeemed++
		}
		out.SoldCount++
		out.SoldValue += c.Value
	}

	out.Outstanding, err = uc.repo.Outstanding(ctx, gymID, filter.Today)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Outstanding es el saldo por redimir de hoy (para Reportes Contables).
func (uc *GiftCardUseCase) Outstanding(ctx context.Context, gymID uuid.UUID, loc *time.Location) (repositories.GiftCardOutstanding, error) {
	return uc.repo.Outstanding(ctx, gymID, todayIn(loc))
}

// RedeemGiftCardInput es lo que pide redimir una tarjeta: a quién se le crea la
// suscripción y, si el plan es grupal, quiénes la acompañan.
type RedeemGiftCardInput struct {
	UserID              uuid.UUID
	AdditionalMemberIDs []uuid.UUID
	// AllowExpired permite redimir una tarjeta vencida. El handler sólo lo deja
	// pasar a quien tiene PermGiftCardsManage.
	AllowExpired bool
	RedeemedByID uuid.UUID
}

// Redeem marca la tarjeta como usada y crea la suscripción, en una sola
// transacción.
//
// La suscripción va SIN cobro: TotalPaid = 0, sin payment_lines y sin
// matrícula — el dinero ya entró el día de la venta. Si la persona ya tiene una
// suscripción vigente, la nueva arranca cuando termine la última (misma regla
// que RenewSubscription), para no regalarle días que ya había pagado.
func (uc *GiftCardUseCase) Redeem(ctx context.Context, id uuid.UUID, in RedeemGiftCardInput, loc *time.Location) (*entities.GiftCard, *entities.Subscription, error) {
	card, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	if card == nil {
		return nil, nil, errors.ErrNotFound
	}
	if card.Status != entities.GiftCardPending {
		return nil, nil, errors.ErrGiftCardNotPending
	}
	if card.IsExpiredOn(todayIn(loc)) && !in.AllowExpired {
		return nil, nil, errors.ErrGiftCardExpired
	}

	user, err := uc.userRepo.FindByID(in.UserID)
	if err != nil {
		return nil, nil, err
	}
	if user == nil {
		return nil, nil, errors.ErrNotFound
	}

	// El plan puede estar desactivado desde la venta: la tarjeta se pagó y se
	// honra igual. Lo que no puede es haber desaparecido.
	plan, err := uc.planRepo.FindByID(card.PlanID)
	if err != nil {
		return nil, nil, err
	}
	if plan == nil {
		return nil, nil, errors.ErrNotFound
	}

	required := plan.MaxMembers - 1
	if plan.MaxMembers > 1 && len(in.AdditionalMemberIDs) != required {
		return nil, nil, fmt.Errorf("%w: el plan '%s' requiere %d persona(s) adicional(es) y se recibieron %d",
			errors.ErrInvalidInput, plan.Name, required, len(in.AdditionalMemberIDs))
	}
	for _, mid := range in.AdditionalMemberIDs {
		if mid == in.UserID {
			return nil, nil, fmt.Errorf("%w: el titular no puede ser incluido como miembro adicional", errors.ErrInvalidInput)
		}
	}

	if loc == nil {
		loc = time.UTC
	}
	start := uc.chainedStart(in.UserID)
	sub := entities.NewSubscription(
		in.UserID, plan.ID, card.GymID,
		start, plan.DurationDays, string(plan.BillingMode),
		0, 0, 0,
	)
	localNow := time.Now().In(loc)
	sub.Date = localNow.Format("2006-01-02")
	sub.Hour = localNow.Format("15:04")
	sub.GiftCardID = &card.ID
	sub.PaymentMethod = "Tarjeta regalo"
	sub.Notes = fmt.Sprintf("Tarjeta regalo %s · regalo de %s · valor %s",
		card.Code, card.BuyerName, formatPesos(card.Value))
	sub.Activate()

	members := buildGroupMembers(sub.ID, in.UserID, in.AdditionalMemberIDs)
	now := time.Now().UTC().Round(0)

	if err := uc.uow.Do(ctx, func(r repositories.Repos) error {
		// Primero la guarda: si otra recepción ya la redimió, no se crea nada.
		if err := r.GiftCards.MarkRedeemed(ctx, card.ID, now, in.RedeemedByID, in.UserID, sub.ID); err != nil {
			return err
		}
		if err := r.Subscriptions.Create(sub); err != nil {
			return err
		}
		for _, m := range members {
			if err := r.Members.Create(m); err != nil {
				return fmt.Errorf("registrando miembro %s del grupo: %w", m.UserID, err)
			}
		}
		return nil
	}); err != nil {
		return nil, nil, err
	}

	if uc.subscriptions != nil {
		uc.subscriptions.notifyAccessChanged(sub)
	}

	card.Status = entities.GiftCardRedeemed
	card.RedeemedAt = &now
	card.RedeemedByID = &in.RedeemedByID
	card.RedeemedUserID = &in.UserID
	card.RedeemedSubscriptionID = &sub.ID
	card.RedeemedUser = user
	return card, sub, nil
}

// chainedStart es el inicio de la suscripción regalada: el fin de la última
// suscripción no cancelada de la persona, o ahora si ya pasó.
func (uc *GiftCardUseCase) chainedStart(userID uuid.UUID) time.Time {
	now := time.Now()
	latest := now
	subs, err := uc.subscriptionRepo.FindByUserID(userID)
	if err != nil {
		return now
	}
	for _, s := range subs {
		if s.Status == entities.SubscriptionStatusCancelled {
			continue
		}
		if s.EndDate.After(latest) {
			latest = s.EndDate
		}
	}
	return latest
}

// Void anula una tarjeta pendiente (vencida o no) y devuelve su dinero: escribe
// líneas espejo negativas fechadas HOY, igual que VoidSale y UncollectDebt —
// nunca se toca un cierre ya enviado. Una tarjeta redimida no se anula: la
// suscripción ya existe.
func (uc *GiftCardUseCase) Void(ctx context.Context, id uuid.UUID, reason string, voidedByID uuid.UUID, loc *time.Location) (*entities.GiftCard, error) {
	card, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if card == nil {
		return nil, errors.ErrNotFound
	}
	if card.Status != entities.GiftCardPending {
		return nil, errors.ErrGiftCardNotPending
	}

	original, err := uc.paymentLineRepo.GetByPayable(ctx, entities.PayableGiftCard, card.ID)
	if err != nil {
		return nil, err
	}

	if loc == nil {
		loc = time.UTC
	}
	now := time.Now().UTC().Round(0)
	localNow := now.In(loc)

	mirrors := make([]entities.PaymentLine, 0, len(original))
	for _, l := range original {
		if l.ReversalOf != nil {
			continue
		}
		reversalOf := l.ID
		mirrors = append(mirrors, entities.PaymentLine{
			ID:              uuid.New(),
			PayableKind:     entities.PayableGiftCard,
			PayableID:       card.ID,
			PaymentMethodID: l.PaymentMethodID,
			Amount:          -l.Amount,
			GymID:           card.GymID,
			Date:            localNow.Format("2006-01-02"),
			Hour:            localNow.Format("15:04"),
			ReversalOf:      &reversalOf,
			CreatedAt:       now,
			UpdatedAt:       now,
		})
	}

	reason = truncate(strings.TrimSpace(reason), 200)
	if err := uc.uow.Do(ctx, func(r repositories.Repos) error {
		if err := r.GiftCards.MarkVoided(ctx, card.ID, now, voidedByID, reason); err != nil {
			return err
		}
		if len(mirrors) == 0 {
			return nil
		}
		attempt := make([]entities.PaymentLine, len(mirrors))
		copy(attempt, mirrors)
		return r.PaymentLines.CreateBatch(ctx, attempt)
	}); err != nil {
		return nil, err
	}

	card.Status = entities.GiftCardVoided
	card.VoidedAt = &now
	card.VoidedByID = &voidedByID
	card.VoidReason = reason
	return card, nil
}

func todayIn(loc *time.Location) string {
	if loc == nil {
		loc = time.UTC
	}
	return time.Now().In(loc).Format("2006-01-02")
}

func truncate(s string, max int) string {
	r := []rune(s)
	if len(r) > max {
		return string(r[:max])
	}
	return s
}

func digitsOnly(s string, max int) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return truncate(b.String(), max)
}

// formatPesos da "$70.000": separador de miles con punto, sin decimales.
func formatPesos(v float64) string {
	n := int64(money.Round(v))
	neg := n < 0
	if neg {
		n = -n
	}
	s := fmt.Sprintf("%d", n)
	var b strings.Builder
	for i, r := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(r)
	}
	if neg {
		return "-$" + b.String()
	}
	return "$" + b.String()
}
