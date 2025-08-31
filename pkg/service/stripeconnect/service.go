package stripeconnect

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/config"
	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/handler/common"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v82"
)

// Service defines the interface for Stripe Connect operations
type Service interface {
	// CreateAccount creates a new Stripe Connect account for a user
	CreateAccount(ctx context.Context, userID uuid.UUID) (*stripe.Account, error)

	// GenerateOnboardingURL generates a Stripe onboarding URL for the user
	GenerateOnboardingURL(ctx context.Context, userID uuid.UUID) (string, error)

	// GetAccount retrieves the Stripe Connect account for a user
	GetAccount(ctx context.Context, userID uuid.UUID) (*stripe.Account, error)

	// IsOnboardingComplete checks if the user has completed Stripe onboarding
	IsOnboardingComplete(ctx context.Context, userID uuid.UUID) (bool, error)
}

type stripeConnectService struct {
	client *stripe.Client
	uow    repository.UnitOfWork
	cfg    *config.Stripe
}

// Config holds the configuration for the Stripe Connect serviced

// New creates a new instance of the Stripe Connect service using the official Stripe client
// Deprecated: Use NewClientService instead for better client management
// New creates a new instance of the Stripe Connect service
func New(
	uow repository.UnitOfWork,
	logger *slog.Logger,
	cfg *config.Stripe,
) Service {

	return &stripeConnectService{
		client: stripe.NewClient(cfg.ApiKey),
		uow:    uow,
		cfg:    cfg,
	}
}

func (s *stripeConnectService) CreateAccount(
	ctx context.Context,
	userID uuid.UUID,
) (*stripe.Account, error) {
	userRepo, err := common.GetUserRepository(s.uow, slog.Default())
	if err != nil {
		return nil, fmt.Errorf("failed to get user repository: %w", err)
	}

	// Check if user already has a Stripe account
	existingAccountID, err := userRepo.GetStripeAccountID(ctx, userID)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return nil, fmt.Errorf("failed to check existing account: %w", err)
	}

	if existingAccountID != "" {
		// Account already exists, return it
		acct, err := s.client.V1Accounts.GetByID(ctx, existingAccountID, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to get existing Stripe account: %w", err)
		}
		return acct, nil
	}

	// Create a new Stripe Connect account
	params := &stripe.AccountCreateParams{
		Type:    stripe.String(string(stripe.AccountTypeExpress)),
		Country: stripe.String("US"), // TODO: Make this configurable
		Capabilities: &stripe.AccountCreateCapabilitiesParams{
			CardPayments: &stripe.AccountCreateCapabilitiesCardPaymentsParams{
				Requested: stripe.Bool(true),
			},
			Transfers: &stripe.AccountCreateCapabilitiesTransfersParams{
				Requested: stripe.Bool(true),
			},
		},
	}

	acct, err := s.client.V1Accounts.Create(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to create Stripe account: %w", err)
	}

	// Save the Stripe account ID to the user
	err = userRepo.UpdateStripeAccount(ctx, userID, acct.ID, false)
	if err != nil {
		// Try to clean up the Stripe account if we can't save the reference
		_, _ = s.client.V1Accounts.Delete(ctx, acct.ID, nil) // nolint:errcheck
		return nil, fmt.Errorf("failed to save Stripe account ID: %w", err)
	}

	return acct, nil
}

func (s *stripeConnectService) GenerateOnboardingURL(
	ctx context.Context,
	userID uuid.UUID,
) (string, error) {
	// Get or create Stripe account
	acct, err := s.CreateAccount(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("failed to get or create Stripe account: %w", err)
	}

	// Create account link for onboarding
	params := &stripe.AccountLinkCreateParams{
		Account:    stripe.String(acct.ID),
		RefreshURL: stripe.String(s.cfg.OnboardingRefreshURL),
		ReturnURL:  stripe.String(s.cfg.OnboardingReturnURL),
		Type:       stripe.String("account_onboarding"),
	}

	result, err := s.client.V1AccountLinks.Create(ctx, params)
	if err != nil {
		return "", fmt.Errorf("failed to create account link: %w", err)
	}

	return result.URL, nil
}

func (s *stripeConnectService) GetAccount(
	ctx context.Context,
	userID uuid.UUID,
) (*stripe.Account, error) {
	userRepo, err := common.GetUserRepository(s.uow, slog.Default())
	if err != nil {
		return nil, fmt.Errorf("failed to get user repository: %w", err)
	}
	stripeAccountID, err := userRepo.GetStripeAccountID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get Stripe account ID: %w", err)
	}

	if stripeAccountID == "" {
		return nil, domain.ErrNotFound
	}

	acct, err := s.client.V1Accounts.GetByID(ctx, stripeAccountID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get Stripe account: %w", err)
	}

	return acct, nil
}

func (s *stripeConnectService) IsOnboardingComplete(
	ctx context.Context,
	userID uuid.UUID,
) (bool, error) {
	userRepo, err := common.GetUserRepository(s.uow, slog.Default())
	if err != nil {
		return false, fmt.Errorf("failed to get user repository: %w", err)
	}
	// First check our local database
	status, err := userRepo.GetStripeOnboardingStatus(ctx, userID)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return false, fmt.Errorf("failed to get local onboarding status: %w", err)
	}

	// If we have a local status, return it
	if status {
		return true, nil
	}

	// Otherwise, check with Stripe
	acct, err := s.GetAccount(ctx, userID)
	if err != nil {
		return false, fmt.Errorf("failed to get account: %w", err)
	}

	// Check if onboarding is complete
	onboardingComplete := acct.DetailsSubmitted && acct.PayoutsEnabled

	// Update our local database with the current status
	err = userRepo.UpdateStripeAccount(ctx, userID, acct.ID, onboardingComplete)
	if err != nil {
		return false, fmt.Errorf("failed to update local onboarding status: %w", err)
	}

	return onboardingComplete, nil
}
