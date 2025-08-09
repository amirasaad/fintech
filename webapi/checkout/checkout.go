package checkout

import (
	"github.com/amirasaad/fintech/pkg/checkout"
	"github.com/amirasaad/fintech/pkg/config"
	"github.com/amirasaad/fintech/pkg/middleware"
	authsvc "github.com/amirasaad/fintech/pkg/service/auth"
	"github.com/amirasaad/fintech/webapi/common"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
	"github.com/golang-jwt/jwt/v5"
)

// Routes registers HTTP routes for checkout-related operations.
func Routes(
	app *fiber.App,
	checkoutSvc *checkout.Service,
	authSvc *authsvc.Service,
	cfg *config.App,
) {
	app.Get(
		"/checkout/sessions/pending",
		middleware.JwtProtected(cfg.Auth.Jwt),
		GetPendingSessions(checkoutSvc, authSvc),
	)
}

// GetPendingSessions returns a Fiber handler for retrieving pending checkout sessions.
// for the current user.
// @Summary Get pending checkout sessions
// @Description Retrieves a list of pending checkout sessions for the authenticated user.
// @Tags checkout
// @Accept json
// @Produce json
// @Success 200 {object} common.Response "Pending sessions fetched"
// @Failure 401 {object} common.ProblemDetails "Unauthorized"
// @Failure 500 {object} common.ProblemDetails "Internal server error"
// @Router /checkout/sessions/pending [get]
// @Security Bearer
func GetPendingSessions(checkoutSvc *checkout.Service, authSvc *authsvc.Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		token, ok := c.Locals("user").(*jwt.Token)
		if !ok {
			return common.ProblemDetailsJSON(c, "Unauthorized", nil, "missing user context")
		}

		userID, err := authSvc.GetCurrentUserId(token)
		if err != nil {
			log.Errorf("Failed to parse user ID from token: %v", err)
			return common.ProblemDetailsJSON(c, "Invalid user ID", err)
		}

		sessions, err := checkoutSvc.GetSessionsByUserID(c.Context(), userID)
		if err != nil {
			log.Errorf("Failed to get pending sessions: %v", err)
			return common.ProblemDetailsJSON(c, "Failed to get pending sessions", err)
		}

		dtos := make([]*SessionDTO, 0, len(sessions))
		for _, s := range sessions {
			if s.Status == "created" {
				dtos = append(dtos, &SessionDTO{
					ID:            s.ID,
					TransactionID: s.TransactionID.String(),
					UserID:        s.UserID.String(),
					AccountID:     s.AccountID.String(),
					Amount:        s.Amount,
					Currency:      s.Currency,
					Status:        s.Status,
					CheckoutURL:   s.CheckoutURL,
					CreatedAt:     s.CreatedAt,
					ExpiresAt:     s.ExpiresAt,
				})
			}
		}

		return common.SuccessResponseJSON(c, fiber.StatusOK, "Pending sessions fetched", dtos)
	}
}
