package handler

import (
	"context"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/domain/account"
)

// DepositOperationHandler executes deposit domain operations
type DepositOperationHandler struct {
	BaseHandler
	logger *slog.Logger
}

// Handle executes the deposit domain operation
func (h *DepositOperationHandler) Handle(ctx context.Context, req *OperationRequest) (*OperationResponse, error) {
	logger := h.logger.With("operation", "deposit")

	tx, err := req.Account.Deposit(req.UserID, req.ConvertedMoney, account.MoneySource(req.MoneySource))
	if err != nil {
		logger.Error("DepositOperationHandler failed: domain operation error", "error", err)
		return &OperationResponse{Error: err}, nil
	}

	req.Transaction = tx
	logger.Info("DepositOperationHandler: domain operation completed", "transactionID", tx.ID)

	return h.BaseHandler.Handle(ctx, req)
}

// WithdrawOperationHandler executes withdraw domain operations
type WithdrawOperationHandler struct {
	BaseHandler
	logger *slog.Logger
}

// Handle executes the withdraw domain operation
func (h *WithdrawOperationHandler) Handle(ctx context.Context, req *OperationRequest) (*OperationResponse, error) {
	logger := h.logger.With("operation", "withdraw")

	tx, err := req.Account.Withdraw(req.UserID, req.ConvertedMoney, account.MoneySource(req.MoneySource))
	if err != nil {
		logger.Error("WithdrawOperationHandler failed: domain operation error", "error", err)
		return &OperationResponse{Error: err}, nil
	}

	req.Transaction = tx
	logger.Info("WithdrawOperationHandler: domain operation completed", "transactionID", tx.ID)

	return h.BaseHandler.Handle(ctx, req)
}

// TransferOperationHandler executes transfer domain operations
type TransferOperationHandler struct {
	BaseHandler
	logger *slog.Logger
}

// Handle executes the transfer domain operation
func (h *TransferOperationHandler) Handle(ctx context.Context, req *OperationRequest) (*OperationResponse, error) {
	logger := h.logger.With("operation", "transfer")

	txIn, txOut, err := req.Account.Transfer(req.UserID, req.DestAccount, req.ConvertedMoney, account.MoneySourceInternal)
	if err != nil {
		logger.Error("TransferOperationHandler failed: domain operation error", "error", err)
		return &OperationResponse{Error: err}, nil
	}

	req.Transaction = txOut
	req.TransactionIn = txIn
	logger.Info("TransferOperationHandler: domain operation completed", "txOutID", txOut.ID, "txInID", txIn.ID)

	return h.BaseHandler.Handle(ctx, req)
}
