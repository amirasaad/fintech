package events_test

import (
	"testing"
	"time"

	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestNewFlowEvent(t *testing.T) {
	t.Run("creates flow event with default values", func(t *testing.T) {
		e := events.NewFlowEvent()

		assert.NotEqual(t, uuid.Nil, e.ID)
		assert.NotEqual(t, uuid.Nil, e.UserID)
		assert.NotEqual(t, uuid.Nil, e.AccountID)
		assert.NotEqual(t, uuid.Nil, e.CorrelationID)
		assert.False(t, e.Timestamp.IsZero())
	})

	t.Run("applies options in order", func(t *testing.T) {
		userID := uuid.New()
		accountID := uuid.New()
		correlationID := uuid.New()
		customTime := time.Now().Add(-24 * time.Hour)

		e := events.NewFlowEvent(
			func(e *events.FlowEvent) { e.UserID = userID },
			func(e *events.FlowEvent) { e.AccountID = accountID },
			func(e *events.FlowEvent) { e.CorrelationID = correlationID },
			func(e *events.FlowEvent) { e.Timestamp = customTime },
		)

		assert.Equal(t, userID, e.UserID)
		assert.Equal(t, accountID, e.AccountID)
		assert.Equal(t, correlationID, e.CorrelationID)
		assert.Equal(t, customTime, e.Timestamp)
	})
}

func TestFlowEvent_WithUserID(t *testing.T) {
	t.Run("sets user ID", func(t *testing.T) {
		e := &events.FlowEvent{}
		userID := uuid.New()

		result := e.WithUserID(userID)

		assert.Equal(t, userID, e.UserID)
		assert.Same(t, e, result) // Should return the same instance for method chaining
	})
}

func TestFlowEvent_WithID(t *testing.T) {
	t.Run("sets ID", func(t *testing.T) {
		e := &events.FlowEvent{}
		id := uuid.New()

		result := e.WithID(id)

		assert.Equal(t, id, e.ID)
		assert.Same(t, e, result)
	})
}

func TestFlowEvent_WithAccountID(t *testing.T) {
	t.Run("sets account ID", func(t *testing.T) {
		e := &events.FlowEvent{}
		accountID := uuid.New()

		result := e.WithAccountID(accountID)

		assert.Equal(t, accountID, e.AccountID)
		assert.Same(t, e, result)
	})
}

func TestFlowEvent_WithCorrelationID(t *testing.T) {
	t.Run("sets correlation ID", func(t *testing.T) {
		e := &events.FlowEvent{}
		correlationID := uuid.New()

		result := e.WithCorrelationID(correlationID)

		assert.Equal(t, correlationID, e.CorrelationID)
		assert.Same(t, e, result)
	})
}

func TestFlowEvent_WithFlowType(t *testing.T) {
	tests := []struct {
		name     string
		flowType string
		expected string
	}{
		{
			name:     "deposit flow",
			flowType: "deposit",
			expected: "deposit",
		},
		{
			name:     "withdraw flow",
			flowType: "withdraw",
			expected: "withdraw",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &events.FlowEvent{}

			result := e.WithFlowType(tt.flowType)

			assert.Equal(t, tt.expected, e.FlowType)
			assert.Same(t, e, result)
		})
	}
}
