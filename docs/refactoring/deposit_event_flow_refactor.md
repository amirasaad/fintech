---
icon: material/sync
---

# 🧾 Deposit/Payment Event Flow Refactor

## 📌 Overview

This document explains the refactored, cycle-free, DRY event-driven flow for deposit and payment initiation. It covers handler responsibilities, idempotency, and anti-cycle design, with a Mermaid diagram and troubleshooting tips.

---

## 📌 Updated Deposit Event Flow Diagram

```mermaid
flowchart TD
    A[Deposit.Requested] --> B[DepositValidationHandler → Deposit.Validated]
    B --> C[DepositPersistenceHandler → DepositPersistedEvent]
    C --> D[ConversionHandler → Deposit.CurrencyConverted]
    D --> E[ConversionPersistenceHandler]
    D --> F[BusinessValidationHandler → Deposit.Validated]
    F --> G[DepositPaymentInitiationHandler → Payment.Initiated]
    G --> H[PaymentPersistenceHandler]
```

---

## 📌 Handler Responsibilities (Deposit Flow)

- **DepositValidationHandler**: Validates deposit request, emits `Deposit.Validated`.
- **DepositPersistenceHandler**: Persists validated deposit, emits `DepositPersistedEvent`.
- **ConversionHandler**: Handles currency conversion, emits `Deposit.CurrencyConverted`.
- **ConversionPersistenceHandler**: Persists conversion data, idempotent, logs all actions.
- **BusinessValidationHandler**: Performs business checks, emits `Deposit.Validated`, idempotent.
- **DepositPaymentInitiationHandler**: Initiates payment, emits `Payment.Initiated`, idempotent.
- **PaymentPersistenceHandler**: Persists payment info, idempotent if needed.

---

## 📌 Idempotency & Anti-Cycle Design

- Each handler uses a `sync.Map` (or persistent store) to track processed TransactionIDs.
- If a duplicate event is received, the handler logs and skips emission (`[SKIP]`).
- No handler emits an event that can re-trigger itself or a previous handler.
- This design prevents infinite loops and duplicate payment initiations.

---

## 📌 Example Logs (Deposit Flow)

```plain/text
[START] Received event handler=DepositValidationHandler event_type=Deposit.Requested ...
✅ [SUCCESS] Account validated, emitting Deposit.Validated ...
[START] Received event handler=DepositPersistenceHandler event_type=Deposit.Validated ...
✅ [SUCCESS] Transaction persisted ...
[START] Received event handler=ConversionHandler event_type=DepositPersistedEvent ...
✅ [SUCCESS] Conversion done, emitting Deposit.CurrencyConverted ...
[START] Received event handler=ConversionPersistenceHandler event_type=Deposit.CurrencyConverted ...
✅ [SUCCESS] Conversion data persisted ...
[START] Received event handler=BusinessValidationHandler event_type=Deposit.CurrencyConverted ...
✅ [SUCCESS] Business validation passed, emitting Deposit.Validated ...
[START] Received event handler=DepositPaymentInitiationHandler event_type=Deposit.Validated ...
✅ [SUCCESS] Initiating payment, emitting Payment.Initiated ...
[START] Received event handler=PaymentPersistenceHandler event_type=Payment.Initiated ...
✅ [SUCCESS] Payment info persisted ...
```

---

## Updated Withdraw Event Flow Diagram

```mermaid
flowchart TD
    A[Withdraw.Requested] --> B[WithdrawValidationHandler → Withdraw.Validated]
    B --> C[WithdrawPersistenceHandler → WithdrawPersistedEvent]
    C --> D[ConversionHandler → Withdraw.CurrencyConverted]
    D --> E[ConversionPersistenceHandler]
    D --> F[BusinessValidationHandler → Withdraw.Validated]
    F --> G[WithdrawPaymentInitiationHandler → Payment.Initiated]
    G --> H[PaymentPersistenceHandler]
```

---

## Updated Transfer Event Flow Diagram

```mermaid
flowchart TD
    A[Transfer.Requested] --> B[TransferValidationHandler → Transfer.Validated]
    B --> C[InitialPersistenceHandler → TransferPersistedEvent]
    C --> D[ConversionHandler → Transfer.CurrencyConverted]
    D --> E[ConversionPersistenceHandler]
    D --> F[BusinessValidationHandler → Transfer.Validated]
    F --> G[TransferPaymentInitiationHandler → Payment.Initiated]
    G --> H[PaymentPersistenceHandler]
```

---

## Handler Responsibilities (Withdraw Flow)

- **WithdrawValidationHandler**: Validates withdraw request, emits `Withdraw.Validated`.
- **WithdrawPersistenceHandler**: Persists validated withdraw, emits `WithdrawPersistedEvent`.
- **ConversionHandler**: Handles currency conversion, emits `Withdraw.CurrencyConverted`.
- **ConversionPersistenceHandler**: Persists conversion data, idempotent, logs all actions.
- **BusinessValidationHandler**: Performs business checks, emits `Withdraw.Validated`, idempotent.
- **WithdrawPaymentInitiationHandler**: Initiates payment, emits `Payment.Initiated`, idempotent.
- **PaymentPersistenceHandler**: Persists payment info, idempotent if needed.

---

## Handler Responsibilities (Transfer Flow)

- **TransferValidationHandler**: Validates transfer request, emits `Transfer.Validated`.
- **InitialPersistenceHandler**: Persists initial transfer, emits `TransferPersistedEvent`.
- **ConversionHandler**: Handles currency conversion, emits `Transfer.CurrencyConverted`.
- **ConversionPersistenceHandler**: Persists conversion data, idempotent, logs all actions.
- **BusinessValidationHandler**: Performs business checks, emits `Transfer.Validated`, idempotent.
- **TransferPaymentInitiationHandler**: Initiates payment, emits `Payment.Initiated`, idempotent.
- **PaymentPersistenceHandler**: Persists payment info, idempotent if needed.

---

## Troubleshooting Tips (All Flows)

- **Multiple events for the same transaction?**
  - Check idempotency logic in each handler.
  - Ensure no handler emits an event that can re-trigger itself or a previous handler.
- **Event chain not progressing?**
  - Check logs for `[SKIP]` or `[ERROR]` messages.
  - Ensure each handler emits only the next event in the chain.
- **Missing logs or correlation IDs?**
  - Add structured logging with `correlation_id`, `transaction_id`, and event type in each handler.

---

## Summary

This refactor ensures a clean, DRY, and cycle-free event-driven deposit/payment flow with robust idempotency and clear logging for easy troubleshooting.
