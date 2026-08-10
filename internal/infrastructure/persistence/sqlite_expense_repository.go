package persistence

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/sebastiancorrales/gym-go/internal/domain/entities"
	"github.com/sebastiancorrales/gym-go/internal/domain/repositories"
	"gorm.io/gorm"
)

// ── Suppliers ────────────────────────────────────────────────────────────────

type SQLiteSupplierRepository struct{ db *gorm.DB }

func NewSQLiteSupplierRepository(db *gorm.DB) repositories.SupplierRepository {
	return &SQLiteSupplierRepository{db: db}
}

func (r *SQLiteSupplierRepository) Create(ctx context.Context, s *entities.Supplier) error {
	return r.db.WithContext(ctx).Create(s).Error
}

func (r *SQLiteSupplierRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.Supplier, error) {
	var s entities.Supplier
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&s).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &s, nil
}

func (r *SQLiteSupplierRepository) List(ctx context.Context, gymID uuid.UUID, status *entities.SupplierStatus) ([]entities.Supplier, error) {
	var suppliers []entities.Supplier
	q := r.db.WithContext(ctx).Where("gym_id = ?", gymID)
	if status != nil {
		q = q.Where("status = ?", *status)
	}
	err := q.Order("name ASC").Limit(maxSupplierRows).Find(&suppliers).Error
	warnIfCapped("List(suppliers)", len(suppliers), maxSupplierRows)
	return suppliers, err
}

// Update writes an explicit column list. Same reasoning as products: Save()
// writes the whole struct, and a field the caller did not intend to touch
// arriving as its zero value silently overwrites real data.
func (r *SQLiteSupplierRepository) Update(ctx context.Context, s *entities.Supplier) error {
	return r.db.WithContext(ctx).Model(&entities.Supplier{}).
		Where("id = ?", s.ID).
		Updates(map[string]interface{}{
			"name": s.Name, "tax_id": s.TaxID, "contact_name": s.ContactName,
			"phone": s.Phone, "email": s.Email, "address": s.Address,
			"notes": s.Notes, "status": s.Status, "updated_at": s.UpdatedAt,
		}).Error
}

func (r *SQLiteSupplierRepository) FindByTaxID(ctx context.Context, gymID uuid.UUID, taxID string) (*entities.Supplier, error) {
	taxID = strings.TrimSpace(taxID)
	if taxID == "" {
		// Sin NIT no hay unicidad que comprobar: varios proveedores pueden no
		// tenerlo, y devolver "encontrado" bloquearía el alta del segundo.
		return nil, nil
	}
	var s entities.Supplier
	err := r.db.WithContext(ctx).Where("gym_id = ? AND tax_id = ?", gymID, taxID).First(&s).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &s, nil
}

// ── Expense categories ───────────────────────────────────────────────────────

type SQLiteExpenseCategoryRepository struct{ db *gorm.DB }

func NewSQLiteExpenseCategoryRepository(db *gorm.DB) repositories.ExpenseCategoryRepository {
	return &SQLiteExpenseCategoryRepository{db: db}
}

func (r *SQLiteExpenseCategoryRepository) Create(ctx context.Context, c *entities.ExpenseCategory) error {
	return r.db.WithContext(ctx).Create(c).Error
}

func (r *SQLiteExpenseCategoryRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.ExpenseCategory, error) {
	var c entities.ExpenseCategory
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&c).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &c, nil
}

// List returns the gym's own categories plus the system ones (gym_id nil).
func (r *SQLiteExpenseCategoryRepository) List(ctx context.Context, gymID uuid.UUID) ([]entities.ExpenseCategory, error) {
	var categories []entities.ExpenseCategory
	err := r.db.WithContext(ctx).
		Where("gym_id = ? OR gym_id = ?", gymID, uuid.Nil).
		Order("name ASC").Find(&categories).Error
	return categories, err
}

// ── Expenses ─────────────────────────────────────────────────────────────────

type SQLiteExpenseRepository struct{ db *gorm.DB }

func NewSQLiteExpenseRepository(db *gorm.DB) repositories.ExpenseRepository {
	return &SQLiteExpenseRepository{db: db}
}

func (r *SQLiteExpenseRepository) Create(ctx context.Context, e *entities.Expense) error {
	return r.db.WithContext(ctx).Create(e).Error
}

func (r *SQLiteExpenseRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.Expense, error) {
	var e entities.Expense
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&e).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &e, nil
}

func (r *SQLiteExpenseRepository) Update(ctx context.Context, e *entities.Expense) error {
	return r.db.WithContext(ctx).Model(&entities.Expense{}).
		Where("id = ?", e.ID).
		Updates(map[string]interface{}{
			"status": e.Status, "updated_at": e.UpdatedAt,
		}).Error
}

func (r *SQLiteExpenseRepository) ListByDateRange(ctx context.Context, gymID uuid.UUID, startDate, endDate string) ([]entities.Expense, error) {
	var expenses []entities.Expense
	err := r.db.WithContext(ctx).
		Where("gym_id = ? AND date >= ? AND date <= ?", gymID, startDate, endDate).
		Order("date DESC, hour DESC").
		Limit(maxExpenseRows).Find(&expenses).Error
	warnIfCapped("ListByDateRange(expenses)", len(expenses), maxExpenseRows)
	return expenses, err
}

// Report aggregates in the database.
//
// A voided expense leaves a mirror row with a negative total (same convention as
// VoidSale), so summing every row over the range already nets the cancellations
// out — no filtering by status needed, and no double-counting.
func (r *SQLiteExpenseRepository) Report(ctx context.Context, gymID uuid.UUID, startDate, endDate string) (repositories.ExpenseReport, error) {
	var report repositories.ExpenseReport

	err := r.db.WithContext(ctx).Model(&entities.Expense{}).
		Select(`COALESCE(SUM(expenses.total), 0) AS total_expenses,
		        COUNT(*) AS expenses_count,
		        COALESCE(SUM(CASE WHEN expense_categories.kind = 'inventory' THEN expenses.total ELSE 0 END), 0) AS inventory_cost,
		        COALESCE(SUM(CASE WHEN expense_categories.kind <> 'inventory' THEN expenses.total ELSE 0 END), 0) AS operating_cost`).
		Joins("LEFT JOIN expense_categories ON expense_categories.id = expenses.category_id").
		Where("expenses.gym_id = ? AND expenses.date >= ? AND expenses.date <= ?", gymID, startDate, endDate).
		Scan(&report).Error

	return report, err
}

func (r *SQLiteExpenseRepository) ReportByCategory(ctx context.Context, gymID uuid.UUID, startDate, endDate string) ([]repositories.ExpenseByCategoryReport, error) {
	var rows []repositories.ExpenseByCategoryReport

	err := r.db.WithContext(ctx).Model(&entities.Expense{}).
		Select(`expenses.category_id AS category_id,
		        COALESCE(expense_categories.name, 'Sin categoría') AS category_name,
		        COALESCE(expense_categories.kind, 'other') AS kind,
		        COALESCE(SUM(expenses.total), 0) AS total,
		        COUNT(*) AS count`).
		Joins("LEFT JOIN expense_categories ON expense_categories.id = expenses.category_id").
		Where("expenses.gym_id = ? AND expenses.date >= ? AND expenses.date <= ?", gymID, startDate, endDate).
		Group("expenses.category_id, expense_categories.name, expense_categories.kind").
		Order("total DESC").
		Scan(&rows).Error

	return rows, err
}

// ── Expense lines ────────────────────────────────────────────────────────────

type SQLiteExpenseLineRepository struct{ db *gorm.DB }

func NewSQLiteExpenseLineRepository(db *gorm.DB) repositories.ExpenseLineRepository {
	return &SQLiteExpenseLineRepository{db: db}
}

func (r *SQLiteExpenseLineRepository) CreateBatch(ctx context.Context, expenseID uuid.UUID, lines []entities.ExpenseLine) error {
	if len(lines) == 0 {
		return nil
	}
	for i := range lines {
		lines[i].ExpenseID = expenseID
		if lines[i].ID == uuid.Nil {
			lines[i].ID = uuid.New()
		}
	}
	return r.db.WithContext(ctx).Create(&lines).Error
}

func (r *SQLiteExpenseLineRepository) GetByExpenseID(ctx context.Context, expenseID uuid.UUID) ([]entities.ExpenseLine, error) {
	var lines []entities.ExpenseLine
	err := r.db.WithContext(ctx).Where("expense_id = ?", expenseID).Find(&lines).Error
	return lines, err
}

// GetByExpenseIDs resolves many expenses' lines in one query, batched to stay
// under SQLite's parameter limit.
func (r *SQLiteExpenseLineRepository) GetByExpenseIDs(ctx context.Context, expenseIDs []uuid.UUID) (map[uuid.UUID][]entities.ExpenseLine, error) {
	grouped := make(map[uuid.UUID][]entities.ExpenseLine, len(expenseIDs))
	if len(expenseIDs) == 0 {
		return grouped, nil
	}

	const batchSize = 500
	for start := 0; start < len(expenseIDs); start += batchSize {
		end := start + batchSize
		if end > len(expenseIDs) {
			end = len(expenseIDs)
		}

		var lines []entities.ExpenseLine
		if err := r.db.WithContext(ctx).
			Where("expense_id IN ?", expenseIDs[start:end]).Find(&lines).Error; err != nil {
			return nil, err
		}
		for _, line := range lines {
			grouped[line.ExpenseID] = append(grouped[line.ExpenseID], line)
		}
	}
	return grouped, nil
}
