package handler

import (
	"context"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/domain/user"
	"github.com/amirasaad/fintech/pkg/repository"
)

// ValidationHandler validates that the account exists and belongs to the user
type ValidationHandler struct {
	BaseHandler
	uow    repository.UnitOfWork
	logger *slog.Logger
}

// Handle validates the account and passes the request to the next handler
func (h *ValidationHandler) Handle(ctx context.Context, req *OperationRequest) (*OperationResponse, error) {
	logger := h.logger.With("userID", req.UserID, "accountID", req.AccountID)
	logger.Info("AccountValidationHandler: starting")

	repo, err := h.uow.AccountRepository()
	if err != nil {
		logger.Error("AccountValidationHandler failed: repository error", "error", err)
		return &OperationResponse{Error: err}, nil
	}

	acc, err := repo.Get(req.AccountID)
	if err != nil {
		logger.Error("AccountValidationHandler failed: account not found", "error", err)
		return &OperationResponse{Error: account.ErrAccountNotFound}, nil
	}

	if acc.UserID != req.UserID {
		logger.Error("AccountValidationHandler failed: user unauthorized", "accountUserID", acc.UserID)
		return &OperationResponse{Error: user.ErrUserUnauthorized}, nil
	}

	req.Account = acc
	logger.Info("AccountValidationHandler: account validated successfully")

	return h.BaseHandler.Handle(ctx, req)
}

// TransferValidationHandler validates both source and destination accounts for transfers
type TransferValidationHandler struct {
	BaseHandler
	uow    repository.UnitOfWork
	logger *slog.Logger
}

// Handle validates both accounts and passes the request to the next handler
func (h *TransferValidationHandler) Handle(ctx context.Context, req *OperationRequest) (*OperationResponse, error) {
	logger := h.logger.With("userID", req.UserID, "sourceAccountID", req.AccountID, "destAccountID", req.DestAccountID)
	logger.Info("TransferValidationHandler: starting")

	repo, err := h.uow.AccountRepository()
	if err != nil {
		logger.Error("TransferValidationHandler failed: repository error", "error", err)
		return &OperationResponse{Error: err}, nil
	}

	// Get and validate source account
	sourceAccount, err := repo.Get(req.AccountID)
	if err != nil {
		logger.Error("TransferValidationHandler failed: source account not found", "error", err)
		return &OperationResponse{Error: account.ErrAccountNotFound}, nil
	}

	if sourceAccount.UserID != req.UserID {
		logger.Error("TransferValidationHandler failed: user unauthorized for source account", "accountUserID", sourceAccount.UserID)
		return &OperationResponse{Error: user.ErrUserUnauthorized}, nil
	}

	// Get and validate destination account
	destAccount, err := repo.Get(req.DestAccountID)
	if err != nil {
		logger.Error("TransferValidationHandler failed: destination account not found", "error", err)
		return &OperationResponse{Error: account.ErrAccountNotFound}, nil
	}

	req.Account = sourceAccount
	req.DestAccount = destAccount
	logger.Info("TransferValidationHandler: both accounts validated successfully")

	return h.BaseHandler.Handle(ctx, req)
}
