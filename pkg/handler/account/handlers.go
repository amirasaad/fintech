// Package account provides event-driven handlers for account domain events.
package account

import (
	"context"

	"github.com/amirasaad/fintech/pkg/domain"
	accountdomain "github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/eventbus"
)

// DepositValidationHandler handles DepositRequestedEvent, performs validation, and publishes DepositValidatedEvent.
func DepositValidationHandler(bus eventbus.EventBus) func(context.Context, domain.Event) {
	return func(ctx context.Context, e domain.Event) {
		de, ok := e.(accountdomain.DepositRequestedEvent)
		if !ok {
			// log error or ignore
			return
		}
		// Simple validation logic
		if de.UserID == "" || de.AccountID == "" || de.Amount <= 0 || de.Currency == "" {
			// Optionally: publish a validation failed event or log
			return
		}
		_ = bus.Publish(ctx, accountdomain.DepositValidatedEvent{DepositRequestedEvent: de})
	}
}

// MoneyCreationHandler handles DepositValidatedEvent, creates money, and publishes MoneyCreatedEvent.
func MoneyCreationHandler(bus eventbus.EventBus) func(context.Context, domain.Event) {
	return func(ctx context.Context, e domain.Event) {
		ve, ok := e.(accountdomain.DepositValidatedEvent)
		if !ok {
			return
		}
		if ve.DepositRequestedEvent.Amount <= 0 { //nolint
			return
		}
		// In a real implementation, you would create a money object here.
		_ = bus.Publish(ctx, accountdomain.MoneyCreatedEvent{DepositValidatedEvent: ve})
	}
}

// DepositPersistenceHandler handles MoneyCreatedEvent, persists to DB, and publishes DepositPersistedEvent.
func DepositPersistenceHandler(bus eventbus.EventBus) func(context.Context, domain.Event) {
	return func(ctx context.Context, e domain.Event) {
		me, ok := e.(accountdomain.MoneyCreatedEvent)
		if !ok {
			return
		}
		// TODO: perform persistence logic
		_ = bus.Publish(ctx, accountdomain.DepositPersistedEvent{MoneyCreatedEvent: me})
	}
}
