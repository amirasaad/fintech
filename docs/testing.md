---
icon: material/test-tube
---

# 🧪 Testing Guide

Comprehensive testing is crucial for ensuring the reliability and correctness of our financial application. This guide covers our testing strategy, tools, and best practices.

## 🎯 Testing Strategy

### Unit Tests

- Test individual functions and methods in isolation
- Located alongside the code they test (e.g., `_test.go` files)
- Focus on pure business logic and domain rules

### Integration Tests

- Test interactions between components
- Use in-memory or test database instances
- Verify event publishing/subscribing behavior

### End-to-End Tests

- Test complete user flows
- Use test containers for external services
- Verify system behavior from API to database

## 🛠️ Test Suite Execution

### Run All Tests

```bash
go test -v ./...
```

### Run Tests with Race Detector

```bash
go test -race ./...
```

### Run Tests in a Specific Package

```bash
cd pkg/account
go test -v
```

## 📈 Code Coverage

### Generate Coverage Report

```bash
make cov_report
```

This will:

1. Run all tests with coverage
2. Generate an HTML report at `docs/coverage.html`
3. Show coverage percentage in the terminal

## 🔄 Event-Driven Testing

### Testing Event Handlers

```go
func TestDepositHandler(t *testing.T) {
    // Setup test dependencies
    bus := NewInMemoryEventBus()
    repo := NewMockAccountRepository()
    handler := NewDepositHandler(repo, bus)

    // Register handler
    bus.Subscribe("Deposit.Requested", handler.Handle)

    // Publish test event
    event := DepositRequestedEvent{
        Amount:   100,
        AccountID: "acc123",
    }
    bus.Emit(context.Background(), event)

    // Verify state changes
    acc, _ := repo.FindByID("acc123")
    assert.Equal(t, 100, acc.Balance)
}
```

### Testing Event Flows

```go
func TestDepositFlow(t *testing.T) {
    // Setup test environment
    container := testutils.NewTestContainer(t)
    defer container.Cleanup()

    // Execute API request
    resp, err := http.Post(
        container.Server.URL + "/deposit",
        "application/json",
        strings.NewReader(`{"amount": 100, "account_id": "acc123"}`),
    )
    require.NoError(t, err)
    require.Equal(t, http.StatusAccepted, resp.StatusCode)

    // Simulate webhook callback
    webhookResp, err := http.Post(
        container.Server.URL + "/webhooks/payment",
        "application/json",
        strings.NewReader(`{"event": "payment.completed", "amount": 100, "account_id": "acc123"}`),
    )
    require.NoError(t, err)
    require.Equal(t, http.StatusOK, webhookResp.StatusCode)

    // Verify final state
    acc, err := container.AccountRepo.FindByID("acc123")
    require.NoError(t, err)
    assert.Equal(t, 100, acc.Balance)
}
```

## 🧪 Test Doubles

### Mocks

Use Mockery to generate mocks for interfaces:

```bash
mockery --name=AccountRepository --dir=pkg/domain --output=pkg/domain/mocks
```

### Test Containers

Use test containers for integration testing:

```go
func TestMain(m *testing.M) {
    container, err := testcontainers.StartPostgresContainer()
    if err != nil {
        log.Fatal(err)
    }
    defer container.Terminate()

    os.Exit(m.Run())
}
```

## 🔍 Test Data Management

### Fixtures

```go
func createTestAccount(t *testing.T, repo AccountRepository) *Account {
    acc := &Account{
        ID:      "test-account",
        Balance: 1000,
        Status:  "active",
    }
    err := repo.Save(acc)
    require.NoError(t, err)
    return acc
}
```

### Test Helpers

```go
func mustParseTime(t *testing.T, value string) time.Time {
    tm, err := time.Parse(time.RFC3339, value)
    require.NoError(t, err)
    return tm
}
```

## 🚀 Continuous Integration

Our CI pipeline runs:

1. Unit tests with race detection
2. Integration tests with test containers
3. Linting and static analysis
4. Code coverage reporting

## 📝 Best Practices

1. **Isolate Tests**: Each test should be independent
2. **Use Table Tests**: For testing multiple scenarios
3. **Test Edge Cases**: Zero values, nil checks, error conditions
4. **Benchmark Critical Paths**: Use Go's built-in benchmarking
5. **Keep Tests Fast**: Use mocks for slow dependencies
6. **Test Error Cases**: Ensure proper error handling
7. **Verify State and Behavior**: Check both state changes and interactions

## 🔗 Related Documentation

- [Go Testing Package](https://pkg.go.dev/testing)
- [Testify](https://github.com/stretchr/testify)
- [Testcontainers Go](https://golang.testcontainers.org/)
- [Mockery](https://github.com/vektra/mockery)
