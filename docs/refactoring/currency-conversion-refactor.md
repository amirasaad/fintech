---
icon: material/sync
---
# 🛠️ Currency Conversion Refactor: Step-by-Step Migration Plan

This document outlines the migration plan to refactor the currency conversion flow for maximum DRY, decoupling, and extensibility in the event-driven fintech system.

---

## 🎯 Rationale

- **Problem:** Previous flows coupled currency conversion with business logic (e.g., payment initiation), leading to code repetition and risk of unintended side effects.
- **Goal:** Make currency conversion a generic, reusable process, with all business-specific logic handled in separate handlers.

---

## 🚦 Step-by-Step Migration Plan

### 1. Define Generic Conversion Events

- **ConversionRequested**: Contains all info needed for conversion (amount, source/target currency, and a `FlowType` or `Purpose` field, or embeds the original business event as a payload).
- **ConversionDone**: Contains the result (converted amount, same context/correlation info as above).

**Example:**

```go
// ConversionRequested event
 type ConversionRequested struct {
     CorrelationID string      // Unique ID to correlate request/response
     FlowType      string      // "deposit", "withdraw", "transfer", etc.
     OriginalEvent interface{} // The original event (DepositValidatedEvent, etc.)
     Amount        money.Money
     SourceCurrency string
     TargetCurrency string
     Timestamp      int64
 }

// ConversionDone event
 type ConversionDone struct {
     CorrelationID string
     FlowType      string
     OriginalEvent interface{}
     ConvertedAmount money.Money
     Timestamp      int64
 }
```

---

### 2. Refactor the Conversion Handler

- Listens for **ConversionRequested**.
- Performs conversion.
- Emits **ConversionDone** with all context preserved.
- **No business logic, no payment logic, no branching.**

---

### 3. Update Business Handlers to Use ConversionDone

- Each business handler (deposit, withdraw, transfer) listens for **ConversionDone**.
- If the `FlowType` or `OriginalEvent` matches their flow, they emit the next business event (`DepositConversionDone`, `WithdrawConversionDone`, etc.).
- Only these business handlers trigger payment or domain operations.

---

### 4. Update Event Bus Wiring

- Subscribe the generic conversion handler to **ConversionRequested**.
- Subscribe each business handler to **ConversionDone**.
- Remove all direct business-specific conversion handler wiring.

---

### 5. Update All Emission Points

- Wherever you previously emitted a business-specific conversion requested event, emit a **ConversionRequested** event with the correct context.

---

### 6. Update Tests

- Ensure tests for all flows (deposit, withdraw, transfer) still pass and that no payment is triggered for transfer.

---

### 7. Document the New Pattern

- Update your architecture docs and diagrams to show the new generic conversion flow and business-specific post-conversion handlers.

---

## 📋 Summary Table

| Step | Action |
|------|--------|
| 1 | Define generic ConversionRequested/ConversionDone events |
| 2 | Refactor conversion handler to only handle these events |
| 3 | Update business handlers to listen for ConversionDone and emit next-step events |
| 4 | Update event bus wiring |
| 5 | Update all emission points to use ConversionRequested |
| 6 | Update and run tests |
| 7 | Update documentation and diagrams |

---

**This plan ensures a clean, extensible, and maintainable event-driven architecture for all currency conversion flows.**
