package usecases

import (
	"context"
	"errors"
	"github.com/yourusername/gym-go/internal/domain/entities"
	"github.com/yourusername/gym-go/internal/domain/repositories"
	"time"

	"github.com/google/uuid"
)

// AttendanceUseCase maneja la lógica de negocio relacionada con asistencias
type AttendanceUseCase struct {
	attendanceRepo repositories.AttendanceRepository
	memberRepo     repositories.MemberRepository
	classRepo      repositories.ClassRepository
}

// NewAttendanceUseCase crea una nueva instancia de AttendanceUseCase
func NewAttendanceUseCase(
	attendanceRepo repositories.AttendanceRepository,
	memberRepo repositories.MemberRepository,
	classRepo repositories.ClassRepository,
) *AttendanceUseCase {
	return &AttendanceUseCase{
		attendanceRepo: attendanceRepo,
		memberRepo:     memberRepo,
		classRepo:      classRepo,
	}
}

// CheckIn registra el ingreso de un miembro al gimnasio
func (uc *AttendanceUseCase) CheckIn(ctx context.Context, memberID string, classID *string) (*entities.Attendance, error) {
	// Verificar que el miembro existe y está activo
	member, err := uc.memberRepo.GetByID(ctx, memberID)
	if err != nil {
		return nil, err
	}
	if member == nil {
		return nil, errors.New("miembro no encontrado")
	}
	if !member.IsActive() {
		return nil, errors.New("el miembro no está activo")
	}

	// Verificar que no tenga un check-in activo
	activeAttendance, err := uc.attendanceRepo.GetActiveAttendance(ctx, memberID)
	if err == nil && activeAttendance != nil {
		return nil, errors.New("el miembro ya tiene un check-in activo")
	}

	// Si es para una clase específica, validar
	if classID != nil {
		class, err := uc.classRepo.GetByID(ctx, *classID)
		if err != nil {
			return nil, err
		}
		if class == nil {
			return nil, errors.New("clase no encontrada")
		}

		enrollmentCount, err := uc.classRepo.GetEnrollmentCount(ctx, *classID)
		if err != nil {
			return nil, err
		}

		if !class.CanEnroll(enrollmentCount) {
			return nil, errors.New("no se puede inscribir en esta clase")
		}
	}

	// Crear registro de asistencia
	attendance := &entities.Attendance{
		ID:        uuid.New().String(),
		MemberID:  memberID,
		ClassID:   classID,
		CheckIn:   time.Now(),
		CreatedAt: time.Now(),
	}

	if err := uc.attendanceRepo.Create(ctx, attendance); err != nil {
		return nil, err
	}

	return attendance, nil
}

// CheckOut registra la salida de un miembro del gimnasio
func (uc *AttendanceUseCase) CheckOut(ctx context.Context, memberID string) error {
	// Obtener asistencia activa
	attendance, err := uc.attendanceRepo.GetActiveAttendance(ctx, memberID)
	if err != nil {
		return err
	}
	if attendance == nil {
		return errors.New("no hay un check-in activo para este miembro")
	}

	// Registrar check-out
	now := time.Now()
	attendance.CheckOut = &now

	return uc.attendanceRepo.Update(ctx, attendance)
}

// GetMemberAttendanceHistory obtiene el historial de asistencias de un miembro
func (uc *AttendanceUseCase) GetMemberAttendanceHistory(
	ctx context.Context,
	memberID string,
	startDate, endDate time.Time,
) ([]*entities.Attendance, error) {
	filters := repositories.AttendanceFilters{
		MemberID:  &memberID,
		StartDate: &startDate,
		EndDate:   &endDate,
	}

	return uc.attendanceRepo.List(ctx, filters)
}

// GetAttendanceStats obtiene estadísticas de asistencia de un miembro
func (uc *AttendanceUseCase) GetAttendanceStats(
	ctx context.Context,
	memberID string,
	startDate, endDate time.Time,
) (int, error) {
	return uc.attendanceRepo.GetMemberAttendanceCount(ctx, memberID, startDate, endDate)
}



