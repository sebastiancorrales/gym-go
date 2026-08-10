package persistence

import (
	"context"

	"github.com/google/uuid"
	"github.com/sebastiancorrales/gym-go/internal/domain/entities"
	"github.com/sebastiancorrales/gym-go/internal/domain/repositories"
	"gorm.io/gorm"
)

// SQLitePaymentLineRepository implements PaymentLineRepository for SQLite.
type SQLitePaymentLineRepository struct {
	db *gorm.DB
}

func NewSQLitePaymentLineRepository(db *gorm.DB) repositories.PaymentLineRepository {
	return &SQLitePaymentLineRepository{db: db}
}

func (r *SQLitePaymentLineRepository) CreateBatch(ctx context.Context, lines []entities.PaymentLine) error {
	if len(lines) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Create(&lines).Error
}

func (r *SQLitePaymentLineRepository) GetByPayable(ctx context.Context, kind entities.PayableKind, payableID uuid.UUID) ([]entities.PaymentLine, error) {
	var lines []entities.PaymentLine
	err := r.db.WithContext(ctx).
		Where("payable_kind = ? AND payable_id = ?", kind, payableID).
		Order("created_at ASC, id ASC").
		Find(&lines).Error
	return lines, err
}

// GetByPayables resolves many documents' lines in one query, batched to stay
// under SQLite's parameter limit.
//
// Asking one document at a time is the N+1 that was already fixed once in the
// daily close report; this exists so it does not come back through payments.
func (r *SQLitePaymentLineRepository) GetByPayables(ctx context.Context, kind entities.PayableKind, payableIDs []uuid.UUID) (map[uuid.UUID][]entities.PaymentLine, error) {
	grouped := make(map[uuid.UUID][]entities.PaymentLine, len(payableIDs))
	if len(payableIDs) == 0 {
		return grouped, nil
	}

	const batchSize = 500
	for start := 0; start < len(payableIDs); start += batchSize {
		end := start + batchSize
		if end > len(payableIDs) {
			end = len(payableIDs)
		}

		var lines []entities.PaymentLine
		if err := r.db.WithContext(ctx).
			Where("payable_kind = ? AND payable_id IN ?", kind, payableIDs[start:end]).
			Order("created_at ASC, id ASC").
			Find(&lines).Error; err != nil {
			return nil, err
		}
		for _, line := range lines {
			grouped[line.PayableID] = append(grouped[line.PayableID], line)
		}
	}
	return grouped, nil
}

// SumByMethod aggregates IN THE DATABASE and is the single source of the
// payment-method breakdown.
//
// Because Amount carries its sign, voids net themselves out here — no status
// filter, no separate subtraction, and the totals add up to the day's cash.
//
// The LEFT JOIN on subscriptions is the one place the clean polymorphic design
// gets muddy: cancelling a subscription does not write a reversing line (the
// money may or may not have been refunded — that is a policy decision nobody has
// made), yet every existing report excludes CANCELLED subscriptions. Excluding
// them here preserves today's numbers exactly. The alternative, writing
// reversing lines on cancellation, is the better model and the eventual fix.
func (r *SQLitePaymentLineRepository) SumByMethod(ctx context.Context, startDate, endDate string) ([]repositories.PaymentMethodTotal, error) {
	var rows []repositories.PaymentMethodTotal

	err := r.db.WithContext(ctx).Model(&entities.PaymentLine{}).
		Select(`payment_lines.payment_method_id AS payment_method_id,
		        COALESCE(sale_payment_methods.name, 'Desconocido') AS payment_method_name,
		        COALESCE(sale_payment_methods.type, 'other') AS payment_method_type,
		        payment_lines.payable_kind AS payable_kind,
		        COALESCE(SUM(payment_lines.amount), 0) AS total,
		        COALESCE(SUM(payment_lines.change_given), 0) AS change_given,
		        COUNT(*) AS count`).
		Joins("LEFT JOIN sale_payment_methods ON sale_payment_methods.id = payment_lines.payment_method_id").
		Joins(`LEFT JOIN subscriptions ON payment_lines.payable_kind = 'subscription'
		        AND subscriptions.id = payment_lines.payable_id`).
		Where("payment_lines.date >= ? AND payment_lines.date <= ?", startDate, endDate).
		Where("payment_lines.payable_kind <> 'subscription' OR subscriptions.status IS NULL OR subscriptions.status <> 'CANCELLED'").
		Group("payment_lines.payment_method_id, sale_payment_methods.name, sale_payment_methods.type, payment_lines.payable_kind").
		Scan(&rows).Error

	return rows, err
}

// ReplaceForPayable swaps a document's payment lines.
//
// DELETE then INSERT in that order, with IDs precomputed by the caller, so a
// retry after SQLITE_BUSY re-executes both and converges to the same state
// instead of duplicating rows — the Unit of Work idempotency contract.
func (r *SQLitePaymentLineRepository) ReplaceForPayable(ctx context.Context, kind entities.PayableKind, payableID uuid.UUID, lines []entities.PaymentLine) error {
	db := r.db.WithContext(ctx)

	if err := db.Where("payable_kind = ? AND payable_id = ?", kind, payableID).
		Delete(&entities.PaymentLine{}).Error; err != nil {
		return err
	}
	if len(lines) == 0 {
		return nil
	}
	return db.Create(&lines).Error
}
