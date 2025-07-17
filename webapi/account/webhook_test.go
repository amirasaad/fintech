package account_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/webapi/account"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
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
	t.Skip("Integration test to be implemented after event-driven refactor is complete")
}
