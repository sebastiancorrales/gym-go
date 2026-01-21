package usecases

import (
	"context"
	"errors"
	"github.com/yourusername/gym-go/internal/domain/entities"
	"github.com/yourusername/gym-go/internal/domain/repositories"
	"time"

	"github.com/google/uuid"
)

// ClassUseCase maneja la lógica de negocio relacionada con clases
type ClassUseCase struct {
	classRepo      repositories.ClassRepository
	instructorRepo repositories.InstructorRepository
}

// NewClassUseCase crea una nueva instancia de ClassUseCase
func NewClassUseCase(
	classRepo repositories.ClassRepository,
	instructorRepo repositories.InstructorRepository,
) *ClassUseCase {
	return &ClassUseCase{
		classRepo:      classRepo,
		instructorRepo: instructorRepo,
	}
}

// CreateClass crea una nueva clase
func (uc *ClassUseCase) CreateClass(ctx context.Context, class *entities.Class) error {
	// Validaciones de negocio
	if class.Name == "" {
		return errors.New("el nombre de la clase es requerido")
	}
	if class.InstructorID == "" {
		return errors.New("el instructor es requerido")
	}
	if class.Capacity <= 0 {
		return errors.New("la capacidad debe ser mayor a cero")
	}

	// Verificar que el instructor existe y está activo
	instructor, err := uc.instructorRepo.GetByID(ctx, class.InstructorID)
	if err != nil {
		return err
	}
	if instructor == nil {
		return errors.New("instructor no encontrado")
	}
	if !instructor.IsActive {
		return errors.New("el instructor no está activo")
	}

	// Generar ID y establecer valores por defecto
	class.ID = uuid.New().String()
	class.Status = entities.ClassStatusScheduled
	class.CreatedAt = time.Now()
	class.UpdatedAt = time.Now()

	return uc.classRepo.Create(ctx, class)
}

// GetClassByID obtiene una clase por su ID
func (uc *ClassUseCase) GetClassByID(ctx context.Context, id string) (*entities.Class, error) {
	if id == "" {
		return nil, errors.New("el ID es requerido")
	}

	return uc.classRepo.GetByID(ctx, id)
}

// UpdateClass actualiza la información de una clase
func (uc *ClassUseCase) UpdateClass(ctx context.Context, class *entities.Class) error {
	if class.ID == "" {
		return errors.New("el ID es requerido")
	}

	// Verificar que la clase existe
	existingClass, err := uc.classRepo.GetByID(ctx, class.ID)
	if err != nil {
		return err
	}
	if existingClass == nil {
		return errors.New("clase no encontrada")
	}

	class.UpdatedAt = time.Now()
	return uc.classRepo.Update(ctx, class)
}

// CancelClass cancela una clase
func (uc *ClassUseCase) CancelClass(ctx context.Context, classID string) error {
	class, err := uc.classRepo.GetByID(ctx, classID)
	if err != nil {
		return err
	}
	if class == nil {
		return errors.New("clase no encontrada")
	}

	class.Cancel()
	return uc.classRepo.Update(ctx, class)
}

// StartClass inicia una clase
func (uc *ClassUseCase) StartClass(ctx context.Context, classID string) error {
	class, err := uc.classRepo.GetByID(ctx, classID)
	if err != nil {
		return err
	}
	if class == nil {
		return errors.New("clase no encontrada")
	}
	if class.Status != entities.ClassStatusScheduled {
		return errors.New("solo se pueden iniciar clases programadas")
	}

	class.Start()
	return uc.classRepo.Update(ctx, class)
}

// CompleteClass completa una clase
func (uc *ClassUseCase) CompleteClass(ctx context.Context, classID string) error {
	class, err := uc.classRepo.GetByID(ctx, classID)
	if err != nil {
		return err
	}
	if class == nil {
		return errors.New("clase no encontrada")
	}
	if class.Status != entities.ClassStatusOngoing {
		return errors.New("solo se pueden completar clases en curso")
	}

	class.Complete()
	return uc.classRepo.Update(ctx, class)
}

// ListClasses lista clases con filtros
func (uc *ClassUseCase) ListClasses(ctx context.Context, filters repositories.ClassFilters) ([]*entities.Class, error) {
	return uc.classRepo.List(ctx, filters)
}



