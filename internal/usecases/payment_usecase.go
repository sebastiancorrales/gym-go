package usecases

import (
	"context"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/sebastiancorrales/gym-go/internal/domain/entities"
	"github.com/sebastiancorrales/gym-go/internal/domain/repositories"
	"github.com/sebastiancorrales/gym-go/pkg/errors"
	"github.com/sebastiancorrales/gym-go/pkg/money"
)

// PaymentUseCase corrige el cobro de un documento ya registrado.
//
// Es un use case propio y no un método de SaleUseCase porque el flujo es
// idéntico para venta, suscripción y gasto: resolver el total autoritativo,
// validar, reemplazar las líneas y dejar constancia. Tenerlo en un solo sitio es
// exactamente lo que impide que las tres copias se desincronicen — el fallo que
// todo este trabajo está arreglando.
type PaymentUseCase struct {
	paymentLineRepo   repositories.PaymentLineRepository
	paymentMethodRepo repositories.PaymentMethodRepository
	changeLogRepo     repositories.ChangeLogRepository
	saleRepo          repositories.SaleRepository
	subscriptionRepo  repositories.SubscriptionRepository
	expenseRepo       repositories.ExpenseRepository
	// debtRepo sólo se usa para la guarda de AmendPayments sobre una venta
	// fiada: corregir el pago por ahí desincronizaría payment_lines y debts.
	debtRepo repositories.DebtRepository
	uow      repositories.UnitOfWork
}

func NewPaymentUseCase(
	paymentLineRepo repositories.PaymentLineRepository,
	paymentMethodRepo repositories.PaymentMethodRepository,
	changeLogRepo repositories.ChangeLogRepository,
	saleRepo repositories.SaleRepository,
	subscriptionRepo repositories.SubscriptionRepository,
	expenseRepo repositories.ExpenseRepository,
	debtRepo repositories.DebtRepository,
	uow repositories.UnitOfWork,
) *PaymentUseCase {
	return &PaymentUseCase{
		paymentLineRepo:   paymentLineRepo,
		paymentMethodRepo: paymentMethodRepo,
		changeLogRepo:     changeLogRepo,
		saleRepo:          saleRepo,
		subscriptionRepo:  subscriptionRepo,
		expenseRepo:       expenseRepo,
		debtRepo:          debtRepo,
		uow:               uow,
	}
}

// AmendPaymentsInput describe una corrección.
//
// El total NO viene de fuera: se lee del documento. Dejar que el cliente
// reafirme cuánto costó algo convertiría una corrección de método en una forma
// de cambiar el importe sin dejar rastro.
type AmendPaymentsInput struct {
	Kind          entities.PayableKind
	PayableID     uuid.UUID
	Lines         []entities.PaymentLine // sin ID ni fecha: los rellena el use case
	Reason        string
	ChangedByID   uuid.UUID
	ChangedByName string
	GymID         uuid.UUID
}

// payableDoc es lo que el use case necesita de un documento, sea del tipo que sea.
type payableDoc struct {
	total float64
	date  string
	hour  string
	gymID uuid.UUID
}

// AmendPayments reemplaza las líneas de pago de un documento y lo registra.
func (uc *PaymentUseCase) AmendPayments(ctx context.Context, in AmendPaymentsInput) ([]entities.PaymentLine, error) {
	if len(in.Lines) == 0 {
		return nil, errors.ErrNoPaymentLines
	}

	// Una venta con un fiado asociado (pendiente, cobrado o anulado) no se
	// corrige por aquí: reescribir sus líneas dejaría payment_lines y debts
	// desincronizados — el fiado seguiría existiendo (o no) sin relación con
	// las líneas reales de la venta. Se corrige desde el módulo de deudas.
	if in.Kind == entities.PayableSale {
		debt, err := uc.debtRepo.GetBySaleID(ctx, in.PayableID)
		if err != nil {
			return nil, err
		}
		if debt != nil {
			return nil, errors.ErrCannotAmendCreditSale
		}
	}

	doc, applyPrimary, err := uc.loadPayable(ctx, in.Kind, in.PayableID)
	if err != nil {
		return nil, err
	}

	current, err := uc.paymentLineRepo.GetByPayable(ctx, in.Kind, in.PayableID)
	if err != nil {
		return nil, err
	}

	// Se exige método activo, SALVO que ya estuviera en las líneas actuales.
	//
	// Sin esa excepción, un admin que sólo corrige una de dos patas sería
	// rechazado porque la otra usa un método que se desactivó después — y se
	// quedaría sin poder arreglar nada.
	alreadyPresent := make(map[uuid.UUID]bool, len(current))
	for _, line := range current {
		alreadyPresent[line.PaymentMethodID] = true
	}

	methods := make(map[uuid.UUID]*entities.SalePaymentMethod, len(in.Lines))
	for _, line := range in.Lines {
		if _, ok := methods[line.PaymentMethodID]; ok {
			continue
		}
		method, err := uc.paymentMethodRepo.GetByID(ctx, line.PaymentMethodID)
		if err != nil {
			return nil, err
		}
		if method == nil {
			return nil, errors.ErrNotFound
		}
		if !method.IsActive() && !alreadyPresent[line.PaymentMethodID] {
			return nil, errors.ErrPaymentMethodNotActive
		}
		methods[line.PaymentMethodID] = method
	}

	if err := entities.ValidatePaymentLines(in.Lines, doc.total); err != nil {
		return nil, err
	}
	if err := entities.ValidateCashChange(in.Lines, methods); err != nil {
		return nil, err
	}
	entities.ApplyChangeGiven(in.Lines)

	// Nombres para la descripción de la bitácora, incluidos los de las líneas
	// actuales (que pueden usar un método ya desactivado).
	names := make(map[uuid.UUID]string, len(methods)+len(current))
	for id, m := range methods {
		names[id] = m.Name
	}
	for _, line := range current {
		if _, ok := names[line.PaymentMethodID]; ok {
			continue
		}
		if m, err := uc.paymentMethodRepo.GetByID(ctx, line.PaymentMethodID); err == nil && m != nil {
			names[line.PaymentMethodID] = m.Name
		}
	}

	oldValue := entities.DescribePaymentLines(current, names)

	now := time.Now().UTC().Round(0)
	newLines := make([]entities.PaymentLine, len(in.Lines))
	for i, line := range in.Lines {
		newLines[i] = entities.PaymentLine{
			ID:              uuid.New(),
			PayableKind:     in.Kind,
			PayableID:       in.PayableID,
			PaymentMethodID: line.PaymentMethodID,
			Amount:          money.Round(line.Amount),
			CashReceived:    money.Round(line.CashReceived),
			ChangeGiven:     money.Round(line.ChangeGiven),
			GymID:           doc.gymID,
			// Se conserva la fecha del documento, no la de hoy: mover un cobro a
			// hoy lo sacaría del cierre diario en el que ya se contó.
			Date:          doc.date,
			Hour:          doc.hour,
			CreatedAt:     now,
			UpdatedAt:     now,
			PaymentMethod: methods[line.PaymentMethodID],
		}
	}
	newValue := entities.DescribePaymentLines(newLines, names)

	// Guarda de no-op: sin esto, cada clic en Guardar ensucia la bitácora con una
	// entrada que no cambió nada.
	if oldValue == newValue {
		return current, nil
	}

	primaryID := entities.PrimaryPaymentMethod(newLines)
	primaryName := ""
	if m := methods[primaryID]; m != nil {
		primaryName = m.Name
	}

	// Si el documento es una venta anulada, hay que reescribir también su espejo.
	// Se localiza AQUÍ, fuera de la transacción, junto con todo lo demás.
	var mirrorLines []entities.PaymentLine
	var mirrorSale *entities.Sale
	if in.Kind == entities.PayableSale {
		mirrorLines, mirrorSale, err = uc.buildMirrorLines(ctx, in.PayableID, newLines, now)
		if err != nil {
			return nil, err
		}
	}

	entry := &entities.ChangeLog{
		ID:            uuid.New(),
		EntityType:    string(in.Kind),
		EntityID:      in.PayableID,
		Field:         entities.ChangeLogFieldPayments,
		OldValue:      oldValue,
		NewValue:      newValue,
		Reason:        in.Reason,
		ChangedByID:   in.ChangedByID,
		ChangedByName: in.ChangedByName,
		GymID:         doc.gymID,
		CreatedAt:     now,
	}

	if err := uc.uow.Do(ctx, func(r repositories.Repos) error {
		if err := r.PaymentLines.ReplaceForPayable(ctx, in.Kind, in.PayableID, newLines); err != nil {
			return err
		}
		if err := applyPrimary(r, primaryID, primaryName, now); err != nil {
			return err
		}
		if mirrorSale != nil {
			if err := r.PaymentLines.ReplaceForPayable(ctx, entities.PayableSale, mirrorSale.ID, mirrorLines); err != nil {
				return err
			}
			// La entidad COMPLETA, mutada sólo en el método principal.
			// SaleRepository.Update usa Save(), que escribe todas las columnas: con
			// un struct parcial borraría voided_sale_id, total, type y date del
			// espejo y lo dejaría huérfano. Es el mismo bug de Save() que ya se
			// cerró en productos.
			mirrorSale.PaymentMethodID = primaryID
			mirrorSale.UpdatedAt = now
			if err := r.Sales.Update(ctx, mirrorSale); err != nil {
				return err
			}
		}
		return r.ChangeLogs.Create(ctx, entry)
	}); err != nil {
		return nil, err
	}

	return newLines, nil
}

// buildMirrorLines construye la negación exacta del conjunto nuevo para la venta
// espejo de una anulación.
//
// Sin esta cascada, corregir "esta venta fue Transferencia, no Efectivo" en una
// venta anulada dejaría el desglose mostrando Transferencia +100.000 y Efectivo
// −100.000: el total del día seguiría en cero, pero aparecería un fantasma en
// efectivo que no corresponde a ningún movimiento real.
//
// La fecha del espejo se CONSERVA: la anulación ocurrió cuando ocurrió, y
// moverla a hoy trasladaría dinero entre dos cierres diarios ya enviados.
func (uc *PaymentUseCase) buildMirrorLines(ctx context.Context, originalSaleID uuid.UUID, newLines []entities.PaymentLine, now time.Time) ([]entities.PaymentLine, *entities.Sale, error) {
	mirror, err := uc.saleRepo.FindVoidBySaleID(ctx, originalSaleID)
	if err != nil {
		return nil, nil, err
	}
	if mirror == nil {
		return nil, nil, nil // la venta no está anulada: nada que cascadear
	}

	existing, err := uc.paymentLineRepo.GetByPayable(ctx, entities.PayableSale, mirror.ID)
	if err != nil {
		return nil, nil, err
	}
	if len(existing) == 0 {
		// Anulación anterior al módulo de pagos, o a medias. La corrección del
		// admin sigue siendo válida; sólo se pierde la cascada.
		log.Printf("⚠️  AmendPayments: la venta anulada %s no tiene líneas espejo; no se cascadea", mirror.ID)
		return nil, nil, nil
	}

	lines := make([]entities.PaymentLine, len(newLines))
	for i, line := range newLines {
		reversalOf := newLines[i].ID
		lines[i] = entities.PaymentLine{
			ID:              uuid.New(),
			PayableKind:     entities.PayableSale,
			PayableID:       mirror.ID,
			PaymentMethodID: line.PaymentMethodID,
			Amount:          -line.Amount,
			GymID:           line.GymID,
			Date:            existing[0].Date, // la fecha de la anulación, no la de hoy
			Hour:            existing[0].Hour,
			ReversalOf:      &reversalOf,
			CreatedAt:       now,
			UpdatedAt:       now,
		}
	}
	return lines, mirror, nil
}

// loadPayable resuelve el total autoritativo del documento y devuelve la función
// que actualiza su método principal denormalizado.
func (uc *PaymentUseCase) loadPayable(ctx context.Context, kind entities.PayableKind, id uuid.UUID) (payableDoc, func(repositories.Repos, uuid.UUID, string, time.Time) error, error) {
	switch kind {
	case entities.PayableSale:
		sale, err := uc.saleRepo.GetByID(ctx, id)
		if err != nil {
			return payableDoc{}, nil, err
		}
		if sale == nil {
			return payableDoc{}, nil, errors.ErrNotFound
		}
		// La contrapartida de una anulación es derivada: se corrige a través de
		// la venta original, y la cascada se encarga del resto. Permitir editarla
		// por separado dejaría las dos mitades diciendo cosas distintas.
		if sale.Type == entities.SaleTypeVoid {
			return payableDoc{}, nil, errors.ErrCannotAmendVoidSale
		}

		doc := payableDoc{total: sale.Total, date: sale.Date, hour: sale.Hour}
		apply := func(r repositories.Repos, primaryID uuid.UUID, _ string, now time.Time) error {
			sale.PaymentMethodID = primaryID
			sale.UpdatedAt = now
			return r.Sales.Update(ctx, sale)
		}
		return doc, apply, nil

	case entities.PayableSubscription:
		sub, err := uc.subscriptionRepo.FindByID(id)
		if err != nil {
			return payableDoc{}, nil, err
		}
		if sub == nil {
			return payableDoc{}, nil, errors.ErrNotFound
		}
		// Una suscripción redimida con tarjeta regalo no tiene pagos: el dinero
		// es de la tarjeta. "Corregirla" crearía líneas por $0 o, peor, por un
		// monto que nunca entró.
		if sub.GiftCardID != nil {
			return payableDoc{}, nil, errors.ErrCannotAmendGiftPayment
		}
		// Se permite aunque esté cancelada: el dinero existió, y corregir con qué
		// se cobró es justo lo que un admin necesita poder hacer.
		doc := payableDoc{total: sub.TotalPaid, date: sub.Date, hour: sub.Hour, gymID: sub.GymID}
		apply := func(r repositories.Repos, primaryID uuid.UUID, primaryName string, now time.Time) error {
			sub.PaymentMethodID = primaryID
			sub.PaymentMethod = primaryName
			sub.UpdatedAt = now
			return r.Subscriptions.Update(sub)
		}
		return doc, apply, nil

	case entities.PayableExpense:
		expense, err := uc.expenseRepo.GetByID(ctx, id)
		if err != nil {
			return payableDoc{}, nil, err
		}
		if expense == nil {
			return payableDoc{}, nil, errors.ErrNotFound
		}
		doc := payableDoc{total: expense.Total, date: expense.Date, hour: expense.Hour, gymID: expense.GymID}
		apply := func(r repositories.Repos, primaryID uuid.UUID, _ string, now time.Time) error {
			// El gasto guarda su método en una columna propia, no en líneas, así
			// que sólo se actualiza el denormalizado.
			expense.PaymentMethodID = primaryID
			expense.UpdatedAt = now
			return r.Expenses.Update(ctx, expense)
		}
		return doc, apply, nil
	}

	return payableDoc{}, nil, errors.ErrInvalidInput
}

// AuditEntry es una línea de la bitácora, ya normalizada para mostrar.
//
// Existe porque hay dos fuentes: SubscriptionAuditLog (sólo fechas, la de antes)
// y ChangeLog (genérica). Devolverlas mezcladas y ordenadas evita que la
// pantalla tenga que saber que son dos tablas.
type AuditEntry struct {
	Description   string    `json:"description"`
	OldValue      string    `json:"old_value,omitempty"`
	NewValue      string    `json:"new_value,omitempty"`
	Reason        string    `json:"reason,omitempty"`
	ChangedByName string    `json:"changed_by_name"`
	CreatedAt     time.Time `json:"created_at"`
}

// AuditFor devuelve la bitácora de correcciones de un documento.
func (uc *PaymentUseCase) AuditFor(ctx context.Context, entityType string, entityID uuid.UUID) ([]AuditEntry, error) {
	entries, err := uc.changeLogRepo.FindByEntity(ctx, entityType, entityID)
	if err != nil {
		return nil, err
	}

	out := make([]AuditEntry, len(entries))
	for i, e := range entries {
		description := "Cambio manual"
		if e.Field == entities.ChangeLogFieldPayments {
			description = "Método de pago corregido"
		}
		out[i] = AuditEntry{
			Description:   description,
			OldValue:      e.OldValue,
			NewValue:      e.NewValue,
			Reason:        e.Reason,
			ChangedByName: e.ChangedByName,
			CreatedAt:     e.CreatedAt,
		}
	}
	return out, nil
}
