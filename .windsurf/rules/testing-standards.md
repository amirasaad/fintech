---
trigger: model_decision
description: Testing standards and TDD approach for the fintech project
globs: *_test.go
---

# Testing Standards (TDD)

## Test-Driven Development Approach

1. Write failing tests first
2. Implement minimum code to pass tests
3. Refactor while maintaining test coverage

## Test Conventions

- Place tests alongside code (`_test.go` files)
- Use descriptive test names
- Test both success and error cases
- Aim for high test coverage, especially for business logic
- Use table-driven tests for multiple scenarios
- Mock external dependencies

## Test Structure

- Unit tests for all packages
- Benchmark tests for performance-critical code
- Integration tests for API endpoints
- Use test utilities from [webapi/test_utils.go](mdc:webapi/test_utils.go)

## Test Examples

- Service tests: [pkg/service/account_test.go](mdc:pkg/service/account_test.go)
- API tests: [webapi/account_test.go](mdc:webapi/account_test.go)
- Repository tests: [infra/repository_test.go](mdc:infra/repository_test.go)
- Benchmark tests: [pkg/service/account_benchmark_test.go](mdc:pkg/service/account_benchmark_test.go)

## Testing Best Practices

- Use table-driven tests for multiple scenarios
- Mock external dependencies (database, external APIs)
- Test error conditions and edge cases
- Use meaningful test data and fixtures
- Keep tests focused and readable
- Use subtests for better organization

## Test Utilities

- Use the test utilities in [webapi/test_utils.go](mdc:webapi/test_utils.go)
- Follow the patterns established in existing test files
- Use proper setup and teardown for integration tests
