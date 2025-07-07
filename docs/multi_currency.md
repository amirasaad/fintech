# Multi-Currency Support

## Overview

This document outlines the multi-currency support feature in the Fintech application. Multi-currency is implemented at both the **account and transaction levels**.

- Each account is assigned a specific currency (e.g., "USD", "EUR") upon creation.
- All financial operations (deposits, withdrawals) for an account **must** be performed in that account's designated currency.
- Each transaction records the currency in which it was performed.
- The currency is specified using the ISO 4217 code. If not provided during account creation, it defaults to "USD".

## API Changes

### Account Creation

To create an account with a specific currency, provide the `currency` code in the request body.

- **Request:**

  ```json
  { "currency": "EUR" }
  ```

- **Response:** The new account object will include the specified currency.

  ```json
  { "id": "...", "currency": "EUR", ... }
  ```

## Money Value Object & Service API

- All monetary operations (deposit, withdraw) use the `Money` value object for currency and amount validation.
- The service layer exposes methods that accept `amount` and `currency` as primitives, constructing and validating `Money` internally.
- This eliminates the need for separate `DepositWithCurrency`/`WithdrawWithCurrency` methods.

### Example Usage

```go
// Service layer usage:
tx, err := accountService.Deposit(userID, accountID, 100.0, "EUR")
if err != nil {
    // handle error (e.g., invalid currency, amount, or business rule)
}
```

- All validation (currency code, amount positivity) is performed in the domain layer via `NewMoney`.
- This approach ensures all operations are currency-aware and future-proofs the system for features like currency conversion.

## Implementation Summary

- **Account and Transaction-Level Currency:** The `Account` and `Transaction` domain models, database schema, and repositories have been updated to include a `currency` field.
- **Service-Layer Validation:** Application services now enforce currency consistency for all transactions.
- **API Enforcement:** The web API validates currency codes in requests and ensures they match the account's currency for all operations.
- **Testing:** Unit and integration tests have been added to cover multi-currency scenarios, including validation and error handling.

## Future Work

- **Currency Conversion:** The system does not currently support currency conversion. Future work could include integrating with an exchange rate API to allow for cross-currency transactions.
- **Reporting:** Enhanced financial reporting that leverages the multi-currency data could be developed.

## Error Handling

- **`400 Bad Request`**: Returned for invalid or unsupported currency codes.
- **`400 Bad Request`**: Returned for currency mismatches between the operation and the account.

## Extending Currency Support

To support a new currency, update the `iso4217` map located in the domain layer and ensure it is a valid ISO 4217 code.

## Security Considerations

- All currency values are validated against a predefined list of supported codes.
- The system does not perform any currency conversion at this stage, preventing complexities related to exchange rate management.
