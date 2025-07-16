---
icon: material/format-paint
---

# Decorator Pattern

## 🏁 Overview

The fintech application previously used the decorator pattern for automatic transaction management across service operations. With the adoption of the UnitOfWork interface pattern, transaction management is now handled directly in the service layer, providing type-safe, context-aware, and testable transaction boundaries.

## 🏗️ Implementation

### 🧰 UnitOfWork Interface

Transaction management is now performed using the `UnitOfWork` interface, which exposes a `Do(ctx, func(uow UnitOfWork) error)` method. This method ensures all repository operations within a transaction use the same DB session, providing atomicity and consistency.

#### 🧪 Example Usage

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

## 🌍 Use Cases

## 🚀 Benefits

- **DRY Principle**: Eliminates transaction boilerplate
- **Separation of Concerns**: Business logic separated from transaction management
- **Readability**: Clean, focused business operations
- **Maintainability**: Transaction logic centralized in the UnitOfWork
- **Automatic Rollback**: On any error or panic
- **Consistent Logging**: Structured logging for all transaction events
- **Easier Mocking**: Business logic can be tested in isolation

## 🧰 Service Integration

All service methods now follow this pattern:

1. **Logging**: Start operation with context
2. **Transaction**: Execute business logic within `uow.Do(ctx, ...)`
