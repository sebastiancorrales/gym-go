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
	now := time.Now()
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

func (r *SQLiteFingerprintRepository) GetByUserID(ctx context.Context, userID int) ([]*entities.Fingerprint, error) {
	var fingerprints []*entities.Fingerprint
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("created_at DESC").Find(&fingerprints).Error
	return fingerprints, err
}

func (r *SQLiteFingerprintRepository) GetActiveByUserID(ctx context.Context, userID int) ([]*entities.Fingerprint, error) {
	var fingerprints []*entities.Fingerprint
	err := r.db.WithContext(ctx).Where("user_id = ? AND is_active = ?", userID, true).Order("created_at DESC").Find(&fingerprints).Error
	return fingerprints, err
}

func (r *SQLiteFingerprintRepository) Update(ctx context.Context, fingerprint *entities.Fingerprint) error {
	fingerprint.UpdatedAt = time.Now()
	return r.db.WithContext(ctx).Save(fingerprint).Error
}

func (r *SQLiteFingerprintRepository) Delete(ctx context.Context, id int) error {
	return r.db.WithContext(ctx).Model(&entities.Fingerprint{}).Where("id = ?", id).Updates(map[string]interface{}{
		"is_active":  false,
		"updated_at": time.Now(),
	}).Error
}

func (r *SQLiteFingerprintRepository) GetAllTemplates(ctx context.Context) ([]*entities.Fingerprint, error) {
	var fingerprints []*entities.Fingerprint
	err := r.db.WithContext(ctx).Where("is_active = ?", true).Order("user_id, created_at DESC").Find(&fingerprints).Error
	return fingerprints, err
}

func (r *SQLiteFingerprintRepository) LogVerification(ctx context.Context, verification *entities.FingerprintVerification) error {
	verification.VerifiedAt = time.Now()
	return r.db.WithContext(ctx).Create(verification).Error
}
