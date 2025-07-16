---
icon: material/credit-card
---

# Stripe Integration & Multi-Currency Deposit Refactor

## 🏁 Overview

We integrated Stripe as a payment provider for deposits, refactored our multi-currency deposit flow, and improved our exchange rate caching logic. This ensures robust, auditable, and correct handling of all deposit scenarios, including cross-currency and zero-decimal currencies.

## 🛠️ What We Did

### 🏦 Stripe Payment Provider Integration

- 🚀 Implemented a `PaymentProvider` interface and a concrete Stripe implementation using the official [stripe-go](https://github.com/stripe/stripe-go/) SDK.
- 🔄 Migrated to the new `stripe.Client` pattern for future-proofing and better testability.
- 🧩 Injected the payment provider into the handler chain for clean separation of concerns.

### 🔗 Deposit Flow Refactor

- 🏗️ The service layer now only emits a deposit event; all business logic is handled in the handler chain.
- ➕ Added a `PaymentProviderHandler` to the chain, which:
  - 💳 Initiates payments with Stripe.
  - 🆔 Handles payment IDs and errors.
- 💱 Ensured currency conversion is always performed before crediting the account, using up-to-date exchange rates.

### 💱 Multi-Currency & Zero-Decimal Currency Handling

- 🐞 Fixed a critical bug: previously, JPY deposits were multiplied by 100, resulting in 100x overcharging on Stripe.
- 🧮 Now, the amount sent to Stripe is calculated using currency metadata (e.g., decimals for USD vs. JPY).
- 📊 All deposit and conversion logic is fully auditable and testable.

### 🗃️ Exchange Rate Cache Logic

- 🔄 Refactored cache lookup to always check both direct and reverse currency pairs, with consistent TTL/freshness logic.
- 🧹 Removed backend-specific logic for more predictable and maintainable caching.

## 🐞 Problems We Faced & Solutions

### 💱 Currency Conversion Mismatches

- ❌ **Problem:** Deposits in a currency different from the account’s currency were not being converted, leading to accounting errors.
- ✅ **Solution:** Enforced conversion in the handler chain, always crediting the account in its own currency.

### 💸 Stripe Amount Calculation for Zero-Decimal Currencies

- ❌ **Problem:** JPY and other zero-decimal currencies were incorrectly multiplied by 100, causing 100x overcharging.
- ✅ **Solution:** Used currency metadata to determine the correct multiplier for Stripe’s smallest unit.

### 🗃️ Inconsistent Exchange Rate Caching

- ❌ **Problem:** Cache logic was inconsistent, sometimes missing valid rates or using stale data.
- ✅ **Solution:** Unified cache lookup logic for both direct and reverse pairs, backend-agnostic.

### 🧩 Clean Architecture & Testability

- ❌ **Problem:** Payment provider logic was mixed into the service layer, making it hard to test and extend.
- ✅ **Solution:** Moved all provider logic into a dedicated handler in the chain, improving modularity and testability.

## 🔮 Next Steps

- 🔔 Integrate and test Stripe sandbox webhooks for real-time payment status updates.
- 📚 Document webhook event handling and reconciliation logic.
- 🧪 Continue to add tests and monitoring for all payment and currency flows.

## 📚 References

- [stripe-go SDK](https://github.com/stripe/stripe-go/)
- [Stripe API Docs](https://stripe.com/docs/api?lang=go)
- Project code: `infra/provider/stripe_payment_provider.go`, `pkg/handler/`, `pkg/service/account/`, `infra/provider/exchange_rates.go`
