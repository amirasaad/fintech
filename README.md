# Fintech App 🚀

**Note:** This project is a personal exploration of building a robust, event-driven financial application with Go. It is a work in progress and serves as a practical playground for applying modern software engineering principles.

[![build](https://github.com/amirasaad/fintech/actions/workflows/ci.yml/badge.svg)](https://github.com/amirasaad/fintech/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/amirasaad/fintech)](https://goreportcard.com/report/github.com/amirasaad/fintech)
[![codecov](https://codecov.io/github/amirasaad/fintech/graph/badge.svg?token=iuU1Fm5BwG)](https://codecov.io/github/amirasaad/fintech)

## Overview 📝

This Fintech App is a personal project exploring the development of a financial transaction system. Built with Go (Golang), it manages user accounts, deposits, withdrawals, transfers, and transaction history. The project now features an **event-driven architecture** with strict domain invariants, robust error handling, and a clear separation between HTTP handlers and service/application logic. All business rules and invariants are enforced in the domain layer, ensuring data integrity and preventing negative balances.

## Vision ✨

- **Reliable:** Accurate, consistent, and auditable transaction processing
- **Event-Driven:** All payment and transaction state changes are driven by events and webhooks
- **Invariant-Enforcing:** No negative balances; all business rules are strictly enforced in the domain layer
- **Maintainable:** Clean architecture, handler/service separation, and modular design
- **Scalable & Performant:** Designed for high throughput and concurrency using Go's strengths

## Features 🌟

- **Account Management:** Create, view, and manage user accounts
- **Fund Operations:**
  - **Deposits/Withdrawals:** Initiate and complete deposits/withdrawals via event-driven payment flows
  - **Transfers:** Move funds between accounts with strict invariant checks
- **Multi-Currency Support:** Currency-aware accounts and transactions, with real exchange rates and fallback strategies
- **Transaction History:** Full audit trail of all account activity
- **User Authentication & Authorization:** JWT-secured API, role-based access
- **Concurrency Safety:** All balance updates are atomic and race-free
- **Unit of Work Pattern:** Ensures atomicity and consistency for all business operations
- **Webhook-Driven Payment Status Updates:** Payment completion is confirmed asynchronously via webhook callbacks from payment providers
- **Internal Event Bus:** All payment and transaction state changes are propagated via an internal event bus, enabling decoupled and extensible event handling

## Event-Driven Architecture ⚡

The system is designed around an event-driven model for all payment and transaction flows:

- **Payment Initiation:** Creating a deposit or withdrawal generates a pending transaction and triggers an event
- **External Payment Provider:** Handles payment processing and calls back via a webhook when payment is completed or failed
- **Webhook Handler:** Receives payment status updates, updates transaction status, and triggers domain events
- **Event Bus:** Internal event bus propagates domain events (e.g., TransactionCompleted, BalanceUpdated) to update account balances and trigger side effects
- **Strict Invariants:** All balance updates and business rules are enforced in the domain layer; negative balances are never allowed

**Example Flow:**

1. User initiates a deposit (API call)
2. System creates a pending transaction and emits a PaymentInitiated event
3. Payment provider processes the payment and calls the webhook with the result
4. Webhook handler updates the transaction status and emits a TransactionCompleted event
5. Event bus handler updates the account balance if the transaction is completed

## Breaking Changes & Migration Notes ⚠️

- **Event-Driven Payments:** All deposit/withdrawal flows are now event-driven. Balances update only after payment completion events (via webhook).
- **Strict Balance Invariants:** Negative balances are strictly prevented at the domain layer. Withdrawals and transfers will fail if funds are insufficient.
- **Handler/Service Refactor:** HTTP handlers are now thin; all business logic is in services and the domain layer. Legacy persistence handler code has been removed.
- **Transaction Creation:** All transaction creation is now centralized and event-driven. Direct DB writes or legacy patterns are no longer supported.
- **API/Contract Changes:** Some endpoints and request/response formats have changed. See the OpenAPI spec for details.
- **Deleted Files:** Legacy persistence handler and e2e test files have been removed.
- **Testing:** Tests must set up event flows and use the new service interfaces. See updated test examples.
- **Migration:** Existing data may need to be migrated to support new transaction and event models. Review migration scripts and test thoroughly.

## Getting Started 🚀

> **Note:** If you are running the application in a development or test environment, ensure that webhook endpoints are accessible to your payment provider (or mock provider). The internal event bus requires no special configuration for local development, but you may need to expose webhook endpoints (e.g., using ngrok) if testing with external services.

These instructions will guide you through setting up and running the Fintech App on your local machine for development and testing.

### Prerequisites 🛠️

Before you begin, ensure you have the following software installed:

- **Go:** Version 1.24.4 or higher. Download from [golang.org/dl](https://golang.org/dl/). 🐹
- **Docker & Docker Compose:** Essential for setting up the PostgreSQL database and running the application in a containerized environment. Download from [docker.com](https://www.docker.com/get-started). 🐳
- **PostgreSQL Client (Optional):** Tools like `psql` or GUI clients (e.g., DBeaver, pgAdmin) can be useful for direct database interaction and inspection. 🐘

### Installation ⬇️

1. **Clone the repository:**
    Begin by cloning the project from its GitHub repository to your local machine:

    ```bash
    git clone https://github.com/amirasaad/fintech.git
    cd fintech
    ```

2. **Set up Environment Variables:**
    The application automatically loads environment variables from a `.env` file in the root directory. Copy the sample file and configure it:

    ```bash
    cp .env_sample .env
    ```

    Then edit the `.env` file with your configuration. At a minimum, you **must** set a strong value for `AUTH_JWT_SECRET`:

    ```dotenv
    AUTH_JWT_SECRET=your_super_secret_jwt_key
    ```

    See `.env_sample` for all available configuration options and their defaults. Other variables (such as database URL, API keys, etc.) can be left as defaults or customized as needed.

    **Note:** The application uses the `godotenv` package to automatically load environment variables from a `.env` file. If no `.env` file is found, the application will use system environment variables. 🔧

### Running the Application ▶️

#### Recommended: Using Docker Compose (for full environment setup) 🐳

This is the easiest way to get the entire application stack (database and application) running. Docker Compose will build the Go application image and start the PostgreSQL container.

```bash
docker compose up --build -d
```

- `--build`: Forces Docker to rebuild the application image, ensuring you're running the latest code changes. 🔄
- `-d`: Runs the services in detached mode (in the background). 🖥️

The application will be accessible at `http://localhost:3000`. 🌐 The PostgreSQL database will be available on port `5432`. 🐘

#### Running Locally (without Docker for the Go app) 💻

If you prefer to run the Go application directly on your host machine while still using Docker for the database:

1. **Start the PostgreSQL database using Docker Compose:**

    ```bash
    docker compose up db -d
    ```

    This will start only the `db` service defined in `docker-compose.yml`. 🐘

2. **Run the Go application:**
    Ensure your `.env` file is correctly configured to point to the Dockerized PostgreSQL instance (`localhost:5432`). Then, execute the main server application:

    ```bash
    go run cmd/server/main.go
    ```

    The application will be accessible at `http://localhost:3000`. 🌐

#### Running the CLI Application 🖥️

The project also includes a command-line interface (CLI) application for direct interaction with the system.

> **Note:** The CLI interacts with the event-driven flow. For deposit/withdrawal operations, the CLI will initiate the transaction, but the account balance will only update after the payment provider (or mock) triggers the webhook callback. You can simulate webhook events in development using the mock provider or by calling the webhook endpoint manually.

To run the CLI:

```bash
go run cmd/cli/main.go
```

##### CLI Commands

Once the CLI is running, you will be prompted to log in. After successful authentication, you can use the following commands:

- `create`: Creates a new account for the logged-in user. 🆕
- `deposit <account_id> <amount>`: Deposits the specified `amount` into the given `account_id`. ⬆️
- `withdraw <account_id> <amount>`: Withdraws the specified `amount` from the given `account_id`. ⬇️
- `balance <account_id>`: Retrieves and displays the current balance of the specified `account_id`. 💲
- `logout`: Logs out the current user. 👋
- `exit`: Exits the CLI application. 🚪

### Migrations 🗄️

Database migrations are managed using the `golang-migrate` library. This allows for version-controlled, incremental changes to the database schema.

#### Creating a New Migration

To create a new migration file, run the following command from the root of the project:

```bash
make migrate-create
```

You will be prompted to enter a name for the migration (e.g., `add_users_table`). This will generate two new SQL files in the `internal/migrations` directory: one for `up` (applying the migration) and one for `down` (reverting the migration).

#### Applying Migrations

To apply all pending migrations, use the following command:

```bash
make migrate-up
```

This will apply all `up` migrations that have not yet been run.

#### Reverting Migrations

To revert the last applied migration, use the following command:

```bash
make migrate-down
```

#### Applying a Specific Number of Migrations

To apply a specific number of pending migrations, you can use the `migrate` tool directly. For example, to apply the next two migrations, you would run:

```bash
migrate -database "postgres://postgres:password@localhost:5432/fintech?sslmode=disable" -path internal/migrations up 2
```

#### Fixing a Dirty Database

If a migration fails, the database may be left in a "dirty" state. To fix this, you will need to manually revert the changes from the failed migration and then force the migration version to the last successful migration. For example, if migration `3` failed, you would force the version to `2`:

```bash
migrate -database "postgres://postgres:password@localhost:5432/fintech?sslmode=disable" -path internal/migrations force 2
```

## Examples 💡

Here are some examples demonstrating how to interact with the Fintech App.

### CLI Interaction

1. **Start the CLI:**

    ```bash
    go run cmd/cli/main.go
    ```

2. **Login (when prompted):**

    ```bash

        ███████╗██╗███╗   ██╗████████╗███████╗ ██████╗██╗  ██╗     ██████╗██╗     ██╗
        ██╔════╝██║████╗  ██║╚══██╔══╝██╔════╝██╔════╝██║  ██║    ██╔════╝██║     ██║
        █████╗  ██║██╔██╗ ██║   ██║   █████╗  ██║     ███████║    ██║     ██║     ██║
        ██╔══╝  ██║██║╚██╗██║   ██║   ██╔══╝  ██║     ██╔══██║    ██║     ██║     ██║
        ██║     ██║██║ ╚████║   ██║   ███████╗╚██████╗██║  ██║    ╚██████╗███████╗██║
        ╚═╝     ╚═╝╚═╝  ╚═══╝   ╚═╝   ╚══════╝ ╚═════╝╚═╝  ╚═╝     ╚═════╝╚══════╝╚═╝
                                                                        Version (v1.0.0)

    Please login to continue.
    Username or Email:
    Username or Email: your_username
    Password: your_password
    Login successful!
    ```

3. **Create an account:**

    ```bash
    > create
    Account created: ID=xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx, Balance=0.00
    ```

4. **Deposit funds:**

    ```bash
    > deposit xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx 100.50
    Deposited 100.50 to account xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx. New balance: 100.50
    ```

5. **Check balance:**

    ```bash
    > balance xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
    Account xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx balance: 100.50
    ```

### API Interaction (using `curl`)

First, ensure the API server is running (e.g., via `docker compose up -d` or `go run cmd/server/main.go`).

1. **Register a new user:**

    ```bash
    curl -X POST http://localhost:3000/user \
      -H "Content-Type: application/json" \
      -d '{"username":"apiuser","email":"api@example.com","password":"apipassword"}'
    ```

    *Expected Output (truncated):*

    ```json
    {
      "status": 201,
      "message": "Created user",
      "data": {
        "id": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
        "username": "apiuser",
        "email": "api@example.com",
        "password": "...",
        "created": "...",
        "updated": "..."
      }
    }
    ```

2. **Login to get a JWT token:**

    ```bash
    curl -X POST http://localhost:3000/login \
      -H "Content-Type: application/json" \
      -d '{"identity":"apiuser","password":"apipassword"}'
    ```

    *Expected Output (truncated):*

    ```json
    {
      "status": 200,
      "message": "Success login",
      "data": {
        "token": "eyJ..."
      }
    }
    ```

    *Note: Copy the `token` value for subsequent requests.*

3. **Create an account (using the JWT token):**
    Replace `YOUR_JWT_TOKEN` with the token obtained from the login step.

    ```bash
    curl -X POST http://localhost:3000/account \
      -H "Authorization: Bearer YOUR_JWT_TOKEN"
    ```

    *Expected Output (truncated):*

    ```json
    {
      "status": 201,
      "message": "Account created",
      "data": {
        "ID": "yyyyyyyy-yyyy-yyyy-yyyy-yyyyyyyyyyyy",
        "UserID": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
        "Balance": 0,
        "CreatedAt": "...",
        "UpdatedAt": "..."
      }
    }
    ```

    *Note: Copy the `ID` value of the newly created account for subsequent requests.*

4. **Deposit funds into the account (asynchronous/event-driven):**
    Replace `YOUR_JWT_TOKEN` and `YOUR_ACCOUNT_ID`.

    ```bash
    curl -X POST http://localhost:3000/account/YOUR_ACCOUNT_ID/deposit \
      -H "Content-Type: application/json" \
      -H "Authorization: Bearer YOUR_JWT_TOKEN" \
      -d '{"amount": 500.75}'
    ```

    > **Note:** This creates a pending transaction. The account balance will only update after the payment provider (or mock) calls the webhook endpoint to confirm payment completion.

5. **Get account balance:**
    Replace `YOUR_JWT_TOKEN` and `YOUR_ACCOUNT_ID`.

    ```bash
    curl -X GET http://localhost:3000/account/YOUR_ACCOUNT_ID/balance \
      -H "Authorization: Bearer YOUR_JWT_TOKEN"
    ```

## API Endpoints 🔗

The Fintech App exposes a comprehensive RESTful API for all its functionalities. The API design prioritizes clear resource naming, standard HTTP methods, and meaningful status codes.

- **Full API Specification:** A detailed OpenAPI (Swagger) specification is available at [openapi.yaml](./docs/openapi.yaml). This file can be used with tools like Swagger UI to explore and test the API interactively. 📄
- **Example Requests:** You can find practical examples of API requests in the [requests.http](./docs/requests.http) file, which can be executed directly using IDE extensions like the REST Client for VS Code or similar tools. 📝

### Webhook Endpoints ⚡

- `POST /webhook/payment-status`: Receives asynchronous payment status updates from payment providers. This endpoint is called by the provider (or mock) to confirm payment completion or failure. **(Event-driven, not called by end users)**

### Asynchronous/Event-Driven Endpoints

- `POST /account/:id/deposit` and `POST /account/:id/withdraw` now initiate a transaction and return immediately. The account balance is updated only after the corresponding webhook event is received and processed.

### Authentication 🔑

- `POST /login`: Authenticates a user with their credentials (username/email and password) and returns a JSON Web Token (JWT) upon successful authentication. This token must be included in the `Authorization` header for all protected endpoints. 🔐

### User Management 👤

- `POST /user`: Registers a new user in the system. ➕
- `GET /user/:id`: Retrieves the profile details of a specific user by their ID. **(Protected)** 🔍
- `PUT /user/:id`: Updates the profile information for a specific user. **(Protected)** ✏️
- `DELETE /user/:id`: Deletes a user account from the system. **(Protected)** 🗑️

### Account Operations 💳

- `POST /account`: Creates a new financial account linked to the authenticated user. **(Protected)** 🆕
- `POST /account/:id/deposit`: Initiates a deposit of funds into the specified account. **(Protected)** ⬆️
- `POST /account/:id/withdraw`: Processes a withdrawal of funds from the specified account, subject to balance availability. **(Protected)** ⬇️
- `GET /account/:id/balance`: Fetches the current balance of the specified account. **(Protected)** 💲
- `GET /account/:id/transactions`: Retrieves a list of all transactions associated with the specified account. **(Protected)** 📜

### Error Handling 🚨

The API follows RESTful conventions for error responses and uses consistent error handling patterns:

#### HTTP Status Codes

- **200 OK**: Request successful
- **201 Created**: Resource created successfully
- **204 No Content**: Request successful, no content to return
- **400 Bad Request**: Invalid request data or validation errors
- **401 Unauthorized**: Authentication required or invalid credentials
- **403 Forbidden**: Authenticated but not authorized for the resource
- **404 Not Found**: Resource not found
- **422 Unprocessable Entity**: Business rule violations (e.g., insufficient funds)
- **429 Too Many Requests**: Rate limit exceeded
- **500 Internal Server Error**: Unexpected server error

#### Error Response Format

All error responses follow the RFC 9457 Problem Details format:

```json
{
  "type": "about:blank",
  "title": "Error Title",
  "status": 404,
  "detail": "Detailed error message",
  "instance": "/user/123"
}
```

#### Common Error Scenarios

- **User Not Found**: Returns 404 when attempting to update/delete a non-existent user
- **Account Not Found**: Returns 404 when accessing non-existent accounts
- **Insufficient Funds**: Returns 422 when withdrawal amount exceeds balance
- **Invalid Currency**: Returns 422 for unsupported currency codes
- **Unauthorized Access**: Returns 403 when users try to access other users' resources
- **Validation Errors**: Returns 400 with detailed field-specific error messages

## Multi-Currency & Money Value Object

- All monetary operations (deposit, withdraw) are now currency-aware and validated using the `Money` value object in the domain layer.
- The service layer (`AccountService`) exposes methods that accept `amount` and `currency` as primitives, and constructs/validates `Money` internally.
- This eliminates the need for separate `DepositWithCurrency`/`WithdrawWithCurrency` methods and reduces code duplication.

### Example Usage

```go
// In your handler or service consumer:
tx, err := accountService.Deposit(userID, accountID, 100.0, "EUR")
if err != nil {
    // handle error (e.g., invalid currency, amount, or business rule)
}
```

- All validation (currency code, amount positivity) is performed in the domain layer via `NewMoney`.
- This pattern ensures consistency, security, and extensibility for future features like currency conversion.

## Service ↔ Domain Layer Communication

This project follows clean architecture principles, with clear separation between the service and domain layers.

### Diagram: Service ↔ Domain Communication

```mermaid
flowchart TD
    A["API Handler / Controller"] --> B["Service Layer<br/>(e.g., AccountService)"]
    B -->|"Constructs/Calls"| C["Domain Entity<br/>(e.g., Account, User)"]
    B -->|"Uses"| D["Domain Service<br/>(e.g., CurrencyConverter)"]
    B -->|"Persists/Loads"| E["Repository Interface"]
    E -->|"Implements"| F["Infrastructure Repo<br/>(DB, Cache, etc.)"]
    C <--> D
```

- **API Handler**: Receives request, calls service.
- **Service Layer**: Orchestrates use case, manages transactions, calls domain logic.
- **Domain Entity/Service**: Contains business rules, invariants.
- **Repository**: Abstracts persistence, injected into service.
- **Infrastructure**: Actual DB/cache implementation.

### Example: Account Deposit

**Domain Layer (`pkg/domain/account.go`):**

```go
func (a *Account) Deposit(userID uuid.UUID, money Money) (*Transaction, error) {
    if userID != a.UserID {
        return nil, ErrUserUnauthorized
    }
    if money.Amount <= 0 {
        return nil, ErrInvalidAmount
    }
    a.Balance += money.Amount
    tx := NewTransaction(a.ID, userID, money)
    return tx, nil
}
```

**Service Layer (`pkg/service/account.go`):**

```go
func (s *AccountService) Deposit(userID, accountID uuid.UUID, amount float64, currencyCode currency.Code) (*domain.Transaction, *domain.ConversionInfo, error) {
    // --- Rest of code  ---
    // --- Domain logic ---
    tx, err := account.Deposit(userID, money)
    if err != nil {
        _ = uow.Rollback()
        return nil, nil, err
    }
    // --- Persist changes ---
    err = repo.Update(account)
    if err != nil {
        _ = uow.Rollback()
        return nil, nil, err
    }
    // --- Rest of code ---
}
```

- **Service Layer**: Orchestrates the use case, manages transactions, and coordinates repositories.
- **Domain Layer**: Enforces business rules and invariants (e.g., only the account owner can deposit, amount must be positive).
- **Repositories**: Abstract persistence, injected into services.
- **Unit of Work**: Ensures atomicity of operations.

## Project Structure 📁

The project is meticulously organized to promote modularity, maintainability, and adherence to Domain-Driven Design (DDD) principles. This structure facilitates clear separation of concerns and simplifies development and testing.

```ascii
fintech/
├── .github/          # GitHub Actions workflows for CI/CD 🚀
├── api/              # Vercel serverless function entry point (for serverless deployments) ☁️
├── cmd/              # Main application entry points
│   ├── cli/          # Command-Line Interface application 💻
│   └── server/       # HTTP server application 🌐
├── docs/             # Project documentation, OpenAPI spec, HTTP request examples, coverage reports 📄
├── infra/            # Infrastructure Layer 🏗️
│   ├── database.go   # Database connection and migration logic 🗄️
│   ├── eventbus/     # Internal event bus for domain/integration events ⚡
│   ├── model.go      # GORM database models (mapping domain entities to database tables) 📊
│   ├── provider/     # Payment/currency providers, webhook simulation 🏦
│   ├── repository/   # Concrete repository implementations 💾
│   └── uow.go        # Unit of Work implementation 🔄
├── pkg/              # Core Application Packages (Domain, Application, and Shared Infrastructure) 📦
│   ├── domain/       # Domain Layer: Core business entities and rules ❤️
│   ├── middleware/   # Shared middleware components 🚦
│   ├── repository/   # Repository interfaces & UoW 🗃️
│   ├── service/      # Application Layer: Orchestrates use cases, emits/handles events ⚙️
│   └── ...
├── webapi/           # Presentation Layer (Web API) 🌐
│   ├── account/      # Account HTTP handlers, DTOs, webhooks, and related tests 💳
│   ├── auth/         # Authentication HTTP handlers and DTOs 🔑
│   ├── currency/     # Currency HTTP handlers and DTOs 💱
│   ├── user/         # User HTTP handlers and DTOs 👤
│   ├── common/       # Shared web API utilities (e.g., error formatting) 🛠️
│   ├── testutils/    # Test helpers for web API layer 🧪
│   ├── app.go        # Fiber application setup and route registration 🚀
│   ├── webapi.go     # Web API entry point or shared logic 🌐
│   └── ratelimit_test.go # Rate limiting tests 🚦
├── go.mod            # Go module definition 📝
├── go.sum            # Go module checksums ✅
├── Makefile          # Automation scripts 🤖
├── Dockerfile        # Docker build instructions 🐳
├── docker-compose.yml# Docker Compose config 🛠️
├── .env.example      # Example environment variables 📄
├── .gitignore        # Ignore rules 🙈
├── README.md         # This README 📖
└── vercel.json       # Vercel deployment config ☁️
```

## Infrastructure & Design Choices 💡

This project leverages a modern tech stack and adheres to robust design principles to ensure a high-quality, performant, and maintainable application.

- **Language: Go (Golang)** 🐹
  - **Why Go?** Chosen for its excellent performance, strong concurrency model (goroutines and channels), fast compilation times, and static typing, which contribute to building highly efficient and reliable backend services. Its simplicity and strong standard library also accelerate development. ⚡
- **Web Framework: [Fiber](https://gofiber.io/)** 🌐
  - **Why Fiber?** A fast and unopinionated web framework inspired by Express.js. Fiber's performance, ease of use, and extensive middleware ecosystem make it an ideal choice for building high-throughput APIs in Go. 🚀
- **ORM: [GORM](https://gorm.io/index.html)** 🗄️
  - **Why GORM?** A developer-friendly ORM library for Go that simplifies database interactions. It provides powerful features like migrations, associations, and a fluent API, reducing boilerplate code while maintaining control over SQL queries. 💾
- **Database: PostgreSQL 12** 🐘
  - **Why PostgreSQL?** A powerful, open-source relational database known for its reliability, feature robustness, and strong support for transactional integrity. It's a proven choice for mission-critical applications requiring data consistency. 💪
- **Authentication: JSON Web Tokens (JWT) & Basic Authentication** 🔐
  - **JWT (Web API):** A compact, URL-safe means of representing claims to be transferred between two parties. JWTs are used for stateless authentication, allowing the API to scale horizontally without session management overhead. They provide a secure way to transmit user identity and authorization information. 🔑
  - **Basic Authentication (CLI):** For the command-line interface, a basic authentication strategy is employed, where credentials are directly validated against the user store without the overhead of token generation. 💻
- **Concurrency Safety: `sync.Mutex`** 🚦
  - **Why `sync.Mutex`?** In a multi-threaded environment, concurrent access to shared resources (like an account balance) can lead to race conditions and data corruption. `sync.Mutex` is used to protect critical sections of code, ensuring that only one goroutine can modify an account's balance at any given time, thus guaranteeing transactional atomicity and data integrity. 🛡️
- **Unit of Work Pattern:** 📦
  - **Implementation:** The Unit of Work (UoW) pattern is implemented to manage a group of business operations that must be treated as a single transaction. It encapsulates all changes to the database within a single transaction, ensuring that either all changes are committed successfully or all are rolled back if any operation fails. This is crucial for maintaining data consistency in financial applications. 🔄
- **Code Quality: [Qodana](https://www.jetbrains.com/qodana/)** 🧹
  - **Why Qodana?** A static code analysis platform by JetBrains that helps maintain code quality, identify potential bugs, and enforce coding standards. Integrated into the CI/CD pipeline, it provides continuous feedback on code health. 📈
- **Deployment: Vercel (Serverless Functions)** ☁️
  - **Why Vercel?** Configured for serverless deployment, allowing the application to be deployed as a serverless function on Vercel's platform. This provides benefits such as automatic scaling, reduced operational overhead, and a pay-per-use cost model. 🚀
- **Event Bus & Webhook-Driven Design:**
  - **Event Bus:** All payment and transaction state changes are propagated via an internal event bus, enabling decoupled, extensible event handling and side effects.
  - **Webhook-Driven:** Payment completion is confirmed asynchronously via webhook callbacks from payment providers, ensuring robust and auditable payment flows.

## Testing 🧪

> **Note:** All tests for deposit, withdrawal, and transfer flows should use the event-driven flow and interact with the new service interfaces. Tests should simulate webhook callbacks to confirm payment completion and verify balance updates.

Comprehensive testing is crucial for ensuring the reliability and correctness of a financial application. This project includes a robust testing suite.

- **Unit Tests:** Located alongside the code they test (e.g., `_test.go` files), these tests verify the functionality of individual components in isolation. 🎯
- **Test Suite Execution:**
    To run the entire test suite, including all unit tests:

    ```bash
    go test -v ./...
    ```

    The `-v` flag provides verbose output, showing details of each test run. 📊

- **Code Coverage:**
    To generate a code coverage report, which indicates the percentage of code exercised by tests:

    ```bash
    make cov_report
    ```

    This command first runs the tests with coverage enabled (`make cov`) and then generates an HTML report. The report will be saved at `docs/cover.html`, which you can open in your web browser to visualize covered and uncovered lines of code. 📈

## Contributing 💡

> **Important:** All new features and changes must follow the event-driven, invariant-enforcing architecture. Business rules must be enforced in the domain layer, and all payment/transaction flows should use the event bus and webhook-driven patterns.

We welcome contributions to the Fintech App! To contribute, please follow these guidelines:

1. **Fork the repository:** Start by forking the `fintech` repository to your GitHub account. 🍴
2. **Create a new branch:** For each new feature or bug fix, create a dedicated branch from `main`:

    ```bash
    git checkout -b feature/your-feature-name # for new features
    git checkout -b bugfix/issue-description  # for bug fixes
    ```

    🌳
3. **Make your changes:** Implement your feature or fix the bug. Ensure your code adheres to the existing coding style and conventions. 📝
4. **Write/Update Tests:** If you're adding new functionality, write corresponding unit and/or integration tests. If you're fixing a bug, add a test that reproduces the bug and ensures your fix resolves it. 🧪
5. **Ensure Tests Pass:** Before committing, run the entire test suite to ensure your changes haven't introduced any regressions:

    ```bash
    go test -v ./...
    ```

    ✅
6. **Commit your changes:** Use [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/) for clear and consistent commit messages. This helps in generating changelogs and understanding the project history. ✉️
    Example:

    ```vim
    feat: add new account creation endpoint

    This commit introduces the /account POST endpoint for creating new user accounts.
    It includes validation for input parameters and integrates with the AccountService.
    ```

7. **Push to the branch:** Push your local branch to your forked repository:

    ```bash
    git push origin feature/your-feature-name
    ```

    ⬆️
8. **Open a Pull Request:** Navigate to the original `fintech` repository on GitHub and open a pull request from your branch. Provide a clear description of your changes and reference any related issues. 📥

## License ©

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details. 📄
