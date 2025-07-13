# Handler Package

This package implements the Chain of Responsibility pattern for account operations in the fintech application. It provides a clean separation of concerns by breaking down complex operations into smaller, focused handlers.

## Overview

The handler package was created to refactor the large `chain.go` file in the account service, splitting it into smaller, more maintainable files. Each handler has a single responsibility and can be easily tested and modified independently.

## Structure

```
pkg/handler/
├── types.go          # Common types and interfaces
├── base.go           # Base handler implementation
├── validation.go     # Validation handlers
├── money.go          # Money creation and conversion handlers
├── operation.go      # Domain operation handlers
├── persistence.go    # Persistence handlers
├── builder.go        # Chain builder
├── chain.go          # Simplified chain interface
└── README.md         # This file
```

## Components

### Types (`types.go`)

- `OperationHandler` interface
- `OperationRequest` struct
- `OperationResponse` struct
- `OperationType` constants

### Base Handler (`base.go`)

- `BaseHandler` provides common functionality for all handlers
- Implements the chain linking mechanism

### Validation Handlers (`validation.go`)

- `ValidationHandler` - validates account ownership and existence
- `TransferValidationHandler` - validates both source and destination accounts for transfers

### Money Handlers (`money.go`)

- `MoneyCreationHandler` - creates Money objects from request data
- `CurrencyConversionHandler` - handles currency conversion when needed

### Operation Handlers (`operation.go`)

- `DepositOperationHandler` - executes deposit domain operations
- `WithdrawOperationHandler` - executes withdraw domain operations
- `TransferOperationHandler` - executes transfer domain operations

### Persistence Handlers (`persistence.go`)

- `PersistenceHandler` - handles persistence for single operations
- `TransferPersistenceHandler` - handles persistence for transfer operations

### Chain Builder (`builder.go`)

- `ChainBuilder` - builds operation-specific chains
- Provides methods to create deposit, withdraw, and transfer chains

### Account Chain (`chain.go`)

- `AccountChain` - provides a simplified interface for executing operations
- Wraps the chain building and execution logic

## Usage

### Basic Usage

```go
// Create dependencies
uow := repository.NewUnitOfWork(db)
converter := currency.NewConverter()
logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

// Create account chain
accountChain := handler.NewAccountChain(uow, converter, logger)

// Execute operations
resp, err := accountChain.Deposit(ctx, userID, accountID, 100.0, currency.USD)
if err != nil {
    // Handle error
}

// Check for business logic errors
if resp.Error != nil {
    // Handle business error
}

// Use results
transaction := resp.Transaction
convInfo := resp.ConvInfo
```

### Backward Compatibility

The original account service maintains backward compatibility by re-exporting types and providing the same interface:

```go
// In pkg/service/account/
type OperationHandler = handler.OperationHandler
type OperationRequest = handler.OperationRequest
type OperationResponse = handler.OperationResponse
type OperationType = handler.OperationType

// Service now uses AccountChain internally
type Service struct {
    uow          repository.UnitOfWork
    converter    money.CurrencyConverter
    logger       *slog.Logger
    accountChain *AccountChain
}
```

## Benefits

1. **Separation of Concerns**: Each handler has a single responsibility
2. **Testability**: Individual handlers can be tested in isolation
3. **Maintainability**: Changes to one handler don't affect others
4. **Reusability**: Handlers can be reused in different chain configurations
5. **Readability**: Smaller files are easier to understand and navigate
6. **Extensibility**: New handlers can be easily added to the chain

## Chain Flow

### Deposit/Withdraw Chain

1. Validation → Money Creation → Currency Conversion → Domain Operation → Persistence

### Transfer Chain

1. Transfer Validation → Money Creation → Currency Conversion → Transfer Operation → Transfer Persistence

## Testing

Each handler can be tested independently:

```go
func TestValidationHandler_Handle(t *testing.T) {
    // Test validation logic
}

func TestMoneyCreationHandler_Handle(t *testing.T) {
    // Test money creation logic
}

func TestCurrencyConversionHandler_Handle(t *testing.T) {
    // Test currency conversion logic
}
```

## Future Improvements

1. **Middleware Support**: Add support for middleware handlers
2. **Conditional Chains**: Support for conditional handler execution
3. **Metrics**: Add metrics collection to handlers
4. **Circuit Breaker**: Add circuit breaker pattern for external services
5. **Caching**: Add caching handlers for frequently accessed data
