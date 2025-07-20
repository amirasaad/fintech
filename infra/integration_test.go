package infra_test

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/eventbus"

	"github.com/amirasaad/fintech/internal/fixtures/mocks"
	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/amirasaad/fintech/pkg/dto"
	"github.com/amirasaad/fintech/pkg/handler/account/deposit"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockBus struct {
	published []domain.Event
}

func (m *mockBus) Emit(ctx context.Context, event domain.Event) error {
	m.published = append(m.published, event)
	return nil
}
func (m *mockBus) Register(eventType string, handler eventbus.HandlerFunc) {}

func TestDrivenDepositFlow_Integration(t *testing.T) {
	// Setup
	uow := mocks.NewMockUnitOfWork(t)
	repo := mocks.NewAccountRepository(t)
	bus := &mockBus{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	// Create test data
	validUser := uuid.New()
	validAccount := uuid.New()

	uow.EXPECT().GetRepository(mock.Anything).Return(repo, nil)
	uow.EXPECT().Do(mock.Anything, mock.Anything).Return(nil)

	repo.EXPECT().Get(mock.Anything, validAccount).Return(&dto.AccountRead{ID: validAccount, UserID: validUser, Balance: 10000, Currency: "USD"}, nil)

	ctx := context.Background()

	// Step 1: Simulate DepositRequested
	depositRequested := events.DepositRequestedEvent{
		ID: uuid.New(),
		FlowEvent: events.FlowEvent{
			FlowType:  "deposit",
			AccountID: validAccount,
			UserID:    validUser,
		},
		Amount: money.NewFromData(10000, "USD"),
		Source: "Cash",
	}

	// Step 2: Validation Handler
	validationHandler := deposit.ValidationHandler(bus, uow, logger)
	validationHandler(ctx, depositRequested) //nolint:errcheck
	assert.Len(t, bus.published, 1, "Validation handler should publish DepositValidated")

	depositValidated, ok := bus.published[0].(events.DepositValidatedEvent)
	require.True(t, ok, "First event should be DepositValidated")
	assert.Equal(t, validUser, depositValidated.UserID)
	assert.Equal(t, validAccount, depositValidated.AccountID)

	// Step 3: Persistence Handler
	persistHandler := deposit.PersistenceHandler(bus, uow, logger)
	persistHandler(ctx, depositValidated) //nolint:errcheck
	assert.Len(t, bus.published, 3, "Persistence handler should publish DepositPersisted and ConversionRequested")

	depositPersisted, ok := bus.published[1].(events.DepositPersistedEvent)
	require.True(t, ok, "Second event should be DepositPersisted")
	assert.Equal(t, validUser, depositPersisted.UserID)
	assert.Equal(t, validAccount, depositPersisted.AccountID)

	conversionRequested, ok := bus.published[2].(events.ConversionRequestedEvent)
	require.True(t, ok, "Third event should be ConversionRequested")
	assert.Equal(t, "deposit", conversionRequested.FlowType)
	assert.Equal(t, validUser, conversionRequested.UserID)
	assert.Equal(t, validAccount, conversionRequested.AccountID)

	t.Logf("Published events: %#v", bus.published)
	t.Logf("✅ -driven deposit flow completed successfully:")
	t.Logf("   DepositRequested → DepositValidated → DepositPersisted")
	t.Logf("   Total events published: %d", len(bus.published))
}
