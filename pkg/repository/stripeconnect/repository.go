package stripeconnect

import (
	"context"

	"github.com/amirasaad/fintech/pkg/domain"
)

// Repository defines the interface for Stripe Connect related operations
type Repository interface {
	// SaveStripeAccountID saves the Stripe Connect account ID for a user
	SaveStripeAccountID(ctx context.Context, userID, accountID string) error

	// GetStripeAccountID retrieves the Stripe Connect account ID for a user
	GetStripeAccountID(ctx context.Context, userID string) (string, error)

	// UpdateOnboardingStatus updates the onboarding status for a user's Stripe account
	UpdateOnboardingStatus(ctx context.Context, userID string, completed bool) error

	// GetOnboardingStatus checks if the user has completed Stripe onboarding
	GetOnboardingStatus(ctx context.Context, userID string) (bool, error)
}

// NewRepository creates a new Stripe Connect repository
func NewRepository(userRepo UserRepository) Repository {
	return &repository{
		userRepo: userRepo,
	}
}

type repository struct {
	userRepo UserRepository
}

// UserRepository defines the minimal user repository interface needed by Stripe Connect
// This avoids circular dependencies between packages
type UserRepository interface {
	// GetStripeAccountID gets the Stripe account ID for a user
	GetStripeAccountID(ctx context.Context, userID string) (string, error)

	// GetStripeOnboardingStatus gets the Stripe onboarding status for a user
	GetStripeOnboardingStatus(ctx context.Context, userID string) (bool, error)

	// UpdateStripeAccount updates the Stripe account information for a user
	UpdateStripeAccount(
		ctx context.Context,
		userID, accountID string,
		onboardingComplete bool,
	) error

	// UpdateStripeOnboardingStatus updates the Stripe onboarding status for a user
	UpdateStripeOnboardingStatus(ctx context.Context, userID string, completed bool) error
}

func (r *repository) SaveStripeAccountID(ctx context.Context, userID, accountID string) error {
	// Delegate to the user repository to handle the actual database operation
	return r.userRepo.UpdateStripeAccount(
		ctx,
		userID,
		accountID,
		false,
	)
}

func (r *repository) GetStripeAccountID(ctx context.Context, userID string) (string, error) {
	// Delegate to the user repository to get the account ID
	accountID, err := r.userRepo.GetStripeAccountID(ctx, userID)
	if err != nil {
		return "", err
	}

	if accountID == "" {
		return "", domain.ErrNotFound
	}

	return accountID, nil
}

func (r *repository) UpdateOnboardingStatus(
	ctx context.Context,
	userID string,
	completed bool,
) error {
	// Delegate to the user repository to update the onboarding status
	return r.userRepo.UpdateStripeOnboardingStatus(ctx, userID, completed)
}

func (r *repository) GetOnboardingStatus(ctx context.Context, userID string) (bool, error) {
	// Delegate to the user repository to get the onboarding status
	status, err := r.userRepo.GetStripeOnboardingStatus(ctx, userID)
	if err != nil {
		return false, err
	}

	return status, nil
}
