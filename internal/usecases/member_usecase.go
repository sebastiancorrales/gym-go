package usecases

import (
	"context"
	"errors"
	"gym-go/internal/domain/entities"
	"gym-go/internal/domain/repositories"
	"time"

	"github.com/google/uuid"
)

// MemberUseCase maneja la lógica de negocio relacionada con miembros
// Sigue el principio de Responsabilidad Única (SRP)
type MemberUseCase struct {
	memberRepo     repositories.MemberRepository
	membershipRepo repositories.MembershipRepository
}

// NewMemberUseCase crea una nueva instancia de MemberUseCase
func NewMemberUseCase(
	memberRepo repositories.MemberRepository,
	membershipRepo repositories.MembershipRepository,
) *MemberUseCase {
	return &MemberUseCase{
		memberRepo:     memberRepo,
		membershipRepo: membershipRepo,
	}
}

// CreateMember crea un nuevo miembro
func (uc *MemberUseCase) CreateMember(ctx context.Context, member *entities.Member) error {
	// Validaciones de negocio
	if member.Email == "" {
		return errors.New("el email es requerido")
	}

	// Verificar si el email ya existe
	existingMember, err := uc.memberRepo.GetByEmail(ctx, member.Email)
	if err == nil && existingMember != nil {
		return errors.New("el email ya está registrado")
	}

	// Generar ID y establecer valores por defecto
	member.ID = uuid.New().String()
	member.JoinDate = time.Now()
	member.Status = entities.MemberStatusActive
	member.CreatedAt = time.Now()
	member.UpdatedAt = time.Now()

	return uc.memberRepo.Create(ctx, member)
}

// GetMemberByID obtiene un miembro por su ID
func (uc *MemberUseCase) GetMemberByID(ctx context.Context, id string) (*entities.Member, error) {
	if id == "" {
		return nil, errors.New("el ID es requerido")
	}

	return uc.memberRepo.GetByID(ctx, id)
}

// UpdateMember actualiza la información de un miembro
func (uc *MemberUseCase) UpdateMember(ctx context.Context, member *entities.Member) error {
	if member.ID == "" {
		return errors.New("el ID es requerido")
	}

	// Verificar que el miembro existe
	existingMember, err := uc.memberRepo.GetByID(ctx, member.ID)
	if err != nil {
		return err
	}
	if existingMember == nil {
		return errors.New("miembro no encontrado")
	}

	member.UpdatedAt = time.Now()
	return uc.memberRepo.Update(ctx, member)
}

// AssignMembership asigna una membresía a un miembro
func (uc *MemberUseCase) AssignMembership(ctx context.Context, memberID, membershipID string) error {
	// Verificar que el miembro existe
	member, err := uc.memberRepo.GetByID(ctx, memberID)
	if err != nil {
		return err
	}
	if member == nil {
		return errors.New("miembro no encontrado")
	}

	// Verificar que la membresía existe y está activa
	membership, err := uc.membershipRepo.GetByID(ctx, membershipID)
	if err != nil {
		return err
	}
	if membership == nil {
		return errors.New("membresía no encontrada")
	}
	if !membership.IsActive {
		return errors.New("la membresía no está activa")
	}

	member.MembershipID = &membershipID
	member.UpdatedAt = time.Now()

	return uc.memberRepo.Update(ctx, member)
}

// ListMembers lista miembros con filtros
func (uc *MemberUseCase) ListMembers(ctx context.Context, filters repositories.MemberFilters) ([]*entities.Member, error) {
	return uc.memberRepo.List(ctx, filters)
}

// SuspendMember suspende la membresía de un miembro
func (uc *MemberUseCase) SuspendMember(ctx context.Context, memberID string) error {
	member, err := uc.memberRepo.GetByID(ctx, memberID)
	if err != nil {
		return err
	}
	if member == nil {
		return errors.New("miembro no encontrado")
	}

	member.Suspend()
	return uc.memberRepo.Update(ctx, member)
}

// ActivateMember activa la membresía de un miembro
func (uc *MemberUseCase) ActivateMember(ctx context.Context, memberID string) error {
	member, err := uc.memberRepo.GetByID(ctx, memberID)
	if err != nil {
		return err
	}
	if member == nil {
		return errors.New("miembro no encontrado")
	}

	member.Activate()
	return uc.memberRepo.Update(ctx, member)
}
