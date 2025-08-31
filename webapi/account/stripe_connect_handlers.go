package account

import (
	"context"

	authsvc "github.com/amirasaad/fintech/pkg/service/auth"
	"github.com/amirasaad/fintech/pkg/service/stripeconnect"
	"github.com/amirasaad/fintech/webapi/account/dto"
	"github.com/amirasaad/fintech/webapi/common"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// getUserIDFromContext extracts the user ID from the JWT token in the context
func (h *StripeConnectHandlers) getUserIDFromContext(c *fiber.Ctx) (uuid.UUID, error) {
	token, ok := c.Locals("user").(*jwt.Token)
	if !ok {
		return uuid.Nil, common.ProblemDetailsJSON(c, "Unauthorized", nil, "missing user context")
	}
	userID, err := h.authSvc.GetCurrentUserId(token)
	if err != nil {
		log.Errorf("Failed to parse user ID from token: %v", err)
		return uuid.Nil, common.ProblemDetailsJSON(c, "Invalid user ID", err)
	}
	return userID, nil
}

type StripeConnectHandlers struct {
	stripeConnectSvc stripeconnect.Service
	authSvc          *authsvc.Service
}

func NewStripeConnectHandlers(
	stripeConnectSvc stripeconnect.Service,
	authSvc *authsvc.Service,
) *StripeConnectHandlers {
	return &StripeConnectHandlers{
		stripeConnectSvc: stripeConnectSvc,
		authSvc:          authSvc,
	}
}

// InitiateOnboarding initiates the Stripe Connect onboarding flow
// @Summary Initiate Stripe Connect onboarding
// @Description Generates a Stripe Connect onboarding URL for the authenticated user
// @Tags account
// @Produce json
// @Security BearerAuth
// @Success 200 {object} dto.InitiateOnboardingResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /stripe/account/onboard [post]
func (h *StripeConnectHandlers) InitiateOnboarding(c *fiber.Ctx) error {
	// Get user ID from JWT token
	userID, err := h.getUserIDFromContext(c)
	if err != nil {
		return common.ProblemDetailsJSON(c, err.Error(), err)
	}

	onboardingURL, err := h.stripeConnectSvc.GenerateOnboardingURL(
		context.Background(),
		userID,
	)
	if err != nil {
		return common.ProblemDetailsJSON(c, "Failed to generate onboarding URL", err)
	}

	return common.SuccessResponseJSON(
		c,
		fiber.StatusOK,
		"Onboarding URL generated successfully",
		onboardingURL,
	)
}

// GetOnboardingStatus checks if the authenticated user has completed
// the Stripe Connect onboarding process
// @Summary Check Stripe Connect onboarding status
// @Description Returns the onboarding completion status
// for the authenticated user's Stripe Connect account
// @Tags account
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} dto.OnboardingStatusResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /stripe/account/onboard/status [get]
func (h *StripeConnectHandlers) GetOnboardingStatus(c *fiber.Ctx) error {
	// Get user ID from JWT token
	userID, err := h.getUserIDFromContext(c)
	if err != nil {
		return common.ProblemDetailsJSON(c, err.Error(), err)
	}

	isComplete, err := h.stripeConnectSvc.IsOnboardingComplete(context.Background(), userID)
	if err != nil {
		return common.ProblemDetailsJSON(c, "Failed to get onboarding status", err)
	}

	return common.SuccessResponseJSON(
		c,
		fiber.StatusOK,
		"Onboarding status retrieved successfully",
		dto.OnboardingStatusResponse{
			IsComplete: isComplete,
		},
	)
}

// MapRoutes maps the Stripe Connect routes to the router with the API version prefix
func (h *StripeConnectHandlers) MapRoutes(router fiber.Router, jwtMiddleware fiber.Handler) {
	// Stripe Connect onboarding routes
	onboardGroup := router.Group("/account/onboard")
	onboardGroup.Use(jwtMiddleware) // Add JWT protection to all routes in this group
	{
		onboardGroup.Post("/", h.InitiateOnboarding)
		onboardGroup.Get("/status", h.GetOnboardingStatus)
	}
}
