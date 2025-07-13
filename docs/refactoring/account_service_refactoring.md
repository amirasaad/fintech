# Account Service Refactoring Guide

## Problem Statement

The original `AccountService` had significant code duplication and branching in the `Deposit` and `Withdraw` methods. Both methods followed nearly identical patterns:

1. Get repositories from Unit of Work
2. Retrieve account by ID
3. Create money object
4. Convert currency if needed
5. Perform the operation (deposit/withdraw)
6. Store conversion info if applicable
7. Update account
8. Create transaction record

## Refactoring Solution: Chain of Responsibility Pattern

### Overview

The account service has been refactored using the **Chain of Responsibility** pattern, which eliminates branching complexity and provides excellent extensibility. Each handler in the chain has a single responsibility and can either process the request or pass it to the next handler.

### Architecture

The refactored solution consists of:

1. **OperationHandler Interface**: Defines the contract for all handlers in the chain
2. **OperationRequest/Response**: Data structures for passing information through the chain
3. **Specialized Handlers**: Each handling a specific aspect of the operation
4. **ChainBuilder**: Constructs the complete operation chain

### Handler Chain Structure

```go
// The complete chain for deposit/withdraw operations:
ValidationHandler → MoneyCreationHandler → CurrencyConversionHandler → DomainOperationHandler → PersistenceHandler
```

### Individual Handlers

#### 1. ValidationHandler

- **Responsibility**: Validates account exists and belongs to the user
- **Input**: UserID, AccountID
- **Output**: Account entity or error

#### 2. MoneyCreationHandler

- **Responsibility**: Creates Money value object from amount and currency
- **Input**: Amount, CurrencyCode
- **Output**: Money object or error

#### 3. CurrencyConversionHandler

- **Responsibility**: Converts currency if different from account currency
- **Input**: Money object, Account currency
- **Output**: Converted Money object and conversion info

#### 4. DomainOperationHandler

- **Responsibility**: Executes the actual domain operation (deposit/withdraw)
- **Input**: Account, Money, Operation type
- **Output**: Transaction object

#### 5. PersistenceHandler

- **Responsibility**: Persists account and transaction changes
- **Input**: Account, Transaction, Conversion info
- **Output**: Final response with transaction and conversion info

### Implementation Example

```go
// Simplified Deposit method using the chain
func (s *AccountService) Deposit(
    userID, accountID uuid.UUID,
    amount float64,
    currencyCode currency.Code,
) (tx *account.Transaction, convInfo *common.ConversionInfo, err error) {
    req := &OperationRequest{
        UserID:       userID,
        AccountID:    accountID,
        Amount:       amount,
        CurrencyCode: currencyCode,
        Operation:    OperationDeposit,
    }

    resp, err := s.chain.Handle(context.Background(), req)
    if err != nil {
        return nil, nil, err
    }

    return resp.Transaction, resp.ConvInfo, resp.Error
}
```

### Chain Builder

```go
// ChainBuilder constructs the complete operation chain
func (b *ChainBuilder) BuildOperationChain() OperationHandler {
    validation := &ValidationHandler{uow: b.uow, logger: b.logger}
    moneyCreation := &MoneyCreationHandler{logger: b.logger}
    conversion := &CurrencyConversionHandler{converter: b.converter, logger: b.logger}
    domainOp := &DomainOperationHandler{logger: b.logger}
    persistence := &PersistenceHandler{uow: b.uow, logger: b.logger}

    // Build the chain
    validation.SetNext(moneyCreation)
    moneyCreation.SetNext(conversion)
    conversion.SetNext(domainOp)
    domainOp.SetNext(persistence)

    return validation
}
```

## Benefits of This Refactoring

### 1. **Zero Branching Complexity**

- Each handler has a single responsibility
- No complex conditional logic in the main service methods
- Clear separation of concerns

### 2. **Excellent Extensibility**

- Easy to add new handlers (e.g., audit logging, notifications)
- Easy to modify existing handlers without affecting others
- Easy to reorder handlers or create different chains

### 3. **Improved Testability**

- Each handler can be tested independently
- Mock handlers can be easily substituted
- Clear input/output contracts

### 4. **Better Maintainability**

- Changes to one aspect don't affect others
- Clear data flow through the chain
- Easy to understand and debug

### 5. **Go Idiomatic**

- Uses interfaces and composition
- Follows Go patterns and conventions
- Clean and readable code

## File Structure

```
pkg/service/account/
├── account.go          # Service definition and account creation
├── types.go            # Common types and interfaces
├── chain.go            # Chain of Responsibility implementation
├── chain_test.go       # Chain tests
├── operations.go       # Operation types and request/response
├── queries.go          # Query operations (GetAccount, GetTransactions, GetBalance)
└── account_test.go     # Service tests
```

## Migration Strategy

The refactoring has been completed with the following approach:

1. ✅ **Phase 1**: Implemented Chain of Responsibility pattern
2. ✅ **Phase 2**: Updated all service methods to use the chain
3. ✅ **Phase 3**: Added comprehensive tests for all handlers
4. ✅ **Phase 4**: Removed old implementation code

## Example Usage

After refactoring, the usage remains exactly the same:

```go
// Deposit example
tx, convInfo, err := accountService.Deposit(userID, accountID, 100.0, currency.Code("EUR"))

// Withdraw example
tx, convInfo, err := accountService.Withdraw(userID, accountID, 50.0, currency.Code("USD"))
```

## Adding New Handlers

To add a new handler (e.g., for audit logging):

```go
// AuditHandler logs all operations for audit purposes
type AuditHandler struct {
    BaseHandler
    logger *slog.Logger
}

func (h *AuditHandler) Handle(ctx context.Context, req *OperationRequest) (*OperationResponse, error) {
    h.logger.Info("Operation executed",
        "userID", req.UserID,
        "accountID", req.AccountID,
        "operation", req.Operation,
        "amount", req.Amount,
        "currency", req.CurrencyCode)
    
    return h.BaseHandler.Handle(ctx, req)
}

// Add to chain in ChainBuilder
audit := &AuditHandler{logger: b.logger}
validation.SetNext(audit)
audit.SetNext(moneyCreation)
```

## Currency HTTP Requests

I've also created a comprehensive `currencies.http` file with all the currency API endpoints, including:

- **Public endpoints**: List currencies, get currency details, search, statistics
- **Admin endpoints**: Register, activate, deactivate, unregister currencies
- **Error handling examples**: Invalid codes, missing fields, unauthorized access
- **Workflow examples**: Complete currency lifecycle management
- **Bulk operations**: Register multiple currencies

The file is located at `docs/requests/currencies.http` and provides ready-to-use HTTP requests for testing all currency-related functionality.
