---
icon: material/architecture
---

# 🧹 Clean Event-Driven Architecture Refactoring

## 🎯 Goals Achieved

This refactoring successfully addressed the messy event handler registration and achieved the following goals:

- ✅ **No misleading events** - Clear event flow and responsibilities
- ✅ **DRY (Don't Repeat Yourself)** - Generic payment handlers instead of operation-specific ones
- ✅ **SRP (Single Responsibility Principle)** - Each handler has one clear responsibility
- ✅ **No payment handlers per operation** - Single generic payment handlers that subscribe to multiple validation events

## 🔄 Before vs After

### Before (Messy)
```go
// Operation-specific payment handlers (violates DRY)
bus.Subscribe("DepositConversionDoneEvent", deposithandler.ConversionDoneHandler(bus, deps.Uow, deps.PaymentProvider, deps.Logger))
bus.Subscribe("WithdrawConversionDoneEvent", withdrawhandler.ConversionDoneHandler(bus, deps.Uow, deps.PaymentProvider, deps.Logger))

// Duplicate payment persistence handlers (violates DRY)
bus.Subscribe("PaymentInitiatedEvent", deposithandler.PaymentPersistenceHandler(bus, deps.Uow, deps.Logger))
bus.Subscribe("PaymentInitiatedEvent", withdrawhandler.PaymentPersistenceHandler(bus, deps.Uow, deps.Logger))

// Conversion handlers doing payment initiation (violates SRP)
// ConversionDoneHandler was doing both business validation AND payment initiation
```

### After (Clean)
```go
// Generic payment handlers (DRY)
bus.Subscribe("DepositValidatedEvent", paymenthandler.PaymentInitiationHandler(bus, deps.PaymentProvider, deps.Logger))
bus.Subscribe("WithdrawValidatedEvent", paymenthandler.PaymentInitiationHandler(bus, deps.PaymentProvider, deps.Logger))

// Single generic payment persistence handler (DRY)
bus.Subscribe("PaymentInitiatedEvent", paymenthandler.PaymentPersistenceHandler(bus, deps.Uow, deps.Logger))

// Conversion handlers focus only on business validation (SRP)
bus.Subscribe("DepositConversionDoneEvent", deposithandler.ConversionDoneHandler(bus, deps.Uow, deps.Logger))
bus.Subscribe("WithdrawConversionDoneEvent", withdrawhandler.ConversionDoneHandler(bus, deps.Uow, deps.Logger))
```

## 🏗️ Architecture Changes

### 1. **Generic Payment Initiation Handler**

**Location**: `pkg/handler/payment/initiation_handler.go`

**Responsibilities**:
- Handles `DepositValidatedEvent` and `WithdrawValidatedEvent`
- Initiates payment with the payment provider
- Emits `PaymentInitiatedEvent`

**Benefits**:
- ✅ **DRY**: Single handler for all payment initiation
- ✅ **SRP**: Only handles payment initiation, not business validation
- ✅ **Extensible**: Easy to add new validation events (e.g., `TransferValidatedEvent`)

### 2. **Generic Payment HandleProcessed Handler**

**Location**: `pkg/handler/payment/persistence_handler.go`

**Responsibilities**:
- Handles `PaymentInitiatedEvent` for all operations
- Updates transaction with payment ID
- Emits `PaymentIdPersistedEvent`

**Benefits**:
- ✅ **DRY**: Single handler for all payment persistence
- ✅ **SRP**: Only handles payment persistence, not business logic
- ✅ **Consistent**: Same persistence logic for all operations

### 3. **Clean Conversion Done Handlers**

**Location**: `pkg/handler/account/deposit/conversion_done.go` and `pkg/handler/account/withdraw/conversion_done.go`

**Responsibilities**:
- Handle `*ConversionDoneEvent` events
- Perform business validation after conversion
- Emit validation events to trigger payment initiation

**Benefits**:
- ✅ **SRP**: Only handles business validation, not payment initiation
- ✅ **Clear Flow**: Validation → Payment Initiation (not Validation + Payment)
- ✅ **Testable**: Easy to test business validation logic separately

## 🔄 Event Flow

### Deposit Flow
```
DepositRequestedEvent → ValidationHandler → DepositValidatedEvent → PaymentInitiationHandler → PaymentInitiatedEvent → PaymentPersistenceHandler
```

### Withdraw Flow
```
WithdrawRequestedEvent → ValidationHandler → WithdrawValidatedEvent → PaymentInitiationHandler → PaymentInitiatedEvent → PaymentPersistenceHandler
```

### Transfer Flow (No Payment)
```
TransferRequestedEvent → ValidationHandler → TransferValidatedEvent → DomainOpHandler → TransferDomainOpDoneEvent → PersistenceHandler
```

## 🧪 Testing

### Updated Tests
- ✅ **Payment Initiation Handler**: Tests for `DepositValidatedEvent` and `WithdrawValidatedEvent`
- ✅ **Payment HandleProcessed Handler**: Tests for `PaymentInitiatedEvent`
- ✅ **Conversion Done Handlers**: Tests for business validation only

### Test Coverage
```bash
go test ./pkg/handler/payment/... -v  # ✅ All passing
go test ./pkg/handler/... -v          # ✅ All passing
```

## 🗂️ File Changes

### Added/Modified
- ✅ `pkg/handler/payment/initiation_handler.go` - Generic payment initiation
- ✅ `pkg/handler/payment/persistence_handler.go` - Generic payment persistence
- ✅ `pkg/handler/account/deposit/conversion_done.go` - Clean business validation
- ✅ `pkg/handler/account/withdraw/conversion_done.go` - Clean business validation
- ✅ `app/app.go` - Updated event handler registration

### Removed
- ❌ `pkg/handler/account/deposit/payment_persistence.go` - Duplicate code
- ❌ `pkg/handler/account/withdraw/payment_persistence.go` - Duplicate code

## 🎯 Key Principles Applied

### 1. **Event-Driven Design**
- Events trigger next steps, not conditional logic
- Clear event flow: Validation → Payment → HandleProcessed
- No if-else statements for control flow

### 2. **Single Responsibility Principle**
- Each handler has one clear responsibility
- Conversion handlers: Business validation only
- Payment handlers: Payment operations only
- HandleProcessed handlers: Database operations only

### 3. **Don't Repeat Yourself**
- Generic payment handlers for all operations
- Single payment persistence logic
- Reusable event structures

### 4. **Separation of Concerns**
- Business validation separated from payment initiation
- Payment logic separated from conversion logic
- HandleProcessed logic separated from business logic

## 🚀 Benefits

### 1. **Maintainability**
- Clear separation of concerns
- Easy to understand and modify individual components
- Consistent patterns across all handlers

### 2. **Testability**
- Each handler can be tested independently
- Clear input/output expectations
- Easy to mock dependencies

### 3. **Extensibility**
- Easy to add new operations (e.g., loan payments)
- Easy to add new payment providers
- Easy to add new validation rules

### 4. **Reliability**
- No duplicate code to maintain
- Clear error handling paths
- Consistent event flow

## 🔮 Future Enhancements

### 1. **Add Transfer Payment Support**
```go
// Easy to extend - just add new event subscription
bus.Subscribe("TransferValidatedEvent", paymenthandler.PaymentInitiationHandler(bus, deps.PaymentProvider, deps.Logger))
```

### 2. **Add Payment Completion Handlers**
```go
// Generic payment completion handling
bus.Subscribe("PaymentCompletedEvent", paymenthandler.PaymentCompletionHandler(bus, deps.Uow, deps.Logger))
```

### 3. **Add Payment Failure Handlers**
```go
// Generic payment failure handling
bus.Subscribe("PaymentFailedEvent", paymenthandler.PaymentFailureHandler(bus, deps.Uow, deps.Logger))
```

## 📚 Related Documentation

- [Event-Driven Architecture](event-driven-architecture.md)
- [Event-Driven Deposit Flow](event-driven-deposit-flow.md)
- [Event-Driven Withdraw Flow](event-driven-withdraw-flow.md)
- [Event-Driven Transfer Flow](event-driven-transfer-flow.md)
- [Domain Events](domain-events.md)
