package usecases

import (
	"github.com/google/uuid"
	"github.com/sebastiancorrales/gym-go/internal/domain/entities"
	"github.com/sebastiancorrales/gym-go/internal/domain/repositories"
)

type UserUseCase struct {
	userRepo repositories.UserRepository
}

func NewUserUseCase(userRepo repositories.UserRepository) *UserUseCase {
	return &UserUseCase{
		userRepo: userRepo,
	}
}

func (uc *UserUseCase) CreateUser(gymID uuid.UUID, email, firstName, lastName, phone string, role entities.UserRole) (*entities.User, error) {
	user := entities.NewUser(gymID, email, firstName, lastName, role)
	user.Phone = phone

	if err := uc.userRepo.Create(user); err != nil {
		return nil, err
	}

	return user, nil
}

func (uc *UserUseCase) GetUserByID(id uuid.UUID) (*entities.User, error) {
	return uc.userRepo.FindByID(id)
}

func (uc *UserUseCase) ListUsers(limit, offset int) ([]*entities.User, error) {
	return uc.userRepo.List(limit, offset)
}

// ListUsersByGym returns the users of a single gym. Prefer this over ListUsers:
// List has no gym filter, so callers were loading every user of every gym and
// discarding the rest in memory.
func (uc *UserUseCase) ListUsersByGym(gymID uuid.UUID) ([]*entities.User, error) {
	return uc.userRepo.FindByGymID(gymID)
}

// CountMembers returns how many members (role MEMBER) belong to a gym.
func (uc *UserUseCase) CountMembers(gymID uuid.UUID) (int64, error) {
	return uc.userRepo.CountByGymIDAndRole(gymID, entities.RoleMember)
}

// GetUsersByIDs resolves many users in one query and returns them keyed by ID,
// ready to be used as a lookup map when assembling a listing response.
func (uc *UserUseCase) GetUsersByIDs(ids []uuid.UUID) (map[uuid.UUID]*entities.User, error) {
	users, err := uc.userRepo.FindByIDs(ids)
	if err != nil {
		return nil, err
	}

	byID := make(map[uuid.UUID]*entities.User, len(users))
	for _, u := range users {
		byID[u.ID] = u
	}
	return byID, nil
}

func (uc *UserUseCase) UpdateUser(user *entities.User) error {
	return uc.userRepo.Update(user)
}

func (uc *UserUseCase) DeleteUser(id uuid.UUID) error {
	return uc.userRepo.Delete(id)
}

func (uc *UserUseCase) FindByDocumentAndGym(docNumber string, gymID uuid.UUID) (*entities.User, error) {
	return uc.userRepo.FindByDocumentAndGym(docNumber, gymID)
}

func (uc *UserUseCase) DeactivateUser(id uuid.UUID) error {
	user, err := uc.userRepo.FindByID(id)
	if err != nil {
		return err
	}

	user.Status = "INACTIVE"
	return uc.userRepo.Update(user)
}
