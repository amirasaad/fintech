package handler

import (
	"context"
	"log/slog"

	account "github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/repository"
)

// DepositPersistenceHandler handles the persistence of deposit operations.
// It updates the account and creates the deposit transaction record.
type DepositPersistenceHandler struct {
	BaseHandler
	uow    repository.UnitOfWork
	logger *slog.Logger
}

// Handle processes deposit events, updates the account, and creates the deposit transaction record.
func (h *DepositPersistenceHandler) Handle(ctx context.Context, req *OperationRequest) (*OperationResponse, error) {
	logger := h.logger
	events := req.Account.PullEvents()
	var transactions []*account.Transaction

	if len(events) == 0 {
		logger.Error("DepositPersistenceHandler: no events found on account")
		return &OperationResponse{Error: context.Canceled}, nil
	}

	if err := h.uow.Do(ctx, func(uow repository.UnitOfWork) error {
		repo, err := uow.AccountRepository()
		if err != nil {
			logger.Error("DepositPersistenceHandler failed: AccountRepository error", "error", err)
			return err
		}
		if err = repo.Update(req.Account); err != nil {
			logger.Error("DepositPersistenceHandler failed: account update error", "error", err)
			return err
		}
		txRepo, err := uow.TransactionRepository()
		if err != nil {
			logger.Error("DepositPersistenceHandler failed: TransactionRepository error", "error", err)
			return err
		}
		for _, evt := range events {
			e, ok := evt.(account.DepositRequestedEvent) //nolint:go-critic
			if !ok {
				continue
			}
			// Replace inline transaction construction with factory helper
			tx := NewDepositTransaction(e)
			if err := txRepo.Create(tx, req.ConvInfo, ""); err != nil {
				logger.Error("DepositPersistenceHandler failed: transaction create error", "error", err)
				return err
			}
			transactions = append(transactions, tx)
		}
		return nil
	}); err != nil {
		return &OperationResponse{Error: err}, nil
	}

	logger.Info("DepositPersistenceHandler: persistence completed successfully")

	return &OperationResponse{
		Transactions: transactions,
		ConvInfo:     req.ConvInfo,
	}, nil
}
