package usecases

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/sebastiancorrales/gym-go/internal/domain/entities"
	"github.com/sebastiancorrales/gym-go/internal/domain/repositories"
	"github.com/sebastiancorrales/gym-go/pkg/errors"
)

// ExpenseUseCase orquesta los egresos: gastos generales y compras a proveedor.
type ExpenseUseCase struct {
	expenseRepo       repositories.ExpenseRepository
	expenseLineRepo   repositories.ExpenseLineRepository
	categoryRepo      repositories.ExpenseCategoryRepository
	supplierRepo      repositories.SupplierRepository
	productRepo       repositories.ProductRepository
	paymentMethodRepo repositories.PaymentMethodRepository
	movementRepo      repositories.StockMovementRepository
	uow               repositories.UnitOfWork
}

func NewExpenseUseCase(
	expenseRepo repositories.ExpenseRepository,
	expenseLineRepo repositories.ExpenseLineRepository,
	categoryRepo repositories.ExpenseCategoryRepository,
	supplierRepo repositories.SupplierRepository,
	productRepo repositories.ProductRepository,
	paymentMethodRepo repositories.PaymentMethodRepository,
	movementRepo repositories.StockMovementRepository,
	uow repositories.UnitOfWork,
) *ExpenseUseCase {
	return &ExpenseUseCase{
		expenseRepo:       expenseRepo,
		expenseLineRepo:   expenseLineRepo,
		categoryRepo:      categoryRepo,
		supplierRepo:      supplierRepo,
		productRepo:       productRepo,
		paymentMethodRepo: paymentMethodRepo,
		movementRepo:      movementRepo,
		uow:               uow,
	}
}

// purchaseLine agrega una compra por producto: si el gasto lista el mismo
// producto en dos líneas, se funden en un movimiento con el costo promediado.
// Es la misma razón por la que CreateSale agrega con qtyByProduct.
type purchaseLine struct {
	qty       int
	totalCost float64
}

// CreateExpense registra un egreso y, si lleva líneas de producto, la entrada de
// inventario correspondiente — todo en una transacción.
//
// El gasto general (el arriendo) y la compra de mercancía recorren exactamente
// el mismo camino: si no hay líneas con producto, el bloque de inventario
// simplemente no hace nada. No hay dos flujos que mantener sincronizados.
func (uc *ExpenseUseCase) CreateExpense(ctx context.Context, expense *entities.Expense, loc *time.Location) error {
	// No se exige GymID: el super admin sembrado por las migraciones tiene el
	// gimnasio nulo, y GymTimezoneMiddleware ya contempla ese estado
	// explícitamente. Rechazarlo aquí haría que la persona con más probabilidades
	// de registrar el primer gasto de una instalación nueva no pudiera hacerlo.
	if expense.UserID == uuid.Nil {
		return errors.ErrInvalidInput
	}
	if len(expense.Lines) == 0 {
		return errors.ErrInvalidInput
	}
	if expense.TaxAmount < 0 {
		return errors.ErrInvalidPrice
	}

	// ── Validaciones que hacen I/O, todas ANTES de abrir la transacción ──

	category, err := uc.categoryRepo.GetByID(ctx, expense.CategoryID)
	if err != nil {
		return err
	}
	if category == nil {
		return errors.ErrNotFound
	}
	// Una categoría de otro gimnasio no es visible aquí; las del sistema
	// (gym_id nulo) sí.
	if category.GymID != uuid.Nil && category.GymID != expense.GymID {
		return errors.ErrNotFound
	}

	if expense.SupplierID != nil {
		supplier, err := uc.supplierRepo.GetByID(ctx, *expense.SupplierID)
		if err != nil {
			return err
		}
		if supplier == nil || supplier.GymID != expense.GymID {
			return errors.ErrNotFound
		}
		if !supplier.IsActive() {
			return errors.ErrInvalidInput
		}
	}

	method, err := uc.paymentMethodRepo.GetByID(ctx, expense.PaymentMethodID)
	if err != nil {
		return err
	}
	if method == nil {
		return errors.ErrNotFound
	}
	if !method.IsActive() {
		return errors.ErrPaymentMethodNotActive
	}

	// Líneas: validación y agregación por producto.
	purchases := make(map[uuid.UUID]*purchaseLine)
	for i := range expense.Lines {
		line := &expense.Lines[i]
		if err := line.Validate(); err != nil {
			return err
		}
		if line.ProductID == nil {
			continue
		}

		product, err := uc.productRepo.GetByID(ctx, *line.ProductID)
		if err != nil {
			return err
		}
		if product == nil {
			return errors.ErrNotFound
		}
		if !product.IsActive() {
			return errors.ErrProductNotActive
		}

		qty := int(line.Quantity)
		if agg, ok := purchases[*line.ProductID]; ok {
			agg.qty += qty
			agg.totalCost += float64(qty) * line.UnitCost
		} else {
			purchases[*line.ProductID] = &purchaseLine{qty: qty, totalCost: float64(qty) * line.UnitCost}
		}
	}

	// ── Todo lo generado se decide AQUÍ, fuera de la transacción: uow.Do
	// reintenta ante contención y exige que la función sea idempotente. ──

	expense.ID = uuid.New()
	now := time.Now().UTC().Round(0)
	if loc == nil {
		loc = time.UTC
	}
	local := now.In(loc)

	expense.CalculateTotals() // el total lo decide el servidor, no el cliente
	expense.Status = entities.ExpenseStatusRegistered
	expense.ExpenseDate = now
	expense.Date = local.Format("2006-01-02")
	expense.Hour = local.Format("15:04")
	expense.CreatedAt = now
	expense.UpdatedAt = now

	for i := range expense.Lines {
		expense.Lines[i].ID = uuid.New()
		expense.Lines[i].ExpenseID = expense.ID
		expense.Lines[i].CreatedAt = now
	}

	type purchaseMove struct {
		movement entities.StockMovement
		unitCost float64
	}
	moves := make([]purchaseMove, 0, len(purchases))
	for productID, agg := range purchases {
		expenseID := expense.ID
		unitCost := agg.totalCost / float64(agg.qty) // promedio de esta compra
		m := entities.NewStockMovement(
			productID, agg.qty,
			entities.StockReasonPurchase, entities.StockRefExpense, &expenseID,
			&expense.UserID, now, expense.Date, expense.Hour,
		)
		m.UnitCost = unitCost
		moves = append(moves, purchaseMove{movement: m, unitCost: unitCost})
	}

	return uc.uow.Do(ctx, func(r repositories.Repos) error {
		if err := r.Expenses.Create(ctx, expense); err != nil {
			return err
		}
		if err := r.ExpenseLines.CreateBatch(ctx, expense.ID, expense.Lines); err != nil {
			return err
		}
		if len(moves) == 0 {
			return nil
		}

		// Copia por intento: StockAfter sólo se conoce dentro de la transacción,
		// y un reintento debe partir de la plantilla limpia. Ver CreateSale.
		applied := make([]entities.StockMovement, len(moves))
		for i, move := range moves {
			applied[i] = move.movement
			newStock, _, err := r.Products.ApplyPurchase(ctx, applied[i].ProductID, applied[i].Quantity, move.unitCost)
			if err != nil {
				return err
			}
			applied[i].StockAfter = &newStock
		}
		return r.StockMovements.CreateBatch(ctx, applied)
	})
}

// VoidExpense anula un gasto creando su espejo negativo, igual que VoidSale.
//
// No se borra ni se edita el original: el espejo con importe negativo es lo que
// permite que los reportes sumen todas las filas del rango y ya salga neteado,
// sin filtrar por estado y sin contar dos veces.
func (uc *ExpenseUseCase) VoidExpense(ctx context.Context, expenseID, userID uuid.UUID, loc *time.Location) (*entities.Expense, error) {
	original, err := uc.expenseRepo.GetByID(ctx, expenseID)
	if err != nil {
		return nil, err
	}
	if original == nil {
		return nil, errors.ErrNotFound
	}
	if !original.CanBeVoided() {
		return nil, errors.ErrInvalidInput
	}

	lines, err := uc.expenseLineRepo.GetByExpenseID(ctx, expenseID)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC().Round(0)
	if loc == nil {
		loc = time.UTC
	}
	local := now.In(loc)

	voidExpense := &entities.Expense{
		ID:               uuid.New(),
		GymID:            original.GymID,
		CategoryID:       original.CategoryID,
		SupplierID:       original.SupplierID,
		PaymentMethodID:  original.PaymentMethodID,
		Description:      "Anulación: " + original.Description,
		DocumentNumber:   original.DocumentNumber,
		Subtotal:         -original.Subtotal,
		TaxAmount:        -original.TaxAmount,
		Total:            -original.Total,
		Status:           entities.ExpenseStatusRegistered,
		AffectsInventory: original.AffectsInventory,
		VoidedExpenseID:  &original.ID,
		UserID:           userID, // quien anula, no quien registró
		ExpenseDate:      now,
		// Fecha propia: la anulación ocurre hoy, no el día del gasto original.
		Date:      local.Format("2006-01-02"),
		Hour:      local.Format("15:04"),
		CreatedAt: now,
		UpdatedAt: now,
	}

	voidLines := make([]entities.ExpenseLine, len(lines))
	returns := make(map[uuid.UUID]int)
	for i, line := range lines {
		voidLines[i] = entities.ExpenseLine{
			ID:          uuid.New(),
			ExpenseID:   voidExpense.ID,
			ProductID:   line.ProductID,
			Description: line.Description,
			Quantity:    line.Quantity, // positiva por claridad, como en SaleDetail
			UnitCost:    line.UnitCost,
			Total:       -line.Total,
			CreatedAt:   now,
		}
		if line.ProductID != nil {
			returns[*line.ProductID] += int(line.Quantity)
		}
	}

	movesTemplate := make([]entities.StockMovement, 0, len(returns))
	for productID, qty := range returns {
		voidID := voidExpense.ID
		movesTemplate = append(movesTemplate, entities.NewStockMovement(
			productID, -qty,
			entities.StockReasonPurchaseVoid, entities.StockRefExpense, &voidID,
			&userID, now, voidExpense.Date, voidExpense.Hour,
		))
	}

	original.Status = entities.ExpenseStatusVoided
	original.UpdatedAt = now

	if err := uc.uow.Do(ctx, func(r repositories.Repos) error {
		if err := r.Expenses.Update(ctx, original); err != nil {
			return err
		}
		if err := r.Expenses.Create(ctx, voidExpense); err != nil {
			return err
		}
		if err := r.ExpenseLines.CreateBatch(ctx, voidExpense.ID, voidLines); err != nil {
			return err
		}

		moves := make([]entities.StockMovement, len(movesTemplate))
		copy(moves, movesTemplate)

		for i := range moves {
			// Si la mercancía comprada ya se vendió, esto falla con
			// ErrInsufficientStock y la anulación entera se revierte. Es
			// deliberado: permitir stock negativo aquí rompería para siempre la
			// guarda que impide sobrevender en el punto de venta, que es la
			// propiedad más valiosa del sistema. El operario tiene que registrar
			// primero una salida manual y anular después.
			newStock, err := r.Products.ApplyStockDelta(ctx, moves[i].ProductID, -moves[i].Quantity)
			if err != nil {
				return err
			}
			moves[i].StockAfter = &newStock
			// El costo promedio NO se revierte: ya se mezcló con compras
			// posteriores y deshacerlo exactamente es imposible en general.
			// Corregirlo, si hace falta, es un ajuste manual consciente.
			moves[i].Note = "Anulación de compra; el costo promedio no se revierte"
		}
		return r.StockMovements.CreateBatch(ctx, moves)
	}); err != nil {
		return nil, err
	}

	return voidExpense, nil
}

// GetExpenseByID devuelve un gasto con sus líneas y sus relaciones resueltas.
func (uc *ExpenseUseCase) GetExpenseByID(ctx context.Context, id uuid.UUID) (*entities.Expense, error) {
	expense, err := uc.expenseRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if expense == nil {
		return nil, errors.ErrNotFound
	}

	lines, err := uc.expenseLineRepo.GetByExpenseID(ctx, id)
	if err != nil {
		return nil, err
	}
	expense.Lines = lines

	uc.hydrate(ctx, []*entities.Expense{expense})
	return expense, nil
}

// ListExpenses devuelve los gastos de un rango con sus líneas y relaciones.
func (uc *ExpenseUseCase) ListExpenses(ctx context.Context, gymID uuid.UUID, startDate, endDate string) ([]entities.Expense, error) {
	expenses, err := uc.expenseRepo.ListByDateRange(ctx, gymID, startDate, endDate)
	if err != nil {
		return nil, err
	}
	if len(expenses) == 0 {
		return expenses, nil
	}

	// Las líneas de todos los gastos en UNA consulta. Pedirlas de una en una es
	// el antipatrón que ya se corrigió en el cierre diario.
	ids := make([]uuid.UUID, len(expenses))
	refs := make([]*entities.Expense, len(expenses))
	for i := range expenses {
		ids[i] = expenses[i].ID
		refs[i] = &expenses[i]
	}

	linesByExpense, err := uc.expenseLineRepo.GetByExpenseIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	for i := range expenses {
		expenses[i].Lines = linesByExpense[expenses[i].ID]
	}

	uc.hydrate(ctx, refs)
	return expenses, nil
}

// hydrate resuelve categoría, proveedor y método de pago en bloque, cacheando
// por ID para no repetir consultas cuando varios gastos comparten proveedor.
func (uc *ExpenseUseCase) hydrate(ctx context.Context, expenses []*entities.Expense) {
	categories := make(map[uuid.UUID]*entities.ExpenseCategory)
	suppliers := make(map[uuid.UUID]*entities.Supplier)
	methods := make(map[uuid.UUID]*entities.SalePaymentMethod)

	for _, e := range expenses {
		if _, ok := categories[e.CategoryID]; !ok {
			c, _ := uc.categoryRepo.GetByID(ctx, e.CategoryID)
			categories[e.CategoryID] = c
		}
		e.Category = categories[e.CategoryID]

		if e.SupplierID != nil {
			if _, ok := suppliers[*e.SupplierID]; !ok {
				s, _ := uc.supplierRepo.GetByID(ctx, *e.SupplierID)
				suppliers[*e.SupplierID] = s
			}
			e.Supplier = suppliers[*e.SupplierID]
		}

		if _, ok := methods[e.PaymentMethodID]; !ok {
			m, _ := uc.paymentMethodRepo.GetByID(ctx, e.PaymentMethodID)
			methods[e.PaymentMethodID] = m
		}
		e.PaymentMethod = methods[e.PaymentMethodID]
	}
}

// Report devuelve los totales de egresos del periodo.
func (uc *ExpenseUseCase) Report(ctx context.Context, gymID uuid.UUID, startDate, endDate string) (repositories.ExpenseReport, error) {
	return uc.expenseRepo.Report(ctx, gymID, startDate, endDate)
}

// ReportByCategory devuelve el desglose de egresos por categoría.
func (uc *ExpenseUseCase) ReportByCategory(ctx context.Context, gymID uuid.UUID, startDate, endDate string) ([]repositories.ExpenseByCategoryReport, error) {
	return uc.expenseRepo.ReportByCategory(ctx, gymID, startDate, endDate)
}

// ListCategories devuelve las categorías disponibles para el gimnasio.
func (uc *ExpenseUseCase) ListCategories(ctx context.Context, gymID uuid.UUID) ([]entities.ExpenseCategory, error) {
	return uc.categoryRepo.List(ctx, gymID)
}
