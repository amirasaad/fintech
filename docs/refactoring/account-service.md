---
icon: material/compare
---

# Account Service Refactor

Welcome to the Account Service Refactor guide! This page explains how we transformed our account service for clarity, maintainability, and extensibility. Whether you're a new contributor or a seasoned team member, you'll find actionable insights and a clear roadmap here.

## ⚠️ Problem Statement

!!! warning "Why Refactor?"
    The original `AccountService` had significant code duplication and branching in the `Deposit` and `Withdraw` methods. Both methods followed nearly identical patterns, making the codebase hard to maintain and extend.

**Issues included:**

- Repeated logic for repository access, validation, and persistence
- Complex, nested branching for currency conversion and transaction handling
- Mixed responsibilities (validation, conversion, persistence, logging)
- Difficult to test and reason about

---

## 🛠️ Refactoring Solution: Chain of Responsibility Pattern

!!! tip "The Big Idea"
    We adopted the **Chain of Responsibility** pattern, breaking the process into focused, testable handlers. Each handler does one thing well and passes control to the next.

**Key changes:**

- Each operation (deposit, withdraw, transfer) is now a chain of handlers
- Handlers are responsible for a single concern (validation, money creation, conversion, domain operation, persistence)
- The service method simply builds and invokes the chain

---

## 🏗️ Architecture

- **OperationHandler Interface:** Contract for all handlers
- **OperationRequest/Response:** Data structures for passing information
- **Specialized Handlers:** Each with a single responsibility
- **ChainBuilder:** Constructs the complete operation chain

---

## 🧩 Handler Chain Structure

```go
// Deposit: ValidationHandler → MoneyCreationHandler → CurrencyConversionHandler → DomainOperationHandler → DepositPersistenceHandler
// Withdraw: ... → WithdrawPersistenceHandler
```

---

## 📝 Migration Note

- The legacy `PersistenceHandler` is removed
- All persistence is now event-driven and operation-specific
- Transaction creation is DRY and unified

---

## 🧪 Example Usage

```go
// Deposit example
tx, convInfo, err := accountService.Deposit(userID, accountID, 100.0, currency.Code("EUR"))
// Withdraw example
tx, convInfo, err := accountService.Withdraw(userID, accountID, 50.0, currency.Code("USD"))
```

---

## ✨ Adding New Handlers

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
    return h.Next(ctx, req)
}
```

---

## 🔮 Looking Forward

This refactor sets the stage for even more powerful features and easier onboarding. If you have ideas for further improvements, or want to contribute a new handler, jump in!

Happy coding! 🎉
