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

	"github.com/amirasaad/fintech/infra/eventbus"

	"github.com/amirasaad/fintech/internal/fixtures/mocks"
	accountdomain "github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/amirasaad/fintech/pkg/dto"
	"github.com/amirasaad/fintech/pkg/handler/payment"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/amirasaad/fintech/webapi/account"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestStripeWebhookHandler_PublishesEvent(t *testing.T) {
	app := fiber.New()
	var called bool
	mockBus := eventbus.NewWithMemory(slog.New(slog.NewTextHandler(io.Discard, nil)))
	// Register for the correct event type emitted by the handler
	mockBus.Register(
		events.EventTypePaymentCompleted,
		func(ctx context.Context, event events.Event) error {
			called = true
			t.Logf("Handler called for event type: %s", event.Type())
			return nil
		},
	)
	t.Logf("Registered event type: %s", events.EventTypePaymentCompleted)
	app.Post("/webhook/stripe", account.StripeWebhookHandler(mockBus, "test_secret"))

	stripeEvent := map[string]any{
		"id":   "evt_test",
		"type": "payment_intent.succeeded",
		"data": map[string]any{
			"object": map[string]any{
				"id":     "pi_test",
				"status": "succeeded",
			},
			"raw": json.RawMessage(`{"id":"pi_test","status":"succeeded"}`),
		},
	}
	body, _ := json.Marshal(stripeEvent)
	req := httptest.NewRequest(http.MethodPost, "/webhook/stripe", bytes.NewReader(body))
	resp, _ := app.Test(req)

	// Debug: print response status
	t.Logf("Stripe webhook response status: %d", resp.StatusCode)
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
	txRepo := mocks.NewMockTransactionRepository(t)
	accRepo := mocks.NewAccountRepository(t)
	uow := mocks.NewMockUnitOfWork(t)

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

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	bus := eventbus.NewWithMemory(logger)
	// Use the correct event type constant from events package
	bus.Register(events.EventTypePaymentCompleted, payment.HandleCompleted(bus, uow, logger))

	// Set up Fiber app with webhook handler
	app := fiber.New()
	app.Post("/webhook/stripe", account.StripeWebhookHandler(bus, "test_secret"))

	// Simulate Stripe webhook call
	stripeEvent := map[string]any{
		"id":   "evt_test",
		"type": "payment_intent.succeeded",
		"data": map[string]any{
			"object": map[string]any{
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
