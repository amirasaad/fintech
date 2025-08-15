---
icon: material/chart-line-variant
---

# Real Exchange Rates & Currency Conversion

## 📈 Overview

The fintech system uses a robust, production-ready exchange rate provider setup:

- **ExchangeRateAPIProvider** (`infra/provider/exchangerate_api.go`): Fetches real-time rates from [exchangerate-api.com](https://www.exchangerate-api.com/), with caching and health checks.
- **exchange.Service** (`pkg/service/exchange/service.go`): Orchestrates provider selection, caching, and fallback logic.

**Example:**

```go
// The exchange service is created in the factory
exchangeService, err := exchange.New(exchange.Config{
 Registry: providerRegistry,
 Cache:    rateCache,
 Logger:   logger,
})

// In the application, you can then use the service to convert money
convertedMoney, err := exchangeService.Convert(ctx, originalMoney, "USD")
```

!!! tip "Why this matters"
    This setup ensures reliable, up-to-date currency conversion with robust fallback and caching.

## :repeat: Conversion Flow

1. The service layer requests a conversion (e.g., deposit/withdraw in a different currency).
2. The domain layer performs the conversion using the latest exchange rate (via the real provider).
3. The result is rounded to the correct number of decimals for the target currency using big.Rat.
4. The value is stored in the smallest unit (e.g., cents for USD) as an integer (BIGINT in the DB).
5. Conversion details (original amount, rate, etc.) are stored as DECIMAL(30,15) for full float64 compatibility.

## 🗄️ Database Schema

- All money values are stored as BIGINT (smallest unit, e.g., cents).
- Conversion fields (original_amount, conversion_rate) in the transactions table are stored as DECIMAL(30,15) to support any float64 value.

## 🏅 Best Practices

- Always pass raw float64 values to the domain layer; do not round in the service or API layers.
- The domain will round and validate as needed.
- All conversion and rounding logic is centralized for consistency and safety.

## 🧪 Example

- Deposit 1,000,000,000 JPY to a USD account:
  - The system fetches the exchange rate, converts, and rounds to 2 decimals for USD.
  - The result is stored as an integer (cents) in the DB.
  - The original amount, rate, and conversion details are stored as DECIMAL(30,15).

## 🚀 Recent Improvements

- Domain-driven rounding and validation using big.Rat
- Full float64 compatibility for conversion fields
- DB schema updated for DECIMAL(30,15) on conversion fields
- All overflow and decimal errors are now handled before DB writes
