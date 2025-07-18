package account_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/amirasaad/fintech/pkg/domain/events"

	"log/slog"

	"github.com/amirasaad/fintech/infra/provider"
	fixturesmocks "github.com/amirasaad/fintech/internal/fixtures/mocks"
	"github.com/amirasaad/fintech/pkg/domain"
	accountdomain "github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/amirasaad/fintech/pkg/handler/payment"
	"github.com/amirasaad/fintech/webapi/account"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// --- Mock EventBus for unit test ---
type mockEventBus struct {
	called    bool
	lastEvent domain.Event
	returnErr error
}

func (m *mockEventBus) Publish(_ context.Context, event domain.Event) error {
	m.called = true
	m.lastEvent = event
	return m.returnErr
}
func (m *mockEventBus) Subscribe(_ string, _ func(context.Context, domain.Event)) {}

func TestStripeWebhookHandler_PublishesEvent(t *testing.T) {
	app := fiber.New()
	mockBus := &mockEventBus{}
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
	assert.True(t, mockBus.called, "EventBus should be called")
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
	uow := fixturesmocks.NewMockUnitOfWork(t)
	uow.On("TransactionRepository").Return(txRepo, nil)

	// Use a real *account.Transaction for the fake transaction
	fakeTx := &accountdomain.Transaction{
		Status:    "initiated",
		PaymentID: "pi_test",
		Amount:    money.NewFromData(0, "USD"), // or a valid amount/currency
		Balance:   money.NewFromData(0, "USD"),
	}

	txRepo.On("GetByPaymentID", "pi_test").Return(fakeTx, nil)
	txRepo.On("Update", fakeTx).Run(func(args mock.Arguments) {
		fakeTx.Status = "succeeded"
	}).Return(nil)

	// Set up event bus and register handler
	bus := eventbus.NewSimpleEventBus()
	logger := slog.New(slog.NewTextHandler(nil, nil))
	bus.Subscribe((events.PaymentCompletedEvent{}).EventType(), payment.CompletedHandler(bus, uow, logger))
	bus.Subscribe((events.PaymentInitiationEvent{}).EventType(), payment.PaymentInitiationHandler(bus, provider.NewMockPaymentProvider(), logger))
	bus.Subscribe((events.PaymentIdPersistedEvent{}).EventType(), payment.PersistenceHandler(bus, uow, logger))

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
