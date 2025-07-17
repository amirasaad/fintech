package infra_test

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/amirasaad/fintech/internal/fixtures/mocks"
	"github.com/amirasaad/fintech/pkg/domain"
	accountdomain "github.com/amirasaad/fintech/pkg/domain/account"
	events "github.com/amirasaad/fintech/pkg/domain/account/events"
	"github.com/amirasaad/fintech/pkg/handler/account/common"
	"github.com/amirasaad/fintech/pkg/queries"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockEventBus struct {
	published []domain.Event
}

func (m *mockEventBus) Publish(ctx context.Context, event domain.Event) error {
	m.published = append(m.published, event)
	return nil
}
func (m *mockEventBus) Subscribe(eventType string, handler func(context.Context, domain.Event)) {}

func TestEventDrivenValidationFlow_Integration(t *testing.T) {
	// Setup
	uow := mocks.NewMockUnitOfWork(t)
	repo := mocks.NewMockAccountRepository(t)
	bus := &mockEventBus{} // Using the existing mock from testutils_test.go
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	// Create test data
	validUser := uuid.New()
	validAccount := uuid.New()
	acc := &accountdomain.Account{ID: validAccount, UserID: validUser}
	query := queries.GetAccountQuery{
		AccountID: validAccount.String(),
		UserID:    validUser.String(),
	}

	// Setup mocks
	uow.On("AccountRepository").Return(repo, nil)
	repo.On("Get", validAccount).Return(acc, nil)

	// Create handlers
	queryHandler := common.GetAccountQueryHandler(uow, bus)
	validationHandler := common.AccountValidationHandler(bus, logger)

	// Execute the flow
	ctx := context.Background()

	// Step 1: Execute query handler
	result, err := queryHandler.HandleQuery(ctx, query)
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Step 2: Verify query events were published
	assert.Len(t, bus.published, 1, "Query handler should publish AccountQuerySucceededEvent")

	queryEvent, ok := bus.published[0].(events.AccountQuerySucceededEvent)
	assert.True(t, ok, "First event should be AccountQuerySucceededEvent")
	assert.Equal(t, query.UserID, queryEvent.Result.UserID)
	assert.Equal(t, query.AccountID, queryEvent.Result.AccountID)

	// Step 3: Execute validation handler (simulating event bus subscription)
	validationHandler(ctx, queryEvent)

	// Step 4: Verify validation events were published
	assert.Len(t, bus.published, 2, "Validation handler should publish DepositValidatedEvent")

	validationEvent, ok := bus.published[1].(events.AccountValidatedEvent)
	assert.True(t, ok, "Second event should be DepositValidatedEvent")
	assert.Equal(t, query.UserID, validationEvent.UserID)
	assert.Equal(t, query.AccountID, validationEvent.AccountID)

	// Verify the complete flow
	t.Logf("✅ Event-driven validation flow completed successfully:")
	t.Logf("   Query Handler → AccountQuerySucceededEvent")
	t.Logf("   Validation Handler → AccountValidatedEvent")
	t.Logf("   Total events published: %d", len(bus.published))
}

func TestEventDrivenValidationFlow_QueryFailure(t *testing.T) {
	// Setup
	uow := mocks.NewMockUnitOfWork(t)
	repo := mocks.NewMockAccountRepository(t)
	bus := &mockEventBus{}

	// Create test data for failure case
	invalidAccount := uuid.New()
	validUser := uuid.New()
	query := queries.GetAccountQuery{
		AccountID: invalidAccount.String(),
		UserID:    validUser.String(),
	}

	// Setup mocks for failure
	uow.On("AccountRepository").Return(repo, nil)
	repo.On("Get", invalidAccount).Return(nil, assert.AnError)

	// Create query handler
	queryHandler := common.GetAccountQueryHandler(uow, bus)

	// Execute query handler
	ctx := context.Background()
	result, err := queryHandler.HandleQuery(ctx, query)

	// Verify failure
	require.Error(t, err)
	assert.Nil(t, result)

	// Verify failure event was published
	assert.Len(t, bus.published, 1, "Query handler should publish AccountQueryFailedEvent")

	failureEvent, ok := bus.published[0].(events.AccountQueryFailedEvent)
	assert.True(t, ok, "Event should be AccountQueryFailedEvent")
	assert.Equal(t, query, failureEvent.Query)
	assert.NotEmpty(t, failureEvent.Reason)

	t.Logf("✅ Query failure flow completed successfully:")
	t.Logf("   Query Handler → AccountQueryFailedEvent")
	t.Logf("   Error: %s", failureEvent.Reason)
}
