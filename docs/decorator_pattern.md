# Decorator Pattern Implementation 🎨

## Overview

The fintech application previously used the decorator pattern for automatic transaction management across service operations. With the adoption of the UnitOfWork interface pattern, transaction management is now handled directly in the service layer, providing type-safe, context-aware, and testable transaction boundaries.

## Current Transaction Management

### UnitOfWork Interface

Transaction management is now performed using the `UnitOfWork` interface, which exposes a `Do(ctx, func(uow UnitOfWork) error)` method. This method ensures all repository operations within a transaction use the same DB session, providing atomicity and consistency.

#### Example Usage

```go
func (s *AccountService) CreateAccount(ctx context.Context, userID uuid.UUID) (*account.Account, error) {
    var a *account.Account
    err := s.uow.Do(ctx, func(uow repository.UnitOfWork) error {
        repo, err := uow.GetRepository[repository.AccountRepository]()
        if err != nil {
            return err
    }
        a, err = account.New().WithUserID(userID).Build()
        if err != nil {
            return err
        }
        return repo.Create(a)
    })
    if err != nil {
        a = nil
    }
    return a, err
}
```

## Benefits

- **DRY Principle**: Eliminates transaction boilerplate
- **Separation of Concerns**: Business logic separated from transaction management
- **Readability**: Clean, focused business operations
- **Maintainability**: Transaction logic centralized in the UnitOfWork
- **Automatic Rollback**: On any error or panic
- **Consistent Logging**: Structured logging for all transaction events
- **Easier Mocking**: Business logic can be tested in isolation

## Service Integration

All service methods now follow this pattern:

1. **Logging**: Start operation with context
2. **Transaction**: Execute business logic within `uow.Do(ctx, ...)`
3. **Error Handling**: Automatic rollback on errors
4. **Logging**: Success/failure with context

## Migration Notes

- The `TransactionDecorator` and `UnitOfWorkTransactionDecorator` are no longer used.
- All transaction management is now handled via the `UnitOfWork` interface and its `Do` method.
- Service constructors and methods should accept and use a `repository.UnitOfWork` directly.

## Testing Strategy

### Unit Testing

```go
func TestCreateAccount_Success(t *testing.T) {
    svc, accountRepo, _, uow := newServiceWithMocks(t)
    uow.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, fn func(uow repository.UnitOfWork) error) error {
        return fn(uow)
    })
    accountRepo.EXPECT().Create(mock.Anything).Return(nil)

    userID := uuid.New()
    gotAccount, err := svc.CreateAccount(context.Background(), userID)
    assert.NoError(t, err)
    assert.NotNil(t, gotAccount)
}
```

### Integration Testing

- **Transaction Scenarios**: Test rollback on errors
- **Error Propagation**: Test error mapping

## Summary

Transaction management is now cleanly and safely handled via the UnitOfWork interface, with all business logic executed within transaction boundaries using `uow.Do(ctx, ...)`. This approach is idiomatic, testable, and easy to maintain.
