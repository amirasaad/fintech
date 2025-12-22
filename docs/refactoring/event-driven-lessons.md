---
icon: material/school
---
# 📚 Event-Driven Architecture: Lessons Learned

## 📌 Overview

This document captures the key lessons learned during our event-driven architecture refactor, including the problems we solved and the design decisions that led to our final architecture.

## 📌 Key Problems Solved

### 1. Coupling Currency Conversion with Payment Initiation

**Problem**: Original design coupled currency conversion with payment initiation, making it difficult to:

- Reuse conversion logic across different operations
- Test conversion independently
- Add new business flows without modifying conversion logic
- Maintain clean separation of concerns

**Solution**: Decoupled currency conversion as a pure, reusable service:

- Conversion events (`ConversionRequestedEvent`, `ConversionDoneEvent`) are generic
- Conversion handler has no business logic or side effects
- Payment is triggered by business validation, not conversion
- Each business operation can use conversion without coupling

**Benefits**:

- Reusable conversion logic across deposit, withdraw, transfer
- Easier to test and mock conversion independently
- Clear separation between conversion and business logic
- No code duplication

### 2. Business Validation Before Currency Conversion

**Problem**: Business invariants (sufficient funds, limits) were sometimes checked before currency conversion:

- Led to incorrect validations when request currency differed from account currency
- Could bypass limits or allow overdrafts due to currency mismatches
- Inconsistent validation behavior across operations

**Solution**: All business validations performed after currency conversion:

- Sufficient funds check in account's native currency
- Maximum/minimum limits in account's native currency
- All business rules applied to converted amounts
- Consistent validation regardless of request currency

**Benefits**:

- Accurate validation regardless of request currency
- Consistent business rule enforcement
- No currency-related validation bugs
- Clear audit trail of validation in correct currency

### 3. If-Statements for Control Flow

**Problem**: Using conditional logic in handlers to determine next steps:

- Led to complex if-else statements
- Made handlers harder to test and reason about
- Violated single responsibility principle
- Made event flow unclear

**Solution**: Event chaining with business-specific events:

- Each handler emits specific events for next steps
- No conditional logic in handlers
- Clear event flow and responsibilities
- Easy to test individual handlers

**Benefits**:

- No if-else statements for control flow
- Clear event flow and responsibilities
- Easy to test individual handlers
- Follows single responsibility principle

### 4. Payment Triggered by Conversion

**Problem**: Payment was triggered by conversion completion, not business validation:

- Payment could occur before all business rules were validated
- Unclear audit trail of validation → payment flow
- Difficult to add new payment triggers

**Solution**: Payment triggered by business validation events:

- `WithdrawValidatedEvent` triggers payment for withdrawals
- `DepositValidatedEvent` triggers payment for deposits
- Business validation ensures all rules pass before payment
- Clear separation between validation and payment

**Benefits**:

- Payment only occurs after all validations pass
- Clear audit trail of validation → payment flow
- Easier to add new payment triggers
- Better error handling and rollback capabilities

## Design Decisions and Motivations

### 1. Generic vs Business-Specific Events

**Decision**: Use both generic and business-specific events

**Generic Events** (reusable):

- `ConversionRequestedEvent`
- `ConversionDoneEvent`

**Business-Specific Events** (context-aware):

- `Deposit.CurrencyConverted`
- `Withdraw.CurrencyConverted`
- `Transfer.CurrencyConverted`

**Motivation**:

- Generic events for reusable logic (conversion)
- Business-specific events for context-aware operations
- Clear event hierarchy and responsibilities
- Avoid if-statements for control flow

### 2. Event Chaining Pattern

**Decision**: Use event chaining for dependent business logic

**Pattern**:

```
User Request → Validation → Conversion (if needed) → Business Validation → Payment/Domain Op → HandleProcessed
```

**Motivation**:

- Each handler has a single responsibility
- Clear flow of dependent operations
- Easy to test individual steps
- No orchestration logic in handlers

### 3. Dependency Injection

**Decision**: Inject all dependencies into handlers

**Pattern**:

```go
func BusinessHandler(deps Dependencies) func(context.Context, domain.Event)
```

**Motivation**:

- Easy to test with mocks
- Clear dependencies
- Follows dependency inversion principle
- Consistent across all handlers

### 4. Structured Logging

**Decision**: Use structured logging with context in all handlers

**Pattern**:

```go
logger := deps.Logger.With("handler", "BusinessHandler")
logger.Info("processing event", "event", evt)
```

**Motivation**:

- Clear audit trail
- Easy debugging
- Consistent observability
- Production-ready logging

## Architecture Benefits Achieved

### 1. Maintainability

- Clear separation of concerns
- Each handler has a single responsibility
- Easy to modify individual components
- No hidden dependencies

### 2. Testability

- Each handler can be tested in isolation
- Mock dependencies easily injected
- Event-driven testing patterns
- Clear test boundaries

### 3. Scalability

- Handlers can be scaled independently
- Event bus can be distributed
- Easy to add new business operations
- No tight coupling

### 4. Flexibility

- New currencies can be added without changing business logic
- New payment providers can be integrated easily
- Business rules can be modified independently
- Clear extension points

### 5. Observability

- Clear event flow for debugging
- Structured logging at each step
- Audit trail of all operations
- Easy to monitor and alert

## Migration Strategy

### Phase 1: Introduce Generic Conversion Events

1. Create `ConversionRequestedEvent` and `ConversionDoneEvent`
2. Implement generic conversion handler
3. Update existing handlers to emit generic conversion events

### Phase 2: Add Business-Specific Conversion Done Handlers

1. Create business-specific conversion done events
2. Implement handlers for each business operation
3. Wire handlers in event bus

### Phase 3: Decouple Payment from Conversion

1. Move payment initiation to business validation handlers
2. Update event flow to trigger payment after validation
3. Test all flows end-to-end

### Phase 4: Update Validation Logic

1. Ensure all validations happen after conversion
2. Update business rules to work with converted amounts
3. Add comprehensive tests for multi-currency scenarios

## Best Practices Established

### 1. Event Handler Structure

```go
func BusinessHandler(deps Dependencies) func(context.Context, domain.Event) {
    return func(ctx context.Context, e domain.Event) {
        logger := deps.Logger.With("handler", "BusinessHandler")

        // 1. Type assertion
        evt, ok := e.(SpecificEvent)
        if !ok {
            logger.Error("unexpected event type", "event", e)
            return
        }

        // 2. Business logic
        logger.Info("processing event", "event", evt)

        // 3. Emit next event
        _ = deps.EventBus.Publish(ctx, NextEvent{})
    }
}
```

### 2. Error Handling

- Log errors with context
- Don't panic on unexpected events
- Consider retry strategies for transient failures
- Emit failure events when appropriate

### 3. Testing

- Use mocks for external dependencies
- Test event flow end-to-end
- Verify event emissions
- Test error scenarios

### 4. Documentation

- Document event flows clearly
- Explain handler responsibilities
- Provide examples and diagrams
- Keep documentation up to date

## Final Thoughts

The event-driven architecture refactor has significantly improved our codebase by:

1. **Eliminating coupling** between currency conversion and payment processing
2. **Ensuring correct validation** by performing all checks after currency conversion
3. **Removing if-statements** for control flow through event chaining
4. **Improving testability** through dependency injection and clear boundaries
5. **Enabling flexibility** for future extensions and new business operations

The key insight was that **currency conversion should be a pure, reusable service** that doesn't trigger business operations, while **business validation should always happen after conversion** to ensure accuracy. This separation enables a clean, maintainable, and extensible architecture for multi-currency financial operations.

## Next Steps

1. **Complete the refactor** for withdraw and transfer validation handlers
2. **Add comprehensive tests** for all event flows
3. **Update documentation** with final architecture diagrams
4. **Monitor performance** and optimize as needed
5. **Plan future extensions** using the established patterns
