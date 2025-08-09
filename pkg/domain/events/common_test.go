package events_test

import (
	"testing"
	"time"

	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestFlowEvent_Fields(t *testing.T) {
	t.Run("check default values", func(t *testing.T) {
		e := events.FlowEvent{
			ID:            uuid.New(),
			FlowType:      "test",
			UserID:        uuid.New(),
			AccountID:     uuid.New(),
			CorrelationID: uuid.New(),
			Timestamp:     time.Now(),
		}

		assert.NotEqual(t, uuid.Nil, e.ID)
		assert.Equal(t, "test", e.FlowType)
		assert.NotEqual(t, uuid.Nil, e.UserID)
		assert.NotEqual(t, uuid.Nil, e.AccountID)
		assert.NotEqual(t, uuid.Nil, e.CorrelationID)
		assert.False(t, e.Timestamp.IsZero())
	})
}
