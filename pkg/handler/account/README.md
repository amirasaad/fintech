# :octicons-folder: Account Handler Package Structure

This package organizes account operation handlers by operation and shared logic, following Go conventions.

## Structure

- `deposit/`   — Deposit operation handlers, validation, persistence, mapping, and tests
- `withdraw/`  — Withdraw operation handlers, validation, persistence, mapping, and tests
- `transfer/`  — Transfer operation handlers, validation, persistence, mapping, and tests
- `common/`    — Shared helpers, queries, validation, and test utilities used by multiple operations
- `types.go`   — Shared types/interfaces (if needed)

## Go Test Convention
- Each file (e.g., `handler.go`) has its test file (`handler_test.go`) in the same folder.
- No separate test/ subfolders—tests live with the code they test.

## Example
```
deposit/
  handler.go
  handler_test.go
  validation.go
  validation_test.go
  ...
common/
  account_query_handler.go
  account_query_handler_test.go
  ...
```

---

This structure keeps each operation focused, maintainable, and easy to navigate. Shared logic is DRY and discoverable in `common/`.
