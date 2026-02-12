package persistence

import (
	"time"

	"github.com/google/uuid"
	"github.com/sebastiancorrales/gym-go/internal/domain/entities"
	"gorm.io/gorm"
)

type SQLitePlanRepository struct {
	db *gorm.DB
}

func NewSQLitePlanRepository(db *gorm.DB) *SQLitePlanRepository {
	return &SQLitePlanRepository{db: db}
}

func (r *SQLitePlanRepository) Create(plan *entities.Plan) error {
	return r.db.Create(plan).Error
}

func (r *SQLitePlanRepository) FindByID(id uuid.UUID) (*entities.Plan, error) {
	var plan entities.Plan
	err := r.db.Where("id = ?", id).First(&plan).Error
	if err != nil {
		return nil, err
	}
	return &plan, nil
}

func (r *SQLitePlanRepository) FindByGymID(gymID uuid.UUID) ([]*entities.Plan, error) {
	var plans []*entities.Plan
	err := r.db.Where("gym_id = ?", gymID).Find(&plans).Error
	return plans, err
}

func (r *SQLitePlanRepository) FindActiveByGymID(gymID uuid.UUID) ([]*entities.Plan, error) {
	var plans []*entities.Plan
	err := r.db.Where("gym_id = ? AND status = ?", gymID, "ACTIVE").Find(&plans).Error
	return plans, err
}

func (r *SQLitePlanRepository) Update(plan *entities.Plan) error {
	plan.UpdatedAt = time.Now()
	return r.db.Save(plan).Error
}

func (r *SQLitePlanRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&entities.Plan{}, id).Error
}

func (r *SQLitePlanRepository) List(limit, offset int) ([]*entities.Plan, error) {
	var plans []*entities.Plan
	err := r.db.Limit(limit).Offset(offset).Find(&plans).Error
	return plans, err
}
