package persistence

import (
	"context"

	"github.com/google/uuid"
	"github.com/sebastiancorrales/gym-go/internal/domain/entities"
	"github.com/sebastiancorrales/gym-go/internal/domain/repositories"
	"gorm.io/gorm"
)

// SQLiteDebtRepository implements repositories.DebtRepository for SQLite.
type SQLiteDebtRepository struct{ db *gorm.DB }

func NewSQLiteDebtRepository(db *gorm.DB) repositories.DebtRepository {
	return &SQLiteDebtRepository{db: db}
}

func (r *SQLiteDebtRepository) Create(ctx context.Context, debt *entities.Debt) error {
	return r.db.WithContext(ctx).Create(debt).Error
}

func (r *SQLiteDebtRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.Debt, error) {
	var d entities.Debt
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&d).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &d, nil
}

func (r *SQLiteDebtRepository) GetBySaleID(ctx context.Context, saleID uuid.UUID) (*entities.Debt, error) {
	var d entities.Debt
	err := r.db.WithContext(ctx).Where("sale_id = ?", saleID).First(&d).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &d, nil
}

// Update escribe una lista explícita de columnas — mismo motivo que
// sqlite_product_repository.Update: un Save() del struct entero escribiría
// también SaleID/Amount/CreatedByID con lo que traiga el caller, y un cero
// accidental ahí silenciaría datos reales.
func (r *SQLiteDebtRepository) Update(ctx context.Context, debt *entities.Debt) error {
	return r.db.WithContext(ctx).Model(&entities.Debt{}).
		Where("id = ?", debt.ID).
		Updates(map[string]interface{}{
			"debtor_user_id":              debt.DebtorUserID,
			"debtor_name":                 debt.DebtorName,
			"status":                      debt.Status,
			"collected_at":                debt.CollectedAt,
			"collected_by_id":             debt.CollectedByID,
			"collected_payment_method_id": debt.CollectedPaymentMethodID,
			"collected_payment_line_id":   debt.CollectedPaymentLineID,
			"updated_at":                  debt.UpdatedAt,
		}).Error
}

func (r *SQLiteDebtRepository) List(ctx context.Context, gymID uuid.UUID, debtorUserID *uuid.UUID, debtorName string, status *entities.DebtStatus) ([]entities.Debt, error) {
	q := r.db.WithContext(ctx).Where("gym_id = ?", gymID)
	if debtorUserID != nil {
		q = q.Where("debtor_user_id = ?", *debtorUserID)
	}
	if debtorName != "" {
		q = q.Where("LOWER(TRIM(debtor_name)) = LOWER(TRIM(?))", debtorName)
	}
	if status != nil {
		q = q.Where("status = ?", *status)
	}
	var debts []entities.Debt
	err := q.Order("date DESC, hour DESC").Find(&debts).Error
	return debts, err
}

// debtorSummaryRow es la forma cruda que devuelve SQLite antes de convertir
// debtor_user_id (TEXT o NULL) a *uuid.UUID: GORM no hace esa conversión sola
// al escanear un SELECT crudo a un struct — hay que pasar por string primero.
type debtorSummaryRow struct {
	DebtorUserID *string `json:"debtor_user_id"`
	DebtorName   string  `json:"debtor_name"`
	TotalOwed    float64 `json:"total_owed"`
	DebtCount    int     `json:"debt_count"`
	OldestDate   string  `json:"oldest_date"`
}

// SummaryByDebtor agrupa las deudas pendientes por deudor: por
// debtor_user_id cuando está vinculado, si no por el nombre libre
// normalizado. Agregación en la base, no en Go — mismo principio que
// SumByMethod: es la única fuente del "quién debe".
func (r *SQLiteDebtRepository) SummaryByDebtor(ctx context.Context, gymID uuid.UUID) ([]repositories.DebtorSummary, error) {
	var rows []debtorSummaryRow

	err := r.db.WithContext(ctx).Model(&entities.Debt{}).
		Select(`debtor_user_id AS debtor_user_id,
		        MAX(debtor_name) AS debtor_name,
		        COALESCE(SUM(amount), 0) AS total_owed,
		        COUNT(*) AS debt_count,
		        MIN(date) AS oldest_date`).
		Where("gym_id = ? AND status = ?", gymID, entities.DebtStatusPending).
		Group(`CASE WHEN debtor_user_id IS NOT NULL THEN debtor_user_id ELSE LOWER(TRIM(debtor_name)) END`).
		Order("total_owed DESC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	summaries := make([]repositories.DebtorSummary, 0, len(rows))
	for _, row := range rows {
		s := repositories.DebtorSummary{
			DebtorName: row.DebtorName,
			TotalOwed:  row.TotalOwed,
			DebtCount:  row.DebtCount,
			OldestDate: row.OldestDate,
		}
		if row.DebtorUserID != nil && *row.DebtorUserID != "" {
			if id, parseErr := uuid.Parse(*row.DebtorUserID); parseErr == nil {
				s.DebtorUserID = &id
			}
		}
		summaries = append(summaries, s)
	}
	return summaries, nil
}
