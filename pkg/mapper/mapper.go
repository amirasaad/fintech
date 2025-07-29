package mapper

import (
	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/amirasaad/fintech/pkg/dto"
)

// MapAccountReadToDomain maps a dto.AccountRead to a domain Account.
func MapAccountReadToDomain(dto *dto.AccountRead) *account.Account {
	balance, err := money.New(dto.Balance, currency.Code(dto.Currency))
	if err != nil {
		panic(err)
	}
	acc, err := account.New().
		WithID(dto.ID).
		WithUserID(dto.UserID).
		WithBalance(balance.Amount()).
		WithCurrency(balance.Currency()).
		WithCreatedAt(dto.CreatedAt).
		WithUpdatedAt(dto.CreatedAt).
		Build()
	if err != nil {
		panic(err)
	}
	return acc
}
