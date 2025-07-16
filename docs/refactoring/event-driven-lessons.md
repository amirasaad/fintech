---

## 📚 Lessons Learned: Mixing Event-Driven and Chain of Responsibility Patterns

### 1. What We Tried

- **Event-Driven (ED):** Used an event bus to decouple business actions (e.g., deposit, payment completed). Each event could trigger multiple handlers, supporting async and modular workflows.
- **Chain of Responsibility (CoR):** Used chains for stepwise processing (validation → conversion → payment → persistence). Each handler in the chain could pass or halt the request.
- **Mix:** Sometimes, event handlers themselves were implemented as chains. Chains were sometimes invoked directly by the event bus.

### 2. What Worked Well

- **Decoupling:** Event bus made it easy to add new reactions to business events.
- **Stepwise Processing:** Chains made it easy to compose complex operations from simple steps.
- **Testability:** Each handler (event or chain) could be tested in isolation.

### 3. What Became Painful

- **Responsibility Confusion:** It was unclear whether to add new business logic to the event handler or the chain.
- **Complexity Growth:** As the system grew, the interplay between event bus and chains became hard to reason about.
- **Debugging:** Tracing the flow of a single business action required jumping between event bus, event handler, and chain.
- **Extensibility:** Adding new event types or business rules sometimes required changes in multiple places, violating the open/closed principle.
- **Testing:** Integration tests became more brittle as the event/chain boundaries blurred.

### 4. Key Lessons

- **Single Responsibility Principle:** Each architectural pattern should have a clear, bounded responsibility.
- **Event Bus = Dispatcher:** The event bus should only dispatch events to handlers, not orchestrate business logic.
- **Chains = Local Orchestration:** Chains are best used for stepwise processing *within* a single event handler, not as a system-wide dispatcher.
- **Handler Registry > Switches:** Use a map of event types to handler functions for clarity and extensibility.
- **Explicit Boundaries:** Keep event-driven and chain-of-responsibility patterns at separate layers.

### 5. How We’ll Move Forward

- **Event Bus:** Will dispatch events to dedicated handler functions via a registry.
- **Event Handlers:** Will orchestrate business logic for a single event type. May use a chain internally for complex, stepwise logic.
- **No More Mixed Dispatch:** Chains will not be registered as event handlers at the bus level.
- **Documentation & Tests:** Each event type and handler will be documented and tested in isolation.

### 6. Sample Refactored Pattern

```go
// Event bus with handler registry
var eventHandlers = map[string]EventHandler{
    "DepositRequestedEvent": handleDepositRequested,
    "PaymentCompletedEvent": handlePaymentCompleted,
    // ...
}

func (bus *EventBus) Publish(event Event) {
    if handler, ok := eventHandlers[event.Type]; ok {
        go handler(event)
    }
}

// Example event handler using a chain internally
func handleDepositRequested(event Event) error {
    chain := NewDepositProcessingChain()
    return chain.Process(event)
}
```

### 7. Final Thought
>
> “Architecture is about boundaries and clarity. When in doubt, make responsibilities explicit and keep patterns focused.”

---
