package usecases

import (
	"github.com/google/uuid"
	"github.com/yourusername/gym-go/internal/domain/entities"
	"github.com/yourusername/gym-go/internal/domain/repositories"
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

func (uc *UserUseCase) UpdateUser(user *entities.User) error {
	return uc.userRepo.Update(user)
}

func (uc *UserUseCase) DeleteUser(id uuid.UUID) error {
	return uc.userRepo.Delete(id)
}
