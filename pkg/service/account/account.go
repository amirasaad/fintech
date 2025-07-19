// Package account provides business logic for interacting with domain entities such as accounts and transactions.
// It defines the Service struct and its methods for creating accounts, depositing and withdrawing funds,
// retrieving account details, listing transactions, and checking account balances.
//
// The service layer follows clean architecture principles and uses the decorator pattern for transaction management.
// All business operations are wrapped with automatic transaction management, error recovery, and structured logging.
package account

import (
	"context"

	"github.com/amirasaad/fintech/pkg/commands"
	"github.com/amirasaad/fintech/pkg/domain/events"

	"github.com/amirasaad/fintech/config"
	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/domain/money"
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

func (s *Service) CreateAccount(ctx context.Context, create dto.AccountCreate) (dto.AccountRead, error) {
	uow := s.deps.Uow
	var result *dto.AccountRead
	err := uow.Do(ctx, func(uow repository.UnitOfWork) error {
		repoAny, err := uow.GetRepository((*repoaccount.Repository)(nil))
		if err != nil {
			return err
		}
		acctRepo := repoAny.(repoaccount.Repository)

		// Enforce domain invariants
		curr := currency.Code(create.Currency)
		if curr == "" {
			curr = currency.DefaultCurrency
		}
		domainAcc, err := account.New().WithUserID(create.UserID).WithCurrency(curr).Build()
		if err != nil {
			return err
		}

		// Map to DTO for persistence
		createDTO := dto.AccountCreate{
			ID:       domainAcc.ID,
			UserID:   domainAcc.UserID,
			Balance:  int64(domainAcc.Balance.Amount()), // or 0 if always zero at creation
			Currency: curr.String(),
			// Add more fields as needed
		}
		if err := acctRepo.Create(ctx, createDTO); err != nil {
			return err
		}

		// Fetch for read DTO
		read, err := acctRepo.Get(ctx, domainAcc.ID)
		if err != nil {
			return err
		}
		result = read
		return nil
	})
	if err != nil {
		return dto.AccountRead{}, err
	}
	return *result, nil
}

// Deposit adds funds to the specified account and creates a transaction record.
func (s *Service) Deposit(ctx context.Context, cmd commands.DepositCommand) error {
	amount, err := money.New(cmd.Amount, currency.Code(cmd.Currency))
	if err != nil {
		return err
	}

	return s.deps.EventBus.Publish(ctx, events.DepositRequestedEvent{
		EventID:   uuid.New(),
		AccountID: cmd.AccountID, // pass uuid.UUID directly
		UserID:    cmd.UserID,    // pass uuid.UUID directly
		Amount:    amount,
		Source:    cmd.MoneySource,
	})
}

// Withdraw removes funds from the specified account to an external target and creates a transaction record.
func (s *Service) Withdraw(ctx context.Context, cmd commands.WithdrawCommand) error {
	amount, err := money.New(cmd.Amount, currency.Code(cmd.Currency))
	if err != nil {
		return err
	}
	var bankAccountNumber, routingNumber, externalWalletAddress string
	if cmd.ExternalTarget != nil {
		bankAccountNumber = cmd.ExternalTarget.BankAccountNumber
		routingNumber = cmd.ExternalTarget.RoutingNumber
		externalWalletAddress = cmd.ExternalTarget.ExternalWalletAddress
	}
	// Publish event with paymentID
	evt := events.WithdrawRequestedEvent{
		EventID:               uuid.New(),
		AccountID:             cmd.AccountID,
		UserID:                cmd.UserID,
		Amount:                amount,
		BankAccountNumber:     bankAccountNumber,
		RoutingNumber:         routingNumber,
		ExternalWalletAddress: externalWalletAddress,
	}
	return s.deps.EventBus.Publish(ctx, evt)
}

// Transfer moves funds from one account to another account.
func (s *Service) Transfer(ctx context.Context, create dto.TransactionCreate, destAccountID uuid.UUID) error {
	amount, err := money.New(float64(create.Amount), currency.Code(create.Currency))
	if err != nil {
		return err
	}
	// Only emit and publish event
	return s.deps.EventBus.Publish(ctx, events.TransferRequestedEvent{
		EventID:         uuid.New(),
		SourceAccountID: create.AccountID,
		DestAccountID:   destAccountID,
		SenderUserID:    create.UserID,
		ReceiverUserID:  create.UserID, // TODO: Look up destination account to get receiver user ID
		Amount:          amount,
		Source:          "transfer", // Use consistent source for transfers
	})
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
