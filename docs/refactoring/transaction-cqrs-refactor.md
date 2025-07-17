---
icon: material/source-branch
---

# :material-source-branch: Transaction CQRS Refactor

## 💡 Philosophy

!!! tip
    Patterns are solutions, not goals. Let real problems lead you to the right design pattern, not the other way around.

Adopting design patterns like CQRS should be a response to real, observed pain points in the system. This ensures architecture remains pragmatic, maintainable, and truly solves business needs, rather than introducing unnecessary complexity.

## Motivation

!!! info "Why CQRS?"
    As our transaction persistence and event-driven flows have grown, we've encountered increasing complexity in balancing domain purity, auditability, and query flexibility.

### Our current approach mixes domain models for both writes and reads, leading to:

- Leaky abstractions between domain and infrastructure
- Difficulty evolving read models for reporting/audit
- Inflexible API responses and DTOs

!!! danger "Pain Points"
    - **Domain Model Pollution:**
    - Read-specific fields and denormalized data are creeping into domain structs.
    - **Query Inflexibility:**
    - Hard to add computed/audit fields to API responses without polluting the domain.
    - **Audit & Reporting:**
    - No clear place for audit trails, event history, or reporting fields.
    - **Separation of Concerns:**
    - Handlers and services are forced to map between domain and API models manually.

## CQRS Overview

CQRS (Command Query Responsibility Segregation) separates write (command) and read (query) models:

- **Write Models:** Domain entities and command DTOs for create/update flows.
- **Read Models:** Read-optimized DTOs for queries, reporting, and API responses.

## Proposed Changes

- **Introduce `TransactionRead` DTO:**
  - A read-optimized struct for queries, API responses, and reporting.
- **Repository Interface Refactor:**
  - Query methods (`Get`, `ListByUser`, `ListByAccount`, etc.) return `TransactionRead` instead of domain `Transaction`.
  - Write methods (`Create`, `Update`, `PartialUpdate`, `Upsert`) continue to use domain models or command DTOs.
- **Handler/Service Refactor:**
  - Handlers/services use the appropriate model for each operation, reducing mapping boilerplate.
- **Documentation & Migration:**
  - Document new flow and migration plan for existing code.

## Benefits

- **Separation of Concerns:**
  - Domain model stays pure; read model evolves independently.
- **Auditability:**
  - Read DTOs can include audit/event history fields.
- **API Flexibility:**
  - Easier to shape API responses for frontend/reporting needs.
- **Maintainability:**
  - Reduces coupling and manual mapping in handlers/services.

## Next Steps

!!! note "Next Steps"
    - Define `TransactionRead` DTO and update repository interfaces.
    - Refactor query methods to return read DTOs.
    - Update handlers/services to use new models.
    - Document migration and update tests.

---
