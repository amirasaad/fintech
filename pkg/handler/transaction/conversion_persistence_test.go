package transaction

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/amirasaad/fintech/internal/fixtures/mocks"
	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestConversionPersistence(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Run("successfully persists conversion data", func(t *testing.T) {
		// Setup
		uow := mocks.NewMockUnitOfWork(t)

		transactionID := uuid.New()
		event := events.ConversionDoneEvent{
			FlowEvent: events.FlowEvent{
				FlowType:      "deposit",
				UserID:        uuid.New(),
				AccountID:     uuid.New(),
				CorrelationID: uuid.New(),
			},
			TransactionID: transactionID,
			ConversionInfo: &domain.ConversionInfo{
				OriginalAmount:    100.0,
				OriginalCurrency:  currency.USD.String(),
				ConvertedAmount:   85.0,
				ConvertedCurrency: currency.EUR.String(),
				ConversionRate:    0.85,
			},
		}

		// Mock expectations - simplify by just mocking the Do function to return success
		uow.On("Do", mock.Anything, mock.AnythingOfType("func(repository.UnitOfWork) error")).Return(nil).Once()

		// Execute
		handler := ConversionPersistence(uow, logger)
		err := handler(ctx, event)

		// Assert
		assert.NoError(t, err)
	})

	t.Run("handles unexpected event type gracefully", func(t *testing.T) {
		// Setup
		uow := mocks.NewMockUnitOfWork(t)

		// Use a different event type
		event := events.DepositRequestedEvent{}

		// Execute
		handler := ConversionPersistence(uow, logger)
		err := handler(ctx, event)

		// Assert
		assert.NoError(t, err)
		// No interactions should occur with mocks
		uow.AssertNotCalled(t, "Do", mock.Anything, mock.Anything)
	})

	t.Run("handles repository error", func(t *testing.T) {
		// Setup
		uow := mocks.NewMockUnitOfWork(t)

		transactionID := uuid.New()
		event := events.ConversionDoneEvent{
			FlowEvent: events.FlowEvent{
				FlowType:      "deposit",
				UserID:        uuid.New(),
				AccountID:     uuid.New(),
				CorrelationID: uuid.New(),
			},
			TransactionID: transactionID,
			ConversionInfo: &domain.ConversionInfo{
				OriginalAmount:    100.0,
				OriginalCurrency:  currency.USD.String(),
				ConvertedAmount:   85.0,
				ConvertedCurrency: currency.EUR.String(),
				ConversionRate:    0.85,
			},
		}

		// Mock repository error
		uow.On("Do", mock.Anything, mock.AnythingOfType("func(repository.UnitOfWork) error")).
			Return(errors.New("repository error")).Once()

		// Execute
		handler := ConversionPersistence(uow, logger)
		err := handler(ctx, event)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "repository error")
	})
}
