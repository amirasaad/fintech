package payment

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/domain/money"

	"log/slog"

	"github.com/amirasaad/fintech/internal/fixtures/mocks"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/amirasaad/fintech/pkg/provider"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockPaymentProvider struct {
	initiateFn func(ctx context.Context, userID, accountID uuid.UUID, amount int64, currency string) (string, error)
}

// GetPaymentStatus implements provider.PaymentProvider.
func (m *mockPaymentProvider) GetPaymentStatus(ctx context.Context, paymentID string) (provider.PaymentStatus, error) {
	panic("unimplemented")
}

func (m *mockPaymentProvider) InitiatePayment(ctx context.Context, userID, accountID uuid.UUID, amount int64, currency string) (string, error) {
	return m.initiateFn(ctx, userID, accountID, amount, currency)
}

func TestPaymentInitiationHandler_BusinessLogic(t *testing.T) {
	userID := uuid.New()
	accountID := uuid.New()
	amount, _ := money.New(100, currency.USD)
	paymentID := "pay-123"

	// Create a mock account for testing
	mockAccount := &account.Account{
		ID:      accountID,
		UserID:  userID,
		Balance: amount,
	}

	tests := []struct {
		name        string
		input       domain.Event
		provider    *mockPaymentProvider
		expectPub   bool
		expectPayID string
		expectUser  uuid.UUID
		expectAcc   uuid.UUID
		expectAmt   int64
		expectCur   string
		setupMocks  func(bus *mocks.MockEventBus)
	}{
		{
			name: "deposit validation success",
			input: events.DepositBusinessValidatedEvent{
				DepositConversionDoneEvent: events.DepositConversionDoneEvent{
					DepositValidatedEvent: events.DepositValidatedEvent{
						DepositRequestedEvent: events.DepositRequestedEvent{
							FlowEvent: events.FlowEvent{
								UserID:        userID,
								AccountID:     accountID,
								CorrelationID: uuid.New(),
								FlowType:      "deposit",
							},
							ID:        uuid.New(),
							Amount:    amount,
							Source:    "deposit",
							Timestamp: time.Now(),
						},
						Account: mockAccount,
					},
					ConversionDoneEvent: events.ConversionDoneEvent{
						FlowEvent: events.FlowEvent{
							UserID:        userID,
							AccountID:     accountID,
							CorrelationID: uuid.New(),
							FlowType:      "deposit",
						},
						ID:         uuid.New(),
						FromAmount: amount,
						ToAmount:   amount,
						RequestID:  uuid.New().String(),
						Timestamp:  time.Now(),
					},
					TransactionID: uuid.New(),
				},
				TransactionID: uuid.New(),
			},
			provider: &mockPaymentProvider{
				initiateFn: func(ctx context.Context, u, a uuid.UUID, amt int64, cur string) (string, error) {
					assert.Equal(t, userID, u)
					assert.Equal(t, accountID, a)
					assert.Equal(t, amount.Amount(), amt)
					assert.Equal(t, amount.Currency().String(), cur)
					return paymentID, nil
				},
			},
			expectPub:   true,
			expectPayID: paymentID,
			expectUser:  userID,
			expectAcc:   accountID,
			expectAmt:   amount.Amount(),
			expectCur:   amount.Currency().String(),
			setupMocks: func(bus *mocks.MockEventBus) {
				bus.On("Publish", mock.Anything, mock.AnythingOfType("events.PaymentInitiatedEvent")).Return(nil)
			},
		},
		{
			name: "withdraw validation success",
			input: events.WithdrawValidatedEvent{
				WithdrawRequestedEvent: events.WithdrawRequestedEvent{
					FlowEvent: events.FlowEvent{
						UserID:        userID,
						AccountID:     accountID,
						CorrelationID: uuid.New(),
						FlowType:      "withdraw",
					},
					ID:        uuid.New(),
					Amount:    amount,
					Timestamp: time.Now(),
				},
				TargetCurrency: amount.Currency().String(),
				Account:        mockAccount,
			},
			provider: &mockPaymentProvider{
				initiateFn: func(ctx context.Context, u, a uuid.UUID, amt int64, cur string) (string, error) {
					assert.Equal(t, userID, u)
					assert.Equal(t, accountID, a)
					assert.Equal(t, amount.Amount(), amt)
					assert.Equal(t, amount.Currency().String(), cur)
					return paymentID, nil
				},
			},
			expectPub:   true,
			expectPayID: paymentID,
			expectUser:  userID,
			expectAcc:   accountID,
			expectAmt:   amount.Amount(),
			expectCur:   amount.Currency().String(),
			setupMocks: func(bus *mocks.MockEventBus) {
				bus.On("Publish", mock.Anything, mock.AnythingOfType("events.PaymentInitiatedEvent")).Return(nil)
			},
		},
		{
			name: "provider error",
			input: events.DepositBusinessValidatedEvent{
				DepositConversionDoneEvent: events.DepositConversionDoneEvent{
					DepositValidatedEvent: events.DepositValidatedEvent{
						DepositRequestedEvent: events.DepositRequestedEvent{
							FlowEvent: events.FlowEvent{
								UserID:        userID,
								AccountID:     accountID,
								CorrelationID: uuid.New(),
								FlowType:      "deposit",
							},
							ID:        uuid.New(),
							Amount:    amount,
							Source:    "deposit",
							Timestamp: time.Now(),
						},
						Account: mockAccount,
					},
					ConversionDoneEvent: events.ConversionDoneEvent{
						FlowEvent: events.FlowEvent{
							UserID:        userID,
							AccountID:     accountID,
							CorrelationID: uuid.New(),
							FlowType:      "deposit",
						},
						ID:         uuid.New(),
						FromAmount: amount,
						ToAmount:   amount,
						RequestID:  uuid.New().String(),
						Timestamp:  time.Now(),
					},
					TransactionID: uuid.New(),
				},
				TransactionID: uuid.New(),
			},
			provider: &mockPaymentProvider{
				initiateFn: func(ctx context.Context, u, a uuid.UUID, amt int64, cur string) (string, error) {
					return "", errors.New("provider failed")
				},
			},
			expectPub:   false,
			expectPayID: "",
			setupMocks:  nil,
		},
		{
			name: "unexpected event type",
			input: events.ConversionDoneEvent{
				FlowEvent: events.FlowEvent{
					UserID:        userID,
					AccountID:     accountID,
					CorrelationID: uuid.New(),
					FlowType:      "conversion",
				},
				ID:         uuid.New(),
				FromAmount: amount,
				ToAmount:   amount,
				RequestID:  uuid.New().String(),
				Timestamp:  time.Now(),
			},
			provider: &mockPaymentProvider{
				initiateFn: func(ctx context.Context, u, a uuid.UUID, amt int64, cur string) (string, error) {
					return paymentID, nil
				},
			},
			expectPub:   false,
			expectPayID: "",
			setupMocks:  nil,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			bus := mocks.NewMockEventBus(t)
			if tc.setupMocks != nil {
				tc.setupMocks(bus)
			}
			handler := PaymentInitiationHandler(bus, tc.provider, slog.Default())
			ctx := context.Background()
			handler(ctx, tc.input)
			if tc.expectPub {
				assert.True(t, bus.AssertCalled(t, "Publish", ctx, mock.AnythingOfType("events.PaymentInitiatedEvent")), "should publish PaymentInitiatedEvent")
			} else {
				bus.AssertNotCalled(t, "Publish", ctx, mock.AnythingOfType("events.PaymentInitiatedEvent"))
			}
		})
	}
}

func TestDepositEventChain_NoInfiniteLoop(t *testing.T) {
	// Setup
	ctx := context.Background()
	userID := uuid.New()
	accountID := uuid.New()
	amount, _ := money.New(100, currency.USD)

	// Track handler invocations
	handlerCalls := make(map[string]int)

	bus := eventbus.NewSimpleEventBus()

	// Minimal stub for account and UoW
	mockAccount := &account.Account{ID: accountID, UserID: userID, Balance: amount}
	txID := uuid.New()

	// Register handlers for the deposit flow
	bus.Subscribe("DepositRequestedEvent", func(ctx context.Context, e domain.Event) {
		handlerCalls["DepositRequestedEvent"]++
		dr := e.(events.DepositRequestedEvent)
		_ = bus.Publish(ctx, events.DepositValidatedEvent{
			DepositRequestedEvent: dr,
			Account:               mockAccount,
		})
	})
	bus.Subscribe("DepositValidatedEvent", func(ctx context.Context, e domain.Event) {
		handlerCalls["DepositValidatedEvent"]++
		ve := e.(events.DepositValidatedEvent)
		_ = bus.Publish(ctx, events.DepositPersistedEvent{
			DepositValidatedEvent: ve,
			TransactionID:         txID,
			Amount:                ve.Amount,
		})
	})
	bus.Subscribe("DepositPersistedEvent", func(ctx context.Context, e domain.Event) {
		handlerCalls["DepositPersistedEvent"]++
		// End of chain for this test
	})

	// Start the chain
	_ = bus.Publish(ctx, events.DepositRequestedEvent{
		FlowEvent: events.FlowEvent{
			UserID:        userID,
			AccountID:     accountID,
			CorrelationID: uuid.New(),
			FlowType:      "deposit",
		},
		ID:        uuid.New(),
		Amount:    amount,
		Source:    "deposit",
		Timestamp: time.Now(),
	})

	// Assert each handler called exactly once
	assert.Equal(t, 1, handlerCalls["DepositRequestedEvent"], "DepositRequestedEvent handler should be called once")
	assert.Equal(t, 1, handlerCalls["DepositValidatedEvent"], "DepositValidatedEvent handler should be called once")
	assert.Equal(t, 1, handlerCalls["DepositPersistedEvent"], "DepositPersistedEvent handler should be called once")
}

func TestDepositAndWithdrawEventChains_NoCrossWorkflowInfiniteLoop(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	accountID := uuid.New()
	amount, _ := money.New(100, currency.USD)

	handlerCalls := make(map[string]int)
	bus := eventbus.NewSimpleEventBus()
	mockAccount := &account.Account{ID: accountID, UserID: userID, Balance: amount}
	txID := uuid.New()

	// Deposit workflow
	bus.Subscribe("DepositRequestedEvent", func(ctx context.Context, e domain.Event) {
		handlerCalls["DepositRequestedEvent"]++
		dr := e.(events.DepositRequestedEvent)
		_ = bus.Publish(ctx, events.DepositValidatedEvent{
			DepositRequestedEvent: dr,
			Account:               mockAccount,
		})
	})
	bus.Subscribe("DepositValidatedEvent", func(ctx context.Context, e domain.Event) {
		handlerCalls["DepositValidatedEvent"]++
		ve := e.(events.DepositValidatedEvent)
		_ = bus.Publish(ctx, events.DepositPersistedEvent{
			DepositValidatedEvent: ve,
			TransactionID:         txID,
			Amount:                ve.Amount,
		})
	})
	bus.Subscribe("DepositPersistedEvent", func(ctx context.Context, e domain.Event) {
		handlerCalls["DepositPersistedEvent"]++
	})

	// Withdraw workflow
	bus.Subscribe("WithdrawRequestedEvent", func(ctx context.Context, e domain.Event) {
		handlerCalls["WithdrawRequestedEvent"]++
		wr := e.(events.WithdrawRequestedEvent)
		_ = bus.Publish(ctx, events.WithdrawValidatedEvent{
			WithdrawRequestedEvent: wr,
			TargetCurrency:         amount.Currency().String(),
			Account:                mockAccount,
		})
	})
	bus.Subscribe("WithdrawValidatedEvent", func(ctx context.Context, e domain.Event) {
		handlerCalls["WithdrawValidatedEvent"]++
		wv := e.(events.WithdrawValidatedEvent)
		_ = bus.Publish(ctx, events.WithdrawPersistedEvent{
			WithdrawValidatedEvent: wv,
			TransactionID:          txID,
		})
	})
	bus.Subscribe("WithdrawPersistedEvent", func(ctx context.Context, e domain.Event) {
		handlerCalls["WithdrawPersistedEvent"]++
	})

	// Start both chains
	_ = bus.Publish(ctx, events.DepositRequestedEvent{
		FlowEvent: events.FlowEvent{
			UserID:        userID,
			AccountID:     accountID,
			CorrelationID: uuid.New(),
			FlowType:      "deposit",
		},
		ID:        uuid.New(),
		Amount:    amount,
		Source:    "deposit",
		Timestamp: time.Now(),
	})
	_ = bus.Publish(ctx, events.WithdrawRequestedEvent{
		FlowEvent: events.FlowEvent{
			UserID:        userID,
			AccountID:     accountID,
			CorrelationID: uuid.New(),
			FlowType:      "withdraw",
		},
		ID:        uuid.New(),
		Amount:    amount,
		Timestamp: time.Now(),
	})

	// Assert each handler called exactly once per event
	assert.Equal(t, 1, handlerCalls["DepositRequestedEvent"], "DepositRequestedEvent handler should be called once")
	assert.Equal(t, 1, handlerCalls["DepositValidatedEvent"], "DepositValidatedEvent handler should be called once")
	assert.Equal(t, 1, handlerCalls["DepositPersistedEvent"], "DepositPersistedEvent handler should be called once")
	assert.Equal(t, 1, handlerCalls["WithdrawRequestedEvent"], "WithdrawRequestedEvent handler should be called once")
	assert.Equal(t, 1, handlerCalls["WithdrawValidatedEvent"], "WithdrawValidatedEvent handler should be called once")
	assert.Equal(t, 1, handlerCalls["WithdrawPersistedEvent"], "WithdrawPersistedEvent handler should be called once")
}

func TestAllWorkflowsEventChains_NoCrossWorkflowInfiniteLoop(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	accountID := uuid.New()
	amount, _ := money.New(100, currency.USD)

	handlerCalls := make(map[string]int)
	bus := eventbus.NewSimpleEventBus()
	mockAccount := &account.Account{ID: accountID, UserID: userID, Balance: amount}
	txID := uuid.New()

	// Deposit workflow
	bus.Subscribe("DepositRequestedEvent", func(ctx context.Context, e domain.Event) {
		handlerCalls["DepositRequestedEvent"]++
		dr := e.(events.DepositRequestedEvent)
		_ = bus.Publish(ctx, events.DepositValidatedEvent{
			DepositRequestedEvent: dr,
			Account:               mockAccount,
		})
	})
	bus.Subscribe("DepositValidatedEvent", func(ctx context.Context, e domain.Event) {
		handlerCalls["DepositValidatedEvent"]++
		ve := e.(events.DepositValidatedEvent)
		_ = bus.Publish(ctx, events.DepositPersistedEvent{
			DepositValidatedEvent: ve,
			TransactionID:         txID,
			Amount:                ve.Amount,
		})
	})
	bus.Subscribe("DepositPersistedEvent", func(ctx context.Context, e domain.Event) {
		handlerCalls["DepositPersistedEvent"]++
	})

	// Withdraw workflow
	bus.Subscribe("WithdrawRequestedEvent", func(ctx context.Context, e domain.Event) {
		handlerCalls["WithdrawRequestedEvent"]++
		wr := e.(events.WithdrawRequestedEvent)
		_ = bus.Publish(ctx, events.WithdrawValidatedEvent{
			WithdrawRequestedEvent: wr,
			TargetCurrency:         amount.Currency().String(),
			Account:                mockAccount,
		})
	})
	bus.Subscribe("WithdrawValidatedEvent", func(ctx context.Context, e domain.Event) {
		handlerCalls["WithdrawValidatedEvent"]++
		wv := e.(events.WithdrawValidatedEvent)
		_ = bus.Publish(ctx, events.WithdrawPersistedEvent{
			WithdrawValidatedEvent: wv,
			TransactionID:          txID,
		})
	})
	bus.Subscribe("WithdrawPersistedEvent", func(ctx context.Context, e domain.Event) {
		handlerCalls["WithdrawPersistedEvent"]++
	})

	// Transfer workflow
	bus.Subscribe("TransferRequestedEvent", func(ctx context.Context, e domain.Event) {
		handlerCalls["TransferRequestedEvent"]++
		tr := e.(events.TransferRequestedEvent)
		_ = bus.Publish(ctx, events.TransferValidatedEvent{TransferRequestedEvent: tr})
	})
	bus.Subscribe("TransferValidatedEvent", func(ctx context.Context, e domain.Event) {
		handlerCalls["TransferValidatedEvent"]++
		tv := e.(events.TransferValidatedEvent)
		_ = bus.Publish(ctx, events.TransferDomainOpDoneEvent{TransferValidatedEvent: tv})
	})
	bus.Subscribe("TransferDomainOpDoneEvent", func(ctx context.Context, e domain.Event) {
		handlerCalls["TransferDomainOpDoneEvent"]++
		do := e.(events.TransferDomainOpDoneEvent)
		_ = bus.Publish(ctx, events.TransferPersistedEvent{TransferDomainOpDoneEvent: do})
	})
	bus.Subscribe("TransferPersistedEvent", func(ctx context.Context, e domain.Event) {
		handlerCalls["TransferPersistedEvent"]++
	})

	// Start all chains
	_ = bus.Publish(ctx, events.DepositRequestedEvent{
		FlowEvent: events.FlowEvent{
			UserID:        userID,
			AccountID:     accountID,
			CorrelationID: uuid.New(),
			FlowType:      "deposit",
		},
		ID:        uuid.New(),
		Amount:    amount,
		Source:    "deposit",
		Timestamp: time.Now(),
	})
	_ = bus.Publish(ctx, events.WithdrawRequestedEvent{
		FlowEvent: events.FlowEvent{
			UserID:        userID,
			AccountID:     accountID,
			CorrelationID: uuid.New(),
			FlowType:      "withdraw",
		},
		ID:        uuid.New(),
		Amount:    amount,
		Timestamp: time.Now(),
	})
	_ = bus.Publish(ctx, events.TransferRequestedEvent{
		FlowEvent: events.FlowEvent{
			UserID:        userID,
			AccountID:     accountID,
			CorrelationID: uuid.New(),
			FlowType:      "transfer",
		},
		ID:             uuid.New(),
		Amount:         amount,
		Source:         "transfer",
		DestAccountID:  accountID,
		ReceiverUserID: userID,
	})

	// Assert each handler called exactly once per event
	assert.Equal(t, 1, handlerCalls["DepositRequestedEvent"], "DepositRequestedEvent handler should be called once")
	assert.Equal(t, 1, handlerCalls["DepositValidatedEvent"], "DepositValidatedEvent handler should be called once")
	assert.Equal(t, 1, handlerCalls["DepositPersistedEvent"], "DepositPersistedEvent handler should be called once")
	assert.Equal(t, 1, handlerCalls["WithdrawRequestedEvent"], "WithdrawRequestedEvent handler should be called once")
	assert.Equal(t, 1, handlerCalls["WithdrawValidatedEvent"], "WithdrawValidatedEvent handler should be called once")
	assert.Equal(t, 1, handlerCalls["WithdrawPersistedEvent"], "WithdrawPersistedEvent handler should be called once")
	assert.Equal(t, 1, handlerCalls["TransferRequestedEvent"], "TransferRequestedEvent handler should be called once")
	assert.Equal(t, 1, handlerCalls["TransferValidatedEvent"], "TransferValidatedEvent handler should be called once")
	assert.Equal(t, 1, handlerCalls["TransferDomainOpDoneEvent"], "TransferDomainOpDoneEvent handler should be called once")
	assert.Equal(t, 1, handlerCalls["TransferPersistedEvent"], "TransferPersistedEvent handler should be called once")
}
