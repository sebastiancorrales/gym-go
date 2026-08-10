package repositories

import (
	"context"

	"github.com/google/uuid"
	"github.com/sebastiancorrales/gym-go/internal/domain/entities"
)

// SupplierRepository defines the interface for supplier data operations.
type SupplierRepository interface {
	Create(ctx context.Context, supplier *entities.Supplier) error
	GetByID(ctx context.Context, id uuid.UUID) (*entities.Supplier, error)
	List(ctx context.Context, gymID uuid.UUID, status *entities.SupplierStatus) ([]entities.Supplier, error)
	Update(ctx context.Context, supplier *entities.Supplier) error
	// FindByTaxID resuelve la unicidad del NIT. Vive aquí y no como índice único
	// porque SQLite trata '' como un valor comparable, y un UNIQUE sobre
	// (gym_id, tax_id) impediría tener dos proveedores sin NIT.
	FindByTaxID(ctx context.Context, gymID uuid.UUID, taxID string) (*entities.Supplier, error)
}

// ExpenseCategoryRepository defines the interface for expense category operations.
type ExpenseCategoryRepository interface {
	Create(ctx context.Context, category *entities.ExpenseCategory) error
	GetByID(ctx context.Context, id uuid.UUID) (*entities.ExpenseCategory, error)
	// List devuelve las categorías del gimnasio MÁS las del sistema (gym_id nulo).
	List(ctx context.Context, gymID uuid.UUID) ([]entities.ExpenseCategory, error)
}

// ExpenseRepository defines the interface for expense data operations.
type ExpenseRepository interface {
	Create(ctx context.Context, expense *entities.Expense) error
	GetByID(ctx context.Context, id uuid.UUID) (*entities.Expense, error)
	Update(ctx context.Context, expense *entities.Expense) error
	// ListByDateRange filtra por la columna `date` local del gimnasio, igual que
	// sales y subscriptions: filtrar por el timestamp UTC metería los gastos de
	// la noche en el día siguiente.
	ListByDateRange(ctx context.Context, gymID uuid.UUID, startDate, endDate string) ([]entities.Expense, error)
	// Report agrega en la base de datos, no en Go.
	Report(ctx context.Context, gymID uuid.UUID, startDate, endDate string) (ExpenseReport, error)
	ReportByCategory(ctx context.Context, gymID uuid.UUID, startDate, endDate string) ([]ExpenseByCategoryReport, error)
}

// ExpenseLineRepository defines the interface for expense line operations.
type ExpenseLineRepository interface {
	CreateBatch(ctx context.Context, expenseID uuid.UUID, lines []entities.ExpenseLine) error
	GetByExpenseID(ctx context.Context, expenseID uuid.UUID) ([]entities.ExpenseLine, error)
	// GetByExpenseIDs resuelve las líneas de muchos gastos en una consulta. Sin
	// esto, un listado de 50 gastos serían 50 consultas — el antipatrón que ya se
	// corrigió en el cierre diario.
	GetByExpenseIDs(ctx context.Context, expenseIDs []uuid.UUID) (map[uuid.UUID][]entities.ExpenseLine, error)
}

// ExpenseReport son los totales de egresos de un periodo.
type ExpenseReport struct {
	TotalExpenses float64 `json:"total_expenses"`
	ExpensesCount int     `json:"expenses_count"`
	// InventoryCost es lo gastado en categorías de tipo `inventory` (mercancía
	// para revender); OperatingCost es el resto. Separarlos es lo que permite
	// calcular utilidad bruta y operativa sin adivinar por el nombre.
	InventoryCost float64 `json:"inventory_cost"`
	OperatingCost float64 `json:"operating_cost"`
}

// ExpenseByCategoryReport es el desglose de egresos por categoría.
type ExpenseByCategoryReport struct {
	CategoryID   uuid.UUID `json:"category_id"`
	CategoryName string    `json:"category_name"`
	Kind         string    `json:"kind"`
	Total        float64   `json:"total"`
	Count        int       `json:"count"`
}
