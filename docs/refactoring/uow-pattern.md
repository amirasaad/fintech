---
icon: material/tools
---

# Unit of Work (UOW) Pattern

## 🏁 Overview

This document outlines the improvements made to the Unit of Work (UOW) pattern in the fintech application, focusing on maintaining transaction safety while improving type safety and developer experience.

## ✅ What Works Well

The current UOW pattern provides excellent transaction management:

```go
// Current pattern - excellent transaction handling
err = s.uow.Do(context.Background(), func(uow repository.UnitOfWork) error {
    // All operations use the same transaction session
    repoAny, err := uow.GetRepository(reflect.TypeOf((*repository.AccountRepository)(nil)).Elem())
    if err != nil {
        return err
    }
    accountRepo := repoAny.(repository.AccountRepository)

    // Business logic here...
    return nil
})
```

**Benefits:**

- ✅ **Automatic transaction boundaries** - begin/commit/rollback handled automatically
- ✅ **Repository coordination** - all repositories use same transaction session
- ✅ **Atomic operations** - all-or-nothing semantics
- ✅ **Error handling** - automatic rollback on any error
- ✅ **Clean architecture** - business logic separated from infrastructure

## ❌ What Needs Improvement

The current pattern has some developer experience issues:

```go
// Problems with current approach:
// 1. Complex reflect syntax
// 2. Type casting required
// 3. Runtime errors possible
// 4. Poor IDE support
repoAny, err := uow.GetRepository(reflect.TypeOf((*repository.AccountRepository)(nil)).Elem())
accountRepo := repoAny.(repository.AccountRepository)
```

## 📦 Implementation

We've added type-safe convenience methods to the UOW interface:

```go
// pkg/repository/uow.go
type UnitOfWork interface {
    // Existing methods
    Do(ctx context.Context, fn func(uow UnitOfWork) error) error
    GetRepository(repoType reflect.Type) (any, error)

    // New type-safe convenience methods
    AccountRepository() (AccountRepository, error)
    TransactionRepository() (TransactionRepository, error)
    UserRepository() (UserRepository, error)
}
```

### Implementation in Infrastructure Layer

```go
// infra/repository/uow.go
func (u *UoW) AccountRepository() (repository.AccountRepository, error) {
    repoAny, err := u.GetRepository(reflect.TypeOf((*repository.AccountRepository)(nil)).Elem())
    if err != nil {
        return nil, err
    }
    return repoAny.(repository.AccountRepository), nil
}

func (u *UoW) TransactionRepository() (repository.TransactionRepository, error) {
    repoAny, err := u.GetRepository(reflect.TypeOf((*repository.TransactionRepository)(nil)).Elem())
    if err != nil {
        return nil, err
    }
    return repoAny.(repository.TransactionRepository), nil
}

func (u *UoW) UserRepository() (repository.UserRepository, error) {
    repoAny, err := u.GetRepository(reflect.TypeOf((*repository.UserRepository)(nil)).Elem())
    if err != nil {
        return nil, err
    }
    return repoAny.(repository.UserRepository), nil
}
```

## 🚀 Migration Notes

### Before (Current Pattern)

```go
func (s *AccountService) Deposit(userID, accountID uuid.UUID, amount float64, currencyCode currency.Code) error {
    return s.uow.Do(context.Background(), func(uow repository.UnitOfWork) error {
        // Complex reflect-based repository access
        repoAny, err := uow.GetRepository(reflect.TypeOf((*repository.AccountRepository)(nil)).Elem())
        if err != nil {
            return err
        }
        accountRepo := repoAny.(repository.AccountRepository)

        txRepoAny, err := uow.GetRepository(reflect.TypeOf((*repository.TransactionRepository)(nil)).Elem())
        if err != nil {
            return err
        }
        txRepo := txRepoAny.(repository.TransactionRepository)

        // Business logic...
        return nil
    })
}
```

### After (Improved Pattern)

```go
func (s *AccountService) Deposit(userID, accountID uuid.UUID, amount float64, currencyCode currency.Code) error {
    return s.uow.Do(context.Background(), func(uow repository.UnitOfWork) error {
        // Type-safe repository access - no reflect needed!
        accountRepo, err := uow.AccountRepository()
        if err != nil {
            return err
        }

        txRepo, err := uow.TransactionRepository()
        if err != nil {
            return err
        }

        // Business logic...
        return nil
    })
}
```

## Alternative Approaches Considered

### 1. String-Based Repository Names

```go
// Alternative: String-based approach
type StringBasedUnitOfWork interface {
    GetRepository(repoName string) (any, error)
}

// Usage
accountRepoAny, err := uow.GetRepository("account")
accountRepo := accountRepoAny.(repository.AccountRepository)
```

**Pros:**

- ✅ Simpler API
- ✅ More readable
- ✅ No reflect in service code

**Cons:**

- ❌ Runtime errors (typos)
- ❌ No IDE support
- ❌ No compile-time safety

### 2. Generic Repositories

```go
// Alternative: Generic repositories
type GenericRepository[T any] interface {
    Get(ctx context.Context, id uuid.UUID) (*T, error)
    Create(ctx context.Context, entity *T) error
    Update(ctx context.Context, entity *T) error
    Delete(ctx context.Context, id uuid.UUID) error
}

type GenericUnitOfWork interface {
    AccountRepository() GenericRepository[account.Account]
    TransactionRepository() GenericRepository[account.Transaction]
}
```

**Pros:**

- ✅ Full type safety
- ✅ No reflect needed
- ✅ Excellent IDE support

**Cons:**

- ❌ More complex implementation
- ❌ Requires significant refactoring
- ❌ May not fit existing patterns

## Recommended Migration Strategy

### Phase 1: Add Type-Safe Methods (✅ Complete)

1. ✅ Add convenience methods to `UnitOfWork` interface
2. ✅ Implement methods in `UoW` struct
3. ✅ Maintain backward compatibility

### Phase 2: Update Service Code (Recommended)

Gradually update service methods to use the new convenience methods:

```go
// Example: Update account service
func (s *AccountService) executeOperation(req operationRequest, handler operationHandler) (result *operationResult, err error) {
    err = s.uow.Do(context.Background(), func(uow repository.UnitOfWork) error {
        // NEW: Use type-safe methods
        accountRepo, err := uow.AccountRepository()
        if err != nil {
            return err
        }

        txRepo, err := uow.TransactionRepository()
        if err != nil {
            return err
        }

        // OLD: Remove reflect-based code
        // repoAny, err := uow.GetRepository(reflect.TypeOf((*repository.AccountRepository)(nil)).Elem())
        // accountRepo := repoAny.(repository.AccountRepository)

        // Business logic remains the same...
        return nil
    })
    return result, err
}
```

### Phase 3: Consider Future Enhancements

1. **Generic repositories** - For new services
2. **String-based approach** - For dynamic repository loading
3. **Hybrid approach** - Combine multiple patterns

## 💡 Benefits

### ✅ **Developer Experience**

1. **Type Safety** - Compile-time error checking
2. **IDE Support** - Autocomplete and refactoring
3. **Readability** - Clean, self-documenting code
4. **Maintainability** - Easier to understand and modify

### ✅ **Transaction Safety**

1. **All existing benefits preserved** - No changes to transaction handling
2. **Same atomicity guarantees** - All-or-nothing operations
3. **Same error handling** - Automatic rollback on errors
4. **Same repository coordination** - All repositories use same session

### ✅ **Backward Compatibility**

1. **Existing code continues to work** - `GetRepository()` method still available
2. **Gradual migration** - Update services one by one
3. **No breaking changes** - Same interface, additional methods

## Code Examples

### Complete Service Example

```go
type AccountService struct {
    uow       repository.UnitOfWork
    converter mon.CurrencyConverter
    logger    *slog.Logger
}

func (s *AccountService) Deposit(userID, accountID uuid.UUID, amount float64, currencyCode currency.Code) error {
    logger := s.logger.With("userID", userID, "accountID", accountID, "amount", amount, "currency", currencyCode)
    logger.Info("Deposit started")

    var tx *account.Transaction
    var convInfo *common.ConversionInfo

    err := s.uow.Do(context.Background(), func(uow repository.UnitOfWork) error {
        // Type-safe repository access
        accountRepo, err := uow.AccountRepository()
        if err != nil {
            logger.Error("Failed to get account repository", "error", err)
            return err
        }

        txRepo, err := uow.TransactionRepository()
        if err != nil {
            logger.Error("Failed to get transaction repository", "error", err)
            return err
        }

        // Get account
        acc, err := accountRepo.Get(accountID)
        if err != nil {
            logger.Error("Account not found", "error", err)
            return account.ErrAccountNotFound
        }

        // Business logic...
        tx, err = acc.Deposit(userID, money)
        if err != nil {
            return err
        }

        // Update account and create transaction
        if err = accountRepo.Update(acc); err != nil {
            return err
        }
        if err = txRepo.Create(tx); err != nil {
            return err
        }

        return nil
    })

    if err != nil {
        logger.Error("Deposit failed", "error", err)
        return err
    }

    logger.Info("Deposit successful", "transactionID", tx.ID)
    return nil
}
```

### Testing Example

```go
func TestAccountService_Deposit(t *testing.T) {
    // Mock UOW with type-safe methods
    mockUOW := &MockUnitOfWork{
        DoFunc: func(ctx context.Context, fn func(repository.UnitOfWork) error) error {
            return fn(mockUOW)
        },
        AccountRepositoryFunc: func() (repository.AccountRepository, error) {
            return mockAccountRepo, nil
        },
        TransactionRepositoryFunc: func() (repository.TransactionRepository, error) {
            return mockTransactionRepo, nil
        },
    }

    service := NewAccountService(mockUOW, converter, logger)

    // Test implementation...
}
```

## Conclusion

The improved UOW pattern provides:

1. **✅ All existing transaction benefits** - No compromise on data integrity
2. **✅ Better developer experience** - Type safety and IDE support
3. **✅ Backward compatibility** - Existing code continues to work
4. **✅ Gradual migration path** - Update services incrementally

**Recommendation:** Use the type-safe convenience methods for new code and gradually migrate existing services. The current UOW pattern is excellent - we've just made it even better! 🎉
