package queries

import "github.com/google/uuid"

type GetAccountQuery struct {
	AccountID uuid.UUID
	UserID    uuid.UUID
}

func (q GetAccountQuery) EventType() string { return "GetAccountQuery" }

type GetAccountResult struct {
	AccountID uuid.UUID
	UserID    uuid.UUID
	Balance   float64
	Currency  string
}
