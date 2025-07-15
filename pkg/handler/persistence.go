package handler

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/amirasaad/fintech/pkg/currency"
	account "github.com/amirasaad/fintech/pkg/domain/account"
	money "github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/google/uuid"
)

// PersistenceHandler handles the persistence of account and transaction changes
type PersistenceHandler struct {
	BaseHandler
	uow    repository.UnitOfWork
	logger *slog.Logger
}

// Handle persists the changes and returns the final response
func (h *PersistenceHandler) Handle(ctx context.Context, req *OperationRequest) (*OperationResponse, error) {
	logger := h.logger

	// Pull all events from the account
	events := req.Account.PullEvents()
	var transactions []*account.Transaction
	var transactionOut *account.Transaction
	var transactionIn *account.Transaction

	if len(events) == 0 {
		logger.Error("PersistenceHandler: no events found on account")
		return &OperationResponse{Error: errors.New("no events found")}, nil
	}

	if req.ConvInfo != nil {
		logger.Info("PersistenceHandler: conversion info available", "originalAmount", req.ConvInfo.OriginalAmount)
	}

	if err := h.uow.Do(ctx, func(uow repository.UnitOfWork) error {
		repo, err := uow.AccountRepository()
		if err != nil {
			logger.Error("PersistenceHandler failed: AccountRepository error", "error", err)
			return err
		}
		if err = repo.Update(req.Account); err != nil {
			logger.Error("PersistenceHandler failed: account update error", "error", err)
			return err
		}
		txRepo, err := uow.TransactionRepository()
		if err != nil {
			logger.Error("PersistenceHandler failed: TransactionRepository error", "error", err)
			return err
		}
		for _, evt := range events {
			switch e := evt.(type) {
			case account.DepositRequestedEvent:
				moneyVal, _ := money.New(e.Amount, currency.Code(e.Currency))
				tx := &account.Transaction{
					ID:                   e.EventID,
					AccountID:            uuid.MustParse(e.AccountID),
					UserID:               uuid.MustParse(e.UserID),
					Amount:               moneyVal,
					MoneySource:          e.Source,
					Status:               account.TransactionStatusInitiated,
					ExternalTargetMasked: req.ExternalTargetMasked,
					CreatedAt:            time.Now().UTC(),
				}

				if err := txRepo.Create(tx, req.ConvInfo); err != nil {
					logger.Error("PersistenceHandler failed: transaction create error", "error", err)
					return err
				}
				transactions = append(transactions, tx)

			case account.WithdrawRequestedEvent:
				moneyVal, _ := money.New(e.Amount, currency.Code(e.Currency))
				tx := &account.Transaction{
					ID:                   e.EventID,
					AccountID:            uuid.MustParse(e.AccountID),
					UserID:               uuid.MustParse(e.UserID),
					Amount:               moneyVal,
					MoneySource:          e.Source,
					Status:               account.TransactionStatusInitiated,
					ExternalTargetMasked: req.ExternalTargetMasked,
					CreatedAt:            time.Now().UTC(),
				}

				if err := txRepo.Create(tx, req.ConvInfo); err != nil {
					logger.Error("PersistenceHandler failed: transaction create error", "error", err)
					return err
				}
				transactions = append(transactions, tx)

			case account.TransferRequestedEvent:
				moneyVal, _ := money.New(e.Amount, currency.Code(e.Currency))
				// Outgoing transaction (source account)
				transactionOut = &account.Transaction{
					ID:                   e.EventID,
					AccountID:            e.SourceAccountID,
					UserID:               e.SenderUserID,
					Amount:               moneyVal.Negate(),
					MoneySource:          e.Source,
					Status:               account.TransactionStatusInitiated,
					ExternalTargetMasked: req.ExternalTargetMasked,
					CreatedAt:            time.Now().UTC(),
				}
				// Incoming transaction (destination account)
				transactionIn = &account.Transaction{
					ID:                   uuid.New(),
					AccountID:            e.DestAccountID,
					UserID:               e.ReceiverUserID,
					Amount:               moneyVal,
					MoneySource:          e.Source,
					Status:               account.TransactionStatusInitiated,
					ExternalTargetMasked: req.ExternalTargetMasked,
					CreatedAt:            time.Now().UTC(),
				}
				if err := txRepo.Create(transactionOut, req.ConvInfo); err != nil {
					logger.Error("PersistenceHandler failed: outgoing transaction create error", "error", err)
					return err
				}
				if err := txRepo.Create(transactionIn, req.ConvInfo); err != nil {
					logger.Error("PersistenceHandler failed: incoming transaction create error", "error", err)
					return err
				}
				transactions = append(transactions, transactionOut, transactionIn)
			}
		}
		return nil
	}); err != nil {
		return &OperationResponse{Error: err}, nil
	}

	logger.Info("PersistenceHandler: persistence completed successfully")

	return &OperationResponse{
		Transaction:    nil, // For backward compatibility, could set to transactions[0] or nil
		Transactions:   transactions,
		TransactionOut: transactionOut,
		TransactionIn:  transactionIn,
		ConvInfo:       req.ConvInfo,
	}, nil
}

// OperationResponse contains the result of an account operation
// For transfers, both TransactionOut (from source) and TransactionIn (to dest) may be set.
// Add ConvInfoOut and ConvInfoIn for transfer conversion info.
// (This comment is for context; actual struct is in types.go)

// TransferPersistenceHandler handles persistence for transfer operations
type TransferPersistenceHandler struct {
	BaseHandler
	uow    repository.UnitOfWork
	logger *slog.Logger
}

// Handle persists transfer changes and returns the final response
func (h *TransferPersistenceHandler) Handle(ctx context.Context, req *OperationRequest) (*OperationResponse, error) {
	logger := h.logger.With("transactionID", req.Account.ID)

	if err := h.uow.Do(ctx, func(uow repository.UnitOfWork) error {
		// Update both accounts
		repo, err := uow.AccountRepository()
		if err != nil {
			logger.Error("TransferPersistenceHandler failed: AccountRepository error", "error", err)
			return err
		}

		if err = repo.Update(req.Account); err != nil {
			logger.Error("TransferPersistenceHandler failed: source account update error", "error", err)
			return err
		}

		if err = repo.Update(req.DestAccount); err != nil {
			logger.Error("TransferPersistenceHandler failed: destination account update error", "error", err)
			return err
		}

		// Create both transaction records
		txRepo, err := uow.TransactionRepository()
		if err != nil {
			logger.Error("TransferPersistenceHandler failed: TransactionRepository error", "error", err)
			return err
		}

		if err = txRepo.Create(req.Transaction, req.ConvInfoOut); err != nil {
			logger.Error("TransferPersistenceHandler failed: outgoing transaction create error", "error", err)
			return err
		}

		if err = txRepo.Create(req.TransactionIn, req.ConvInfoIn); err != nil {
			logger.Error("TransferPersistenceHandler failed: incoming transaction create error", "error", err)
			return err
		}

		return nil
	}); err != nil {
		return &OperationResponse{Error: err}, nil
	}

	logger.Info("TransferPersistenceHandler: persistence completed successfully")

	return &OperationResponse{
		Transaction:    req.Transaction,
		TransactionOut: req.Transaction,
		TransactionIn:  req.TransactionIn,
		ConvInfoOut:    req.ConvInfoOut,
		ConvInfoIn:     req.ConvInfoIn,
	}, nil
}
