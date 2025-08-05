// Package account provides business logic for interacting with
// domain entities such as accounts and transactions.
// It defines the Service struct and its
// methods for creating accounts, depositing and withdrawing funds,
// retrieving account details, listing transactions, and checking account balances.
//
// The service layer follows clean architecture principles
// and uses the decorator pattern for transaction management.
// All business operations are wrapped with automatic transaction management,
//
//	error recovery, and structured logging.
package account

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/amirasaad/fintech/pkg/provider"

	"github.com/amirasaad/fintech/pkg/commands"
	"github.com/amirasaad/fintech/pkg/domain/events"

	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/amirasaad/fintech/pkg/dto"
	"github.com/amirasaad/fintech/pkg/repository"
	repoaccount "github.com/amirasaad/fintech/pkg/repository/account"
	"github.com/google/uuid"
)

// Service provides business logic for account operations including
// creation, deposits, withdrawals, and balance inquiries.
type Service struct {
	bus    eventbus.Bus
	uow    repository.UnitOfWork
	logger *slog.Logger
}

// New creates a new Service with the provided dependencies.
func New(bus eventbus.Bus, uow repository.UnitOfWork, logger *slog.Logger) *Service {
	return &Service{
		bus:    bus,
		uow:    uow,
		logger: logger,
	}
}

func (s *Service) CreateAccount(
	ctx context.Context,
	create dto.AccountCreate,
) (dto.AccountRead, error) {
	uow := s.uow
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
			curr = currency.DefaultCode
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
		if err = acctRepo.Create(ctx, createDTO); err != nil {
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
func (s *Service) Deposit(
	ctx context.Context,
	cmd commands.Deposit,
) error {
	// Always use the source currency for the initial deposit event
	amount, err := money.New(cmd.Amount, currency.Code(cmd.Currency))
	if err != nil {
		return err
	}
	dr := events.NewDepositRequested(
		cmd.UserID,
		cmd.AccountID,
		uuid.New(),
		events.WithDepositAmount(amount),
	)
	return s.bus.Emit(ctx, dr)
}

// Withdraw removes funds from the specified account
// to an external target and creates a transaction record.
func (s *Service) Withdraw(
	ctx context.Context,
	cmd commands.Withdraw,
) error {
	amount, err := money.New(cmd.Amount, currency.Code(cmd.Currency))
	if err != nil {
		return err
	}

	// Create event with amount and bank account number if provided
	opts := []events.WithdrawRequestedOpt{
		events.WithWithdrawAmount(amount),
	}

	if cmd.ExternalTarget != nil && cmd.ExternalTarget.BankAccountNumber != "" {
		opts = append(
			opts,
			events.WithWithdrawBankAccountNumber(
				cmd.ExternalTarget.BankAccountNumber,
			),
		)
	}

	wr := events.NewWithdrawRequested(
		cmd.UserID,
		cmd.AccountID,
		uuid.New(),
		opts...,
	)
	return s.bus.Emit(ctx, wr)
}

// Transfer moves funds from one account to another account.
func (s *Service) Transfer(
	ctx context.Context,
	cmd commands.Transfer,
) error {
	amount, err := money.New(cmd.Amount, currency.Code(cmd.Currency))
	if err != nil {
		return err
	}
	tr := events.NewTransferRequested(
		cmd.UserID,
		cmd.AccountID,
		uuid.New(),
		events.WithTransferDestAccountID(cmd.ToAccountID),
		events.WithTransferRequestedAmount(amount),
	)
	return s.bus.Emit(ctx, tr)
}

// UpdateTransaction updates a transaction by its ID with the provided update data.
// It handles updating the account balance if the status is changing to "completed".
func (s *Service) UpdateTransaction(
	ctx context.Context,
	transactionID uuid.UUID,
	update dto.TransactionUpdate,
) error {
	logger := s.logger.With("transactionID", transactionID)
	logger.Info("Updating transaction", "update", update)

	return s.uow.Do(ctx, func(uow repository.UnitOfWork) error {
		// Get transaction repository
		txRepo, err := uow.TransactionRepository()
		if err != nil {
			return fmt.Errorf("failed to get transaction repository: %w", err)
		}

		// Get the existing transaction
		tx, err := txRepo.Get(transactionID)
		if err != nil {
			return fmt.Errorf("failed to get transaction: %w", err)
		}

		// Only process account balance update if status is changing to completed
		if update.Status != nil && *update.Status == string(provider.PaymentCompleted) &&
			tx.Status != account.TransactionStatusCompleted {

			accRepo, err := uow.AccountRepository()
			if err != nil {
				return fmt.Errorf("failed to get account repository: %w", err)
			}

			acc, err := accRepo.Get(tx.AccountID)
			if err != nil {
				return fmt.Errorf("failed to get account: %w", err)
			}

			newBalance, err := acc.Balance.Add(tx.Amount)
			if err != nil {
				return fmt.Errorf("failed to update account balance: %w", err)
			}

			acc.Balance = newBalance
			if err := accRepo.Update(acc); err != nil {
				return fmt.Errorf("failed to save updated account: %w", err)
			}

			logger.Info("Account balance updated",
				"accountID", acc.ID,
				"newBalance", acc.Balance,
			)
		}

		// Update transaction fields
		if update.Status != nil {
			tx.Status = account.TransactionStatus(*update.Status)
		}
		if update.PaymentID != nil {
			tx.PaymentID = *update.PaymentID
		}

		// Save the updated transaction
		if err := txRepo.Update(tx); err != nil {
			return fmt.Errorf("failed to update transaction: %w", err)
		}

		logger.Info("Transaction updated successfully")
		return nil
	})
}

// UpdateTransactionStatusByPaymentID updates the status of a transaction
// identified by its payment ID.
// If the status is "completed", it also updates the account balance accordingly.
//
// Deprecated: Use UpdateTransaction with transaction ID instead.
func (s *Service) UpdateTransactionStatusByPaymentID(
	ctx context.Context,
	paymentID, status string,
) error {
	// Use a unit of work for atomicity
	logger := s.logger.With("paymentID", paymentID)
	logger.Info("Updating transaction with payment Id", "status", status)

	// First get the transaction ID by payment ID
	var txID uuid.UUID
	err := s.uow.Do(ctx, func(uow repository.UnitOfWork) error {
		txRepo, err := uow.TransactionRepository()
		if err != nil {
			return err
		}

		tx, err := txRepo.GetByPaymentID(paymentID)
		if err != nil {
			return err
		}
		txID = tx.ID
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to find transaction by payment ID: %w", err)
	}

	// Now update using the transaction ID
	return s.UpdateTransaction(ctx, txID, dto.TransactionUpdate{
		Status: &status,
	})
}
