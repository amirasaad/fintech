# Multi-Currency Support

## Overview

The fintech platform supports multi-currency accounts and transactions with robust, precise, and domain-driven conversion and rounding logic. All currency conversion and rounding is handled in the domain layer, ensuring:

- Any float64 value can be safely converted and stored.
- Rounding is always performed to the correct number of decimals for the target currency using big.Rat.
- The domain layer is responsible for all rounding and validation, not the service or API layers.

## How It Works

- Users can deposit, withdraw, and transfer in any supported currency.
- The system fetches real-time exchange rates and performs conversion in the domain layer.
- The result is rounded to the correct number of decimals for the target currency.
- All values are stored in the smallest unit (e.g., cents for USD) as BIGINT.
- Conversion details (original amount, rate, etc.) are stored as DECIMAL(30,15) for full float64 compatibility.

## Database Schema

- Money values: BIGINT (smallest unit)
- Conversion fields: DECIMAL(30,15)

## Best Practices

- Pass raw float64 values to the domain layer; do not round in the service or API layers.
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
