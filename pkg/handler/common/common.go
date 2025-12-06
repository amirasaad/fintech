package common

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/dto"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/amirasaad/fintech/pkg/repository/account"
	"github.com/amirasaad/fintech/pkg/repository/transaction"
	"github.com/amirasaad/fintech/pkg/repository/user"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var ErrInvalidRepositoryType = errors.New("invalid repository type")

func GetAccountRepository(
	uow repository.UnitOfWork,
	log *slog.Logger,
) (
	account.Repository,
	error,
) {
	accRepoAny, err := uow.GetRepository(
		(*account.Repository)(nil),
	)
	if err != nil {
		log.Error(
			"failed to get account repository",
			"error", err,
		)
		return nil, err
	}
	if accRepo, ok := accRepoAny.(account.Repository); ok {
		return accRepo, nil
	}
	return nil, ErrInvalidRepositoryType
}

func GetTransactionRepository(
	uow repository.UnitOfWork,
	log *slog.Logger,
) (
	transaction.Repository,
	error,
) {
	txRepoAny, err := uow.GetRepository(
		(*transaction.Repository)(nil),
	)
	if err != nil {
		log.Error(
			"failed to get transaction repository",
			"error", err,
		)
		return nil, err
	}
	if txRepo, ok := txRepoAny.(transaction.Repository); ok {

		return txRepo, nil
	}
	return nil, ErrInvalidRepositoryType
}

func GetUserRepository(
	uow repository.UnitOfWork,
	log *slog.Logger,
) (
	user.Repository,
	error,
) {
	userRepoAny, err := uow.GetRepository(
		(*user.Repository)(nil),
	)
	if err != nil {
		log.Error(
			"failed to get user repository",
			"error", err,
		)
		return nil, err
	}
	if userRepo, ok := userRepoAny.(user.Repository); ok {
		return userRepo, nil
	}
	return nil, ErrInvalidRepositoryType
}

// TransactionLookupResult contains transaction lookup result
type TransactionLookupResult struct {
	Transaction   *dto.TransactionRead
	TransactionID uuid.UUID
	Found         bool
	Error         error
}

// LookupTransactionByPaymentOrID looks up a transaction by payment ID or transaction ID
func LookupTransactionByPaymentOrID(
	ctx context.Context,
	txRepo transaction.Repository,
	paymentID *string,
	transactionID uuid.UUID,
	log *slog.Logger,
) TransactionLookupResult {
	result := TransactionLookupResult{TransactionID: transactionID}

	if paymentID != nil && *paymentID != "" {
		tx, err := txRepo.GetByPaymentID(ctx, *paymentID)
		if err == nil {
			result.Transaction = tx
			result.TransactionID = tx.ID
			result.Found = true
			return result
		}
		if errors.Is(err, gorm.ErrRecordNotFound) && transactionID != uuid.Nil {
			tx, err = txRepo.Get(ctx, transactionID)
			if err == nil {
				result.Transaction = tx
				result.Found = true
				return result
			}
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Warn(
				"⚠️ Transaction not found",
				"payment_id", *paymentID,
				"transaction_id", transactionID,
			)
			result.Found = false
			return result
		}
		result.Error = fmt.Errorf("failed to get transaction: %w", err)
		return result
	}

	if transactionID != uuid.Nil {
		tx, err := txRepo.Get(ctx, transactionID)
		if err == nil {
			result.Transaction = tx
			result.Found = true
			return result
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Warn("⚠️ Transaction not found", "transaction_id", transactionID)
			result.Found = false
			return result
		}
		result.Error = fmt.Errorf("failed to get transaction: %w", err)
		return result
	}

	log.Warn("⚠️ No transaction identifiers provided")
	result.Found = false
	return result
}
