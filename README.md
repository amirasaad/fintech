# Fintech App 🚀

**Note:** This project is primarily a learning endeavor and a personal exploration of building a robust financial application with Go. It's a work in progress and serves as a practical playground for applying software engineering principles.

[![build](https://github.com/amirasaad/fintech/actions/workflows/ci.yml/badge.svg)](https://github.com/amirasaad/fintech/actions/workflows/ci.yml)

## Overview 📝

This Fintech App is a personal project exploring the development of a financial transaction system. Built with Go (Golang), it's designed to manage user accounts, deposits, withdrawals, and transaction history. The project focuses on applying modern software engineering principles like Domain-Driven Design (DDD) and clean architecture to understand and implement maintainable, testable, and performant solutions, with an emphasis on data integrity and concurrency safety.

## Vision ✨

This project's vision is to build a foundational backend service for financial operations, primarily as a learning exercise. The goal is to explore and implement a system that is:

- **Reliable:** To understand how to ensure accurate and consistent transaction processing. ✅
- **Scalable:** To learn about designing systems that can handle increasing loads. 📈
- **Secure:** To practice protecting sensitive financial data and preventing unauthorized access. 🔒
- **Maintainable:** To apply principles like clear separation of concerns for easier understanding and extension. 🛠️
- **Performant:** To leverage Go's concurrency features for high throughput and low latency in a practical context. ⚡

## Features 🌟

- **Account Management:** 💳
  - **Opening Accounts:** Users can securely create new financial accounts. 🆕
  - **Account Details:** Retrieve comprehensive information about any account. ℹ️
- **Fund Operations:** 💰
  - **Deposits:** Safely add funds to an account with real-time balance updates. ⬆️
  - **Withdrawals:** Securely remove funds from an account, with checks for insufficient funds. ⬇️
- **Multi-Currency Support:** 💸
  - Accounts and transactions support multiple currencies (e.g., USD, EUR, GBP).
  - All operations are currency-aware, ensuring consistency.
  - For more details, see the [Multi-Currency Documentation](./docs/multi_currency.md).
- **Real-time Balances:** Instantly query and display the current balance of any account, crucial for immediate financial oversight. ⏱️
- **Transaction History:** Access a detailed, chronological record of all financial movements associated with an account, providing transparency and auditability. 📜
- **User Authentication & Authorization:** 🤝
  - Secure user registration and login using industry-standard JSON Web Tokens (JWT) for the web API. 🔑
  - Basic authentication for CLI interactions. 💻
  - Role-based access control ensures that users can only perform operations relevant to their accounts. 🛡️
- **Concurrency Safety:** Implemented using Go's native concurrency primitives (`sync.Mutex`) to prevent race conditions and ensure atomic operations during simultaneous deposits and withdrawals, guaranteeing data integrity. 🚦
- **Unit of Work Pattern:** A core design pattern that ensures all operations within a single business transaction are treated as a single, atomic unit. This guarantees data consistency and integrity, especially during complex sequences of database operations. 📦

### Breaking Changes ⚠️

With the introduction of multi-currency support, the following changes may affect existing API clients:

- **Deposit and Withdrawal Operations:** The `POST /account/:id/deposit` and `POST /account/:id/withdraw` endpoints now require a `currency` field in the request body. The provided currency must match the account's currency. Requests without a `currency` field may fail if the account's currency is not the default ("USD").

## Getting Started 🚀

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
    The application relies on environment variables for configuration, particularly for database connection and JWT secrets. Create a file named `.env` in the root directory of the project and populate it with the following:

    ```dotenv
    DATABASE_URL=postgres://postgres:password@localhost:5432/fintech?sslmode=disable
    JWT_SECRET_KEY=your_super_secret_jwt_key_replace_this_in_production
    ```

    - `DATABASE_URL`: Specifies the connection string for your PostgreSQL database. The provided value is suitable for local development using Docker Compose. 🗄️
    - `JWT_SECRET_KEY`: A secret key used for signing and verifying JWTs. **For production environments, it is critical to use a strong, randomly generated key and manage it securely (e.g., via Kubernetes secrets, AWS Secrets Manager, or similar services). Never hardcode sensitive keys.** ⚠️

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

4. **Deposit funds into the account:**
    Replace `YOUR_JWT_TOKEN` and `YOUR_ACCOUNT_ID`.

    ```bash
    curl -X POST http://localhost:3000/account/YOUR_ACCOUNT_ID/deposit \
      -H "Content-Type: application/json" \
      -H "Authorization: Bearer YOUR_JWT_TOKEN" \
      -d '{"amount": 500.75}'
    ```

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

## Project Structure 📁

The project is meticulously organized to promote modularity, maintainability, and adherence to Domain-Driven Design (DDD) principles. This structure facilitates clear separation of concerns and simplifies development and testing.

```ascii
.
├── .github/          # GitHub Actions workflows for CI/CD 🚀
├── api/              # Vercel serverless function entry point (for serverless deployments) ☁️
├── cmd/              # Main application entry points
│   ├── cli/          # Command-Line Interface application 💻
│   └── server/       # HTTP server application 🌐
├── docs/             # Project documentation, OpenAPI spec, HTTP request examples, coverage reports 📄
├── infra/            # Infrastructure Layer 🏗️
│   ├── database.go   # Database connection and migration logic 🗄️
│   ├── model.go      # GORM database models (mapping domain entities to database tables) 📊
│   ├── repository.go # Concrete implementations of repository interfaces using GORM 💾
│   └── uow.go        # Unit of Work implementation for transactional integrity 🔄
├── pkg/              # Core Application Packages (Domain, Application, and Shared Infrastructure) 📦
│   ├── domain/       # Domain Layer: Contains core business entities (e.g., Account, User, Transaction) and their encapsulated business logic. This is the heart of the application. ❤️
│   ├── middleware/   # Shared middleware components for the web API (e.g., authentication, rate limiting) 🚦
│   ├── repository/   # Repository Layer: Defines interfaces for data persistence operations, abstracting away database specifics. This allows for interchangeable data storage solutions. 🗃️
│   └── service/      # Application Layer: Contains application services that orchestrate domain objects and repositories to fulfill use cases. This layer handles business logic coordination. ⚙️
├── webapi/           # Presentation Layer (Web API) 🌐
│   ├── account.go    # HTTP handlers for account-related operations 💳
│   ├── auth.go       # HTTP handlers for authentication (login) 🔑
│   ├── app.go        # Fiber application setup, middleware, and route registration 🚀
│   ├── user.go       # HTTP handlers for user-related operations 👤
│   └── utils.go      # Utility functions for HTTP responses and error handling 🔧
├── go.mod            # Go module definition and dependencies 📝
├── go.sum            # Go module checksums ✅
├── Makefile          # Automation scripts for common tasks (tests, coverage) 🤖
├── Dockerfile        # Docker build instructions for the application 🐳
├── docker-compose.yml# Docker Compose configuration for local development environment (app + database) 🛠️
├── .env.example      # Example environment variables file 📄
├── .gitignore        # Specifies intentionally untracked files to ignore 🙈
├── README.md         # This comprehensive README file 📖
└── vercel.json       # Vercel deployment configuration ☁️
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

## Testing 🧪

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
