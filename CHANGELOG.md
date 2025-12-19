# CHANGELOG

## v1.5.2 (2025-12-19)

### ♻️ Refactorings

- enhance logging and repository access in account handlers

### ♻️  refactor

- improve idempotency handling with singleflight

### ✅🤡🧪 Tests

- add idempotency tracking tests for event handlers

### 💚👷 CI & Build

- add missing dependency for commitizen-branch hook

## v1.5.2 (2025-12-19)

### docs

- synchronize event names, API endpoints, and file paths across documentation

### 🧹 chore

- migrate cz to use cz-gitmoji adapter

## v1.5.2 (2025-12-06)

### Feat

- **repository**: implement GORM error mapping and wrapping for domain errors
- **payment**: implement idempotency checks for PaymentCompleted and PaymentProcessed events
- **stripepayment**: enhance Stripe client configuration with TLS skip option for development
- **eventbus**: enhance DLQ retry mechanism with configurable backoff and max retries
- **docs**: add comprehensive guidelines for API development, architecture patterns, coding standards, security, and testing

### Fix

- **eventbus**: DLQ retry mechanism and add tests (#25)

### Refactor

- **payment**: enhance logging by safely handling nil payment IDs
- **payment**: simplify idempotency tracking and transaction lookup logic
- **payment**: streamline transaction lookup and idempotency checks

## v1.4.0 (2025-08-31)

### Feat

- implement Stripe Connect integration (#24)
- **payment**: add payout support and Stripe Connect integration  (#23)

### Fix

- **eventbus**: DLQ retry mechanism and add tests

## v1.3.0 (2025-08-21)

### Feat

- 💳 Implement payment processing (#20)

### Refactor

- **service**: ♻️ enhance exchange rate service with improved configuration and error handling (#21)

## v1.2.0 (2025-08-09)

### Feat

- **checkout**: add checkout service with session management and webapi integration
- **auth**: add support for basic auth strategy
- **api**: initialize currency and checkout registries with Redis support
- implement payment processing with fees and currency conversion
- **refactor**: Decompose monolith repository and adopt CQRS

### Fix

- **user**: handle partial updates in user repository

### Refactor

- **testutils**: add checkout registry provider to app deps
- ♻️ reorganize code structure and improve error handling
- **app**: convert Deps to pointer type and extract initialization logic
- **app**: centralize app initialization and dependencies
- **cmd/server**: restructure main.go into modular initialization functions
- move config package to pkg/config and update imports
- **webhooks**: Remove duplicate Stripe webhook logic

## v1.1.0 (2025-08-05)

### BREAKING CHANGE

- Deposit API and service now require a money_source argument

### Feat

- add Stripe webhook handler and improve error handling
- **checkout**: add session service for managing checkout sessions
- **payment**: add PaymentProcessed event type and set paymentID
- **stripe**: add checkout session and webhook handling
- **eventbus**: add redis based eventbus and refactor events streaming
- 🎯 implement Stripe webhook integration with event-driven payment processing (#18)
- stripe integration (#17)
- **core**: ✨ event-driven payments, strict balance invariants, and handler refactor (#15)
- **account**: 🎉 add extensible money source, external target masking, and refactor account operations (#14)
- **currency**: finalize robust multi-currency conversion with domain-driven precision and schema alignment (#11)
- **infra**: add Redis support for exchange rate caching
- **currency**: add USD and EUR constants
feat(common): add common types and errors
feat(money): implement money value object with currency support
feat(account): implement account and transaction with builder pattern
feat(user): implement user domain model
- **api**: add swagger documentation for all endpoints (#6)
- add cli interface  (#4)

### Fix

- **mkdocs**: remove python tags from mkdocs.yml
- **transfer**: update persistence handler tests to match event emission behavior
- add idempotency check for conversion persistence handler
- prevent DepositConversionDoneEvent emission to avoid event cycle
- **webapi**: validate currency codes in account operations
- **money**: replace custom error with common.ErrInvalidDecimalPlaces
Use common error type for invalid decimal places validation in money conversion.
- **account**: update account creation to use builder pattern
- host swagger.yaml
- base url swagger.json
- base url for swagger
- **cli**: use parsed account ID for balance check
- **account**: remove logging from Deposit method
- **account**: add rollback when account not found in Deposit
- Update Go version, fix OpenAPI schema, and refactor error handling (#2)

### Refactor

- update Stripe webhook handler to use event bus
- ⚰️ remove dead code
- **registry**: rename registry types and functions for consistency
- **currency**: use maps.Copy for metadata copying
- **payment**: improve payment provider interface and stripe implementation
- **payment**: simplify handler function signatures to use eventbus.HandlerFunc
- **events**: standardize event types and handlers
- **eventbus**: migrate event type definitions to pkg/domain/events/types.go and update event bus implementations
- update event handling and fix type safety issues
- **event-bus**: ♻️ enhance MemoryRegistryEventBus with logger and improve event processing
- ♻️ update service dependencies to use event bus and unit of work for improved architecture
- **event-driven**: ♻️ major event handler and flow refactor: simplify, modularize, and improve test coverage (#19)
- deposit event flow with simplified handlers and idempotency
- payment initiation to separate deposit and withdraw handlers
- Account Service Refactoring and Test Organization Improvements (#13)
- reorganize codebase structure and extract DTOs to separate files (#12)
- **auth**: improve unauthorized error handling
- Return domain.ErrUserUnauthorized for invalid credentials
- Update webapi to handle unauthorized errors properly
- **currency**: update entity interface and validation logic
- **domain**: use currency.Code type for money and account currency fields
- **service**: add logging and improve currency conversion handling
- **exchange**: update exchange rate service to use last update tracking
Update exchange rate service to check last update timestamp before fetching rates and mask API keys in logs.
- add nolint directives for unchecked errors in examples
- **webapi**: replace uowFactory with service instances in routes (#8)
- **tests**: migrate test suites to testify suite pattern (#7)
- **build**: move coverage report to docs directory
- **test**: replace custom mocks with shared test mocks in service tests

## v1.0.1 (2025-06-30)

### Refactor

- add service layer (#1)

## v1.0.0 (2025-06-23)

### Feat

- **transaction**: add balance field to track account state
- **account**: add thread-safe handling for transactions
- **account**: add more logging to account handler routes
- **domain**: implement account domain logic with tests
- **account**: add endpoint to get account balance
- **transactions**: add transaction listing endpoint and timestamps
- **database**: add model migrations and transaction repository
- implement account management with database integration
- **account**: add timestamp fields for update tracking

### Fix

- **tests**: add nolint directive for resp.Body.Close to suppress checkerr warnings
- **uow**: add recovery mechanism in transaction Begin method to handle panics
- **account**: handle errors during transaction initialization and improve test clarity
- handle error when starting the server
- **account**: standardize error messages and improve test assertions
- **account**: handle deposit errors and improve test coverage
- **repository**: change transaction list order
- **account**: replace max int check with balance overflow validation
- **account**: prevent deposit overflow by checking max safe integer value

### Refactor

- **account**: replace fmt.Println with slog for logging in Deposit, Withdraw, and GetBalance methods
- **pkg**: add pkg to include core app logic
- **database**: implement unit of work pattern for transaction management
- **infra**: migrate from sqlite to postgres and reorganize database layer
- **repository**: add ordering and limit to transaction list query
- **account**: rename transaction variable to tx for consistency
- **handler**: move account logic to domain package
docs: update README with docker and endpoint details
test: update tests to use domain package
- **account**: add account pkg under internal
- **account**: rename New to NewAccount and fix formatting
- **web**: move server.go to web/main.go
- restructure account management code and tests into internal package
- account balance assertions in tests to use GetBalance method for consistency
