---
icon: material/test-tube
---

# Testing

Comprehensive testing is crucial for ensuring the reliability and correctness of a financial application. This project includes a robust testing suite.

## 🎯 Unit Tests

- Located alongside the code they test (e.g., `_test.go` files)
- Verify the functionality of individual components in isolation

## 🧪 Test Suite Execution

To run all tests:

```bash
go test -v ./...
```

The `-v` flag provides verbose output.

## 📈 Code Coverage

To generate a code coverage report:

```bash
make cov_report
```

This runs tests with coverage and generates an HTML report at `docs/coverage.html`.

## 🧪 Event-Driven Testing

- All tests for deposit, withdrawal, and transfer flows use the event-driven flow and interact with the new service interfaces.
- Tests should simulate webhook callbacks to confirm payment completion and verify balance updates.

## 💡 Notes

- All new features and changes must follow the event-driven, invariant-enforcing architecture.
- Business rules must be enforced in the domain layer, and all payment/transaction flows should use the event bus and webhook-driven patterns.
