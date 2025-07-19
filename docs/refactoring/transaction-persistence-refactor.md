---
icon: material/database
---

# :material-database: Transaction Persistence Refactor

## Problem Statement

!!! warning "Why Refactor?"
    As our event-driven payment flow has matured, we have encountered several architectural and maintainability issues with how transaction persistence is currently handled. These issues are impeding our ability to extend, test, and reason about the system as it grows.

!!! danger "Pain Points"
    - **Update vs. Create:**
        - The current persistence logic sometimes creates new transactions at multiple points in the event chain (e.g., after conversion), rather than updating the original transaction created at the start of the flow.
        - This leads to duplicate records, broken audit trails, and confusion about the true lifecycle of a transaction.
    - **Repository Rigidity:**
        - The `TransactionRepository` interface is too rigid, supporting only basic `Create` and `Update` methods.
        - There is no support for partial updates, upserts, or updating by business keys (e.g., event ID, payment ID).
        - This makes it hard to evolve the event-driven flow and add new business requirements (e.g., updating payment ID after initiation, or conversion info after currency conversion).
    - **Domain Model Pollution:**
        - Infrastructure-specific fields (e.g., `external_target`, `money_source`) are being added to the domain `Transaction` struct, rather than being kept in the infrastructure persistence model.
        - This violates clean architecture principles and makes the domain model harder to reason about and test.
    - **Event Handler Complexity:**
        - Handlers use switch-cases or type assertions to handle multiple event types, rather than being single-responsibility.
        - This increases coupling and reduces clarity.

## Why This Matters

- **Auditability:**
  - Financial systems require a clear, auditable trail of all transaction state changes. Duplicates or missing updates undermine trust and compliance.
- **Extensibility:**
  - As we add new payment methods, currencies, and business rules, we need a flexible persistence layer that can evolve without breaking existing flows.
- **Separation of Concerns:**
  - Clean separation between domain logic and infrastructure is essential for maintainability, testability, and onboarding new developers.

## Goals for the Refactor

- **Single Source of Truth:**
  - Each transaction should be created once (at the start of the flow) and updated in-place as its state evolves.
- **Flexible Repository Interface:**
  - Support for partial updates, upserts, and updates by business keys (event ID, payment ID, etc.).
- **Domain/Infra Separation:**
  - Keep the domain model pure; use separate persistence models for the database layer.
- **Handler Simplicity:**
  - Each event handler should have a single responsibility: either create or update, never both.
- **Auditability and Traceability:**
  - Every state change should be recorded and attributable to a specific event in the flow.

!!! note "Next Steps"
    - Design new repository interfaces and persistence models.
    - Refactor event handlers to update transactions in-place.
    - Update migrations and DTOs as needed.
    - Document the new flow and migration plan.

---
