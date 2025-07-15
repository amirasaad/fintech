package handler

import (
	"context"
	"log/slog"

	account "github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/repository"
)

// WithdrawPersistenceHandler handles the persistence of withdraw operations.
// It updates the account and creates the withdrawal transaction record.
type WithdrawPersistenceHandler struct {
	BaseHandler
	uow    repository.UnitOfWork
	logger *slog.Logger
}

// Handle processes withdraw events, updates the account, and creates the withdrawal transaction record.
func (h *WithdrawPersistenceHandler) Handle(ctx context.Context, req *OperationRequest) (*OperationResponse, error) {
	logger := h.logger
	events := req.Account.PullEvents()
	var transactions []*account.Transaction

	if len(events) == 0 {
		logger.Error("WithdrawPersistenceHandler: no events found on account")
		return &OperationResponse{Error: context.Canceled}, nil
	}

	if err := h.uow.Do(ctx, func(uow repository.UnitOfWork) error {
		repo, err := uow.AccountRepository()
		if err != nil {
			logger.Error("WithdrawPersistenceHandler failed: AccountRepository error", "error", err)
			return err
		}
		if err = repo.Update(req.Account); err != nil {
			logger.Error("WithdrawPersistenceHandler failed: account update error", "error", err)
			return err
		}
		txRepo, err := uow.TransactionRepository()
		if err != nil {
			logger.Error("WithdrawPersistenceHandler failed: TransactionRepository error", "error", err)
			return err
		}
		for _, evt := range events {
			e, ok := evt.(account.WithdrawRequestedEvent) //nolint:go-critic
			if !ok {
				continue
			}
			// Replace inline transaction construction with factory helper
			tx := NewWithdrawTransaction(e, req.ExternalTarget)
			if err := txRepo.Create(tx, req.ConvInfo, maskExternalTarget(req.ExternalTarget)); err != nil {
				logger.Error("WithdrawPersistenceHandler failed: transaction create error", "error", err)
				return err
			}
			transactions = append(transactions, tx)
		}
		return nil
	}); err != nil {
		return &OperationResponse{Error: err}, nil
	}

	logger.Info("WithdrawPersistenceHandler: persistence completed successfully")

	return &OperationResponse{
		Transactions: transactions,
		ConvInfo:     req.ConvInfo,
	}, nil
}
