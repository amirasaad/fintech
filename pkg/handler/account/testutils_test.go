package account

import (
	"context"

	"github.com/amirasaad/fintech/pkg/domain"
)

type mockEventBus struct {
	published []domain.Event
}

func (m *mockEventBus) Publish(ctx context.Context, event domain.Event) error {
	m.published = append(m.published, event)
	return nil
}
func (m *mockEventBus) Subscribe(eventType string, handler func(context.Context, domain.Event)) {}
