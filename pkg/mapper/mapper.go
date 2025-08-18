package mapper

import (
	"fmt"

	"github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/dto"
	"github.com/amirasaad/fintech/pkg/money"
)

// MapAccountReadToDomain maps a dto.AccountRead to a domain Account.
func MapAccountReadToDomain(dto *dto.AccountRead) (*account.Account, error) {
	balance, err := money.New(dto.Balance, money.Code(dto.Currency))
	if err != nil {
		return nil, fmt.Errorf("error creating money from dto: %w", err)
	}
	acc, err := account.New().
		WithID(dto.ID).
		WithUserID(dto.UserID).
		WithBalance(balance.Amount()).
		WithCurrency(money.Code(balance.Currency().String())).
		WithCreatedAt(dto.CreatedAt).
		WithUpdatedAt(dto.UpdatedAt).
		Build()
	if err != nil {
		return nil, fmt.Errorf("error creating account from dto: %w", err)
	}
	return acc, nil
}
