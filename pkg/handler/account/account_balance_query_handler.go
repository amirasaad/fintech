package account

import (
	"context"
	"errors"

	"github.com/amirasaad/fintech/pkg/handler"
)

type GetAccountBalanceQuery struct {
	AccountID string
	UserID    string
}

type GetAccountBalanceResult struct {
	Balance  float64
	Currency string
}

type AccountBalanceService interface {
	GetBalance(ctx context.Context, userID, accountID string) (float64, string, error)
}

type getAccountBalanceQueryHandler struct {
	service AccountBalanceService
}

func (h *getAccountBalanceQueryHandler) HandleQuery(ctx context.Context, query any) (any, error) {
	q, ok := query.(GetAccountBalanceQuery)
	if !ok {
		return nil, errors.New("invalid query type")
	}
	bal, cur, err := h.service.GetBalance(ctx, q.UserID, q.AccountID)
	if err != nil {
		return nil, err
	}
	return GetAccountBalanceResult{Balance: bal, Currency: cur}, nil
}

func GetAccountBalanceQueryHandler(service AccountBalanceService) handler.QueryHandler {
	return &getAccountBalanceQueryHandler{service: service}
}
