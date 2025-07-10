# Real Exchange Rates and Currency Conversion

## Overview

The fintech system supports robust, precise, and domain-driven currency conversion for all account and transaction operations. All currency conversion, rounding, and decimal enforcement is handled in the domain layer, ensuring:

- Any float64 value can be safely passed for conversion and storage.
- Rounding is always performed to the correct number of decimals for the target currency.
- All math is performed using Go's big.Rat for arbitrary-precision, avoiding floating-point imprecision.
- The domain layer is responsible for all rounding and validation, not the service or API layers.

## Conversion Flow

1. The service layer requests a conversion (e.g., deposit/withdraw in a different currency).
2. The domain layer performs the conversion using the latest exchange rate.
3. The result is rounded to the correct number of decimals for the target currency using big.Rat.
4. The value is stored in the smallest unit (e.g., cents for USD) as an integer (BIGINT in the DB).
5. Conversion details (original amount, rate, etc.) are stored as DECIMAL(30,15) for full float64 compatibility.

## Database Schema

- All money values are stored as BIGINT (smallest unit, e.g., cents).
- Conversion fields (original_amount, conversion_rate) in the transactions table are stored as DECIMAL(30,15) to support any float64 value.

## Best Practices

- Always pass raw float64 values to the domain layer; do not round in the service or API layers.
- The domain will round and validate as needed.
- All conversion and rounding logic is centralized for consistency and safety.

## Example

- Deposit 1,000,000,000 JPY to a USD account:
  - The system fetches the exchange rate, converts, and rounds to 2 decimals for USD.
  - The result is stored as an integer (cents) in the DB.
  - The original amount, rate, and conversion details are stored as DECIMAL(30,15).

## Recent Improvements

- Domain-driven rounding and validation using big.Rat
- Full float64 compatibility for conversion fields
- DB schema updated for DECIMAL(30,15) on conversion fields
- All overflow and decimal errors are now handled before DB writes
