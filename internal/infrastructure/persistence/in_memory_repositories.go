package persistence

import (
	"context"
	"github.com/yourusername/gym-go/internal/domain/entities"
	"github.com/yourusername/gym-go/internal/domain/repositories"
	"sync"
	"time"
)

// InMemoryMemberRepository es una implementación en memoria para pruebas
type InMemoryMemberRepository struct {
	members map[string]*entities.Member
	mu      sync.RWMutex
}

func NewInMemoryMemberRepository() repositories.MemberRepository {
	return &InMemoryMemberRepository{
		members: make(map[string]*entities.Member),
	}
}

func (r *InMemoryMemberRepository) Create(ctx context.Context, member *entities.Member) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.members[member.ID] = member
	return nil
}

func (r *InMemoryMemberRepository) GetByID(ctx context.Context, id string) (*entities.Member, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	member, exists := r.members[id]
	if !exists {
		return nil, nil
	}
	return member, nil
}

func (r *InMemoryMemberRepository) GetByEmail(ctx context.Context, email string) (*entities.Member, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, member := range r.members {
		if member.Email == email {
			return member, nil
		}
	}
	return nil, nil
}

func (r *InMemoryMemberRepository) Update(ctx context.Context, member *entities.Member) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.members[member.ID] = member
	return nil
}

func (r *InMemoryMemberRepository) Delete(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.members, id)
	return nil
}

func (r *InMemoryMemberRepository) List(ctx context.Context, filters repositories.MemberFilters) ([]*entities.Member, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []*entities.Member
	for _, member := range r.members {
		result = append(result, member)
	}
	return result, nil
}

func (r *InMemoryMemberRepository) Count(ctx context.Context, filters repositories.MemberFilters) (int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return int64(len(r.members)), nil
}

// Implementaciones stub para otros repositorios
type InMemoryMembershipRepository struct{}

func NewInMemoryMembershipRepository() repositories.MembershipRepository {
	return &InMemoryMembershipRepository{}
}

func (r *InMemoryMembershipRepository) Create(ctx context.Context, membership *entities.Membership) error {
	return nil
}
func (r *InMemoryMembershipRepository) GetByID(ctx context.Context, id string) (*entities.Membership, error) {
	return &entities.Membership{ID: id, IsActive: true}, nil
}
func (r *InMemoryMembershipRepository) Update(ctx context.Context, membership *entities.Membership) error {
	return nil
}
func (r *InMemoryMembershipRepository) Delete(ctx context.Context, id string) error {
	return nil
}
func (r *InMemoryMembershipRepository) List(ctx context.Context, filters repositories.MembershipFilters) ([]*entities.Membership, error) {
	return []*entities.Membership{}, nil
}
func (r *InMemoryMembershipRepository) GetActive(ctx context.Context) ([]*entities.Membership, error) {
	return []*entities.Membership{}, nil
}

type InMemoryClassRepository struct{}

func NewInMemoryClassRepository() repositories.ClassRepository {
	return &InMemoryClassRepository{}
}

func (r *InMemoryClassRepository) Create(ctx context.Context, class *entities.Class) error {
	return nil
}
func (r *InMemoryClassRepository) GetByID(ctx context.Context, id string) (*entities.Class, error) {
	return &entities.Class{ID: id}, nil
}
func (r *InMemoryClassRepository) Update(ctx context.Context, class *entities.Class) error {
	return nil
}
func (r *InMemoryClassRepository) Delete(ctx context.Context, id string) error {
	return nil
}
func (r *InMemoryClassRepository) List(ctx context.Context, filters repositories.ClassFilters) ([]*entities.Class, error) {
	return []*entities.Class{}, nil
}
func (r *InMemoryClassRepository) GetEnrollmentCount(ctx context.Context, classID string) (int, error) {
	return 0, nil
}

type InMemoryInstructorRepository struct{}

func NewInMemoryInstructorRepository() repositories.InstructorRepository {
	return &InMemoryInstructorRepository{}
}

func (r *InMemoryInstructorRepository) Create(ctx context.Context, instructor *entities.Instructor) error {
	return nil
}
func (r *InMemoryInstructorRepository) GetByID(ctx context.Context, id string) (*entities.Instructor, error) {
	return &entities.Instructor{ID: id, IsActive: true}, nil
}
func (r *InMemoryInstructorRepository) GetByEmail(ctx context.Context, email string) (*entities.Instructor, error) {
	return nil, nil
}
func (r *InMemoryInstructorRepository) Update(ctx context.Context, instructor *entities.Instructor) error {
	return nil
}
func (r *InMemoryInstructorRepository) Delete(ctx context.Context, id string) error {
	return nil
}
func (r *InMemoryInstructorRepository) List(ctx context.Context, filters repositories.InstructorFilters) ([]*entities.Instructor, error) {
	return []*entities.Instructor{}, nil
}
func (r *InMemoryInstructorRepository) GetActive(ctx context.Context) ([]*entities.Instructor, error) {
	return []*entities.Instructor{}, nil
}

type InMemoryAttendanceRepository struct{}

func NewInMemoryAttendanceRepository() repositories.AttendanceRepository {
	return &InMemoryAttendanceRepository{}
}

func (r *InMemoryAttendanceRepository) Create(ctx context.Context, attendance *entities.Attendance) error {
	return nil
}
func (r *InMemoryAttendanceRepository) GetByID(ctx context.Context, id string) (*entities.Attendance, error) {
	return nil, nil
}
func (r *InMemoryAttendanceRepository) Update(ctx context.Context, attendance *entities.Attendance) error {
	return nil
}
func (r *InMemoryAttendanceRepository) GetActiveAttendance(ctx context.Context, memberID string) (*entities.Attendance, error) {
	return nil, nil
}
func (r *InMemoryAttendanceRepository) List(ctx context.Context, filters repositories.AttendanceFilters) ([]*entities.Attendance, error) {
	return []*entities.Attendance{}, nil
}
func (r *InMemoryAttendanceRepository) GetMemberAttendanceCount(ctx context.Context, memberID string, startDate, endDate time.Time) (int, error) {
	return 0, nil
}



