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

// SaleUseCase handles business logic for sales
type SaleUseCase struct {
	saleRepo          repositories.SaleRepository
	saleDetailRepo    repositories.SaleDetailRepository
	productRepo       repositories.ProductRepository
	paymentMethodRepo repositories.PaymentMethodRepository
	paymentLineRepo   repositories.PaymentLineRepository
	uow               repositories.UnitOfWork
}

// NewSaleUseCase creates a new SaleUseCase
func NewSaleUseCase(
	saleRepo repositories.SaleRepository,
	saleDetailRepo repositories.SaleDetailRepository,
	productRepo repositories.ProductRepository,
	paymentMethodRepo repositories.PaymentMethodRepository,
	paymentLineRepo repositories.PaymentLineRepository,
	uow repositories.UnitOfWork,
) *SaleUseCase {
	return &SaleUseCase{
		saleRepo:          saleRepo,
		saleDetailRepo:    saleDetailRepo,
		productRepo:       productRepo,
		paymentMethodRepo: paymentMethodRepo,
		paymentLineRepo:   paymentLineRepo,
		uow:               uow,
	}
}

// CreateSale creates a new sale with its details
// This function handles inventory updates and transaction management
func (uc *SaleUseCase) CreateSale(ctx context.Context, sale *entities.Sale) error {
	// Validate sale
	if sale.UserID == uuid.Nil {
		return errors.ErrInvalidInput
	}
	// Un cobro necesita o bien el método único (clientes anteriores al pago
	// combinado) o bien sus líneas.
	if sale.PaymentMethodID == uuid.Nil && len(sale.PaymentLines) == 0 {
		return errors.ErrInvalidInput
	}
	if len(sale.Details) == 0 {
		return errors.ErrInvalidInput
	}

	// Validar cada método distinto que aparezca en el pago, memorizando para no
	// consultar dos veces el mismo.
	methods, err := uc.loadPaymentMethodsFor(ctx, sale.PaymentMethodID, sale.PaymentLines)
	if err != nil {
		return err
	}

	// Validate all details and check stock
	productMap := make(map[uuid.UUID]*entities.Product)
	for i := range sale.Details {
		detail := &sale.Details[i]

		// Validate detail
		if err := detail.Validate(); err != nil {
			return err
		}

		// Check if product exists and is active
		product, exists := productMap[detail.ProductID]
		if !exists {
			product, err = uc.productRepo.GetByID(ctx, detail.ProductID)
			if err != nil {
				return err
			}
			if product == nil {
				return errors.ErrNotFound
			}
			if !product.IsActive() {
				return errors.ErrProductNotActive
			}
			productMap[detail.ProductID] = product
		}

		// Check stock availability
		if !product.HasStock(detail.Quantity) {
			return errors.ErrInsufficientStock
		}

		// Set unit price from product if not set
		if detail.UnitPrice == 0 {
			detail.UnitPrice = product.UnitPrice
		}

		// Calculate subtotal
		detail.CalculateSubtotal()
	}

	// Set sale defaults
	sale.ID = uuid.New()
	if sale.Type == "" {
		sale.Type = entities.SaleTypeNormal
	}
	if sale.Status == "" {
		sale.Status = entities.SaleStatusCompleted
	}
	if sale.SaleDate.IsZero() {
		sale.SaleDate = time.Now().UTC().Round(0)
	}
	sale.CreatedAt = time.Now().UTC().Round(0)
	sale.UpdatedAt = time.Now().UTC().Round(0)

	// Calculate totals
	sale.CalculateTotal()

	// Las líneas de pago se resuelven DESPUÉS de calcular el total, porque el
	// cliente nunca manda el total: lo decide el servidor a partir del carrito.
	paymentLines, err := uc.buildPaymentLines(
		sale.PaymentMethodID, sale.PaymentLines, methods, sale.Total,
		entities.PayableSale, sale.ID, sale.GymID, sale.Date, sale.Hour, sale.CreatedAt,
	)
	if err != nil {
		return err
	}
	sale.PaymentLines = paymentLines
	// El método "principal" (la línea de mayor monto) mantiene rellena la columna
	// vieja, que leen cinco sitios de producción y la respuesta de la API.
	sale.PaymentMethodID = entities.PrimaryPaymentMethod(paymentLines)

	// Total quantity per product, computed before the transaction so a retry does
	// not accumulate: everything inside uow.Do must be idempotent.
	qtyByProduct := make(map[uuid.UUID]int, len(productMap))
	for _, detail := range sale.Details {
		qtyByProduct[detail.ProductID] += detail.Quantity
	}

	// Ledger entries, built here with their UUIDs and timestamps already fixed —
	// the Unit of Work retries on lock contention and requires that. StockAfter
	// is the one field that cannot be filled in yet: it is only known inside the
	// transaction, after the UPDATE lands.
	//
	// One movement per product, not per line: qtyByProduct already merges a sale
	// that lists the same product twice, and merging keeps the stock guard intact
	// (6+6 units against a stock of 10 must fail as a whole, not half-succeed).
	movesTemplate := make([]entities.StockMovement, 0, len(qtyByProduct))
	for productID, totalQty := range qtyByProduct {
		saleID := sale.ID
		movesTemplate = append(movesTemplate, entities.NewStockMovement(
			productID, -totalQty,
			entities.StockReasonSale, entities.StockRefSale, &saleID,
			&sale.UserID, sale.SaleDate, sale.Date, sale.Hour,
		))
	}

	// Sale, line items and stock movements are one atomic unit. They used to be
	// separate autocommit statements: a failure partway through left the sale
	// recorded with the stock only partially deducted, and the API still answered
	// as if everything had worked.
	return uc.uow.Do(ctx, func(r repositories.Repos) error {
		if err := r.Sales.Create(ctx, sale); err != nil {
			return err
		}

		if err := r.SaleDetails.CreateBatch(ctx, sale.ID, sale.Details); err != nil {
			return err
		}

		if err := r.PaymentLines.CreateBatch(ctx, sale.PaymentLines); err != nil {
			return err
		}

		if !sale.IsNormal() {
			return nil
		}

		// Fresh copy per attempt. StockAfter is the only field computed inside the
		// transaction, so a retry has to start from the clean template: without
		// this copy, an attempt that filled 3 of 5 rows before failing would leave
		// those 3 holding balances from the aborted attempt.
		moves := make([]entities.StockMovement, len(movesTemplate))
		copy(moves, movesTemplate)

		for i := range moves {
			// Conditional UPDATE: the database enforces stock >= quantity, which
			// closes the gap between the availability check above and this write.
			newStock, err := r.Products.ApplyStockDelta(ctx, moves[i].ProductID, -moves[i].Quantity)
			if err != nil {
				return err
			}
			moves[i].StockAfter = &newStock
		}

		return r.StockMovements.CreateBatch(ctx, moves)
	})
}

// VoidSale voids an existing sale and creates a void transaction
// This function restores inventory and maintains traceability
//
// loc is the gym's timezone, needed to stamp the local date/hour columns that all
// the date-range reports filter on.
func (uc *SaleUseCase) VoidSale(ctx context.Context, saleID uuid.UUID, userID uuid.UUID, loc *time.Location) (*entities.Sale, error) {
	// Get original sale
	originalSale, err := uc.saleRepo.GetByID(ctx, saleID)
	if err != nil {
		return nil, err
	}
	if originalSale == nil {
		return nil, errors.ErrNotFound
	}

	// Check if sale can be voided
	if !originalSale.CanBeVoided() {
		return nil, errors.ErrSaleCannotBeVoided
	}

	// Get sale details
	details, err := uc.saleDetailRepo.GetBySaleID(ctx, saleID)
	if err != nil {
		return nil, err
	}

	// Las líneas de pago de la venta original: el espejo es su negación exacta,
	// pata por pata. Si la venta se cobró con dos métodos, la anulación devuelve
	// por los dos.
	originalLines, err := uc.paymentLineRepo.GetByPayable(ctx, entities.PayableSale, saleID)
	if err != nil {
		return nil, err
	}

	if loc == nil {
		loc = time.UTC
	}
	now := time.Now().UTC().Round(0)
	localNow := now.In(loc)

	// Everything below is built before the transaction so that a retry replays
	// identical values instead of generating new IDs or timestamps.
	originalSale.Status = entities.SaleStatusVoided
	originalSale.UpdatedAt = now

	voidSale := &entities.Sale{
		ID:              uuid.New(),
		SaleDate:        now,
		Total:           -originalSale.Total,
		TotalDiscount:   -originalSale.TotalDiscount,
		UserID:          userID,
		Type:            entities.SaleTypeVoid,
		Status:          entities.SaleStatusCompleted,
		PaymentMethodID: originalSale.PaymentMethodID,
		VoidedSaleID:    &originalSale.ID,
		// Without these the void was invisible to every date-range report and to
		// the daily close, so an annulled sale never subtracted from the takings.
		Date:      localNow.Format("2006-01-02"),
		Hour:      localNow.Format("15:04"),
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Void details mirror the originals with negative amounts.
	voidDetails := make([]entities.SaleDetail, len(details))
	qtyByProduct := make(map[uuid.UUID]int, len(details))
	for i, detail := range details {
		voidDetails[i] = entities.SaleDetail{
			SaleID:     voidSale.ID,
			ProductID:  detail.ProductID,
			UnitPrice:  detail.UnitPrice,
			Quantity:   detail.Quantity, // Keep positive for clarity
			TotalPrice: -detail.TotalPrice,
			Discount:   -detail.Discount,
			Subtotal:   -detail.Subtotal,
			CreatedAt:  now,
		}
		qtyByProduct[detail.ProductID] += detail.Quantity
	}

	// Ledger entries for the return, built before the transaction like in
	// CreateSale. They carry the void sale's own local date/hour: the stock came
	// back today, not on the day of the original sale.
	movesTemplate := make([]entities.StockMovement, 0, len(qtyByProduct))
	for productID, totalQty := range qtyByProduct {
		voidID := voidSale.ID
		movesTemplate = append(movesTemplate, entities.NewStockMovement(
			productID, totalQty,
			entities.StockReasonSaleVoid, entities.StockRefSale, &voidID,
			&userID, voidSale.SaleDate, voidSale.Date, voidSale.Hour,
		))
	}

	// Espejo negado de cada línea original. CashReceived/ChangeGiven se quedan en
	// cero: una reversión es un movimiento neto de caja, no se "des-recibe" un
	// billete que ya se entregó.
	voidLines := make([]entities.PaymentLine, len(originalLines))
	for i, line := range originalLines {
		reversalOf := originalLines[i].ID
		voidLines[i] = entities.PaymentLine{
			ID:              uuid.New(),
			PayableKind:     entities.PayableSale,
			PayableID:       voidSale.ID,
			PaymentMethodID: line.PaymentMethodID,
			Amount:          -line.Amount,
			GymID:           line.GymID,
			// Fecha propia: la anulación ocurre hoy, no el día de la venta.
			// Estamparla con la fecha original movería dinero entre dos cierres
			// diarios que ya se enviaron por correo.
			Date:       voidSale.Date,
			Hour:       voidSale.Hour,
			ReversalOf: &reversalOf,
			CreatedAt:  now,
			UpdatedAt:  now,
		}
	}

	if err := uc.uow.Do(ctx, func(r repositories.Repos) error {
		if err := r.Sales.Update(ctx, originalSale); err != nil {
			return err
		}
		if err := r.Sales.Create(ctx, voidSale); err != nil {
			return err
		}
		if err := r.SaleDetails.CreateBatch(ctx, voidSale.ID, voidDetails); err != nil {
			return err
		}
		if err := r.PaymentLines.CreateBatch(ctx, voidLines); err != nil {
			return err
		}

		// Copy per attempt — see the comment in CreateSale.
		moves := make([]entities.StockMovement, len(movesTemplate))
		copy(moves, movesTemplate)

		for i := range moves {
			newStock, err := r.Products.ApplyStockDelta(ctx, moves[i].ProductID, moves[i].Quantity)
			if err != nil {
				return err
			}
			moves[i].StockAfter = &newStock
		}

		return r.StockMovements.CreateBatch(ctx, moves)
	}); err != nil {
		return nil, err
	}

	// Con su método resuelto, para que la respuesta de la anulación muestre por
	// qué patas se devolvió y no sólo los importes.
	uc.attachMethodsToLines(ctx, voidLines)
	voidSale.PaymentLines = voidLines
	return voidSale, nil
}

// GetSaleByID retrieves a sale by ID with its details
func (uc *SaleUseCase) GetSaleByID(ctx context.Context, id uuid.UUID) (*entities.Sale, error) {
	sale, err := uc.saleRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if sale == nil {
		return nil, errors.ErrNotFound
	}

	// Load details
	details, err := uc.saleDetailRepo.GetBySaleID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Load product info for each detail
	for i := range details {
		product, err := uc.productRepo.GetByID(ctx, details[i].ProductID)
		if err == nil && product != nil {
			details[i].Product = product
		}
	}
	sale.Details = details

	// Load payment method
	pm, err := uc.paymentMethodRepo.GetByID(ctx, sale.PaymentMethodID)
	if err == nil && pm != nil {
		sale.PaymentMethod = pm
	}

	// Y las patas del pago, con su método resuelto: es lo que permite mostrar
	// "Efectivo $40.000 + Transferencia $60.000" en vez de un único nombre.
	lines, err := uc.paymentLineRepo.GetByPayable(ctx, entities.PayableSale, id)
	if err != nil {
		return nil, err
	}
	uc.attachMethodsToLines(ctx, lines)
	sale.PaymentLines = lines

	return sale, nil
}

// attachMethodsToLines resuelve el método de cada línea, memorizando por id.
func (uc *SaleUseCase) attachMethodsToLines(ctx context.Context, lines []entities.PaymentLine) {
	cache := make(map[uuid.UUID]*entities.SalePaymentMethod)
	for i := range lines {
		id := lines[i].PaymentMethodID
		if _, ok := cache[id]; !ok {
			method, err := uc.paymentMethodRepo.GetByID(ctx, id)
			if err == nil {
				cache[id] = method
			}
		}
		lines[i].PaymentMethod = cache[id]
	}
}

// loadPaymentLines resuelve las líneas de muchas ventas en UNA consulta.
//
// Pedirlas de una en una es el N+1 que ya se corrigió una vez en el cierre
// diario; esto existe para que no vuelva por la puerta de los pagos.
func (uc *SaleUseCase) loadPaymentLines(ctx context.Context, sales []entities.Sale) {
	if len(sales) == 0 {
		return
	}

	ids := make([]uuid.UUID, len(sales))
	for i := range sales {
		ids[i] = sales[i].ID
	}

	byPayable, err := uc.paymentLineRepo.GetByPayables(ctx, entities.PayableSale, ids)
	if err != nil {
		return // el listado sigue siendo útil sin el desglose
	}

	cache := make(map[uuid.UUID]*entities.SalePaymentMethod)
	for i := range sales {
		lines := byPayable[sales[i].ID]
		for j := range lines {
			id := lines[j].PaymentMethodID
			if _, ok := cache[id]; !ok {
				method, err := uc.paymentMethodRepo.GetByID(ctx, id)
				if err == nil {
					cache[id] = method
				}
			}
			lines[j].PaymentMethod = cache[id]
		}
		sales[i].PaymentLines = lines
	}
}

// GetAllSales retrieves all sales
func (uc *SaleUseCase) GetAllSales(ctx context.Context) ([]entities.Sale, error) {
	sales, err := uc.saleRepo.GetAll(ctx)
	if err != nil {
		return nil, err
	}
	uc.loadPaymentMethods(ctx, sales)
	uc.loadPaymentLines(ctx, sales)
	return sales, nil
}

// GetSalesByDateRange retrieves sales within a date range using local date strings (YYYY-MM-DD)
func (uc *SaleUseCase) GetSalesByDateRange(ctx context.Context, startDate, endDate string, userID *uuid.UUID) ([]entities.Sale, error) {
	sales, err := uc.saleRepo.GetByDateRange(ctx, startDate, endDate, userID)
	if err != nil {
		return nil, err
	}
	uc.loadPaymentMethods(ctx, sales)
	uc.loadPaymentLines(ctx, sales)
	return sales, nil
}

// GetSalesReport generates a sales report for a date range using local date strings (YYYY-MM-DD)
func (uc *SaleUseCase) GetSalesReport(ctx context.Context, startDate, endDate string, userID *uuid.UUID) ([]repositories.SaleReport, error) {
	return uc.saleRepo.GetSalesReport(ctx, startDate, endDate, userID)
}

// loadPaymentMethods batch-loads payment methods for a list of sales
func (uc *SaleUseCase) loadPaymentMethods(ctx context.Context, sales []entities.Sale) {
	pmMap := make(map[uuid.UUID]*entities.SalePaymentMethod)
	for i := range sales {
		pmID := sales[i].PaymentMethodID
		if _, exists := pmMap[pmID]; exists {
			continue
		}
		pm, err := uc.paymentMethodRepo.GetByID(ctx, pmID)
		if err == nil && pm != nil {
			pmMap[pmID] = pm
		}
	}
	for i := range sales {
		if pm, ok := pmMap[sales[i].PaymentMethodID]; ok {
			sales[i].PaymentMethod = pm
		}
	}
}

// GetSalesReportByProduct generates a sales report grouped by product using local date strings (YYYY-MM-DD)
func (uc *SaleUseCase) GetSalesReportByProduct(ctx context.Context, startDate, endDate string) ([]repositories.SaleProductReport, error) {
	return uc.saleRepo.GetSalesReportByProduct(ctx, startDate, endDate)
}

// loadPaymentMethodsFor carga los métodos de pago implicados en un cobro y
// comprueba que todos existan y estén activos.
//
// Memoriza por id porque un pago combinado puede repetir la consulta y porque
// esto corre en el camino caliente de la caja.
func (uc *SaleUseCase) loadPaymentMethodsFor(ctx context.Context, primaryID uuid.UUID, lines []entities.PaymentLine) (map[uuid.UUID]*entities.SalePaymentMethod, error) {
	wanted := make([]uuid.UUID, 0, len(lines)+1)
	if primaryID != uuid.Nil {
		wanted = append(wanted, primaryID)
	}
	for _, line := range lines {
		wanted = append(wanted, line.PaymentMethodID)
	}

	methods := make(map[uuid.UUID]*entities.SalePaymentMethod, len(wanted))
	for _, id := range wanted {
		if id == uuid.Nil {
			return nil, errors.ErrInvalidInput
		}
		if _, ok := methods[id]; ok {
			continue
		}

		method, err := uc.paymentMethodRepo.GetByID(ctx, id)
		if err != nil {
			return nil, err
		}
		if method == nil {
			return nil, errors.ErrNotFound
		}
		if !method.IsActive() {
			return nil, errors.ErrPaymentMethodNotActive
		}
		methods[id] = method
	}
	return methods, nil
}

// buildPaymentLines deja las líneas de pago listas para escribir: rellena lo que
// falta, valida el invariante y calcula el vuelto.
//
// Compatibilidad hacia atrás: un llamador que sólo fije PaymentMethodID —los
// tests, y cualquier cliente anterior al pago combinado— obtiene una única línea
// por el total. Un pago con un solo método no es un caso especial: es el general
// con N=1, y por eso la síntesis vive aquí y no en el DTO.
//
// Todos los UUID y timestamps se deciden AQUÍ, fuera de la transacción, porque
// el Unit of Work reintenta ante contención y exige idempotencia.
func (uc *SaleUseCase) buildPaymentLines(
	primaryID uuid.UUID,
	lines []entities.PaymentLine,
	methods map[uuid.UUID]*entities.SalePaymentMethod,
	total float64,
	kind entities.PayableKind,
	payableID, gymID uuid.UUID,
	date, hour string,
	now time.Time,
) ([]entities.PaymentLine, error) {
	if len(lines) == 0 {
		lines = []entities.PaymentLine{{PaymentMethodID: primaryID, Amount: total}}
	}

	if err := entities.ValidatePaymentLines(lines, total); err != nil {
		return nil, err
	}
	if err := entities.ValidateCashChange(lines, methods); err != nil {
		return nil, err
	}
	// El vuelto lo calcula siempre el servidor: nunca se confía en un importe
	// monetario que venga del navegador.
	entities.ApplyChangeGiven(lines)

	built := make([]entities.PaymentLine, len(lines))
	for i, line := range lines {
		built[i] = entities.PaymentLine{
			ID:              uuid.New(),
			PayableKind:     kind,
			PayableID:       payableID,
			PaymentMethodID: line.PaymentMethodID,
			Amount:          money.Round(line.Amount),
			CashReceived:    money.Round(line.CashReceived),
			ChangeGiven:     money.Round(line.ChangeGiven),
			GymID:           gymID,
			Date:            date,
			Hour:            hour,
			CreatedAt:       now,
			UpdatedAt:       now,
			// El método ya está cargado y validado: adjuntarlo evita que la
			// respuesta del alta salga sin nombre y obligue a releerla.
			PaymentMethod: methods[line.PaymentMethodID],
		}
	}
	return built, nil
}
