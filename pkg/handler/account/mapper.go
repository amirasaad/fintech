package account

import (
	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/amirasaad/fintech/pkg/queries"
	"github.com/google/uuid"
)

// MapDTOToAccount maps a queries.GetAccountResult DTO to a domain Account model.
func MapDTOToAccount(dto queries.GetAccountResult) (*account.Account, error) {
	accID, err := uuid.Parse(dto.AccountID)
	if err != nil {
		return nil, err
	}
	userID, err := uuid.Parse(dto.UserID)
	if err != nil {
		return nil, err
	}
	bal, err := money.New(dto.Balance, currency.Code(dto.Currency))
	if err != nil {
		return nil, err
	}
	return &account.Account{
		ID:      accID,
		UserID:  userID,
		Balance: bal,
		// Add other fields if needed
	}, nil
}
