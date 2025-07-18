---
icon: octicons/sync-24
---
# ⚡ Event-Driven Transfer Flow

This document describes the event-driven architecture for the transfer workflow in the fintech system.

---

## 🏁 Overview

The transfer process is fully event-driven, with each business step handled by a dedicated event handler. This enables modularity, testability, and clear separation of concerns, following the same principles as the deposit and withdraw flows.

---

## 🖼️ Sequence Diagram

```mermaid
sequenceDiagram
    participant U as "User"
    participant API as "API Handler"
    participant EB as "EventBus"
    participant VH as "ValidationHandler"
    participant CC as "CurrencyConversionHandler"
    participant DO as "DomainOpHandler"
    participant P as "PersistenceHandler"

    U->>API: POST /account/:id/transfer (TransferRequest)
    API->>EB: TransferRequestedEvent
    EB->>VH: ValidationHandler (validates source/target accounts)
    VH->>EB: TransferValidatedEvent
    EB->>CC: CurrencyConversionHandler (if needed)
    CC->>EB: MoneyConvertedEvent
    EB->>DO: DomainOpHandler (executes transfer on domain)
    DO->>EB: TransferDomainOpDoneEvent
    EB->>P: PersistenceHandler (persists to DB)
    P->>EB: TransferPersistedEvent
```

---

## 🔄 Workflow Clarification: Event-Driven Transfer Flow

The transfer workflow is orchestrated through a series of events and handlers:

1. **User submits transfer request** (amount as `float64`, main unit). API emits `TransferRequestedEvent`.
2. **Validation Handler** loads source and target accounts, checks domain validation (`ValidateTransfer`), emits `TransferValidatedEvent`.
3. **Currency Conversion Handler** (if needed) converts currency, emits `MoneyConvertedEvent`.
4. **Domain Operation Handler** executes the transfer on the domain model, emits `TransferDomainOpDoneEvent`.
5. **Persistence Handler** persists the transaction(s), emits `TransferPersistedEvent`.
6. **Webhook Handler** (optional) updates transaction status and account balances on payment confirmation.

### 🖼️ Transfer Workflow Diagram

```mermaid
flowchart TD
    A["TransferRequestedEvent"] --> B["Validation Handler (domain validation)"]
    B --> C["TransferValidatedEvent"]
    E --> F["Currency Conversion Handler (if needed)"]
    F --> G["MoneyConvertedEvent"]
    G --> H["Domain Operation Handler"]
    H --> I["TransferDomainOpDoneEvent"]
    I --> J["Persistence Handler (persists)"]
    J --> K["TransferPersistedEvent"]
    K --> L["Webhook Handler (optional)"]
```

---

## 🧩 Event-Driven Components

### 1. Validation Handler

- **Purpose:** Performs business validation on source and target accounts
- **Events Consumed:** `TransferRequestedEvent`
- **Events Emitted:**
  - `TransferValidatedEvent` - When validation passes
  - (TODO: `TransferValidationFailedEvent` - When validation fails)
- **Validation Rules:**
  - Source and target accounts exist and belong to user
  - Accounts have valid IDs
  - Accounts are in valid state for operations
  - Sufficient balance in source account

### 2. Money Creation Handler

- **Purpose:** Creates a `money.Money` value object for the transfer
- **Events Consumed:** `TransferValidatedEvent`
- **Events Emitted:** `MoneyCreatedEvent`

### 3. Currency Conversion Handler

- **Purpose:** Converts currency if source and target accounts use different currencies
- **Events Consumed:** `MoneyCreatedEvent`
- **Events Emitted:** `MoneyConvertedEvent`

### 4. Domain Operation Handler

- **Purpose:** Executes the transfer on the domain model
- **Events Consumed:** `MoneyConvertedEvent` (or `MoneyCreatedEvent` if no conversion)
- **Events Emitted:** `TransferDomainOpDoneEvent`

### 5. Persistence Handler

- **Purpose:** Persists the transfer transaction(s) to the database
- **Events Consumed:** `TransferDomainOpDoneEvent`
- **Events Emitted:** `TransferPersistedEvent`

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
// Validation handler listens to transfer request events
func TransferValidationHandler(bus eventbus.EventBus, logger *slog.Logger) func(context.Context, domain.Event) {
    return func(ctx context.Context, e domain.Event) {
        event, ok := e.(accountdomain.TransferRequestedEvent)
        if !ok {
            return
        }

        // Perform business validation
        if validationFails {
            // TODO: Emit TransferValidationFailedEvent
            return
        }

        // Emit validation success
        _ = bus.Publish(ctx, accountdomain.TransferValidatedEvent{...})
    }
}
```

### Money Creation Handler Pattern

```go
// Money creation handler listens to validated transfer events
func MoneyCreationHandler(bus eventbus.EventBus, logger *slog.Logger) func(context.Context, domain.Event) {
    return func(ctx context.Context, e domain.Event) {
        event, ok := e.(accountdomain.TransferValidatedEvent)
        if !ok {
            return
        }
        // Create money.Money object
        // ...
        _ = bus.Publish(ctx, accountdomain.MoneyCreatedEvent{...})
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

1. Validation handler emits `TransferValidationFailedEvent` (TODO)
2. Money creation and subsequent handlers are not triggered
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
