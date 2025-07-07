# Multi-Currency Support

## Overview

- Each account and transaction has a `currency` field (ISO 4217 code, e.g., "USD", "EUR").
- All operations (create, deposit, withdraw) must use the account's currency.
- Currency is validated against a supported list; defaults to "USD" if not specified.

## API Changes

### Account Creation

- **Request:**  

  ```json
  { "currency": "EUR" }
  ```

- **Response:**  

  ```json
  { "id": "...", "currency": "EUR", ... }
  ```

### Deposit/Withdraw

- **Request:**  

  ```json
  { "amount": 100.0, "currency": "EUR" }
  ```

- **Validation:**  
  - If `currency` does not match the account, return `400 Bad Request` with error:  
    `"currency mismatch: account has EUR, operation is USD"`

### Supported Currencies

- "USD", "EUR", "GBP", ...

### Error Handling

- `400 Bad Request` for invalid or unsupported currency codes.
- `400 Bad Request` for currency mismatches.

## Extending Support

- To add a new currency, update the `iso4217` map in the domain layer.

## Security Considerations

- All currency values are validated and sanitized.
- No currency conversion is performed unless explicitly implemented.

## Examples

...

## Design

- Add a currency field to the Account and Transaction domain models.
- Update services to handle currency conversions where necessary.
- Ensure all monetary operations respect the currency type.
- Add validation to prevent mixing currencies in operations.

## Implementation Plan

1. Modify domain models to include currency.
2. Update repositories and database schema.
3. Update service methods to handle currency.
4. Add currency conversion service if needed.
5. Add tests for multi-currency scenarios.

## Notes

- Default currency will be USD unless specified.
- Currency codes will follow ISO 4217 standard.
