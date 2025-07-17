// Package account provides business logic for interacting with domain entities such as accounts and transactions.
// It defines the Service struct and its methods for creating accounts, depositing and withdrawing funds,
// retrieving account details, listing transactions, and checking account balances.
//
// The service layer follows clean architecture principles and uses the decorator pattern for transaction management.
// All business operations are wrapped with automatic transaction management, error recovery, and structured logging.
package account

import (
	"context"
	"errors"
	"time"

	"github.com/amirasaad/fintech/config"
	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/domain/account/events"
	"github.com/amirasaad/fintech/pkg/dto"
	"github.com/amirasaad/fintech/pkg/repository"
	repoaccount "github.com/amirasaad/fintech/pkg/repository/account"
	"github.com/google/uuid"
)

// Service provides business logic for account operations including creation, deposits, withdrawals, and balance inquiries.
type Service struct {
	deps config.Deps
}

// NewService creates a new Service with the provided dependencies.
func NewService(deps config.Deps) *Service {
	return &Service{
		deps: deps,
	}
}

// CreateAccount creates a new account for the specified user in a transaction.
func (s *Service) CreateAccount(ctx context.Context, userID uuid.UUID) (a *account.Account, err error) {
	uow := s.deps.Uow
	err = uow.Do(ctx, func(uow repository.UnitOfWork) error {
		repo, err := uow.AccountRepository()
		if err != nil {
			return err
		}
		a, err = account.New().WithUserID(userID).Build()
		if err != nil {
			return err
		}
		return repo.Create(a)
	})
	if err != nil {
		a = nil
	}
	return
}

// CreateAccountWithCurrency creates a new account for the specified user with a specific currency.
// This method allows creating accounts in different currencies, which is useful for multi-currency
// applications where users may need accounts in various currencies.
//
// The operation is wrapped with automatic transaction management and includes comprehensive
// error handling and logging.
//
// Parameters:
//   - userID: The UUID of the user who will own the account
//   - currencyCode: The ISO 4217 currency code for the account (e.g., "USD", "EUR", "JPY")
//
// Returns:
//   - A pointer to the created account with the specified currency
//   - An error if the operation fails (e.g., invalid currency, user not found, database error)
//
// The method validates the currency code and ensures it's supported by the system.
// If the currency is not supported, an appropriate domain error is returned.
//
// Example:
//
//	account, err := service.CreateAccountWithCurrency(userID, currency.Code("EUR"))
//	if err != nil {
//	    log.Error("Failed to create EUR account", "error", err)
//	    return
//	}
//	log.Info("EUR account created", "accountID", account.ID, "currency", account.Currency)
func (s *Service) CreateAccountWithCurrency(
	userID uuid.UUID,
	currencyCode currency.Code,
) (acct *account.Account, err error) {
	logger := s.deps.Logger.With("userID", userID, "currency", currencyCode)
	logger.Info("CreateAccountWithCurrency started")
	uow := s.deps.Uow
	err = uow.Do(context.Background(), func(uow repository.UnitOfWork) error {
		repo, err := uow.AccountRepository()
		if err != nil {
			logger.Error("CreateAccountWithCurrency failed: AccountRepository error", "error", err)
			return err
		}
		acct, err = account.New().
			WithUserID(userID).
			WithCurrency(currencyCode).
			Build()
		if err != nil {
			logger.Error("CreateAccountWithCurrency failed: domain error", "error", err)
			return err
		}
		if err = repo.Create(acct); err != nil {
			logger.Error("CreateAccountWithCurrency failed: repo create error", "error", err)
			return err
		}
		return nil
	})
	if err != nil {
		acct = nil
		logger.Error("CreateAccountWithCurrency failed: transaction error", "error", err)
		return
	}
	logger.Info("CreateAccountWithCurrency successful", "accountID", acct.ID)
	return
}

// CreateAccountCQRS demonstrates using the CQRS-style account.Repository via UoW.GetRepository.
func (s *Service) CreateAccountCQRS(ctx context.Context, uow repository.UnitOfWork, create dto.AccountCreate) error {
	repoAny, err := uow.GetRepository((*repoaccount.Repository)(nil))
	if err != nil {
		return err
	}
	acctRepo, ok := repoAny.(repoaccount.Repository)
	if !ok {
		return errors.New("invalid repository type for account.Repository")
	}
	return acctRepo.Create(ctx, create)
}

// Deposit adds funds to the specified account and creates a transaction record.
func (s *Service) Deposit(
	userID, accountID uuid.UUID,
	amount float64,
	currencyCode currency.Code,
	moneySource string,
) error {
	if amount <= 0 {
		return errors.New("amount must be positive")
	}
	// Publish event with paymentID
	evt := events.DepositRequestedEvent{
		EventID:   uuid.New(),
		AccountID: accountID.String(),
		UserID:    userID.String(),
		Amount:    amount, // float64
		Currency:  currencyCode.String(),
		Source:    moneySource,
		Timestamp: time.Now().Unix(),
	}
	return s.deps.EventBus.Publish(context.Background(), evt)
}

// Withdraw removes funds from the specified account to an external target and creates a transaction record.
func (s *Service) Withdraw(
	userID, accountID uuid.UUID,
	amount float64,
	currencyCode currency.Code,
	externalTarget account.ExternalTarget,
) error {
	if amount <= 0 {
		return errors.New("amount must be positive")
	}
	// Publish event with paymentID
	evt := events.WithdrawRequestedEvent{
		EventID:               uuid.New(),
		AccountID:             accountID.String(),
		UserID:                userID.String(),
		Amount:                amount, // float64
		Currency:              string(currencyCode),
		BankAccountNumber:     externalTarget.BankAccountNumber,
		RoutingNumber:         externalTarget.RoutingNumber,
		ExternalWalletAddress: externalTarget.ExternalWalletAddress,
		Timestamp:             time.Now().Unix(),
		PaymentID:             "", // Set if available
	}
	err := s.deps.EventBus.Publish(context.Background(), evt)
	if err != nil {
		return err
	}
	return nil
}

// Transfer moves funds from one account to another account.
func (s *Service) Transfer(
	userID uuid.UUID,
	sourceAccountID, destAccountID uuid.UUID,
	amount float64,
	currencyCode currency.Code,
) error {
	if amount <= 0 {
		return errors.New("amount must be positive")
	}
	// Only emit and publish event
	evt := events.TransferRequestedEvent{
		EventID:         uuid.New(),
		SourceAccountID: sourceAccountID,
		DestAccountID:   destAccountID,
		SenderUserID:    userID,
		Amount:          amount, // float64
		Currency:        string(currencyCode),
		Source:          string(account.MoneySourceInternal), // or appropriate value
		Timestamp:       time.Now().Unix(),
	}
	return s.deps.EventBus.Publish(context.Background(), evt)
}

// UpdateTransactionStatusByPaymentID updates the status of a transaction identified by its payment ID.
// If the status is "completed", it also updates the account balance accordingly.
func (s *Service) UpdateTransactionStatusByPaymentID(paymentID, status string) error {
	// Use a unit of work for atomicity
	logger := s.deps.Logger.With("paymentID", paymentID)
	logger.Info("Updating transaction with payment Id", "status", status)
	return s.deps.Uow.Do(context.Background(), func(uow repository.UnitOfWork) error {
		txRepo, err := uow.TransactionRepository()
		if err != nil {
			return err
		}
		accRepo, err := uow.AccountRepository()
		if err != nil {
			return err
		}
		tx, err := txRepo.GetByPaymentID(paymentID)
		if err != nil {
			return err
		}
		// Only update account if status is changing to completed and wasn't already completed
		if status == "completed" && tx.Status != account.TransactionStatusCompleted {
			acc, err := accRepo.Get(tx.AccountID)
			if err != nil {
				return err
			}
			newBalance, errAdd := acc.Balance.Add(tx.Amount)
			if errAdd != nil {
				return errAdd
			}
			acc.Balance = newBalance

			if err := accRepo.Update(acc); err != nil {
				return err
			}
		}
		tx.Status = account.TransactionStatus(status)
		return txRepo.Update(tx)
	})
}
