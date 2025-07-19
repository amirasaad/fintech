---
icon: material/call-split
---
# Layered Dependency Injection & Dependency Direction in Go

## Overview

As the fintech project grows, managing dependencies in a scalable, maintainable way becomes critical. This document outlines best practices for **layered dependency injection** and **dependency direction**, so that future enhancements can be made with confidence and clarity.

---

## Why Layered Dependency Injection?

- **Expresses architectural intent:** Lower-level components (e.g., DB, UOW) are independent; higher-level components (e.g., EventBus, services) depend on them, not the other way around.
- **Prevents cyclic dependencies:** Each layer only depends on the layer below.
- **Improves testability:** You can mock or swap out dependencies at each layer.
- **Eases refactoring:** Clear boundaries make it easier to change or replace parts of the system.

---

## Best Practices

1. **Construct lower-level dependencies first** (e.g., DB, UOW).
2. **Pass them to mid-level dependencies** (e.g., repositories, services).
3. **Pass those to high-level dependencies** (e.g., EventBus, HTTP handlers).
4. **Each layer only depends on interfaces from the layer below.**
5. **No cyclic dependencies:** Lower-level code never imports or depends on higher-level code.

---

## Example: Layered Construction

```go
// 1. Infrastructure
infra := InfraDeps{
    DB:  NewDB(cfg),
    UOW: NewUnitOfWork(db),
}

// 2. Domain/Service
services := ServiceDeps{
    Infra:         infra,
    AccountRepo:   NewAccountRepository(infra.UOW),
    AccountService: NewAccountService(accountRepo, ...),
}

// 3. Event Bus
bus := NewSimpleEventBus(services.AccountService, ...)

// 4. App/HTTP
app := NewApp(bus, ...)
```

---

## Example: Layered Deps Structs

```go
type InfraDeps struct {
    DB  *sql.DB
    UOW UnitOfWork
}

type ServiceDeps struct {
    Infra InfraDeps
    AccountRepo AccountRepository
    AccountService AccountService
}

type AppDeps struct {
    Services ServiceDeps
    EventBus EventBus
    Fiber    *fiber.App
}
```

---

## Summary Table

| Layer         | Depends On         | Example Components         |
|---------------|--------------------|---------------------------|
| Infrastructure| -                  | DB, UOW                   |
| Domain/Service| Infrastructure     | Repos, Services           |
| Event Bus     | Service, Infra     | EventBus, Handlers        |
| App/API       | EventBus, Services | Fiber, HTTP Handlers      |

---

## When to Enhance

- As the project grows and the number of dependencies increases
- When you want to enforce clean architecture boundaries
- When you need to swap out or mock dependencies for testing

---

## Final Thought

Layered dependency injection and clear dependency direction are powerful tools for scaling Go projects. This document serves as a reference for future enhancements—when you’re ready, you can refactor your wiring to follow these patterns for even greater maintainability and clarity.
