---
icon: octicons/sync-24
---
# ⚡ Event-Driven Transfer Flow (Current Architecture)

## 🏁 Overview

The current transfer process is a fully event-driven, linear workflow. Each business step is handled by a dedicated, single-responsibility handler with defensive validation. This design ensures modularity, testability, and a clear, robust separation of concerns, preventing unintended side effects like accidental payment initiation.

---

## 🖼️ Event Flow Diagram

```mermaid
flowchart TD
    subgraph "Refactored Transfer Event Flow"
        A[API Request] --> B(TransferRequestedEvent);
        B --> C[Validation Handler];
        C --> D(TransferValidatedEvent);
        D --> E[Initial Persistence Handler];
        E --> F(ConversionRequestedEvent);
        F --> G[Conversion Handler];
        G --> H(TransferConversionDoneEvent);
        H --> I[Business Validation Handler];
        I --> J(TransferDomainOpDoneEvent);
        I -->|On Failure| K(TransferFailedEvent);
        J --> L[Final Persistence Handler];
        L --> M(TransferCompletedEvent);
        L -->|On Failure| K;
    end
```

---

## 🧩 Event Handler Responsibilities

### 1. **Validation Handler**

- **Consumes:** `TransferRequestedEvent`
- **Responsibility:** Performs basic structural validation on the request (e.g., non-nil UUIDs, positive amount). Malformed events are logged and discarded.
- **Emits:** `TransferValidatedEvent` on success.

### 2. **Initial Persistence Handler**

- **Consumes:** `TransferValidatedEvent`
- **Responsibility:** Creates the initial outgoing transaction (`tx_out`) with a `pending` status. This provides a durable record of the request early.
- **Emits:** `ConversionRequestedEvent` to trigger currency conversion (if needed).

### 3. **Conversion Handler (Generic)**

- **Consumes:** `ConversionRequestedEvent`
- **Responsibility:** Performs currency conversion.
- **Emits:** `TransferConversionDoneEvent` (a context-specific event).

### 4. **Business Validation Handler**

- **Consumes:** `TransferConversionDoneEvent`
- **Responsibility:** Performs all business-level validation against the current state of the system (e.g., sufficient funds in the source account).
- **Emits:**
  - `TransferDomainOpDoneEvent` on success.
  - `TransferFailedEvent` on business rule failure (e.g., insufficient funds).

### 5. **Final Persistence Handler**

- **Consumes:** `TransferDomainOpDoneEvent`
- **Responsibility:** Atomically performs the final state changes:
  - Creates the incoming transaction (`tx_in`) for the receiver with a `completed` status.
  - Updates the outgoing transaction (`tx_out`) to `completed`.
  - Updates the balances of both the source and destination accounts.
- **Emits:**
  - `TransferCompletedEvent` on success.
  - `TransferFailedEvent` if the atomic database operation fails.
