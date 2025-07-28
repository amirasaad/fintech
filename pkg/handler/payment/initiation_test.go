package payment

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain/common"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/domain/money"

	"log/slog"

	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/amirasaad/fintech/pkg/provider"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
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

type mockBus struct {
	handlers map[string][]eventbus.HandlerFunc
}

func (m *mockBus) Emit(ctx context.Context, event common.Event) error {
	handlers := m.handlers[event.Type()]
	for _, handler := range handlers {
		if err := handler(ctx, event); err != nil {
			return err
		}
	}
	return nil
}

func (m *mockBus) Register(eventType string, handler eventbus.HandlerFunc) {
	if m.handlers == nil {
		m.handlers = make(map[string][]eventbus.HandlerFunc)
	}
	m.handlers[eventType] = append(m.handlers[eventType], handler)
}

func TestPaymentInitiation_BusinessLogic(t *testing.T) {
	userID := uuid.New()
	accountID := uuid.New()
	amount, _ := money.New(100, currency.USD)
	paymentID := "pay-123"

	tests := []struct {
		name        string
		input       common.Event
		provider    *mockPaymentProvider
		expectPub   bool
		expectPayID string
		expectUser  uuid.UUID
		expectAcc   uuid.UUID
		expectAmt   int64
		expectCur   string
		setupMocks  func(bus *mockBus)
	}{
		{
			name: "deposit validation success",
			input: events.DepositBusinessValidatedEvent{
				DepositBusinessValidationEvent: events.DepositBusinessValidationEvent{
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
					},
					ConversionDoneEvent: events.ConversionDoneEvent{
						FlowEvent: events.FlowEvent{
							UserID:        userID,
							AccountID:     accountID,
							CorrelationID: uuid.New(),
							FlowType:      "deposit",
						},
						ID:        uuid.New(),
						RequestID: uuid.New().String(),
						Timestamp: time.Now(),
					},
				},
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
			setupMocks: func(bus *mockBus) {
				bus.Register("PaymentInitiatedEvent", func(ctx context.Context, e common.Event) error {
					pi := e.(*events.PaymentInitiatedEvent)
					assert.Equal(t, paymentID, pi.PaymentID)
					assert.Equal(t, userID, pi.UserID)
					assert.Equal(t, accountID, pi.AccountID)
					return nil
				})
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
			setupMocks: func(bus *mockBus) {
				bus.Register("PaymentInitiatedEvent", func(ctx context.Context, e common.Event) error {
					pi := e.(*events.PaymentInitiatedEvent)
					assert.Equal(t, paymentID, pi.PaymentID)
					assert.Equal(t, userID, pi.UserID)
					assert.Equal(t, accountID, pi.AccountID)
					return nil
				})
			},
		},
		{
			name: "provider error",
			input: events.DepositBusinessValidatedEvent{
				DepositBusinessValidationEvent: events.DepositBusinessValidationEvent{
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
					},
					ConversionDoneEvent: events.ConversionDoneEvent{
						FlowEvent: events.FlowEvent{
							UserID:        userID,
							AccountID:     accountID,
							CorrelationID: uuid.New(),
							FlowType:      "deposit",
						},
						ID:        uuid.New(),
						RequestID: uuid.New().String(),
						Timestamp: time.Now(),
					},
				},
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
				ID:        uuid.New(),
				RequestID: uuid.New().String(),
				Timestamp: time.Now(),
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
			bus := &mockBus{}
			if tc.setupMocks != nil {
				tc.setupMocks(bus)
			}
			handler := Initiation(bus, tc.provider, slog.Default())
			ctx := context.Background()
			handler(ctx, tc.input) //nolint:errcheck
			if tc.expectPub {
				assert.True(t, bus.handlers["PaymentInitiatedEvent"] != nil, "should publish PaymentInitiatedEvent")
			} else {
				assert.True(t, bus.handlers["PaymentInitiatedEvent"] == nil, "should not publish PaymentInitiatedEvent")
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

	bus := &mockBus{}

	// Minimal stub for account and UoW
	txID := uuid.New()

	// Register handlers for the deposit flow
	bus.Register("DepositRequestedEvent", func(ctx context.Context, e common.Event) error {
		handlerCalls["DepositRequestedEvent"]++
		dr := e.(events.DepositRequestedEvent)
		_ = bus.Emit(ctx, events.DepositValidatedEvent{
			DepositRequestedEvent: dr,
		})
		return nil
	})
	bus.Register("DepositValidatedEvent", func(ctx context.Context, e common.Event) error {
		handlerCalls["DepositValidatedEvent"]++
		ve := e.(events.DepositValidatedEvent)
		_ = bus.Emit(ctx, events.DepositPersistedEvent{
			DepositValidatedEvent: ve,
			TransactionID:         txID,
			Amount:                ve.Amount,
		})
		return nil
	})
	bus.Register("DepositPersistedEvent", func(ctx context.Context, e common.Event) error {
		handlerCalls["DepositPersistedEvent"]++
		// End of chain for this test
		return nil
	})

	// Start the chain
	_ = bus.Emit(ctx, events.DepositRequestedEvent{
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
	bus := &mockBus{}
	txID := uuid.New()

	// Deposit workflow
	bus.Register("DepositRequestedEvent", func(ctx context.Context, e common.Event) error {
		handlerCalls["DepositRequestedEvent"]++
		dr := e.(events.DepositRequestedEvent)
		_ = bus.Emit(ctx, events.DepositValidatedEvent{
			DepositRequestedEvent: dr,
		})
		return nil
	})
	bus.Register("DepositValidatedEvent", func(ctx context.Context, e common.Event) error {
		handlerCalls["DepositValidatedEvent"]++
		ve := e.(events.DepositValidatedEvent)
		_ = bus.Emit(ctx, events.DepositPersistedEvent{
			DepositValidatedEvent: ve,
			TransactionID:         txID,
			Amount:                ve.Amount,
		})
		return nil
	})
	bus.Register("DepositPersistedEvent", func(ctx context.Context, e common.Event) error {
		handlerCalls["DepositPersistedEvent"]++
		return nil
	})

	// Withdraw workflow
	bus.Register("WithdrawRequestedEvent", func(ctx context.Context, e common.Event) error {
		handlerCalls["WithdrawRequestedEvent"]++
		wr := e.(events.WithdrawRequestedEvent)
		_ = bus.Emit(ctx, events.WithdrawValidatedEvent{
			WithdrawRequestedEvent: wr,
			TargetCurrency:         amount.Currency().String(),
		})
		return nil
	})
	bus.Register("WithdrawValidatedEvent", func(ctx context.Context, e common.Event) error {
		handlerCalls["WithdrawValidatedEvent"]++
		wv := e.(events.WithdrawValidatedEvent)
		_ = bus.Emit(ctx, events.WithdrawPersistedEvent{
			WithdrawValidatedEvent: wv,
			TransactionID:          txID,
		})
		return nil
	})
	bus.Register("WithdrawPersistedEvent", func(ctx context.Context, e common.Event) error {
		handlerCalls["WithdrawPersistedEvent"]++
		return nil
	})

	// Start both chains
	_ = bus.Emit(ctx, events.DepositRequestedEvent{
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
	_ = bus.Emit(ctx, events.WithdrawRequestedEvent{
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
	bus := &mockBus{}
	txID := uuid.New()

	// Deposit workflow
	bus.Register("DepositRequestedEvent", func(ctx context.Context, e common.Event) error {
		handlerCalls["DepositRequestedEvent"]++
		dr := e.(events.DepositRequestedEvent)
		_ = bus.Emit(ctx, events.DepositValidatedEvent{
			DepositRequestedEvent: dr,
		})
		return nil
	})
	bus.Register("DepositValidatedEvent", func(ctx context.Context, e common.Event) error {
		handlerCalls["DepositValidatedEvent"]++
		ve := e.(events.DepositValidatedEvent)
		_ = bus.Emit(ctx, events.DepositPersistedEvent{
			DepositValidatedEvent: ve,
			TransactionID:         txID,
			Amount:                ve.Amount,
		})
		return nil
	})
	bus.Register("DepositPersistedEvent", func(ctx context.Context, e common.Event) error {
		handlerCalls["DepositPersistedEvent"]++
		return nil
	})

	// Withdraw workflow
	bus.Register("WithdrawRequestedEvent", func(ctx context.Context, e common.Event) error {
		handlerCalls["WithdrawRequestedEvent"]++
		wr := e.(events.WithdrawRequestedEvent)
		_ = bus.Emit(ctx, events.WithdrawValidatedEvent{
			WithdrawRequestedEvent: wr,
			TargetCurrency:         amount.Currency().String(),
		})
		return nil
	})
	bus.Register("WithdrawValidatedEvent", func(ctx context.Context, e common.Event) error {
		handlerCalls["WithdrawValidatedEvent"]++
		wv := e.(events.WithdrawValidatedEvent)
		_ = bus.Emit(ctx, events.WithdrawPersistedEvent{
			WithdrawValidatedEvent: wv,
			TransactionID:          txID,
		})
		return nil
	})
	bus.Register("WithdrawPersistedEvent", func(ctx context.Context, e common.Event) error {
		handlerCalls["WithdrawPersistedEvent"]++
		return nil
	})

	// Transfer workflow
	bus.Register("TransferRequestedEvent", func(ctx context.Context, e common.Event) error {
		handlerCalls["TransferRequestedEvent"]++
		tr := e.(events.TransferRequestedEvent)
		_ = bus.Emit(ctx, events.TransferValidatedEvent{TransferRequestedEvent: tr})
		return nil
	})
	bus.Register("TransferValidatedEvent", func(ctx context.Context, e common.Event) error {
		handlerCalls["TransferValidatedEvent"]++
		tv := e.(events.TransferValidatedEvent)
		_ = bus.Emit(ctx, events.TransferDomainOpDoneEvent{TransferValidatedEvent: tv})
		return nil
	})
	bus.Register("TransferDomainOpDoneEvent", func(ctx context.Context, e common.Event) error {
		handlerCalls["TransferDomainOpDoneEvent"]++
		do := e.(events.TransferDomainOpDoneEvent)
		_ = bus.Emit(ctx, events.TransferPersistedEvent{TransferDomainOpDoneEvent: do})
		return nil
	})
	bus.Register("TransferPersistedEvent", func(ctx context.Context, e common.Event) error {
		handlerCalls["TransferPersistedEvent"]++
		return nil
	})

	// Start all chains
	_ = bus.Emit(ctx, events.DepositRequestedEvent{
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
	_ = bus.Emit(ctx, events.WithdrawRequestedEvent{
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
	_ = bus.Emit(ctx, events.TransferRequestedEvent{
		FlowEvent: events.FlowEvent{
			UserID:        userID,
			AccountID:     accountID,
			CorrelationID: uuid.New(),
			FlowType:      "transfer",
		},
		ID:            uuid.New(),
		Amount:        amount,
		Source:        "transfer",
		DestAccountID: accountID,
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
