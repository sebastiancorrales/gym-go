package persistence

import (
	"context"
	"database/sql"
	"github.com/sebastiancorrales/gym-go/internal/domain/entities"
	"github.com/sebastiancorrales/gym-go/internal/domain/repositories"
	"time"
)

// PostgresMemberRepository es la implementación de MemberRepository usando PostgreSQL
type PostgresMemberRepository struct {
	db *sql.DB
}

// NewPostgresMemberRepository crea una nueva instancia
func NewPostgresMemberRepository(db *sql.DB) repositories.MemberRepository {
	return &PostgresMemberRepository{db: db}
}

// Create crea un nuevo miembro
func (r *PostgresMemberRepository) Create(ctx context.Context, member *entities.Member) error {
	query := `
		INSERT INTO members (id, first_name, last_name, email, phone, date_of_birth, 
			join_date, status, membership_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	_, err := r.db.ExecContext(ctx, query,
		member.ID,
		member.FirstName,
		member.LastName,
		member.Email,
		member.Phone,
		member.DateOfBirth,
		member.JoinDate,
		member.Status,
		member.MembershipID,
		member.CreatedAt,
		member.UpdatedAt,
	)

	return err
}

// GetByID obtiene un miembro por su ID
func (r *PostgresMemberRepository) GetByID(ctx context.Context, id string) (*entities.Member, error) {
	query := `
		SELECT id, first_name, last_name, email, phone, date_of_birth,
			join_date, status, membership_id, created_at, updated_at
		FROM members
		WHERE id = $1
	`

	member := &entities.Member{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&member.ID,
		&member.FirstName,
		&member.LastName,
		&member.Email,
		&member.Phone,
		&member.DateOfBirth,
		&member.JoinDate,
		&member.Status,
		&member.MembershipID,
		&member.CreatedAt,
		&member.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return member, nil
}

// GetByEmail obtiene un miembro por su email
func (r *PostgresMemberRepository) GetByEmail(ctx context.Context, email string) (*entities.Member, error) {
	query := `
		SELECT id, first_name, last_name, email, phone, date_of_birth,
			join_date, status, membership_id, created_at, updated_at
		FROM members
		WHERE email = $1
	`

	member := &entities.Member{}
	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&member.ID,
		&member.FirstName,
		&member.LastName,
		&member.Email,
		&member.Phone,
		&member.DateOfBirth,
		&member.JoinDate,
		&member.Status,
		&member.MembershipID,
		&member.CreatedAt,
		&member.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return member, nil
}

// Update actualiza un miembro
func (r *PostgresMemberRepository) Update(ctx context.Context, member *entities.Member) error {
	query := `
		UPDATE members
		SET first_name = $2, last_name = $3, phone = $4, status = $5,
			membership_id = $6, updated_at = $7
		WHERE id = $1
	`

	_, err := r.db.ExecContext(ctx, query,
		member.ID,
		member.FirstName,
		member.LastName,
		member.Phone,
		member.Status,
		member.MembershipID,
		time.Now(),
	)

	return err
}

// Delete elimina un miembro
func (r *PostgresMemberRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM members WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// List lista miembros con filtros
func (r *PostgresMemberRepository) List(ctx context.Context, filters repositories.MemberFilters) ([]*entities.Member, error) {
	// Implementación básica - se puede extender con más filtros
	query := `
		SELECT id, first_name, last_name, email, phone, date_of_birth,
			join_date, status, membership_id, created_at, updated_at
		FROM members
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	limit := 10
	if filters.Limit > 0 {
		limit = filters.Limit
	}

	rows, err := r.db.QueryContext(ctx, query, limit, filters.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []*entities.Member
	for rows.Next() {
		member := &entities.Member{}
		err := rows.Scan(
			&member.ID,
			&member.FirstName,
			&member.LastName,
			&member.Email,
			&member.Phone,
			&member.DateOfBirth,
			&member.JoinDate,
			&member.Status,
			&member.MembershipID,
			&member.CreatedAt,
			&member.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		members = append(members, member)
	}

	return members, nil
}

// Count cuenta los miembros con filtros
func (r *PostgresMemberRepository) Count(ctx context.Context, filters repositories.MemberFilters) (int64, error) {
	query := `SELECT COUNT(*) FROM members`
	
	var count int64
	err := r.db.QueryRowContext(ctx, query).Scan(&count)
	return count, err
}



