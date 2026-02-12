package repositories

import (
	"context"
	"github.com/sebastiancorrales/gym-go/internal/domain/entities"
	"time"
)

// AttendanceRepository define las operaciones para persistencia de asistencias
type AttendanceRepository interface {
	Create(ctx context.Context, attendance *entities.Attendance) error
	GetByID(ctx context.Context, id string) (*entities.Attendance, error)
	Update(ctx context.Context, attendance *entities.Attendance) error
	GetActiveAttendance(ctx context.Context, memberID string) (*entities.Attendance, error)
	List(ctx context.Context, filters AttendanceFilters) ([]*entities.Attendance, error)
	GetMemberAttendanceCount(ctx context.Context, memberID string, startDate, endDate time.Time) (int, error)
}

// AttendanceFilters representa los filtros para buscar asistencias
type AttendanceFilters struct {
	MemberID  *string
	ClassID   *string
	StartDate *time.Time
	EndDate   *time.Time
	Limit     int
	Offset    int
}



