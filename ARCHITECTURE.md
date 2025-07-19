# 🏗️ Architecture Overview

This document describes the architecture of the Fintech Platform, including its core principles, structure, and best practices.

---

## 🏁 Principles

- Clean architecture
- Domain-driven design (DDD)
- Separation of concerns
- Dependency injection
- Testability

---

## 🧭 Project Structure

- `cmd/` — Application entry points
- `pkg/` — Domain, service, repository, middleware, etc.
- `webapi/` — HTTP handlers and API endpoints
- `infra/` — Infrastructure layer (database, models)
- `internal/` — Internal utilities and fixtures
- `docs/` — Documentation and OpenAPI specs

---

## 🧰 Key Technologies

- Go (Fiber, GORM, JWT)
- go-playground/validator
- Google UUID

---

## 🏅 Best Practices

- Keep business logic in the domain layer
- Use interfaces for dependency inversion
- Implement repository pattern with Unit of Work
- Use property-style getters for entities
- Centralize validation and error handling

---

## 🔮 Looking Forward

- Expand event-driven architecture
- Add more payment providers
- Enhance observability and monitoring

## Event-Driven Deposit Workflow (CQRS, Money Conversion)

This workflow illustrates how a deposit is processed in an event-driven, CQRS-based fintech system, including money conversion if needed:

```mermaid
sequenceDiagram
    participant U as User
    participant API as API/Service
    participant EB as EventBus
    participant MC as MoneyCreationHandler
    participant CC as CurrencyConverter
    participant DB as Persistence
    participant PP as PaymentProvider

    U->>API: DepositRequest (amount, currency)
    API->>EB: DepositRequestedEvent
    EB->>MC: DepositRequestedEvent
    MC->>CC: MoneyConversionRequestedEvent (if needed)
    CC-->>MC: MoneyConvertedEvent (converted amount)
    MC->>EB: MoneyCreatedEvent (account currency)
    EB->>DB: MoneyCreatedEvent
    DB->>EB: TransactionPersistedEvent
    EB->>PP: TransactionPersistedEvent
    PP->>EB: PaymentInitiatedEvent (paymentId)
    EB->>DB: PaymentInitiatedEvent (persist paymentId)
    PP->>API: Payment confirmation webhook
    API->>DB: Update transaction status, update balance if completed
```

**Workflow Steps:**

1. User submits a deposit request (amount, currency, etc.).
2. API emits a `DepositRequestedEvent`.
3. MoneyCreationHandler creates a Money value object. If conversion is needed, emits a `MoneyConversionRequestedEvent` and waits for a `MoneyConvertedEvent`.
4. Emits a `MoneyCreatedEvent` in the account's currency.
5. Persistence handler saves the transaction and emits a `TransactionPersistedEvent`.
6. Payment provider handler initiates payment and emits a `PaymentInitiatedEvent` (with paymentId).
7. Persistence handler updates the transaction with the paymentId.
8. On payment provider webhook, the API updates the transaction status and account balance if payment is completed.
