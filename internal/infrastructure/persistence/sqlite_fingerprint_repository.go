package persistence

import (
	"context"
	"time"

	"github.com/sebastiancorrales/gym-go/internal/domain/entities"
	"gorm.io/gorm"
)

type SQLiteFingerprintRepository struct {
	db *gorm.DB
}

func NewSQLiteFingerprintRepository(db *gorm.DB) *SQLiteFingerprintRepository {
	return &SQLiteFingerprintRepository{db: db}
}

func (r *SQLiteFingerprintRepository) Create(ctx context.Context, fingerprint *entities.Fingerprint) error {
	now := time.Now().UTC().Round(0)
	fingerprint.CreatedAt = now
	fingerprint.UpdatedAt = now
	fingerprint.IsActive = true

	return r.db.WithContext(ctx).Create(fingerprint).Error
}

func (r *SQLiteFingerprintRepository) GetByID(ctx context.Context, id int) (*entities.Fingerprint, error) {
	var fingerprint entities.Fingerprint
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&fingerprint).Error
	if err != nil {
		return nil, err
	}
	return &fingerprint, nil
}

func (r *SQLiteFingerprintRepository) GetByUserID(ctx context.Context, userID string) ([]*entities.Fingerprint, error) {
	var fingerprints []*entities.Fingerprint
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("created_at DESC").Find(&fingerprints).Error
	return fingerprints, err
}

func (r *SQLiteFingerprintRepository) GetActiveByUserID(ctx context.Context, userID string) ([]*entities.Fingerprint, error) {
	var fingerprints []*entities.Fingerprint
	err := r.db.WithContext(ctx).Where("user_id = ? AND is_active = ?", userID, true).Order("created_at DESC").Find(&fingerprints).Error
	return fingerprints, err
}

func (r *SQLiteFingerprintRepository) Update(ctx context.Context, fingerprint *entities.Fingerprint) error {
	fingerprint.UpdatedAt = time.Now().UTC().Round(0)
	return r.db.WithContext(ctx).Save(fingerprint).Error
}

func (r *SQLiteFingerprintRepository) Delete(ctx context.Context, id int) error {
	return r.db.WithContext(ctx).Model(&entities.Fingerprint{}).Where("id = ?", id).Updates(map[string]interface{}{
		"is_active":  false,
		"updated_at": time.Now().UTC().Round(0),
	}).Error
}

// GetAllTemplates returns only the DigitalPersona templates, which are the
// ones the C# matching service can read.
//
// The filter is not cosmetic: Hikvision terminals match on the device and
// store their templates here purely as a backup copy, in a format the
// DigitalPersona SDK cannot parse. Handing it those rows would make every
// kiosk verification try to match against garbage. Rows written before the
// source column existed are empty and are DigitalPersona's.
func (r *SQLiteFingerprintRepository) GetAllTemplates(ctx context.Context) ([]*entities.Fingerprint, error) {
	var fingerprints []*entities.Fingerprint
	err := r.db.WithContext(ctx).
		Where("is_active = ?", true).
		Where("source IS NULL OR source = '' OR source = ?", entities.FingerprintSourceDigitalPersona).
		Order("user_id, created_at DESC").
		Find(&fingerprints).Error
	return fingerprints, err
}

// GetActiveByUserAndSource returns a user's active templates from one reader
// kind, so an enrollment flow can reason about its own device's slots without
// tripping over the other reader's rows.
func (r *SQLiteFingerprintRepository) GetActiveByUserAndSource(ctx context.Context, userID, source string) ([]*entities.Fingerprint, error) {
	query := r.db.WithContext(ctx).Where("user_id = ? AND is_active = ?", userID, true)
	if source == entities.FingerprintSourceDigitalPersona {
		query = query.Where("source IS NULL OR source = '' OR source = ?", source)
	} else {
		query = query.Where("source = ?", source)
	}

	var fingerprints []*entities.Fingerprint
	err := query.Order("created_at DESC").Find(&fingerprints).Error
	return fingerprints, err
}

func (r *SQLiteFingerprintRepository) CountActiveByUser(ctx context.Context) (map[string]int, error) {
	var rows []struct {
		UserID string
		Total  int
	}

	err := r.db.WithContext(ctx).
		Model(&entities.Fingerprint{}).
		Select("user_id, COUNT(*) AS total").
		Where("is_active = ?", true).
		Group("user_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	counts := make(map[string]int, len(rows))
	for _, row := range rows {
		counts[row.UserID] = row.Total
	}
	return counts, nil
}

// DistinctUserIDsBySource lists who is enrolled on a given reader kind. The
// validity sync walks this instead of every member: pushing a window for
// somebody who never enrolled would create people on the terminal that
// nobody can use.
func (r *SQLiteFingerprintRepository) DistinctUserIDsBySource(ctx context.Context, source string) ([]string, error) {
	query := r.db.WithContext(ctx).
		Model(&entities.Fingerprint{}).
		Distinct("user_id").
		Where("is_active = ?", true)

	if source == entities.FingerprintSourceDigitalPersona {
		query = query.Where("source IS NULL OR source = '' OR source = ?", source)
	} else {
		query = query.Where("source = ?", source)
	}

	var userIDs []string
	err := query.Pluck("user_id", &userIDs).Error
	return userIDs, err
}

func (r *SQLiteFingerprintRepository) LogVerification(ctx context.Context, verification *entities.FingerprintVerification) error {
	verification.VerifiedAt = time.Now().UTC().Round(0)
	return r.db.WithContext(ctx).Create(verification).Error
}
