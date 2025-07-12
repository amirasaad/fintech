# Account Service Refactoring Guide

## Problem Statement

The original `AccountService` has significant code duplication and branching in the `Deposit` and `Withdraw` methods. Both methods follow nearly identical patterns:

1. Get repositories from Unit of Work
2. Retrieve account by ID
3. Create money object
4. Convert currency if needed
5. Perform the operation (deposit/withdraw)
6. Store conversion info if applicable
7. Update account
8. Create transaction record

## Refactoring Solution

### 1. Extract Common Logic

Create a generic `processTransaction` method that handles the common flow:

```go
// TransactionOperation defines the signature for deposit/withdraw operations
type TransactionOperation func(*account.Account, uuid.UUID, mon.Money) (*account.Transaction, error)

// processTransaction handles the common logic for both deposit and withdraw operations
func (s *AccountService) processTransaction(
    userID, accountID uuid.UUID,
    amount float64,
    currencyCode currency.Code,
    operation TransactionOperation,
) (tx *account.Transaction, convInfo *common.ConversionInfo, err error) {
    // Common implementation here
}
```

### 2. Create Operation-Specific Functions

Extract the actual deposit and withdraw logic into separate functions:

```go
// depositOperation performs the actual deposit operation
func (s *AccountService) depositOperation(account *account.Account, userID uuid.UUID, money mon.Money) (*account.Transaction, error) {
    return account.Deposit(userID, money)
}

// withdrawOperation performs the actual withdraw operation
func (s *AccountService) withdrawOperation(account *account.Account, userID uuid.UUID, money mon.Money) (*account.Transaction, error) {
    return account.Withdraw(userID, money)
}
```

### 3. Simplify Main Methods

The main `Deposit` and `Withdraw` methods become much simpler:

```go
// Deposit adds funds to the specified account and creates a transaction record.
func (s *AccountService) Deposit(
    userID, accountID uuid.UUID,
    amount float64,
    currencyCode currency.Code,
) (tx *account.Transaction, convInfo *common.ConversionInfo, err error) {
    s.logger.Info("Deposit started", "userID", userID, "accountID", accountID, "amount", amount, "currency", currencyCode)
    defer func() {
        if err != nil {
            s.logger.Error("Deposit failed", "userID", userID, "accountID", accountID, "amount", amount, "currency", currencyCode, "error", err)
        } else {
            s.logger.Info("Deposit successful", "userID", userID, "accountID", accountID, "amount", amount, "currency", currencyCode, "transactionID", tx.ID)
        }
    }()

    return s.processTransaction(userID, accountID, amount, currencyCode, s.depositOperation)
}

// Withdraw removes funds from the specified account and creates a transaction record.
func (s *AccountService) Withdraw(
    userID, accountID uuid.UUID,
    amount float64,
    currencyCode currency.Code,
) (tx *account.Transaction, convInfo *common.ConversionInfo, err error) {
    s.logger.Info("Withdraw started", "userID", userID, "accountID", accountID, "amount", amount, "currency", currencyCode)
    defer func() {
        if err != nil {
            s.logger.Error("Withdraw failed", "userID", userID, "accountID", accountID, "amount", amount, "currency", currencyCode, "error", err)
        } else {
            s.logger.Info("Withdraw successful", "userID", userID, "accountID", accountID, "amount", amount, "currency", currencyCode, "transactionID", tx.ID)
        }
    }()

    return s.processTransaction(userID, accountID, amount, currencyCode, s.withdrawOperation)
}
```

### 4. Extract Helper Methods

Break down the `processTransaction` method into smaller, focused helper methods:

```go
// getRepositories retrieves the required repositories from the unit of work
func (s *AccountService) getRepositories(uow repository.UnitOfWork) (repository.AccountRepository, repository.TransactionRepository, error)

// getAccount retrieves an account by ID
func (s *AccountService) getAccount(repo repository.AccountRepository, accountID uuid.UUID) (*account.Account, error)

// createMoney creates a money object from amount and currency
func (s *AccountService) createMoney(amount float64, currencyCode currency.Code) (mon.Money, error)

// convertCurrencyIfNeeded converts the money to the account's currency if different
func (s *AccountService) convertCurrencyIfNeeded(money mon.Money, accountCurrency currency.Code, logger *slog.Logger) (mon.Money, *common.ConversionInfo, error)

// storeConversionInfo stores conversion information in the transaction
func (s *AccountService) storeConversionInfo(tx *account.Transaction, convInfo *common.ConversionInfo, logger *slog.Logger)
```

## Benefits of This Refactoring

### 1. **Reduced Code Duplication**
- Common logic is centralized in `processTransaction`
- Helper methods are reusable across different operations

### 2. **Improved Readability**
- Each method has a single responsibility
- The main `Deposit` and `Withdraw` methods are now very clear about their intent

### 3. **Easier Testing**
- Helper methods can be tested independently
- Mocking becomes simpler with smaller, focused functions

### 4. **Better Maintainability**
- Changes to common logic only need to be made in one place
- New transaction types can easily reuse the same pattern

### 5. **Reduced Branching**
- The complex conditional logic is broken down into smaller, more manageable pieces
- Each helper method handles a specific concern

## Implementation Steps

1. **Create the `TransactionOperation` type** to define the function signature
2. **Extract helper methods** one by one, starting with the simplest ones
3. **Create the `processTransaction` method** that orchestrates the common flow
4. **Create operation-specific functions** (`depositOperation`, `withdrawOperation`)
5. **Refactor the main methods** to use the new structure
6. **Update tests** to reflect the new structure

## Example Usage

After refactoring, the usage remains exactly the same:

```go
// Deposit example
tx, convInfo, err := accountService.Deposit(userID, accountID, 100.0, currency.Code("EUR"))

// Withdraw example
tx, convInfo, err := accountService.Withdraw(userID, accountID, 50.0, currency.Code("USD"))
```

## Migration Strategy

1. **Phase 1**: Create the new structure alongside the existing code
2. **Phase 2**: Update tests to use the new structure
3. **Phase 3**: Replace the old implementation with the refactored version
4. **Phase 4**: Remove the old code

This approach ensures zero downtime and allows for easy rollback if issues arise.

## Currency HTTP Requests

I've also created a comprehensive `currencies.http` file with all the currency API endpoints, including:

- **Public endpoints**: List currencies, get currency details, search, statistics
- **Admin endpoints**: Register, activate, deactivate, unregister currencies
- **Error handling examples**: Invalid codes, missing fields, unauthorized access
- **Workflow examples**: Complete currency lifecycle management
- **Bulk operations**: Register multiple currencies

The file is located at `docs/requests/currencies.http` and provides ready-to-use HTTP requests for testing all currency-related functionality. 