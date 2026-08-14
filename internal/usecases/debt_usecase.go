package usecases

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/sebastiancorrales/gym-go/internal/domain/entities"
	"github.com/sebastiancorrales/gym-go/internal/domain/repositories"
	"github.com/sebastiancorrales/gym-go/pkg/errors"
	"github.com/sebastiancorrales/gym-go/pkg/money"
)

// DebtUseCase orquesta el ciclo de vida de un fiado: cobrarlo, revertir un
// cobro mal hecho, vincular un deudor de nombre libre a un socio real, y las
// dos consultas de "quién debe" (agrupada y en detalle).
//
// La apertura de la deuda NO vive aquí: ocurre dentro de SaleUseCase.CreateSale,
// en la misma transacción que la venta que la origina — abrir la deuda es
// parte de vender fiado, no una operación aparte.
type DebtUseCase struct {
	debtRepo          repositories.DebtRepository
	paymentMethodRepo repositories.PaymentMethodRepository
	paymentLineRepo   repositories.PaymentLineRepository
	userRepo          repositories.UserRepository
	// saleDetailRepo/productRepo sólo respaldan SaleItems: qué se llevó cada
	// fiado, para que el mostrador sepa qué está cobrando y no sólo un monto.
	saleDetailRepo repositories.SaleDetailRepository
	productRepo    repositories.ProductRepository
	uow            repositories.UnitOfWork
}

func NewDebtUseCase(
	debtRepo repositories.DebtRepository,
	paymentMethodRepo repositories.PaymentMethodRepository,
	paymentLineRepo repositories.PaymentLineRepository,
	userRepo repositories.UserRepository,
	saleDetailRepo repositories.SaleDetailRepository,
	productRepo repositories.ProductRepository,
	uow repositories.UnitOfWork,
) *DebtUseCase {
	return &DebtUseCase{
		debtRepo:          debtRepo,
		paymentMethodRepo: paymentMethodRepo,
		paymentLineRepo:   paymentLineRepo,
		userRepo:          userRepo,
		saleDetailRepo:    saleDetailRepo,
		productRepo:       productRepo,
		uow:               uow,
	}
}

// CollectDebtInput es lo que pide cobrar una deuda: con qué método real (nunca
// otro fiado) y, si es efectivo, cuánto entregó el deudor.
type CollectDebtInput struct {
	PaymentMethodID uuid.UUID
	CashReceived    float64
	CollectedByID   uuid.UUID
}

// CollectDebt cobra una deuda pendiente por su monto completo — no hay abonos
// parciales, se cobra entera de una vez.
//
// La línea de pago que registra el dinero se fecha HOY (en la zona horaria del
// gimnasio), no el día en que se fio: es lo que hace que el ingreso aparezca en
// el cuadre de caja del día en que de verdad entró, no en el de la venta
// original — cada cierre diario ya enviado por correo se queda intacto.
func (uc *DebtUseCase) CollectDebt(ctx context.Context, debtID uuid.UUID, in CollectDebtInput, loc *time.Location) (*entities.Debt, error) {
	debt, err := uc.debtRepo.GetByID(ctx, debtID)
	if err != nil {
		return nil, err
	}
	if debt == nil {
		return nil, errors.ErrNotFound
	}
	if !debt.CanBeCollected() {
		return nil, errors.ErrDebtNotPending
	}

	method, err := uc.paymentMethodRepo.GetByID(ctx, in.PaymentMethodID)
	if err != nil {
		return nil, err
	}
	if method == nil {
		return nil, errors.ErrNotFound
	}
	if !method.IsActive() {
		return nil, errors.ErrPaymentMethodNotActive
	}
	// No se puede cobrar un fiado con otro fiado: eso sólo movería la deuda de
	// un lado a otro sin que entrara nada a caja.
	if method.Type.IsCredit() {
		return nil, errors.ErrCannotPayDebtWithCredit
	}

	if loc == nil {
		loc = time.UTC
	}
	now := time.Now().UTC().Round(0)
	localNow := now.In(loc)

	line := entities.PaymentLine{
		ID:              uuid.New(),
		PayableKind:     entities.PayableDebt,
		PayableID:       debt.ID,
		PaymentMethodID: method.ID,
		Amount:          money.Round(debt.Amount),
		CashReceived:    money.Round(in.CashReceived),
		GymID:           debt.GymID,
		Date:            localNow.Format("2006-01-02"),
		Hour:            localNow.Format("15:04"),
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	// Reusa las mismas reglas que una venta (N=1, trivialmente válidas): el
	// vuelto lo calcula siempre el servidor, nunca el importe que venga del
	// cliente.
	lines := []entities.PaymentLine{line}
	if err := entities.ValidatePaymentLines(lines, debt.Amount); err != nil {
		return nil, err
	}
	if err := entities.ValidateCashChange(lines, map[uuid.UUID]*entities.SalePaymentMethod{method.ID: method}); err != nil {
		return nil, err
	}
	entities.ApplyChangeGiven(lines)
	line = lines[0]

	debt.Status = entities.DebtStatusCollected
	debt.CollectedAt = &now
	debt.CollectedByID = &in.CollectedByID
	debt.CollectedPaymentMethodID = &method.ID
	debt.CollectedPaymentLineID = &line.ID
	debt.UpdatedAt = now

	if err := uc.uow.Do(ctx, func(r repositories.Repos) error {
		if err := r.PaymentLines.CreateBatch(ctx, []entities.PaymentLine{line}); err != nil {
			return err
		}
		return r.Debts.Update(ctx, debt)
	}); err != nil {
		return nil, err
	}

	debt.PaymentMethod = method
	return debt, nil
}

// UncollectDebt revierte un cobro mal hecho: la deuda vuelve a pendiente y se
// escribe una línea espejo negativa (mismo principio que VoidSale) fechada
// HOY, no el día del cobro original — nunca se toca un cierre ya cerrado.
func (uc *DebtUseCase) UncollectDebt(ctx context.Context, debtID uuid.UUID, loc *time.Location) (*entities.Debt, error) {
	debt, err := uc.debtRepo.GetByID(ctx, debtID)
	if err != nil {
		return nil, err
	}
	if debt == nil {
		return nil, errors.ErrNotFound
	}
	if debt.Status != entities.DebtStatusCollected {
		return nil, errors.ErrDebtNotCollected
	}

	if loc == nil {
		loc = time.UTC
	}
	now := time.Now().UTC().Round(0)
	localNow := now.In(loc)

	reversalOf := *debt.CollectedPaymentLineID
	mirror := entities.PaymentLine{
		ID:              uuid.New(),
		PayableKind:     entities.PayableDebt,
		PayableID:       debt.ID,
		PaymentMethodID: *debt.CollectedPaymentMethodID,
		Amount:          -debt.Amount,
		GymID:           debt.GymID,
		Date:            localNow.Format("2006-01-02"),
		Hour:            localNow.Format("15:04"),
		ReversalOf:      &reversalOf,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	debt.Status = entities.DebtStatusPending
	debt.CollectedAt = nil
	debt.CollectedByID = nil
	debt.CollectedPaymentMethodID = nil
	debt.CollectedPaymentLineID = nil
	debt.UpdatedAt = now

	if err := uc.uow.Do(ctx, func(r repositories.Repos) error {
		if err := r.PaymentLines.CreateBatch(ctx, []entities.PaymentLine{mirror}); err != nil {
			return err
		}
		return r.Debts.Update(ctx, debt)
	}); err != nil {
		return nil, err
	}
	return debt, nil
}

// LinkDebtor vincula una deuda de nombre libre a un socio real. No mueve
// dinero, así que no necesita transacción ni cambia el estado de la deuda.
func (uc *DebtUseCase) LinkDebtor(ctx context.Context, debtID uuid.UUID, userID uuid.UUID) (*entities.Debt, error) {
	debt, err := uc.debtRepo.GetByID(ctx, debtID)
	if err != nil {
		return nil, err
	}
	if debt == nil {
		return nil, errors.ErrNotFound
	}

	user, err := uc.userRepo.FindByID(userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.ErrNotFound
	}

	debt.DebtorUserID = &userID
	debt.UpdatedAt = time.Now().UTC().Round(0)
	if err := uc.debtRepo.Update(ctx, debt); err != nil {
		return nil, err
	}
	return debt, nil
}

// GetByID devuelve una deuda con su método de cobro resuelto (si ya se cobró).
func (uc *DebtUseCase) GetByID(ctx context.Context, id uuid.UUID) (*entities.Debt, error) {
	debt, err := uc.debtRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if debt == nil {
		return nil, errors.ErrNotFound
	}
	// attachExtras muta el slice que recibe; sin copiar el resultado de vuelta
	// a *debt, lo que resolviera (PaymentMethod, SaleItems) se quedaría en la
	// copia temporal y la respuesta saldría sin nada de eso.
	wrapped := []entities.Debt{*debt}
	uc.attachExtras(ctx, wrapped)
	*debt = wrapped[0]
	return debt, nil
}

// List devuelve las deudas individuales de un deudor concreto (o de todos,
// filtrando sólo por estado) — es el drill-down tras GET /debts/summary.
func (uc *DebtUseCase) List(ctx context.Context, gymID uuid.UUID, debtorUserID *uuid.UUID, debtorName string, status *entities.DebtStatus) ([]entities.Debt, error) {
	debts, err := uc.debtRepo.List(ctx, gymID, debtorUserID, debtorName, status)
	if err != nil {
		return nil, err
	}
	uc.attachExtras(ctx, debts)
	return debts, nil
}

// Summary agrupa las deudas pendientes por deudor — la consulta que de verdad
// motiva este módulo: "quién debe y cuánto".
func (uc *DebtUseCase) Summary(ctx context.Context, gymID uuid.UUID) ([]repositories.DebtorSummary, error) {
	return uc.debtRepo.SummaryByDebtor(ctx, gymID)
}

// attachExtras resuelve el método de cobro de cada deuda ya cobrada y los
// productos de la venta que la originó, memorizando por id — el mismo patrón
// que SaleUseCase.attachMethodsToLines/loadPaymentLines.
func (uc *DebtUseCase) attachExtras(ctx context.Context, debts []entities.Debt) {
	methodCache := make(map[uuid.UUID]*entities.SalePaymentMethod)
	for i := range debts {
		id := debts[i].CollectedPaymentMethodID
		if id == nil {
			continue
		}
		if _, ok := methodCache[*id]; !ok {
			method, err := uc.paymentMethodRepo.GetByID(ctx, *id)
			if err == nil {
				methodCache[*id] = method
			}
		}
		debts[i].PaymentMethod = methodCache[*id]
	}

	// Qué se llevó cada fiado: sin esto, "quién debe" sólo dice un monto y
	// nadie del mostrador sabe qué confirmar con el cliente al cobrar.
	saleIDs := make([]uuid.UUID, len(debts))
	for i := range debts {
		saleIDs[i] = debts[i].SaleID
	}
	detailsBySale, err := uc.saleDetailRepo.GetBySaleIDs(ctx, saleIDs)
	if err != nil {
		return // la lista de deudas sigue siendo útil sin el detalle de productos
	}

	productCache := make(map[uuid.UUID]*entities.Product)
	for i := range debts {
		details := detailsBySale[debts[i].SaleID]
		for j := range details {
			pid := details[j].ProductID
			if _, ok := productCache[pid]; !ok {
				product, err := uc.productRepo.GetByID(ctx, pid)
				if err == nil {
					productCache[pid] = product
				}
			}
			details[j].Product = productCache[pid]
		}
		debts[i].SaleItems = details
	}
}
