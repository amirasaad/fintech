package conversion

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"testing"

	"github.com/amirasaad/fintech/internal/fixtures/mocks"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/money"
	"github.com/amirasaad/fintech/pkg/provider/exchange"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestHandleCurrencyConverted(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(
		slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}),
	)

	t.Run("successfully persists conversion data", func(t *testing.T) {

		uow := mocks.NewUnitOfWork(t)
		txRepo := mocks.NewTransactionRepository(t)

		// Create test data
		transactionID := uuid.New()
		convertedAmount, _ := money.New(85.0, money.EUR)
		convInfo := &exchange.RateInfo{
			FromCurrency: "USD",
			ToCurrency:   "EUR",
			Rate:         0.85,
		}

		event := &events.CurrencyConverted{
			CurrencyConversionRequested: events.CurrencyConversionRequested{
				FlowEvent: events.FlowEvent{
					ID:            uuid.New(),
					FlowType:      "deposit",
					UserID:        uuid.New(),
					AccountID:     uuid.New(),
					CorrelationID: uuid.New(),
				},
				TransactionID: transactionID,
			},
			TransactionID:   transactionID,
			ConvertedAmount: convertedAmount,
			ConversionInfo:  convInfo,
		}

		// Mock expectations
		uow.EXPECT().Do(mock.Anything, mock.Anything).Return(nil).Run(
			func(ctx context.Context, fn func(uow repository.UnitOfWork) error) {
				uow.EXPECT().GetRepository(mock.Anything).Return(txRepo, nil).Once()
				txRepo.EXPECT().Update(ctx, transactionID, mock.Anything).Return(nil).Once()
				err := fn(uow)
				require.NoError(t, err)
			},
		).Once()

		handler := HandleCurrencyConverted(uow, logger)
		err := handler(ctx, event)
		require.NoError(t, err)
	})

	t.Run("handles unexpected event type gracefully", func(
		t *testing.T,
	) {
		t.Parallel()
		uow := mocks.NewUnitOfWork(t)

		// Use a different event type
		event := &events.DepositRequested{
			FlowEvent: events.FlowEvent{
				ID:            uuid.New(),
				FlowType:      "deposit",
				UserID:        uuid.New(),
				AccountID:     uuid.New(),
				CorrelationID: uuid.New(),
			},
		}

		handler := HandleCurrencyConverted(uow, logger)
		err := handler(ctx, event)
		require.NoError(t, err) // Should return nil for unexpected event types
		uow.AssertNotCalled(t, "Do", mock.Anything, mock.Anything)
	})

	t.Run("handles nil TransactionID", func(t *testing.T) {
		uow := mocks.NewUnitOfWork(t)

		event := &events.CurrencyConverted{
			CurrencyConversionRequested: events.CurrencyConversionRequested{
				FlowEvent: events.FlowEvent{
					ID:            uuid.New(),
					FlowType:      "deposit",
					UserID:        uuid.New(),
					AccountID:     uuid.New(),
					CorrelationID: uuid.New(),
				},
				TransactionID: uuid.Nil, // Nil TransactionID
			},
			TransactionID: uuid.Nil,
		}

		handler := HandleCurrencyConverted(uow, logger)
		err := handler(ctx, event)
		require.NoError(t, err) // Should return nil for nil TransactionID
		uow.AssertNotCalled(t, "Do", mock.Anything, mock.Anything)
	})

	t.Run("handles nil Info", func(t *testing.T) {
		uow := mocks.NewUnitOfWork(t)

		transactionID := uuid.New()
		convertedAmount, _ := money.New(85.0, money.EUR)

		event := &events.CurrencyConverted{
			CurrencyConversionRequested: events.CurrencyConversionRequested{
				FlowEvent: events.FlowEvent{
					ID:            uuid.New(),
					FlowType:      "deposit",
					UserID:        uuid.New(),
					AccountID:     uuid.New(),
					CorrelationID: uuid.New(),
				},
				TransactionID: transactionID,
			},
			TransactionID:   transactionID,
			ConvertedAmount: convertedAmount,
			ConversionInfo:  nil, // Nil Info
		}

		handler := HandleCurrencyConverted(uow, logger)
		err := handler(ctx, event)
		require.NoError(t, err) // Should return nil for nil Info
		uow.AssertNotCalled(
			t, "Do", mock.Anything, mock.Anything)
	})

	t.Run("handles repository error", func(t *testing.T) {
		uow := mocks.NewUnitOfWork(t)
		txRepo := mocks.NewTransactionRepository(t)

		transactionID := uuid.New()
		convertedAmount, _ := money.New(85.0, money.EUR)
		convInfo := &exchange.RateInfo{
			FromCurrency: fmt.Sprintf("%f", 100.0),
			ToCurrency:   "EUR",
			Rate:         0.85,
		}

		event := &events.CurrencyConverted{
			CurrencyConversionRequested: events.CurrencyConversionRequested{
				FlowEvent: events.FlowEvent{
					ID:            uuid.New(),
					FlowType:      "deposit",
					UserID:        uuid.New(),
					AccountID:     uuid.New(),
					CorrelationID: uuid.New(),
				},
				TransactionID: transactionID,
			},
			TransactionID:   transactionID,
			ConvertedAmount: convertedAmount,
			ConversionInfo:  convInfo,
		}

		expectedErr := errors.New("repository error")
		uow.EXPECT().Do(
			mock.Anything, mock.Anything).Return(expectedErr).Run(
			func(
				ctx context.Context, fn func(uow repository.UnitOfWork) error,
			) {
				uow.EXPECT().GetRepository(mock.Anything).Return(txRepo, nil).Once()
				txRepo.EXPECT().Update(
					ctx, transactionID, mock.Anything).Return(expectedErr).Once()
				err := fn(uow)
				require.Error(t, err)
			},
		).Once()

		handler := HandleCurrencyConverted(uow, logger)
		err := handler(ctx, event)
		require.Error(t, err)
		require.Equal(t, expectedErr, err)
	})

	t.Run("handles invalid repository type", func(t *testing.T) {
		uow := mocks.NewUnitOfWork(t)

		transactionID := uuid.New()
		convertedAmount, _ := money.New(85.0, money.EUR)
		convInfo := &exchange.RateInfo{
			FromCurrency: fmt.Sprintf("%f", 100.0),
			ToCurrency:   "EUR",
			Rate:         0.85,
		}

		event := &events.CurrencyConverted{
			CurrencyConversionRequested: events.CurrencyConversionRequested{
				FlowEvent: events.FlowEvent{
					ID:            uuid.New(),
					FlowType:      "deposit",
					UserID:        uuid.New(),
					AccountID:     uuid.New(),
					CorrelationID: uuid.New(),
				},
				TransactionID: transactionID,
			},
			TransactionID:   transactionID,
			ConvertedAmount: convertedAmount,
			ConversionInfo:  convInfo,
		}

		expectedErr := errors.New("repository type assertion error")
		uow.EXPECT().Do(
			mock.Anything, mock.Anything).Return(expectedErr).Run(
			func(
				ctx context.Context, fn func(uow repository.UnitOfWork) error,
			) {
				uow.EXPECT().
					GetRepository(
						mock.Anything,
					).Return(
					nil,
					expectedErr,
				).Once() // Simulate error from GetRepository
				err := fn(uow)
				require.Error(t, err)
			},
		).Once()

		handler := HandleCurrencyConverted(uow, logger)
		err := handler(ctx, event)
		require.Error(t, err)
		require.Equal(t, expectedErr, err)
	})
}
