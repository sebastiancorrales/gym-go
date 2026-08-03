package persistence

import (
	"time"

	"github.com/google/uuid"
	"github.com/sebastiancorrales/gym-go/internal/domain/entities"
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

func (r *SQLiteUserRepository) FindByDocumentAndGym(docNumber string, gymID uuid.UUID) (*entities.User, error) {
	var user entities.User
	err := r.db.Where("document_number = ? AND gym_id = ?", docNumber, gymID).First(&user).Error
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
	err := r.db.Where("gym_id = ?", gymID).Order("created_at DESC").Find(&users).Error
	return users, err
}

// FindByIDs resolves many users in one query, in batches so the number of bound
// parameters stays well under SQLITE_MAX_VARIABLE_NUMBER.
func (r *SQLiteUserRepository) FindByIDs(ids []uuid.UUID) ([]*entities.User, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	const batchSize = 500
	users := make([]*entities.User, 0, len(ids))

	for start := 0; start < len(ids); start += batchSize {
		end := start + batchSize
		if end > len(ids) {
			end = len(ids)
		}

		var batch []*entities.User
		if err := r.db.Where("id IN ?", ids[start:end]).Find(&batch).Error; err != nil {
			return nil, err
		}
		users = append(users, batch...)
	}

	return users, nil
}

// CountByGymIDAndRole counts users of a given role in a gym. The dashboard needs
// the member count and nothing else, so it should not have to download the whole
// user list to compute it.
func (r *SQLiteUserRepository) CountByGymIDAndRole(gymID uuid.UUID, role entities.UserRole) (int64, error) {
	var count int64
	err := r.db.Model(&entities.User{}).
		Where("gym_id = ? AND role = ? AND deleted_at IS NULL", gymID, role).
		Count(&count).Error
	return count, err
}

func (r *SQLiteUserRepository) Update(user *entities.User) error {
	user.UpdatedAt = time.Now().UTC().Round(0)
	return r.db.Save(user).Error
}

func (r *SQLiteUserRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&entities.User{}, id).Error
}

// List returns users across ALL gyms. Prefer FindByGymID for anything user-facing.
// A limit of 0 used to mean "no LIMIT clause at all"; it now falls back to a
// default so this can never emit an unbounded SELECT over the whole table.
func (r *SQLiteUserRepository) List(limit, offset int) ([]*entities.User, error) {
	if limit <= 0 {
		limit = defaultUserRows
	}

	var users []*entities.User
	err := r.db.Order("created_at DESC").Offset(offset).Limit(limit).Find(&users).Error
	warnIfCapped("List(users)", len(users), limit)
	return users, err
}

func (r *SQLiteUserRepository) Count() (int64, error) {
	var count int64
	err := r.db.Model(&entities.User{}).Count(&count).Error
	return count, err
}



