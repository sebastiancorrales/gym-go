package repositories

import (
	"context"

	"github.com/yourusername/gym-go/internal/domain/entities"
)

// FingerprintRepository defines the interface for fingerprint data access
type FingerprintRepository interface {
	// Create stores a new fingerprint template
	Create(ctx context.Context, fingerprint *entities.Fingerprint) error

	// GetByID retrieves a fingerprint by its ID
	GetByID(ctx context.Context, id int) (*entities.Fingerprint, error)

	// GetByUserID retrieves all fingerprints for a user
	GetByUserID(ctx context.Context, userID int) ([]*entities.Fingerprint, error)

	// GetActiveByUserID retrieves all active fingerprints for a user
	GetActiveByUserID(ctx context.Context, userID int) ([]*entities.Fingerprint, error)

	// Update updates a fingerprint
	Update(ctx context.Context, fingerprint *entities.Fingerprint) error

	// Delete soft deletes a fingerprint (sets IsActive to false)
	Delete(ctx context.Context, id int) error

	// GetAllTemplates retrieves all active fingerprint templates for verification
	GetAllTemplates(ctx context.Context) ([]*entities.Fingerprint, error)

	// LogVerification logs a fingerprint verification attempt
	LogVerification(ctx context.Context, verification *entities.FingerprintVerification) error
}
