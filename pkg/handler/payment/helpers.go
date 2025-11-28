package payment

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"github.com/amirasaad/fintech/pkg/dto"
	"github.com/amirasaad/fintech/pkg/repository/transaction"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// idempotencyTracker tracks processed events for idempotency checks
type idempotencyTracker struct {
	processed sync.Map // map[string]struct{}
}

// newIdempotencyTracker creates a new idempotency tracker
func newIdempotencyTracker() *idempotencyTracker {
	return &idempotencyTracker{}
}

// store marks an event as processed (for testing)
func (t *idempotencyTracker) store(key string) {
	t.processed.Store(key, struct{}{})
}

// delete removes an event from the processed map (for testing)
func (t *idempotencyTracker) delete(key string) {
	t.processed.Delete(key)
}

// checkAndMarkProcessed checks if already processed and marks it if not
func (t *idempotencyTracker) checkAndMarkProcessed(
	key string,
	log *slog.Logger,
	handlerName string,
) bool {
	if _, already := t.processed.LoadOrStore(key, struct{}{}); already {
		log.Info(
			"🔁 [SKIP] Event already processed",
			"handler", handlerName,
			"idempotency_key", key,
		)
		return true
	}
	return false
}

// Global idempotency trackers for different event types
var (
	processedPaymentProcessed = newIdempotencyTracker()
	processedPaymentCompleted = newIdempotencyTracker()
	processedPaymentInitiated = newIdempotencyTracker()
)

// TransactionLookupResult contains the result of a transaction lookup
type TransactionLookupResult struct {
	Transaction   *dto.TransactionRead
	TransactionID uuid.UUID
	Found         bool
	Error         error
}

// lookupTransactionByPaymentOrID attempts to find a transaction by payment ID or transaction ID.
// It handles missing transactions gracefully for idempotent behavior.
func lookupTransactionByPaymentOrID(
	ctx context.Context,
	txRepo transaction.Repository,
	paymentID *string,
	transactionID uuid.UUID,
	log *slog.Logger,
) TransactionLookupResult {
	result := TransactionLookupResult{
		TransactionID: transactionID,
	}

	// First try to get by payment ID if provided
	if paymentID != nil && *paymentID != "" {
		tx, err := txRepo.GetByPaymentID(ctx, *paymentID)
		if err == nil {
			result.Transaction = tx
			result.TransactionID = tx.ID
			result.Found = true
			return result
		}
		// If not found by payment ID and we have transaction ID, try that
		if errors.Is(err, gorm.ErrRecordNotFound) && transactionID != uuid.Nil {
			tx, err = txRepo.Get(ctx, transactionID)
			if err == nil {
				result.Transaction = tx
				result.Found = true
				return result
			}
		}
		// If still not found, handle gracefully
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Warn(
				"⚠️ [SKIP] Transaction not found",
				"payment_id", *paymentID,
				"transaction_id", transactionID,
			)
			result.Found = false
			return result
		}
		result.Error = fmt.Errorf("failed to get transaction: %w", err)
		return result
	}

	// Try by transaction ID if provided
	if transactionID != uuid.Nil {
		tx, err := txRepo.Get(ctx, transactionID)
		if err == nil {
			result.Transaction = tx
			result.Found = true
			return result
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Warn(
				"⚠️ [SKIP] Transaction not found",
				"transaction_id", transactionID,
			)
			result.Found = false
			return result
		}
		result.Error = fmt.Errorf("failed to get transaction: %w", err)
		return result
	}

	// No identifiers provided
	log.Warn(
		"⚠️ [SKIP] No transaction identifiers provided",
	)
	result.Found = false
	return result
}

// checkTransactionIdempotency checks if a transaction with the given status
// has already been processed and marks it if not.
func checkTransactionIdempotency(
	tracker *idempotencyTracker,
	tx *dto.TransactionRead,
	expectedStatus string,
	paymentID *string,
	transactionID uuid.UUID,
	log *slog.Logger,
	handlerName string,
) bool {
	if tx == nil || tx.Status != expectedStatus {
		return false
	}

	idempotencyKey := ""
	switch {
	case paymentID != nil && *paymentID != "":
		idempotencyKey = *paymentID
	case transactionID != uuid.Nil:
		idempotencyKey = transactionID.String()
	case tx.ID != uuid.Nil:
		idempotencyKey = tx.ID.String()
	}

	if idempotencyKey == "" {
		return false
	}

	return tracker.checkAndMarkProcessed(idempotencyKey, log, handlerName)
}
