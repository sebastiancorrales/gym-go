package repositories

import (
	"context"

	"github.com/sebastiancorrales/gym-go/internal/domain/entities"
)

// FingerprintRepository defines the interface for fingerprint data access
type FingerprintRepository interface {
	// Create stores a new fingerprint template
	Create(ctx context.Context, fingerprint *entities.Fingerprint) error

	// GetByID retrieves a fingerprint by its ID
	GetByID(ctx context.Context, id int) (*entities.Fingerprint, error)

	// GetByUserID retrieves all fingerprints for a user
	GetByUserID(ctx context.Context, userID string) ([]*entities.Fingerprint, error)

	// GetActiveByUserID retrieves all active fingerprints for a user
	GetActiveByUserID(ctx context.Context, userID string) ([]*entities.Fingerprint, error)

	// Update updates a fingerprint
	Update(ctx context.Context, fingerprint *entities.Fingerprint) error

	// Delete soft deletes a fingerprint (sets IsActive to false)
	Delete(ctx context.Context, id int) error

	// GetAllTemplates retrieves the active templates the DigitalPersona
	// matcher can read. It deliberately excludes templates captured on a
	// Hikvision terminal: those are a different, incompatible format that the
	// device matches by itself.
	GetAllTemplates(ctx context.Context) ([]*entities.Fingerprint, error)

	// GetActiveByUserAndSource retrieves a user's active templates for one
	// reader kind (see entities.FingerprintSource*).
	GetActiveByUserAndSource(ctx context.Context, userID, source string) ([]*entities.Fingerprint, error)

	// DistinctUserIDsBySource lists the users with at least one active
	// template from a reader kind — i.e. who is actually enrolled on it.
	DistinctUserIDsBySource(ctx context.Context, source string) ([]string, error)

	// CountActiveByUser returns how many active fingerprints each user has, keyed
	// by user ID. Users with none are absent from the map. This exists so a list
	// screen can resolve the whole column in one query instead of one request per
	// row, and it deliberately does not read the template blobs.
	CountActiveByUser(ctx context.Context) (map[string]int, error)

	// LogVerification logs a fingerprint verification attempt
	LogVerification(ctx context.Context, verification *entities.FingerprintVerification) error
}
