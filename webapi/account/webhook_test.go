package account_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/amirasaad/fintech/pkg/domain/events"

	"io"
	"log/slog"

	fixturesmocks "github.com/amirasaad/fintech/internal/fixtures/mocks"
	"github.com/amirasaad/fintech/pkg/domain"
	accountdomain "github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/amirasaad/fintech/pkg/dto"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/amirasaad/fintech/pkg/handler/payment"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/amirasaad/fintech/webapi/account"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// --- Mock EventBus for unit test ---
type mockBus struct {
	handlers map[string][]eventbus.HandlerFunc
}

func (m *mockBus) Emit(ctx context.Context, event domain.Event) error {
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

func TestStripeWebhookHandler_PublishesEvent(t *testing.T) {
	app := fiber.New()
	var called bool
	mockBus := &mockBus{}
	mockBus.Register("SomeEventType", func(ctx context.Context, event domain.Event) error {
		called = true
		return nil
	})
	app.Post("/webhook/stripe", account.StripeWebhookHandler(mockBus, "test_secret"))

	stripeEvent := map[string]interface{}{
		"id":   "evt_test",
		"type": "payment_intent.succeeded",
		"data": map[string]interface{}{
			"object": map[string]interface{}{
				"id":     "pi_test",
				"status": "succeeded",
			},
			"raw": json.RawMessage(`{"id":"pi_test","status":"succeeded"}`),
		},
	}
	body, _ := json.Marshal(stripeEvent)
	req := httptest.NewRequest(http.MethodPost, "/webhook/stripe", bytes.NewReader(body))
	resp, _ := app.Test(req)

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.True(t, called, "EventBus should be called")
	// Optionally, assert on mockBus.lastEvent fields if needed
}

// --- Integration-style test (pseudo, to be expanded with real infra/mocks) ---
// This is a placeholder for a full integration test that would:
// - Set up a real/mocked event bus
// - Register a payment event handler that updates a transaction/account
// - Simulate the webhook call
// - Assert on the updated state
//
// For now, this is a stub to be filled in as the event-driven refactor proceeds.
func TestStripeWebhookHandler_Integration(t *testing.T) {
	// Mocks
	txRepo := fixturesmocks.NewMockTransactionRepository(t)
	accRepo := fixturesmocks.NewAccountRepository(t)
	uow := fixturesmocks.NewMockUnitOfWork(t)

	// Use a real *account.Transaction for the fake transaction
	fakeTx := &accountdomain.Transaction{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		AccountID: uuid.New(),
		Status:    "initiated",
		PaymentID: "pi_test",
		Amount:    money.NewFromData(1000, "USD"), // Use a valid amount
		Balance:   money.NewFromData(0, "USD"),
	}

	// Mock account data
	fakeAccount := &dto.AccountRead{
		ID:       fakeTx.AccountID,
		UserID:   fakeTx.UserID,
		Balance:  0,
		Currency: "USD",
	}

	// Set up transaction repository mocks
	txRepo.On("GetByPaymentID", "pi_test").Return(fakeTx, nil)
	txRepo.On("Update", fakeTx).Run(func(args mock.Arguments) {
		fakeTx.Status = "succeeded"
	}).Return(nil)

	// Set up account repository mocks
	accRepo.On("Get", mock.Anything, fakeTx.AccountID).Return(fakeAccount, nil)
	accRepo.On("Update", mock.Anything, fakeTx.AccountID, mock.Anything).Return(nil)

	// Set up unit of work mocks
	uow.On("TransactionRepository").Return(txRepo, nil)
	uow.On("GetRepository", mock.Anything).Return(accRepo, nil)
	uow.On("Do", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		// Execute the function passed to Do()
		fn := args.Get(1).(func(repository.UnitOfWork) error)
		fn(uow) //nolint:errcheck
	})

	// Set up event bus and register handler
	bus := eventbus.NewSimpleEventBus()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	bus.Subscribe((events.PaymentCompletedEvent{}).Type(), payment.Completed(bus, uow, logger))
	// Remove other handlers that might interfere with the test
	// bus.Subscribe((events.PaymentInitiationEvent{}).Type(), payment.PaymentInitiationHandler(bus, provider.NewMockPaymentProvider(), logger))
	// bus.Subscribe((events.PaymentIdPersistedEvent{}).Type(), payment.Persistence(bus, uow, logger))

	// Set up Fiber app with webhook handler
	app := fiber.New()
	app.Post("/webhook/stripe", account.StripeWebhookHandler(bus, "test_secret"))

	// Simulate Stripe webhook call
	stripeEvent := map[string]interface{}{
		"id":   "evt_test",
		"type": "payment_intent.succeeded",
		"data": map[string]interface{}{
			"object": map[string]interface{}{
				"id":     "pi_test",
				"status": "succeeded",
			},
			"raw": json.RawMessage(`{"id":"pi_test","status":"succeeded"}`),
		},
	}
	body, _ := json.Marshal(stripeEvent)
	req := httptest.NewRequest(http.MethodPost, "/webhook/stripe", bytes.NewReader(body))
	resp, _ := app.Test(req)

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	txRepo.AssertCalled(t, "GetByPaymentID", "pi_test")
	txRepo.AssertCalled(t, "Update", fakeTx)
	assert.Equal(t, "succeeded", string(fakeTx.Status))
}
