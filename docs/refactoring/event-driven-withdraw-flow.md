---
icon: octicons/sync-24
---
# ⚡ Event-Driven Withdraw Flow

This document describes the event-driven architecture for the withdraw workflow in the fintech system.

---

## 🏁 Overview

The withdraw process is fully event-driven, with each business step handled by a dedicated event handler. This enables modularity, testability, and clear separation of concerns, following the same principles as the deposit flow.

---

## 🖼️ Sequence Diagram

```mermaid
sequenceDiagram
    participant U as "User"
    participant API as "API Handler"
    participant EB as "EventBus"
    participant VH as "WithdrawValidationHandler"
    participant WC as "WithdrawConversionHandler"
    participant WCD as "WithdrawConversionDone"
    participant PI as "PaymentInitiationHandler"
    participant P as "PersistenceHandler"

    U->>API: POST /account/:id/withdraw (WithdrawRequest)
    API->>EB: WithdrawRequestedEvent
    EB->>VH: WithdrawValidationHandler
    VH->>EB: WithdrawValidatedEvent
    EB->>WC: WithdrawConversionHandler (if needed)
    WC->>EB: WithdrawConversionDone
    EB->>PI: PaymentInitiationHandler
    PI->>EB: PaymentInitiatedEvent
    EB->>P: PersistenceHandler
    P->>EB: WithdrawPersistedEvent
```

> **Note:** Payment initiation only happens after WithdrawConversionDone. This avoids coupling conversion with payment for other flows (e.g., transfer).

---

## 🔄 Workflow Clarification: Event-Driven Withdraw Flow

The withdraw workflow is orchestrated through a series of events and handlers:

1. **User submits withdraw request** (amount as `float64`, main unit). API emits `WithdrawRequestedEvent`.
2. **Validation Handler** loads the account, checks balance and domain validation (`ValidateWithdraw`), emits `WithdrawValidatedEvent`.
3. **Persistence Handler** converts the amount to a `money.Money` value object and persists the transaction, emits `WithdrawPersistedEvent`.
4. **Payment Initiation Handler** initiates payment, emits `PaymentInitiatedEvent`.
5. **PaymentId Persistence Handler** updates transaction with paymentId, emits `PaymentIdPersistedEvent`.
6. **Webhook Handler** (optional) updates transaction status and account balance on payment confirmation.

### 🖼️ Withdraw Workflow Diagram

```mermaid
flowchart TD
    A["WithdrawRequestedEvent"] --> B["Validation Handler (domain validation)"]
    B --> C["WithdrawValidatedEvent"]
    C --> D["Persistence Handler (creates money object, persists)"]
    D --> E["WithdrawPersistedEvent"]
    E --> F["Payment Initiation Handler"]
    F --> G["PaymentInitiatedEvent"]
    G --> H["PaymentId Persistence Handler"]
    H --> I["PaymentIdPersistedEvent"]
    I --> J["Webhook Handler (optional)"]
```

---

## 🧩 Event-Driven Components

### 1. Validation Handler

- **Purpose:** Performs business validation on the account and balance
- **Events Consumed:** `WithdrawRequestedEvent`
- **Events Emitted:**
  - `WithdrawValidatedEvent` - When validation passes
  - (TODO: `WithdrawValidationFailedEvent` - When validation fails)
- **Validation Rules:**
  - Account exists and belongs to user
  - Account has valid ID
  - Account is in valid state for operations
  - Sufficient balance for withdrawal

### 2. Persistence Handler

- **Purpose:** Converts the amount to a `money.Money` value object and persists the withdraw transaction to the database
- **Events Consumed:** `WithdrawValidatedEvent`
- **Events Emitted:** `WithdrawPersistedEvent`

### 3. Payment Initiation Handler

- **Purpose:** Initiates payment with external providers
- **Events Consumed:** `WithdrawPersistedEvent`
- **Events Emitted:** `PaymentInitiatedEvent`

---

## 🛠️ Key Benefits

### 1. **Modularity**

Each handler has a single responsibility and can be developed, tested, and deployed independently.

### 2. **Testability**

- Unit tests for each handler
- Integration tests for event flows
- Easy mocking of dependencies

### 3. **Scalability**

- Handlers can be scaled independently
- Event-driven architecture supports async processing
- Easy to add new handlers without modifying existing code

### 4. **Maintainability**

- Clear separation of concerns
- Easy to understand and modify individual components
- Consistent patterns across all handlers

### 5. **Event Sourcing Ready**

- All business events are captured
- Easy to implement event sourcing patterns
- Audit trail of all operations

---

## 🛠️ Implementation Details

### Validation Handler Pattern

```go
// Validation handler listens to withdraw request events
func WithdrawValidationHandler(bus eventbus.EventBus, logger *slog.Logger) func(context.Context, domain.Event) {
    return func(ctx context.Context, e domain.Event) {
        event, ok := e.(accountdomain.WithdrawRequestedEvent)
        if !ok {
            return
        }

        // Perform business validation
        if validationFails {
            // TODO: Emit WithdrawValidationFailedEvent
            return
        }

        // Emit validation success
        _ = bus.Publish(ctx, accountdomain.WithdrawValidatedEvent{...})
    }
}
```

### Persistence Handler Pattern

```go
// Persistence handler listens to validated withdraw events
func WithdrawPersistenceHandler(bus eventbus.EventBus, uow repository.UnitOfWork, logger *slog.Logger) func(context.Context, domain.Event) {
    return func(ctx context.Context, e domain.Event) {
        event, ok := e.(accountdomain.WithdrawValidatedEvent)
        if !ok {
            return
        }
        // Convert amount to money.Money and persist transaction
        // ...
        _ = bus.Publish(ctx, accountdomain.WithdrawPersistedEvent{...})
    }
}
```

---

## 🛠️ Error Handling

### Validation Failures

- Account inactive
- Insufficient balance
- Business rule violations
- Invalid account state

### Event Flow on Errors

1. Validation handler emits `WithdrawValidationFailedEvent` (TODO)
2. Persistence handler is not triggered
3. Error is returned to the caller
4. Audit trail is maintained through events

---

## 🧪 Testing Strategy

### Unit Tests

- Test each handler independently
- Mock event bus and dependencies
- Test success and failure scenarios

### Integration Tests

- Test complete event flows
- Use real event bus
- Verify event sequences

### End-to-End Tests

- Test full API endpoints
- Verify business outcomes
