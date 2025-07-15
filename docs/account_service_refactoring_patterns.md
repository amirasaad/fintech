# Account Service Refactoring: Design Patterns Analysis

## Overview

This document summarizes the analysis and implementation of various design patterns for refactoring the account service in the fintech application. The goal was to reduce branching complexity and improve code organization in the `Deposit` and `Withdraw` methods.

## Initial Problem

The original `Deposit` and `Withdraw` methods in `pkg/service/account/account.go` had:

- **Significant code duplication** (~150 lines of nearly identical logic)
- **Complex branching** around currency conversion and transaction handling
- **Mixed responsibilities** (validation, conversion, persistence, logging)
- **Poor maintainability** due to tightly coupled logic

## Implemented Solutions

### 1. Strategy Pattern (Implemented)

**Approach:** Extract common operation logic into a shared method using strategy pattern for the specific operation type.

**Implementation:**

- Created `OperationType` enum and `operationHandler` interface
- Implemented `depositHandler` and `withdrawHandler` concrete strategies
- Extracted common logic into `executeOperation()` method
- Split code into multiple files for better organization

**File Structure:**

```ascii
pkg/service/account/
├── account.go          # Service definition and account creation
├── types.go            # Common types and interfaces
├── handlers.go         # Strategy implementations
├── operations.go       # Core operation execution logic
├── deposit.go          # Deposit-specific logic
├── withdraw.go         # Withdraw-specific logic
└── queries.go          # Query operations (GetAccount, GetTransactions, GetBalance)
```

**Complete Example:**

```go
// types.go
type OperationType string

const (
    OperationDeposit  OperationType = "deposit"
    OperationWithdraw OperationType = "withdraw"
)

type operationHandler interface {
    execute(account *account.Account, userID uuid.UUID, money mon.Money) (*account.Transaction, error)
}

// handlers.go
type depositHandler struct{}

func (h *depositHandler) execute(account *account.Account, userID uuid.UUID, money mon.Money) (*account.Transaction, error) {
    return account.Deposit(userID, money)
}

type withdrawHandler struct{}

func (h *withdrawHandler) execute(account *account.Account, userID uuid.UUID, money mon.Money) (*account.Transaction, error) {
    return account.Withdraw(userID, money)
}

// operations.go
func (s *AccountService) executeOperation(req operationRequest, handler operationHandler) (result *operationResult, err error) {
    // Common logic: fetch account, convert currency, execute operation, persist
    // Uses handler.execute() for the specific operation
}

// deposit.go
func (s *AccountService) Deposit(userID, accountID uuid.UUID, amount float64, currencyCode currency.Code) (*account.Transaction, *common.ConversionInfo, error) {
    req := operationRequest{
        userID: userID, accountID: accountID, amount: amount,
        currencyCode: currencyCode, operation: OperationDeposit,
    }
    result, err := s.executeOperation(req, &depositHandler{})
    return result.transaction, result.convInfo, err
}
```

**Benefits:**

- ✅ Eliminated ~150 lines of duplicated code
- ✅ Reduced branching complexity
- ✅ Improved maintainability
- ✅ Better separation of concerns
- ✅ Easy to add new operations

**Limitations:**

- ❌ Core branching logic still exists in `executeOperation()`
- ❌ Currency conversion logic remains complex

### 2. Command Pattern (Analyzed)

**Approach:** Encapsulate each operation as a command object with a uniform interface.

**Complete Example:**

```go
// Command interface
type AccountCommand interface {
    Execute(ctx context.Context) (*account.Transaction, *common.ConversionInfo, error)
}

// Base command with common functionality
type BaseCommand struct {
    Service     *AccountService
    UserID      uuid.UUID
    AccountID   uuid.UUID
    Amount      float64
    Currency    currency.Code
    logger      *slog.Logger
}

func (c *BaseCommand) getRepositories(uow repository.UnitOfWork) (repository.AccountRepository, repository.TransactionRepository, error) {
    repoAny, err := uow.GetRepository(reflect.TypeOf((*repository.AccountRepository)(nil)).Elem())
    if err != nil {
        return nil, nil, err
    }
    accountRepo := repoAny.(repository.AccountRepository)

    txRepoAny, err := uow.GetRepository(reflect.TypeOf((*repository.TransactionRepository)(nil)).Elem())
    if err != nil {
        return nil, nil, err
    }
    txRepo := txRepoAny.(repository.TransactionRepository)

    return accountRepo, txRepo, nil
}

// Deposit command
type DepositCommand struct {
    BaseCommand
}

func NewDepositCommand(service *AccountService, userID, accountID uuid.UUID, amount float64, currency currency.Code) *DepositCommand {
    return &DepositCommand{
        BaseCommand: BaseCommand{
            Service:  service,
            UserID:   userID,
            AccountID: accountID,
            Amount:   amount,
            Currency: currency,
            logger:   service.logger,
        },
    }
}

func (c *DepositCommand) Execute(ctx context.Context) (*account.Transaction, *common.ConversionInfo, error) {
    logger := c.logger.With("userID", c.UserID, "accountID", c.AccountID, "amount", c.Amount, "currency", c.Currency)
    logger.Info("DepositCommand.Execute started")

    var txLocal *account.Transaction
    var convInfoLocal *common.ConversionInfo

    err := c.Service.uow.Do(ctx, func(uow repository.UnitOfWork) error {
        accountRepo, txRepo, err := c.getRepositories(uow)
        if err != nil {
            return err
        }

        // Get account
        account, err := accountRepo.Get(c.AccountID)
        if err != nil {
            return account.ErrAccountNotFound
        }

        // Create money
        money, err := mon.NewMoney(c.Amount, c.Currency)
        if err != nil {
            return err
        }

        // Convert currency if needed
        convertedMoney, convInfo, err := c.Service.handleCurrencyConversion(money, account.Currency, logger)
        if err != nil {
            return err
        }
        convInfoLocal = convInfo

        // Execute deposit
        txLocal, err = account.Deposit(c.UserID, convertedMoney)
        if err != nil {
            return err
        }

        // Store conversion info
        if convInfoLocal != nil {
            txLocal.OriginalAmount = &convInfoLocal.OriginalAmount
            txLocal.OriginalCurrency = &convInfoLocal.OriginalCurrency
            txLocal.ConversionRate = &convInfoLocal.ConversionRate
        }

        // Persist changes
        if err = accountRepo.Update(account); err != nil {
            return err
        }
        if err = txRepo.Create(txLocal); err != nil {
            return err
        }

        return nil
    })

    if err != nil {
        return nil, nil, err
    }

    return txLocal, convInfoLocal, nil
}

// Withdraw command
type WithdrawCommand struct {
    BaseCommand
}

func NewWithdrawCommand(service *AccountService, userID, accountID uuid.UUID, amount float64, currency currency.Code) *WithdrawCommand {
    return &WithdrawCommand{
        BaseCommand: BaseCommand{
            Service:  service,
            UserID:   userID,
            AccountID: accountID,
            Amount:   amount,
            Currency: currency,
            logger:   service.logger,
        },
    }
}

func (c *WithdrawCommand) Execute(ctx context.Context) (*account.Transaction, *common.ConversionInfo, error) {
    // Similar implementation to DepositCommand but calls account.Withdraw()
    // ... implementation details
}

// Updated service
func (s *AccountService) Deposit(userID, accountID uuid.UUID, amount float64, currencyCode currency.Code) (*account.Transaction, *common.ConversionInfo, error) {
    cmd := NewDepositCommand(s, userID, accountID, amount, currencyCode)
    return cmd.Execute(context.Background())
}

func (s *AccountService) Withdraw(userID, accountID uuid.UUID, amount float64, currencyCode currency.Code) (*account.Transaction, *common.ConversionInfo, error) {
    cmd := NewWithdrawCommand(s, userID, accountID, amount, currencyCode)
    return cmd.Execute(context.Background())
}
```

**Benefits:**

- ✅ Complete decoupling of operations
- ✅ Excellent extensibility
- ✅ Easy to queue, log, or undo operations
- ✅ No branching in service layer

**Drawbacks:**

- ❌ More boilerplate code
- ❌ Can feel verbose for simple operations
- ❌ Not always idiomatic Go

### 3. Chain of Responsibility (Analyzed)

**Approach:** Break operations into a chain of focused handlers, each responsible for one step.

**Complete Example:**

```go
// Handler interface
type OperationHandler interface {
    Handle(ctx context.Context, req *OperationRequest) (*OperationResponse, error)
    SetNext(handler OperationHandler)
}

// Request/Response structures
type OperationRequest struct {
    UserID         uuid.UUID
    AccountID      uuid.UUID
    Amount         float64
    CurrencyCode   currency.Code
    Operation      OperationType
    Account        *account.Account
    Money          mon.Money
    ConvertedMoney mon.Money
    ConvInfo       *common.ConversionInfo
    Transaction    *account.Transaction
}

type OperationResponse struct {
    Transaction *account.Transaction
    ConvInfo    *common.ConversionInfo
    Error       error
}

// Base handler
type BaseHandler struct {
    next OperationHandler
}

func (h *BaseHandler) SetNext(handler OperationHandler) {
    h.next = handler
}

func (h *BaseHandler) Handle(ctx context.Context, req *OperationRequest) (*OperationResponse, error) {
    if h.next != nil {
        return h.next.Handle(ctx, req)
    }
    return &OperationResponse{}, nil
}

// Account validation handler
type AccountValidationHandler struct {
    BaseHandler
    uow    repository.UnitOfWork
    logger *slog.Logger
}

func (h *AccountValidationHandler) Handle(ctx context.Context, req *OperationRequest) (*OperationResponse, error) {
    logger := h.logger.With("userID", req.UserID, "accountID", req.AccountID)

    repoAny, err := h.uow.GetRepository(reflect.TypeOf((*repository.AccountRepository)(nil)).Elem())
    if err != nil {
        logger.Error("AccountValidationHandler failed: repository error", "error", err)
        return &OperationResponse{Error: err}, nil
    }

    repo := repoAny.(repository.AccountRepository)
    account, err := repo.Get(req.AccountID)
    if err != nil {
        logger.Error("AccountValidationHandler failed: account not found", "error", err)
        return &OperationResponse{Error: account.ErrAccountNotFound}, nil
    }

    req.Account = account
    logger.Info("AccountValidationHandler: account validated successfully")

    return h.BaseHandler.Handle(ctx, req)
}

// Money creation handler
type MoneyCreationHandler struct {
    BaseHandler
    logger *slog.Logger
}

func (h *MoneyCreationHandler) Handle(ctx context.Context, req *OperationRequest) (*OperationResponse, error) {
    logger := h.logger.With("amount", req.Amount, "currency", req.CurrencyCode)

    money, err := mon.NewMoney(req.Amount, req.CurrencyCode)
    if err != nil {
        logger.Error("MoneyCreationHandler failed: invalid money", "error", err)
        return &OperationResponse{Error: err}, nil
    }

    req.Money = money
    logger.Info("MoneyCreationHandler: money created successfully")

    return h.BaseHandler.Handle(ctx, req)
}

// Currency conversion handler
type CurrencyConversionHandler struct {
    BaseHandler
    converter mon.CurrencyConverter
    logger    *slog.Logger
}

func (h *CurrencyConversionHandler) Handle(ctx context.Context, req *OperationRequest) (*OperationResponse, error) {
    logger := h.logger.With("fromCurrency", req.Money.Currency(), "toCurrency", req.Account.Currency)

    if req.Money.Currency() == req.Account.Currency {
        req.ConvertedMoney = req.Money
        logger.Info("CurrencyConversionHandler: no conversion needed")
        return h.BaseHandler.Handle(ctx, req)
    }

    convInfo, err := h.converter.Convert(req.Money.AmountFloat(), string(req.Money.Currency()), string(req.Account.Currency))
    if err != nil {
        logger.Error("CurrencyConversionHandler failed: conversion error", "error", err)
        return &OperationResponse{Error: err}, nil
    }

    convertedMoney, err := mon.NewMoney(convInfo.ConvertedAmount, req.Account.Currency)
    if err != nil {
        logger.Error("CurrencyConversionHandler failed: converted money creation error", "error", err)
        return &OperationResponse{Error: err}, nil
    }

    req.ConvertedMoney = convertedMoney
    req.ConvInfo = convInfo
    logger.Info("CurrencyConversionHandler: conversion completed", "rate", convInfo.ConversionRate)

    return h.BaseHandler.Handle(ctx, req)
}

// Domain operation handler
type DomainOperationHandler struct {
    BaseHandler
    logger *slog.Logger
}

func (h *DomainOperationHandler) Handle(ctx context.Context, req *OperationRequest) (*OperationResponse, error) {
    logger := h.logger.With("operation", req.Operation)

    var tx *account.Transaction
    var err error

    switch req.Operation {
    case OperationDeposit:
        tx, err = req.Account.Deposit(req.UserID, req.ConvertedMoney)
    case OperationWithdraw:
        tx, err = req.Account.Withdraw(req.UserID, req.ConvertedMoney)
    default:
        err = fmt.Errorf("unsupported operation: %s", req.Operation)
    }

    if err != nil {
        logger.Error("DomainOperationHandler failed: domain operation error", "error", err)
        return &OperationResponse{Error: err}, nil
    }

    req.Transaction = tx
    logger.Info("DomainOperationHandler: domain operation completed", "transactionID", tx.ID)

    return h.BaseHandler.Handle(ctx, req)
}

// Persistence handler
type PersistenceHandler struct {
    BaseHandler
    uow    repository.UnitOfWork
    logger *slog.Logger
}

func (h *PersistenceHandler) Handle(ctx context.Context, req *OperationRequest) (*OperationResponse, error) {
    logger := h.logger.With("transactionID", req.Transaction.ID)

    if req.ConvInfo != nil {
        req.Transaction.OriginalAmount = &req.ConvInfo.OriginalAmount
        req.Transaction.OriginalCurrency = &req.ConvInfo.OriginalCurrency
        req.Transaction.ConversionRate = &req.ConvInfo.ConversionRate
        logger.Info("PersistenceHandler: conversion info stored")
    }

    repoAny, err := h.uow.GetRepository(reflect.TypeOf((*repository.AccountRepository)(nil)).Elem())
    if err != nil {
        logger.Error("PersistenceHandler failed: AccountRepository error", "error", err)
        return &OperationResponse{Error: err}, nil
    }
    repo := repoAny.(repository.AccountRepository)

    if err = repo.Update(req.Account); err != nil {
        logger.Error("PersistenceHandler failed: account update error", "error", err)
        return &OperationResponse{Error: err}, nil
    }

    txRepoAny, err := h.uow.GetRepository(reflect.TypeOf((*repository.TransactionRepository)(nil)).Elem())
    if err != nil {
        logger.Error("PersistenceHandler failed: TransactionRepository error", "error", err)
        return &OperationResponse{Error: err}, nil
    }
    txRepo := txRepoAny.(repository.TransactionRepository)

    if err = txRepo.Create(req.Transaction); err != nil {
        logger.Error("PersistenceHandler failed: transaction create error", "error", err)
        return &OperationResponse{Error: err}, nil
    }

    logger.Info("PersistenceHandler: persistence completed successfully")

    return &OperationResponse{
        Transaction: req.Transaction,
        ConvInfo:    req.ConvInfo,
    }, nil
}

// Chain builder
type ChainBuilder struct {
    uow       repository.UnitOfWork
    converter mon.CurrencyConverter
    logger    *slog.Logger
}

func NewChainBuilder(uow repository.UnitOfWork, converter mon.CurrencyConverter, logger *slog.Logger) *ChainBuilder {
    return &ChainBuilder{
        uow:       uow,
        converter: converter,
        logger:    logger,
    }
}

func (b *ChainBuilder) BuildOperationChain() OperationHandler {
    accountValidation := &AccountValidationHandler{
        uow:    b.uow,
        logger: b.logger,
    }

    moneyCreation := &MoneyCreationHandler{
        logger: b.logger,
    }

    currencyConversion := &CurrencyConversionHandler{
        converter: b.converter,
        logger:    b.logger,
    }

    domainOperation := &DomainOperationHandler{
        logger: b.logger,
    }

    persistence := &PersistenceHandler{
        uow:    b.uow,
        logger: b.logger,
    }

    // Chain them together
    accountValidation.SetNext(moneyCreation)
    moneyCreation.SetNext(currencyConversion)
    currencyConversion.SetNext(domainOperation)
    domainOperation.SetNext(persistence)

    return accountValidation
}

// Updated service
type AccountService struct {
    uow       repository.UnitOfWork
    converter mon.CurrencyConverter
    logger    *slog.Logger
    chain     OperationHandler
}

func NewAccountService(uow repository.UnitOfWork, converter mon.CurrencyConverter, logger *slog.Logger) *AccountService {
    builder := NewChainBuilder(uow, converter, logger)
    return &AccountService{
        uow:       uow,
        converter: converter,
        logger:    logger,
        chain:     builder.BuildOperationChain(),
    }
}

func (s *AccountService) Deposit(userID, accountID uuid.UUID, amount float64, currencyCode currency.Code) (*account.Transaction, *common.ConversionInfo, error) {
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

func (s *AccountService) Withdraw(userID, accountID uuid.UUID, amount float64, currencyCode currency.Code) (*account.Transaction, *common.ConversionInfo, error) {
    req := &OperationRequest{
        UserID:       userID,
        AccountID:    accountID,
        Amount:       amount,
        CurrencyCode: currencyCode,
        Operation:    OperationWithdraw,
    }

    resp, err := s.chain.Handle(context.Background(), req)
    if err != nil {
        return nil, nil, err
    }

    return resp.Transaction, resp.ConvInfo, resp.Error
}
```

**Benefits:**

- ✅ Single responsibility per handler
- ✅ Zero branching in service layer
- ✅ Easy to extend with new handlers
- ✅ Excellent testability
- ✅ Clear, linear flow

### 4. Event-Driven Architecture (Analyzed)

**Approach:** Convert operations into events that trigger cascading reactions.

**Complete Example:**

```go
// Event interface
type Event interface {
    EventType() string
    Timestamp() time.Time
    CorrelationID() string
}

// Account operation events
type AccountDepositRequested struct {
    UserID        uuid.UUID
    AccountID     uuid.UUID
    Amount        float64
    CurrencyCode  currency.Code
    Timestamp     time.Time
    CorrelationID string
}

func (e AccountDepositRequested) EventType() string { return "AccountDepositRequested" }
func (e AccountDepositRequested) Timestamp() time.Time { return e.Timestamp }
func (e AccountDepositRequested) CorrelationID() string { return e.CorrelationID }

type AccountWithdrawalRequested struct {
    UserID        uuid.UUID
    AccountID     uuid.UUID
    Amount        float64
    CurrencyCode  currency.Code
    Timestamp     time.Time
    CorrelationID string
}

func (e AccountWithdrawalRequested) EventType() string { return "AccountWithdrawalRequested" }
func (e AccountWithdrawalRequested) Timestamp() time.Time { return e.Timestamp }
func (e AccountWithdrawalRequested) CorrelationID() string { return e.CorrelationID }

// Domain events
type AccountValidated struct {
    Account       *account.Account
    UserID        uuid.UUID
    AccountID     uuid.UUID
    Timestamp     time.Time
    CorrelationID string
}

func (e AccountValidated) EventType() string { return "AccountValidated" }
func (e AccountValidated) Timestamp() time.Time { return e.Timestamp }
func (e AccountValidated) CorrelationID() string { return e.CorrelationID }

type CurrencyConversionCompleted struct {
    OriginalAmount    float64
    ConvertedAmount   float64
    OriginalCurrency  currency.Code
    TargetCurrency    currency.Code
    ConversionRate    float64
    Timestamp         time.Time
    CorrelationID     string
}

func (e CurrencyConversionCompleted) EventType() string { return "CurrencyConversionCompleted" }
func (e CurrencyConversionCompleted) Timestamp() time.Time { return e.Timestamp }
func (e CurrencyConversionCompleted) CorrelationID() string { return e.CorrelationID }

type AccountOperationCompleted struct {
    Transaction    *account.Transaction
    ConvInfo       *common.ConversionInfo
    Timestamp      time.Time
    CorrelationID  string
}

func (e AccountOperationCompleted) EventType() string { return "AccountOperationCompleted" }
func (e AccountOperationCompleted) Timestamp() time.Time { return e.Timestamp }
func (e AccountOperationCompleted) CorrelationID() string { return e.CorrelationID }

type AccountOperationFailed struct {
    Error          error
    UserID         uuid.UUID
    AccountID      uuid.UUID
    Timestamp      time.Time
    CorrelationID  string
}

func (e AccountOperationFailed) EventType() string { return "AccountOperationFailed" }
func (e AccountOperationFailed) Timestamp() time.Time { return e.Timestamp }
func (e AccountOperationFailed) CorrelationID() string { return e.CorrelationID }

// Event bus
type EventBus interface {
    Publish(event Event) error
    Subscribe(eventType string, handler EventHandler) error
    Unsubscribe(eventType string, handler EventHandler) error
}

type EventHandler func(ctx context.Context, event Event) error

// In-memory event bus
type InMemoryEventBus struct {
    handlers map[string][]EventHandler
    mu       sync.RWMutex
}

func NewInMemoryEventBus() *InMemoryEventBus {
    return &InMemoryEventBus{
        handlers: make(map[string][]EventHandler),
    }
}

func (b *InMemoryEventBus) Publish(event Event) error {
    b.mu.RLock()
    handlers := b.handlers[event.EventType()]
    b.mu.RUnlock()

    for _, handler := range handlers {
        if err := handler(context.Background(), event); err != nil {
            return fmt.Errorf("handler failed for event %s: %w", event.EventType(), err)
        }
    }
    return nil
}

func (b *InMemoryEventBus) Subscribe(eventType string, handler EventHandler) error {
    b.mu.Lock()
    defer b.mu.Unlock()
    b.handlers[eventType] = append(b.handlers[eventType], handler)
    return nil
}

func (b *InMemoryEventBus) Unsubscribe(eventType string, handler EventHandler) error {
    // Implementation for removing handlers
    return nil
}

// Event handlers
type AccountValidationHandler struct {
    uow    repository.UnitOfWork
    bus    EventBus
    logger *slog.Logger
}

func NewAccountValidationHandler(uow repository.UnitOfWork, bus EventBus, logger *slog.Logger) *AccountValidationHandler {
    handler := &AccountValidationHandler{
        uow:    uow,
        bus:    bus,
        logger: logger,
    }

    bus.Subscribe("AccountDepositRequested", handler.Handle)
    bus.Subscribe("AccountWithdrawalRequested", handler.Handle)

    return handler
}

func (h *AccountValidationHandler) Handle(ctx context.Context, event Event) error {
    logger := h.logger.With("eventType", event.EventType(), "correlationID", event.CorrelationID())

    var userID, accountID uuid.UUID
    var amount float64
    var currencyCode currency.Code

    switch e := event.(type) {
    case AccountDepositRequested:
        userID, accountID, amount, currencyCode = e.UserID, e.AccountID, e.Amount, e.CurrencyCode
    case AccountWithdrawalRequested:
        userID, accountID, amount, currencyCode = e.UserID, e.AccountID, e.Amount, e.CurrencyCode
    default:
        return fmt.Errorf("unsupported event type: %T", event)
    }

    repoAny, err := h.uow.GetRepository(reflect.TypeOf((*repository.AccountRepository)(nil)).Elem())
    if err != nil {
        logger.Error("AccountValidationHandler failed: repository error", "error", err)
        return h.bus.Publish(AccountOperationFailed{
            Error:         err,
            UserID:        userID,
            AccountID:     accountID,
            Timestamp:     time.Now(),
            CorrelationID: event.CorrelationID(),
        })
    }

    repo := repoAny.(repository.AccountRepository)
    account, err := repo.Get(accountID)
    if err != nil {
        logger.Error("AccountValidationHandler failed: account not found", "error", err)
        return h.bus.Publish(AccountOperationFailed{
            Error:         account.ErrAccountNotFound,
            UserID:        userID,
            AccountID:     accountID,
            Timestamp:     time.Now(),
            CorrelationID: event.CorrelationID(),
        })
    }

    logger.Info("AccountValidationHandler: account validated successfully")

    return h.bus.Publish(AccountValidated{
        Account:       account,
        UserID:        userID,
        AccountID:     accountID,
        Timestamp:     time.Now(),
        CorrelationID: event.CorrelationID(),
    })
}

// Updated service
type AccountService struct {
    bus    EventBus
    logger *slog.Logger
}

func NewAccountService(uow repository.UnitOfWork, converter mon.CurrencyConverter, logger *slog.Logger) *AccountService {
    bus := NewInMemoryEventBus()

    // Initialize all handlers
    NewAccountValidationHandler(uow, bus, logger)
    // Add more handlers as needed

    return &AccountService{
        bus:    bus,
        logger: logger,
    }
}

func (s *AccountService) Deposit(userID, accountID uuid.UUID, amount float64, currencyCode currency.Code) (*account.Transaction, *common.ConversionInfo, error) {
    correlationID := uuid.New().String()

    event := AccountDepositRequested{
        UserID:        userID,
        AccountID:     accountID,
        Amount:        amount,
        CurrencyCode:  currencyCode,
        Timestamp:     time.Now(),
        CorrelationID: correlationID,
    }

    // Publish the event
    if err := s.bus.Publish(event); err != nil {
        return nil, nil, err
    }

    // In a real implementation, you'd need to wait for completion
    // This is a simplified version
    return nil, nil, nil
}
```

**Benefits:**

- ✅ Complete decoupling
- ✅ Excellent scalability
- ✅ Built-in observability
- ✅ Async processing capabilities
- ✅ Easy to add cross-cutting concerns

**Challenges:**

- ❌ Complex data flow across events
- ❌ Synchronous operations become complex
- ❌ Transaction management difficulties
- ❌ Error propagation complexity
- ❌ Event correlation challenges

## Mock Payment Provider Pattern

The MockPaymentProvider (see infra/provider/mock_payment_provider.go) is used for simulating payment flows in tests and local development.

- InitiateDeposit/InitiateWithdraw simulate async payment completion after a short delay.
- The service polls GetPaymentStatus until PaymentCompleted is returned (see pkg/service/account/account.go).
- This pattern is suitable for local development, integration tests, and MVPs.
- **Not for production:** Real payment providers use webhooks or callbacks for async confirmation.

### When to Use

- Local development
- Integration and E2E tests
- MVPs or internal tools

### When NOT to Use

- Production payment flows
- When integrating with real payment providers (Stripe, banks, etc.)

### Migration Path

For production, migrate to an event-driven model:

- Initiate payment, return "pending" status
- Listen for webhook/callback from provider
- Update transaction status asynchronously
- Notify user if needed

## Pattern Comparison

| Pattern | Branching | Extensibility | Testability | Complexity | Go Idiomatic |
|---------|-----------|---------------|-------------|------------|--------------|
| **Strategy** | Low | Good | Good | Medium | ✅ |
| **Command** | None | Excellent | Excellent | High | ⚠️ |
| **Chain of Responsibility** | None | Excellent | Excellent | Medium | ✅ |
| **Event-Driven** | None | Excellent | Good | High | ⚠️ |

## Recommendations

### For Current Use Case

**Chain of Responsibility** is the best fit because:

- Eliminates all branching in the service layer
- Maintains Go idioms and simplicity
- Provides excellent extensibility
- Each handler has a single, clear responsibility
- Easy to test and maintain

### For Future Extensions

Consider **hybrid approaches**:

- **Strategy + Chain of Responsibility**: Use strategy for operation type, chain for execution steps
- **Synchronous + Event-Driven**: Keep core business logic synchronous, use events for side effects (audit, notifications)

## Implementation Status

- ✅ **Strategy Pattern**: Fully implemented and working
- 🔄 **Chain of Responsibility**: Ready for implementation
- 📋 **Command Pattern**: Analyzed, ready for implementation if needed
- 📋 **Event-Driven**: Analyzed, suitable for specific use cases

## Next Steps

1. **Implement Chain of Responsibility** to further reduce complexity
2. **Add comprehensive tests** for all patterns
3. **Consider hybrid approaches** for specific requirements
4. **Document pattern selection criteria** for future development

## Code Quality Metrics

### Before Refactoring

- **Lines of Code**: ~566 lines in single file
- **Cyclomatic Complexity**: High (multiple nested if-else blocks)
- **Code Duplication**: ~150 lines duplicated between Deposit/Withdraw
- **Maintainability**: Poor (tightly coupled logic)

### After Strategy Pattern

- **Lines of Code**: ~700 lines across 7 focused files
- **Cyclomatic Complexity**: Reduced (linear flow in executeOperation)
- **Code Duplication**: Eliminated
- **Maintainability**: Excellent (clear separation of concerns)

### Expected After Chain of Responsibility

- **Lines of Code**: ~800 lines across 10+ focused files
- **Cyclomatic Complexity**: Minimal (linear handler chain)
- **Code Duplication**: None
- **Maintainability**: Outstanding (single responsibility per handler)

## Conclusion

The refactoring journey demonstrates how different design patterns can address the same problem with varying trade-offs. The **Strategy Pattern** provided immediate benefits, while **Chain of Responsibility** offers the best long-term solution for this specific use case.

The key insight is that **pattern selection should be driven by specific requirements** rather than following a one-size-fits-all approach. For fintech applications requiring high reliability and maintainability, the Chain of Responsibility pattern provides the optimal balance of simplicity, extensibility, and Go idiomaticity.

## Mega Refactor: Event-Driven, Operation-Specific Handlers

### Overview

The account service now uses **operation-specific persistence handlers** and an event-driven approach for all account operations (deposit, withdraw, transfer). The legacy monolithic `PersistenceHandler` has been replaced by:

- `DepositPersistenceHandler`
- `WithdrawPersistenceHandler`
- `TransferPersistenceHandler`

Transaction creation logic is now centralized in `transaction_factory.go` as reusable helpers, ensuring DRY and consistent transaction records.

### Handler Chain Example (Deposit)

```go
// Chain for deposit operation:
ValidationHandler → MoneyCreationHandler → CurrencyConversionHandler → DomainOperationHandler → DepositPersistenceHandler
```

### Event-Driven Persistence

- Domain methods emit events (e.g., `DepositRequestedEvent`).
- Persistence handlers pull these events and use factory helpers to create transactions.
- Each handler is focused, testable, and only responsible for its operation.

### Updated File Structure

```ascii
pkg/handler/
├── base.go
├── builder.go
├── deposit_persistence.go
├── withdraw_persistence.go
├── transfer_persistence.go
├── transaction_factory.go
└── ...
```

### Example: DepositPersistenceHandler

```go
// DepositPersistenceHandler handles deposit events and persistence
func (h *DepositPersistenceHandler) Handle(ctx context.Context, req *OperationRequest) (*OperationResponse, error) {
    events := req.Account.PullEvents()
    for _, evt := range events {
        e, ok := evt.(account.DepositRequestedEvent) //nolint:go-critic
        if !ok { continue }
        tx := NewDepositTransaction(e)
        // persist tx and update account
    }
    // ...
}
```

### Benefits

- Each operation is isolated and testable
- Transaction creation is DRY and consistent
- Event-driven: domain emits events, handlers persist them
