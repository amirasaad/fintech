---
icon: material/lightbulb
---

# Refactoring Patterns

## 💡 Overview

This document summarizes the analysis and implementation of various design patterns for refactoring the account service in the fintech application. The goal was to reduce branching complexity and improve code organization in the `Deposit` and `Withdraw` methods.

## 📦 Reference: pkg/registry

The [pkg/registry](https://pkg.go.dev/github.com/amirasaad/fintech/pkg/registry) package provides a **flexible, extensible registry system** for managing entities (users, accounts, currencies, etc.) with support for:

- **Abstractions:**
  - `Entity` interface (property-style getters: `ID()`, `Name()`, `Active()`, `Metadata()`, etc.)
  - `RegistryProvider` interface for CRUD, search, metadata, and lifecycle operations
  - Observer/event bus, validation, caching, persistence, metrics, and health interfaces

- **Patterns & Architecture:**
  - Clean separation of interface and implementation layers
  - Builder and factory patterns for registry construction
  - Event-driven and observer patterns for entity lifecycle events
  - Caching and persistence strategies (in-memory, file-based, etc.)

- **Usage Examples:**
  - Register, retrieve, update, and unregister entities
  - Use memory cache, file persistence, and custom validation
  - Event-driven hooks for entity changes

- **Best Practices:**
  - Use property-style getters for all entities (e.g., `Name()`, not `GetName()`)
  - Prefer registry interfaces for dependency inversion and testability
  - Leverage event bus and observer for decoupled side effects (metrics, logging, etc.)
  - Use the builder for complex configuration (caching, validation, persistence)

### 🧪 Example: Registering an Entity

```go
user := registry.NewBaseEntity("user-1", "John Doe")
user.Metadata()["email"] = "john@example.com"
err := registry.Register(ctx, user)
```

### 🧪 Example: Custom Registry with Caching & Persistence

```go
reg, err := registry.NewRegistryBuilder().
    WithName("prod-reg").
    WithCache(1000, 10*time.Minute).
    WithPersistence("/data/entities.json", 30*time.Second).
    BuildRegistry()
```

!!! tip "Why use the registry?"
    The registry pattern centralizes entity management, supports extensibility (events, validation, caching), and enforces clean architecture boundaries.

**See also:**
- [`pkg/registry/README.md`](https://github.com/amirasaad/fintech/blob/main/pkg/registry/README.md) for full documentation
- [`pkg/registry/interface.go`](https://github.com/amirasaad/fintech/blob/main/pkg/registry/interface.go) for all abstractions
- [`pkg/registry/examples_test.go`](https://github.com/amirasaad/fintech/blob/main/pkg/registry/examples_test.go) for usage patterns

---

## ⚠️ Initial Problem

- Significant code duplication (~150 lines of nearly identical logic)
- Complex branching around currency conversion and transaction handling
- Mixed responsibilities (validation, conversion, persistence, logging)
- Poor maintainability due to tightly coupled logic

---

## 🛠️ Strategy Pattern

**Approach:**

- Extract common operation logic into a shared method using the strategy pattern for operation type.
- Use an `operationHandler` interface and concrete strategies for deposit/withdraw.

**Key Code:**

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
```

**When to Use:**

- You have similar operations (deposit/withdraw) with shared logic but different details.

**Benefits:**

| Pattern | Branching | Extensibility | Testability | Complexity | Go Idiomatic |
|---------|-----------|---------------|-------------|------------|--------------|
| **Strategy** | Low | Good | Good | Medium | ✅ |
| **Command** | None | Excellent | Excellent | High | ⚠️ |
| **Chain of Responsibility** | None | Excellent | Excellent | Medium | ✅ |
| **Event-Driven** | None | Excellent | Good | High | ⚠️ |

---

## 🧰 Implementation Status

- ✅ **Strategy Pattern**:  Implemented and fully discarded
- ✅ **Chain of Responsibility**: Implemented
- 📋 **Command Pattern**: Analyzed, ready for implementation if needed
- ✅ **Event-Driven**: Implemented

---

## 🧪 Code Quality Metrics

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

---

## 🏅 Recommendations

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

---

## 🔮 Conclusion

The refactoring journey demonstrates how different design patterns can address the same problem with varying trade-offs. The **Strategy Pattern** provided immediate benefits, while **Chain of Responsibility** offers the best long-term solution for this specific use case.

The key insight is that **pattern selection should be driven by specific requirements** rather than following a one-size-fits-all approach. For fintech applications requiring high reliability and maintainability, the Chain of Responsibility pattern provides the optimal balance of simplicity, extensibility, and Go idiomaticity.
