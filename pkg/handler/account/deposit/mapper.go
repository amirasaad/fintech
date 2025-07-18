package deposit

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
	return account.NewAccountFromData(dto.ID, dto.UserID, balance, dto.CreatedAt, dto.CreatedAt)
}
