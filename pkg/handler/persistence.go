package handler

import (
	"context"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/repository"
)

// PersistenceHandler handles the persistence of account and transaction changes
type PersistenceHandler struct {
	BaseHandler
	uow    repository.UnitOfWork
	logger *slog.Logger
}

// Handle persists the changes and returns the final response
func (h *PersistenceHandler) Handle(ctx context.Context, req *OperationRequest) (*OperationResponse, error) {
	logger := h.logger.With("transactionID", req.Transaction.ID)

	// Store conversion info if conversion occurred
	if req.ConvInfo != nil {
		logger.Info("PersistenceHandler: conversion info available", "originalAmount", req.ConvInfo.OriginalAmount)
	}
	if err := h.uow.Do(ctx, func(uow repository.UnitOfWork) error {

		// Update account using type-safe method
		repo, err := uow.AccountRepository()

		if err != nil {
			logger.Error("PersistenceHandler failed: AccountRepository error", "error", err)
			return err
		}

		if err = repo.Update(req.Account); err != nil {
			logger.Error("PersistenceHandler failed: account update error", "error", err)
			return err
		}

		// Create transaction using type-safe method
		txRepo, err := uow.TransactionRepository()
		if err != nil {
			logger.Error("PersistenceHandler failed: TransactionRepository error", "error", err)
			return err
		}

		if err = txRepo.Create(req.Transaction, req.ConvInfo, req.ExternalTargetMasked); err != nil {
			logger.Error("PersistenceHandler failed: transaction create error", "error", err)
			return err
		}

		return err
	}); err != nil {
		return &OperationResponse{Error: err}, nil
	}

	logger.Info("PersistenceHandler: persistence completed successfully")

	return &OperationResponse{
		Transaction: req.Transaction,
		ConvInfo:    req.ConvInfo,
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
	logger := h.logger.With("transactionID", req.Transaction.ID)

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

		if err = txRepo.Create(req.Transaction, req.ConvInfoOut, ""); err != nil {
			logger.Error("TransferPersistenceHandler failed: outgoing transaction create error", "error", err)
			return err
		}

		if err = txRepo.Create(req.TransactionIn, req.ConvInfoIn, ""); err != nil {
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
