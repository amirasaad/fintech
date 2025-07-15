package handler

import (
	"context"
	"log/slog"
	"strings"

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

	err := req.Account.Deposit(req.UserID, req.ConvertedMoney, account.MoneySource(req.MoneySource), req.PaymentID)
	if err != nil {
		logger.Error("DepositOperationHandler failed: domain operation error", "error", err)
		return &OperationResponse{Error: err}, nil
	}

	logger.Info("DepositOperationHandler: domain operation completed")
	return h.BaseHandler.Handle(ctx, req)
}

// WithdrawOperationHandler executes withdraw domain operations
type WithdrawOperationHandler struct {
	BaseHandler
	logger *slog.Logger
}

func maskExternalTarget(target *ExternalTarget) string {
	if target == nil {
		return ""
	}
	if target.BankAccountNumber != "" {
		return maskString(target.BankAccountNumber)
	}
	if target.ExternalWalletAddress != "" {
		return maskString(target.ExternalWalletAddress)
	}
	if target.RoutingNumber != "" {
		return maskString(target.RoutingNumber)
	}
	return ""
}

func maskString(s string) string {
	if len(s) <= 4 {
		return s
	}
	return strings.Repeat("*", len(s)-4) + s[len(s)-4:]
}

// Handle executes the withdraw domain operation
func (h *WithdrawOperationHandler) Handle(ctx context.Context, req *OperationRequest) (*OperationResponse, error) {
	logger := h.logger.With("operation", "withdraw")

	// Enforce that ExternalTarget is present for withdrawals
	if req.ExternalTarget == nil || (req.ExternalTarget.BankAccountNumber == "" && req.ExternalTarget.RoutingNumber == "" && req.ExternalTarget.ExternalWalletAddress == "") {
		logger.Error("WithdrawOperationHandler: missing or invalid external target")
		return &OperationResponse{Error: account.ErrAccountNotFound}, nil // Use a more specific error if desired
	}
	logger.Info("WithdrawOperationHandler: external target details", "bank_account_number", req.ExternalTarget.BankAccountNumber, "routing_number", req.ExternalTarget.RoutingNumber, "external_wallet_address", req.ExternalTarget.ExternalWalletAddress)
	req.ExternalTargetMasked = maskExternalTarget(req.ExternalTarget)

	err := req.Account.Withdraw(req.UserID, req.ConvertedMoney, account.ExternalTarget{RoutingNumber: "42342423"}, req.PaymentID)
	if err != nil {
		logger.Error("WithdrawOperationHandler failed: domain operation error", "error", err)
		return &OperationResponse{Error: err}, nil
	}

	logger.Info("WithdrawOperationHandler: domain operation completed")
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

	err := req.Account.Transfer(req.UserID, req.DestUserID, req.DestAccount, req.ConvertedMoney, account.MoneySourceInternal)
	if err != nil {
		logger.Error("TransferOperationHandler failed: domain operation error", "error", err)
		return &OperationResponse{Error: err}, nil
	}

	logger.Info("TransferOperationHandler: domain operation completed")
	return h.BaseHandler.Handle(ctx, req)
}
