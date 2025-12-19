# WARP.md

This file provides guidance to WARP (warp.dev) when working with code in this repository.

## Project Overview

A Go-based fintech platform using event-driven architecture for payment processing, account management, and currency conversion. The project follows clean architecture principles with domain-driven design (DDD).

**Tech Stack:**

- Go 1.24.4
- Fiber (web framework)
- GORM (ORM with PostgreSQL)
- JWT authentication
- Stripe payment integration
- Redis (event bus)
- Docker & Docker Compose

## Essential Commands

### Development

```bash
# Run the application
make run
# Or directly:
go run cmd/server/main.go

# Run with hot reload (using Air)
air

# Build the application
go build ./...
```

### Testing

```bash
# Run all tests (excludes internal, api, cmd directories)
make test

# Run unit tests only (pkg/ directory)
make test-unit

# Run integration tests only (infra/ directory)
make test-integration

# Run E2E tests only (webapi/ directory)
make test-e2e

# Generate coverage report
make cov
make cov_report  # Opens HTML coverage in docs/coverage.html
```

### Linting & Formatting

```bash
# Run linter
make lint

# Format code
go fmt ./...

# Tidy dependencies
go mod tidy
```

### Database Migrations

```bash
# Create a new migration
make migrate-create name=<migration_name>

# Apply migrations
make migrate-up

# Rollback migrations
make migrate-down n=<number_of_steps>

# Force migration version (use with caution)
make migrate-force n=<version>
```

### Docker

```bash
# Start all services (app + database)
docker compose up --build -d

# Start only database
docker compose up db -d

# View logs
docker compose logs -f

# Stop all services
docker compose down
```

## Architecture & Code Structure

### Layered Architecture

The codebase follows strict layering:

1. **Domain Layer** (`pkg/domain/`) - Core business entities and interfaces
   - Pure business logic, no infrastructure dependencies
   - Defines repository interfaces
   - Contains domain events and errors

2. **Service Layer** (`pkg/service/`) - Business logic orchestration
   - Coordinates between repositories
   - Implements business rules
   - Uses Unit of Work for transactions

3. **Repository Layer** (`pkg/repository/` and `infra/repository/`) - Data access
   - Repository interfaces defined in `pkg/repository/`
   - Repository implementations in `infra/repository/`
   - Uses GORM for database operations

4. **Infrastructure Layer** (`infra/`) - Database configuration and models
   - GORM models and database setup
   - Connection management
   - Migration configuration

5. **Web API Layer** (`webapi/`) - HTTP handlers and API endpoints
   - RESTful API implementation
   - Request validation
   - Authentication middleware

6. **Event Handlers** (`pkg/handler/`) - Event-driven business flows
   - Organized by domain: `account/`, `payment/`, `conversion/`, `fees/`
   - Each handler is responsible for a single event type
   - Handlers emit subsequent events to form event chains

### Event-Driven Architecture

This platform is built on event-driven patterns:

**Core Principle:** All business flows (deposit, withdraw, transfer) are modeled as event chains. Each handler processes one event and emits the next.

**Event Flows:**

- **Deposit:** `Requested` → `CurrencyConverted` → `Validated` → `Payment.Initiated`
- **Withdraw:** `Requested` → `CurrencyConverted` → `Validated` → `Payment.Initiated`
- **Transfer:** `Requested` → `Validated` → `CurrencyConverted` → `Completed`

**Key Components:**

- Event bus initialization in `infra/initializer/`
- Event bus registration in `pkg/app/setup_eventbus.go`
- Event handlers in `pkg/handler/<domain>/`
- Event definitions in `pkg/domain/events/`
- Common `FlowEvent` struct for shared fields (UserID, AccountID, CorrelationID, FlowType)

**Design Pattern:**

- Handlers register for specific event types (no central switch/if logic)
- Each handler has Single Responsibility
- Cycle detection via `scripts/event_cycle_check.go` (runs in pre-commit)

### Unit of Work Pattern

All transactional operations use the Unit of Work (UoW) pattern:

```go
// Automatic error mapping in UoW.Do()
uow.Do(ctx, func(uow repository.UnitOfWork) error {
    accountRepo, _ := uow.AccountRepository()
    // GORM errors are automatically mapped to domain errors
    return accountRepo.Create(account)
})
```

**Benefits:**

- Ensures atomic transactions
- Automatic GORM → domain error mapping
- Consistent error handling across the codebase

### Error Handling

**Two-layer error translation:**

1. **GORM Layer:** `TranslateError: true` normalizes database-specific errors to GORM errors
2. **Domain Layer:** `MapGormErrorToDomain()` converts GORM errors to domain errors

**Mappings:**

- `gorm.ErrDuplicatedKey` → `domain.ErrAlreadyExists` → HTTP 422
- `gorm.ErrRecordNotFound` → `domain.ErrNotFound` → HTTP 404

**All errors in `UoW.Do()` are automatically mapped.** For non-transactional operations, use `WrapError()` helper.

### Directory Structure

```
cmd/                    # Application entry points
├── server/            # HTTP server
└── cli/               # CLI tools

pkg/                    # Main application code
├── domain/            # Domain entities, interfaces, events
├── service/           # Business logic services
├── repository/        # Repository implementations
├── handler/           # Event handlers (account, payment, conversion, fees)
├── middleware/        # HTTP middleware (auth, logging, etc.)
├── provider/          # External service integrations (Stripe)
├── currency/          # Currency and money types
├── exchange/          # Exchange rate services
├── eventbus/          # Event bus implementation
├── config/            # Configuration management
└── utils/             # Shared utilities

webapi/                 # HTTP API layer
├── account/           # Account endpoints
├── auth/              # Authentication endpoints
├── user/              # User endpoints
├── payment/           # Payment endpoints
├── checkout/          # Checkout endpoints
└── currency/          # Currency endpoints

infra/                  # Infrastructure layer
├── initializer/       # Dependency injection setup
├── repository/        # Repository base implementations
└── model.go           # GORM models

internal/               # Internal utilities
└── migrations/        # Database migrations

docs/                   # Documentation
```

## Coding Conventions

### Go Style

- **Indentation:** Use tabs (not spaces)
- **Naming:**
  - Exported functions: `CamelCase` (e.g., `CreateAccount`)
  - Unexported functions: `camelCase` (e.g., `validateInput`)
  - Package names: `lowercase`
- **Property-style getters:** Use `Name()`, not `GetName()`
- **Logging:** Use `slog/log` or `charmbracelet/log` for structured logging, never `fmt.Print`
- **Error handling:** Always check errors explicitly, never ignore them
- **Line length:** Maximum 100 characters (enforced by revive linter)

### Error Handling Rules

- Use `ErrorResponseJSON` for HTTP error responses (see `webapi/common/utils.go`)
- Return `fiber.Error` for API errors
- Log errors with context using structured logging
- For transactional operations, rely on automatic error mapping in `UoW.Do()`
- For non-transactional operations, use `WrapError()` or `MapGormErrorToDomain()` directly

### Event Handler Patterns

- Each handler is responsible for a single event type
- Handlers must be registered with the event bus
- Use structured, emoji-rich logging for traceability
- All event structs embed `FlowEvent` for common fields
- All IDs use `uuid.UUID`
- Handlers emit the next event in the flow chain

### Testing Standards (TDD)

**Test Organization:**

1. Write failing tests first
2. Implement minimum code to pass
3. Refactor while maintaining coverage

**Test Types:**

- Unit tests: `pkg/**/*_test.go`
- Integration tests: `infra/**/*_test.go`
- E2E tests: `webapi/**/*_test.go`

**Test Conventions:**

- Use table-driven tests for multiple scenarios
- Mock external dependencies (database, external APIs)
- Test both success and error cases
- Use descriptive test names
- Keep tests focused and readable
- Use test utilities from `webapi/testutils/` and `pkg/handler/testutils/`

### Security Guidelines

- JWT tokens for authentication (see `pkg/middleware/auth.go`)
- Use `go-playground/validator` for input validation
- Never log passwords, tokens, or sensitive data
- Use environment variables for configuration (never hardcode secrets)
- Parameterized queries via GORM (automatic protection)
- Implement rate limiting for public endpoints

## Pre-commit Hooks

The repository uses pre-commit hooks (`.pre-commit-config.yaml`):

- Trailing whitespace check
- End-of-file fixer
- Event cycle detection (`scripts/event_cycle_check.go`)
- `go vet`
- Unit tests (`go test ./pkg/...`)
- `gocritic check`
- `golangci-lint run`
- `go build`
- `go fmt`
- `go mod tidy`
- Commitizen for conventional commits

**To install:** Follow instructions in `.pre-commit-config.yaml`

## Environment Configuration

Copy `.env_sample` to `.env` and configure:

**Required:**

- `AUTH_JWT_SECRET` - JWT signing secret
- `DATABASE_URL` - PostgreSQL connection string

**Optional:**

- Stripe configuration (API keys, webhook secrets, redirect URLs)
- Redis configuration (for event bus)
- Server configuration (host, port, scheme)

## API Documentation

- OpenAPI specification: `docs/api/openapi.yaml`
- Swagger UI available when server is running
- Full documentation in `docs/` directory

## Important Files

- `ARCHITECTURE.md` - Detailed architecture documentation
- `CONTRIBUTING.md` - Contribution guidelines
- `docs/domain-events.md` - Event-driven architecture guide
- `docs/error-handling.md` - Complete error handling reference
- `docs/getting-started.md` - Setup and installation guide
- `docs/service-domain-communication.md` - Service/domain boundaries

## Common Development Patterns

### Creating a New Service

1. Define domain interface in `pkg/domain/<entity>.go`
2. Implement service in `pkg/service/<entity>.go`
3. Use dependency injection for repositories
4. Handle transactions with `UoW.Do()`
5. Add tests in `pkg/service/<entity>_test.go`

### Adding a New API Endpoint

1. Create handler in `webapi/<domain>/<handler>.go`
2. Use existing error utilities (`ErrorResponseJSON`)
3. Validate input with `go-playground/validator`
4. Apply auth middleware if needed
5. Add tests in `webapi/<domain>/<handler>_test.go`
6. Update OpenAPI spec in `docs/api/openapi.yaml`

### Creating a New Event Flow

1. Define event types in `pkg/domain/events/`
2. Create handler in `pkg/handler/<domain>/<handler>.go`
3. Register handler with event bus in `infra/initializer/`
4. Add E2E flow tests in `pkg/handler/event_flows_test.go`
5. Verify no cycles with `scripts/event_cycle_check.go`

### Adding a Repository Method

1. Define interface in `pkg/domain/<entity>.go`
2. Implement in `infra/repository/<entity>/`
3. Use `WrapError()` for error mapping if outside UoW
4. Add unit tests
5. Use GORM best practices (preloading, proper contexts)
