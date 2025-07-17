package common

import (
	"context"
	"errors"
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

// QueryHandler defines the interface for handling queries
type QueryHandler interface {
	HandleQuery(ctx context.Context, query any) (any, error)
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

func GetAccountBalanceQueryHandler(service AccountBalanceService) QueryHandler {
	return &getAccountBalanceQueryHandler{service: service}
}
