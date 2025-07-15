package handler

import (
	"context"
	"log/slog"

	account "github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/repository"
)

// TransferPersistenceHandler handles the persistence of transfer operations.
// It updates both source and destination account balances and creates both transaction records.
type TransferPersistenceHandler struct {
	BaseHandler
	uow    repository.UnitOfWork
	logger *slog.Logger
}

// Handle processes transfer events, updates both accounts, and creates both transaction records.
func (h *TransferPersistenceHandler) Handle(ctx context.Context, req *OperationRequest) (*OperationResponse, error) {
	logger := h.logger
	events := req.Account.PullEvents()
	var transactions []*account.Transaction
	var transactionOut *account.Transaction
	var transactionIn *account.Transaction

	if len(events) == 0 {
		logger.Error("TransferPersistenceHandler: no events found on account")
		return &OperationResponse{Error: context.Canceled}, nil
	}

	if err := h.uow.Do(ctx, func(uow repository.UnitOfWork) error {
		repo, err := uow.AccountRepository()
		if err != nil {
			logger.Error("TransferPersistenceHandler failed: AccountRepository error", "error", err)
			return err
		}
		txRepo, err := uow.TransactionRepository()
		if err != nil {
			logger.Error("TransferPersistenceHandler failed: TransactionRepository error", "error", err)
			return err
		}
		for _, evt := range events {
			e, ok := evt.(account.TransferRequestedEvent) //nolint:go-critic
			if !ok {
				continue
			}
			// Fetch destination account to get its UserID
			destAcc, err := repo.Get(e.DestAccountID)
			if err != nil {
				logger.Error("TransferPersistenceHandler failed: destination account fetch error", "error", err)
				return err
			}
			transactionOut, transactionIn := NewTransferTransactions(e, destAcc.UserID)
			if err := txRepo.Create(transactionOut, req.ConvInfo, ""); err != nil {
				logger.Error("TransferPersistenceHandler failed: outgoing transaction create error", "error", err)
				return err
			}
			if err := txRepo.Create(transactionIn, req.ConvInfo, ""); err != nil {
				logger.Error("TransferPersistenceHandler failed: incoming transaction create error", "error", err)
				return err
			}
			transactions = append(transactions, transactionOut, transactionIn)

			// Unified balance update for transfer
			sourceAcc, err := repo.Get(e.SourceAccountID)
			if err != nil {
				logger.Error("TransferPersistenceHandler failed: source account fetch error", "error", err)
				return err
			}
			sourceAcc.Balance, err = sourceAcc.Balance.Add(transactionOut.Amount)
			if err != nil {
				logger.Error("TransferPersistenceHandler failed: source balance update error", "error", err)
				return err
			}
			destAcc.Balance, err = destAcc.Balance.Add(transactionIn.Amount)
			if err != nil {
				logger.Error("TransferPersistenceHandler failed: destination balance update error", "error", err)
				return err
			}
			if err := repo.Update(sourceAcc); err != nil {
				logger.Error("TransferPersistenceHandler failed: source account update error", "error", err)
				return err
			}
			if err := repo.Update(destAcc); err != nil {
				logger.Error("TransferPersistenceHandler failed: destination account update error", "error", err)
				return err
			}
			// End unified balance update
		}
		return nil
	}); err != nil {
		return &OperationResponse{Error: err}, nil
	}

	logger.Info("TransferPersistenceHandler: persistence completed successfully")

	return &OperationResponse{
		Transactions:   transactions,
		TransactionOut: transactionOut,
		TransactionIn:  transactionIn,
		ConvInfo:       req.ConvInfo,
	}, nil
}
