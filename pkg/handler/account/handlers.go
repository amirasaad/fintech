// Package account provides event-driven handlers for account domain events.
package account

import (
	"context"

	"log/slog"

	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/domain/account/events"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/amirasaad/fintech/pkg/queries"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/google/uuid"
)

// MoneyCreationHandler handles DepositValidatedEvent, converts float64 amount to int64 (smallest unit), and publishes MoneyCreatedEvent.
func MoneyCreationHandler(bus eventbus.EventBus) func(context.Context, domain.Event) {
	return func(ctx context.Context, e domain.Event) {
		ve, ok := e.(events.DepositValidatedEvent)
		if !ok {
			return
		}
		if ve.Amount <= 0 {
			return
		}
		m, err := money.New(ve.Amount, currency.Code(ve.Currency))
		if err != nil {
			return
		}
		_ = bus.Publish(ctx, events.MoneyCreatedEvent{
			DepositValidatedEvent: ve,
			Amount:                m.Amount(), // int64, smallest unit
			Currency:              m.Currency().String(),
			TargetCurrency:        ve.Currency,
		})
	}
}

// DepositPersistenceHandler handles MoneyConvertedEvent, persists to DB, and publishes DepositPersistedEvent.
func DepositPersistenceHandler(bus eventbus.EventBus, uow repository.UnitOfWork) func(context.Context, domain.Event) {
	return func(ctx context.Context, e domain.Event) {
		mce, ok := e.(events.MoneyConvertedEvent)
		if !ok {
			return
		}
		// TODO: Implement actual DB persistence logic using uow.Do
		// Example:
		// err := uow.Do(ctx, func(uow repository.UnitOfWork) error {
		//     repo, err := uow.AccountRepository()
		//     if err != nil { return err }
		//     // Persist transaction, update account, etc.
		//     return nil
		// })
		// if err != nil { return }
		_ = bus.Publish(ctx, events.DepositPersistedEvent{
			MoneyCreatedEvent: mce.MoneyCreatedEvent,
			// Add DB transaction info if needed
		})
	}
}

// PaymentIdPersistenceHandler handles PaymentInitiatedEvent, updates the transaction with the paymentId, and publishes PaymentIdPersistedEvent.
func PaymentIdPersistenceHandler(bus eventbus.EventBus, uow repository.UnitOfWork) func(context.Context, domain.Event) {
	return func(ctx context.Context, e domain.Event) {
		pie, ok := e.(events.PaymentInitiatedEvent)
		if !ok {
			return
		}
		// TODO: Implement actual DB update logic using uow.Do
		// Example:
		// err := uow.Do(ctx, func(uow repository.UnitOfWork) error {
		//     repo, err := uow.TransactionRepository()
		//     if err != nil { return err }
		//     // Update transaction with paymentId
		//     return nil
		// })
		// if err != nil { return }
		_ = bus.Publish(ctx, events.PaymentIdPersistedEvent{
			PaymentInitiatedEvent: pie,
			// Add DB transaction info if needed
		})
	}
}

// DepositValidationHandler handles DepositRequestedEvent, maps DTO to domain, validates, and publishes DepositValidatedEvent.
func DepositValidationHandler(bus eventbus.EventBus, logger *slog.Logger) func(context.Context, domain.Event) {
	return func(ctx context.Context, e domain.Event) {
		de, ok := e.(events.DepositRequestedEvent)
		if !ok {
			logger.Error("DepositValidationHandler: unexpected event type", "event", e)
			return
		}
		getAccountResult := queries.GetAccountResult{
			AccountID: de.AccountID,
			UserID:    de.UserID,
			Balance:   de.Amount,
			Currency:  de.Currency,
		}
		acc, err := MapDTOToAccount(getAccountResult)
		if err != nil {
			logger.Error("DepositValidationHandler: failed to map DTO to domain Account", "error", err, "result", getAccountResult)
			return
		}
		userUUID, err := uuid.Parse(de.UserID)
		if err != nil {
			logger.Error("DepositValidationHandler: invalid userID", "error", err, "userID", de.UserID)
			return
		}
		amount, err := money.New(de.Amount, currency.Code(de.Currency))
		if err != nil {
			return
		}
		if err := acc.ValidateDeposit(userUUID, amount); err != nil {
			logger.Error("DepositValidationHandler: domain validation failed", "error", err)
			return
		}
		_ = bus.Publish(ctx, events.DepositValidatedEvent{
			DepositRequestedEvent: de,
			AccountID:             acc.ID.String(),
		})
	}
}
