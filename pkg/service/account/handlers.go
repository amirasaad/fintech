package account

import (
	"github.com/amirasaad/fintech/pkg/domain/account"
	mon "github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/google/uuid"
)

// depositHandler implements operationHandler for deposit operations
type depositHandler struct{}

func (h *depositHandler) execute(account *account.Account, userID uuid.UUID, money mon.Money) (*account.Transaction, error) {
	return account.Deposit(userID, money)
}

// withdrawHandler implements operationHandler for withdraw operations
type withdrawHandler struct{}

func (h *withdrawHandler) execute(account *account.Account, userID uuid.UUID, money mon.Money) (*account.Transaction, error) {
	return account.Withdraw(userID, money)
}