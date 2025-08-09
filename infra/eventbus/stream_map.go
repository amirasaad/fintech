package eventbus

import (
	"fmt"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"strings"
)

func streamNameFor(eventType events.EventType) string {
	return nameFor("events", eventType)
}

// dlqStreamName returns the DLQ stream name for the given event type.
func dlqStreamName(eventType events.EventType) string {
	return nameFor("dlq", eventType)
}

// groupNameFor returns the Redis consumer group name for the event type.
func groupNameFor(eventType events.EventType) string {
	return nameFor("group", eventType)
}

// consumerNameFor returns the Redis consumer name for the event type.
func consumerNameFor(eventType events.EventType) string {
	return nameFor("consumer", eventType)
}

func nameFor(prefix string, eventType events.EventType) string {
	parts := strings.Split(eventType.String(), ".")
	if len(parts) == 2 {
		return fmt.Sprintf(
			"%s:%s:%s",
			prefix,
			strings.ToLower(parts[0]),
			strings.ToLower(parts[1]))
	}
	return fmt.Sprintf("%s:%s", prefix, strings.ToLower(eventType.String()))
}
