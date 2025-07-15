package account

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/amirasaad/fintech/infra/eventbus"
	"github.com/amirasaad/fintech/internal/fixtures/mocks"
	"github.com/amirasaad/fintech/pkg/config"
	"github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/domain/money"
	accountsvc "github.com/amirasaad/fintech/pkg/service/account"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestPaymentWebhookHandler_Integration(t *testing.T) {
	uow := mocks.NewMockUnitOfWork(t)
	mockRepo := mocks.NewMockTransactionRepository(t)
	uow.EXPECT().TransactionRepository().Return(mockRepo, nil)
	eventBus := &eventbus.MemoryEventBus{}
	svc := accountsvc.NewService(config.Deps{
		Uow:      uow,
		EventBus: eventBus,
		// Add other required deps as needed (mock or real)
	})

	// Pre-create a transaction
	tx := &account.Transaction{
		ID:        uuid.New(),
		AccountID: uuid.New(),
		UserID:    uuid.New(),
		Amount:    money.NewFromData(1000, "USD"), // Use your test money constructor
		Balance:   money.NewFromData(1000, "USD"),
		Status:    account.TransactionStatusPending,
		PaymentID: "pid123",
	}
	mockRepo.EXPECT().GetByPaymentID(mock.Anything).Return(tx, nil)
	mockRepo.EXPECT().Update(mock.Anything).Return(nil)
	app := fiber.New()
	app.Post("/webhook/payment", PaymentWebhookHandler(svc))

	// Send webhook
	request := httptest.NewRequest(
		"POST", "/webhook/payment",
		bytes.NewBufferString(
			`{"payment_id": "pid123", "status": "completed"}`),
	)
	request.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(request)
	assert.NoError(t, err)
	// Log response for debugging
	var respBody bytes.Buffer
	_, _ = respBody.ReadFrom(resp.Body) //nolint:errcheck
	if resp.StatusCode != http.StatusOK {
		t.Logf("Webhook response: %s", respBody.String())
	}
	defer resp.Body.Close() //nolint:errcheck
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	// Assert transaction status updated
	assert.Equal(t, string(account.TransactionStatusCompleted), tx.Status)
}
