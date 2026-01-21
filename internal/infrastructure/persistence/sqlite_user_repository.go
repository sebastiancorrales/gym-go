package persistence

import (
	"time"

	"github.com/google/uuid"
	"github.com/yourusername/gym-go/internal/domain/entities"
	"gorm.io/gorm"
)

// SQLiteUserRepository implements UserRepository for SQLite
type SQLiteUserRepository struct {
	db *gorm.DB
}

// NewSQLiteUserRepository creates a new SQLite user repository
func NewSQLiteUserRepository(db *gorm.DB) *SQLiteUserRepository {
	return &SQLiteUserRepository{db: db}
}

func (r *SQLiteUserRepository) Create(user *entities.User) error {
	return r.db.Create(user).Error
}

func (r *SQLiteUserRepository) FindByID(id uuid.UUID) (*entities.User, error) {
	var user entities.User
	err := r.db.Where("id = ?", id).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *SQLiteUserRepository) FindByEmail(email string) (*entities.User, error) {
	var user entities.User
	err := r.db.Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *SQLiteUserRepository) FindByGymID(gymID uuid.UUID) ([]*entities.User, error) {
	var users []*entities.User
	err := r.db.Where("gym_id = ?", gymID).Find(&users).Error
	return users, err
}

func (r *SQLiteUserRepository) Update(user *entities.User) error {
	user.UpdatedAt = time.Now()
	return r.db.Save(user).Error
}

func (r *SQLiteUserRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&entities.User{}, id).Error
}

func (r *SQLiteUserRepository) List(limit, offset int) ([]*entities.User, error) {
	var users []*entities.User
	err := r.db.Limit(limit).Offset(offset).Find(&users).Error
	return users, err
}

func (r *SQLiteUserRepository) Count() (int64, error) {
	var count int64
	err := r.db.Model(&entities.User{}).Count(&count).Error
	return count, err
}



