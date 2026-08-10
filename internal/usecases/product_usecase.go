package usecases

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/sebastiancorrales/gym-go/internal/domain/entities"
	"github.com/sebastiancorrales/gym-go/internal/domain/repositories"
	"github.com/sebastiancorrales/gym-go/pkg/errors"
)

// ProductUseCase handles business logic for products
type ProductUseCase struct {
	productRepo   repositories.ProductRepository
	movementRepo  repositories.StockMovementRepository
	uow           repositories.UnitOfWork
	defaultTZName string
}

// NewProductUseCase creates a new ProductUseCase.
//
// Recibe el Unit of Work porque desde que existe el kardex ningún cambio de
// stock puede ocurrir solo: el movimiento y el saldo se escriben juntos o no se
// escribe ninguno.
func NewProductUseCase(
	productRepo repositories.ProductRepository,
	movementRepo repositories.StockMovementRepository,
	uow repositories.UnitOfWork,
) *ProductUseCase {
	return &ProductUseCase{
		productRepo:  productRepo,
		movementRepo: movementRepo,
		uow:          uow,
	}
}

// StockChange describe un cambio de stock pedido por una persona.
//
// Loc es la zona horaria del gimnasio: el movimiento se estampa con la fecha
// local, no con la UTC, porque es la columna por la que filtran el kardex y el
// cuadre del día.
type StockChange struct {
	ProductID uuid.UUID
	Reason    entities.StockMovementReason
	Note      string
	UserID    uuid.UUID
	Loc       *time.Location
}

// applyStockChange es el único camino por el que un cambio manual de stock llega
// a la base: aplica el delta y escribe el asiento en la misma transacción.
func (uc *ProductUseCase) applyStockChange(ctx context.Context, change StockChange, delta int) (*entities.StockMovement, error) {
	if delta == 0 {
		return nil, nil // nada que registrar; un asiento de cero solo ensucia el libro
	}
	if !entities.IsManualStockReason(change.Reason) {
		// Sin esto, un cliente podría mandar reason="sale" y meter movimientos en
		// el kardex que no corresponden a ninguna venta.
		return nil, errors.ErrInvalidInput
	}

	loc := change.Loc
	if loc == nil {
		loc = time.UTC
	}
	now := time.Now().UTC().Round(0)
	local := now.In(loc)

	var userID *uuid.UUID
	if change.UserID != uuid.Nil {
		userID = &change.UserID
	}

	// UUID y timestamps fijados ANTES de entrar: el Unit of Work reintenta ante
	// contención y exige que la función sea idempotente.
	template := entities.NewStockMovement(
		change.ProductID, delta,
		change.Reason, entities.StockRefManual, nil,
		userID, now, local.Format("2006-01-02"), local.Format("15:04"),
	)
	template.Note = change.Note

	var applied entities.StockMovement
	err := uc.uow.Do(ctx, func(r repositories.Repos) error {
		move := template // copia por intento: StockAfter solo se conoce aquí dentro

		newStock, err := r.Products.ApplyStockDelta(ctx, move.ProductID, delta)
		if err != nil {
			return err
		}
		move.StockAfter = &newStock

		if err := r.StockMovements.CreateBatch(ctx, []entities.StockMovement{move}); err != nil {
			return err
		}
		applied = move
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &applied, nil
}

// CreateProduct creates a new product.
//
// Si nace con stock, el alta escribe también su asiento inicial: sin él, el
// kardex arrancaría con un saldo que no viene de ningún movimiento y la
// comprobación "suma del libro == products.stock" fallaría desde el primer día.
func (uc *ProductUseCase) CreateProduct(ctx context.Context, product *entities.Product, change StockChange) error {
	if product.Name == "" {
		return errors.ErrInvalidInput
	}
	if product.UnitPrice < 0 {
		return errors.ErrInvalidPrice
	}
	if product.Stock < 0 {
		return errors.ErrInvalidQuantity
	}
	if product.MinStock < 0 {
		return errors.ErrInvalidQuantity
	}

	// Set default values
	product.ID = uuid.New()
	if product.Status == "" {
		product.Status = entities.ProductStatusActive
	}
	now := time.Now().UTC().Round(0)
	product.CreatedAt = now
	product.UpdatedAt = now

	if product.Stock == 0 {
		return uc.productRepo.Create(ctx, product)
	}

	loc := change.Loc
	if loc == nil {
		loc = time.UTC
	}
	local := now.In(loc)

	var userID *uuid.UUID
	if change.UserID != uuid.Nil {
		userID = &change.UserID
	}

	productID := product.ID
	initialStock := product.Stock

	template := entities.NewStockMovement(
		productID, initialStock,
		entities.StockReasonInitial, entities.StockRefProduct, &productID,
		userID, now, local.Format("2006-01-02"), local.Format("15:04"),
	)
	template.Note = "Stock inicial al crear el producto"
	template.StockAfter = &initialStock // el producto nace con este saldo, no hay delta que leer

	return uc.uow.Do(ctx, func(r repositories.Repos) error {
		if err := r.Products.Create(ctx, product); err != nil {
			return err
		}
		return r.StockMovements.CreateBatch(ctx, []entities.StockMovement{template})
	})
}

// GetProductByID retrieves a product by ID
func (uc *ProductUseCase) GetProductByID(ctx context.Context, id uuid.UUID) (*entities.Product, error) {
	return uc.productRepo.GetByID(ctx, id)
}

// GetAllProducts retrieves all products, optionally filtered by status
func (uc *ProductUseCase) GetAllProducts(ctx context.Context, status *entities.ProductStatus) ([]entities.Product, error) {
	return uc.productRepo.GetAll(ctx, status)
}

// UpdateProduct updates an existing product.
//
// Ya no valida el stock porque ya no lo toca: el repositorio escribe una lista
// explícita de columnas que no lo incluye. Mover stock es asunto del kardex.
func (uc *ProductUseCase) UpdateProduct(ctx context.Context, product *entities.Product) error {
	if product.ID == uuid.Nil {
		return errors.ErrInvalidInput
	}
	if product.Name == "" {
		return errors.ErrInvalidInput
	}
	if product.UnitPrice < 0 {
		return errors.ErrInvalidPrice
	}
	if product.MinStock < 0 {
		return errors.ErrInvalidQuantity
	}

	// Check if product exists
	existing, err := uc.productRepo.GetByID(ctx, product.ID)
	if err != nil {
		return err
	}
	if existing == nil {
		return errors.ErrNotFound
	}

	product.UpdatedAt = time.Now().UTC().Round(0)
	product.CreatedAt = existing.CreatedAt // Preserve creation date

	return uc.productRepo.Update(ctx, product)
}

// DeleteProduct deletes a product
func (uc *ProductUseCase) DeleteProduct(ctx context.Context, id uuid.UUID) error {
	// Check if product exists
	existing, err := uc.productRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if existing == nil {
		return errors.ErrNotFound
	}

	return uc.productRepo.Delete(ctx, id)
}

// CountProductStock registra un conteo físico: el stock pasa a valer `counted` y
// queda un asiento con la diferencia.
//
// Antes se llamaba SetProductStock y escribía el valor absoluto sin dejar rastro
// — el saldo anterior se perdía para siempre y nadie sabía quién lo había
// cambiado. Ahora el ajuste se expresa como el delta que realmente ocurrió, que
// es lo que un libro de inventario tiene que poder contar.
func (uc *ProductUseCase) CountProductStock(ctx context.Context, counted int, change StockChange) (*entities.StockMovement, error) {
	if counted < 0 {
		return nil, errors.ErrInvalidQuantity
	}

	product, err := uc.productRepo.GetByID(ctx, change.ProductID)
	if err != nil {
		return nil, err
	}
	if product == nil {
		return nil, errors.ErrNotFound
	}

	change.Reason = entities.StockReasonCount
	// El delta se calcula desde el saldo leído, pero quien decide si cabe es el
	// UPDATE condicional: si otra caja vende entre la lectura y la escritura, el
	// conteo aplica su delta sobre el saldo real y no pisa esa venta.
	return uc.applyStockChange(ctx, change, counted-product.Stock)
}

// AdjustProductStock registra una entrada o salida manual de inventario.
//
// Ya no lee el stock para comprobar que no quede negativo: esa comprobación en
// Go era una carrera (leer, decidir, escribir). Ahora la guarda vive en el WHERE
// de ApplyStockDelta y la decide la base de datos.
func (uc *ProductUseCase) AdjustProductStock(ctx context.Context, delta int, change StockChange) (*entities.StockMovement, error) {
	if delta == 0 {
		return nil, errors.ErrInvalidQuantity
	}
	if change.Reason == "" {
		change.Reason = entities.StockReasonAdjustment
	}
	return uc.applyStockChange(ctx, change, delta)
}

// ListStockMovements devuelve el kardex de un rango con filtros opcionales.
func (uc *ProductUseCase) ListStockMovements(ctx context.Context, startDate, endDate string, productID *uuid.UUID, reason *entities.StockMovementReason) ([]entities.StockMovement, error) {
	return uc.movementRepo.List(ctx, startDate, endDate, productID, reason)
}

// ListProductMovements devuelve el kardex de un producto concreto.
func (uc *ProductUseCase) ListProductMovements(ctx context.Context, productID uuid.UUID, startDate, endDate string) ([]entities.StockMovement, error) {
	return uc.movementRepo.ListByProduct(ctx, productID, startDate, endDate)
}

// SearchProducts searches for products by name or description
func (uc *ProductUseCase) SearchProducts(ctx context.Context, searchTerm string) ([]entities.Product, error) {
	return uc.productRepo.Search(ctx, searchTerm)
}
