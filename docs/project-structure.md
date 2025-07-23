---
icon: material/folder
---

# Project Structure

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
│   ├── eventbus/     # Internal event bus for domain/integration events ⚡
│   ├── provider/     # Payment/currency providers, webhook simulation 🏦
│   └── repository/   # Concrete repository implementations 💾
├── pkg/              # Core Application Packages (Domain, Application, and Shared Infrastructure) 📦
│   ├── cache/        # Caching interfaces and implementations 🗄️
│   ├── commands/     # Command pattern implementations ⚡
│   ├── currency/     # Currency domain logic and utilities 💱
│   ├── domain/       # Domain Layer: Core business entities and rules ❤️
│   │   ├── account/  # Account domain entities and business logic 💳
│   │   ├── events/   # Domain events for event-driven architecture 📡
│   │   ├── money/    # Money value object and currency handling 💰
│   │   └── user/     # User domain entities 👤
│   ├── dto/          # Data Transfer Objects for API communication 📋
│   ├── eventbus/     # Event bus interfaces and implementations 🚌
│   ├── handler/      # Event handlers for business flows 🎯
│   │   ├── account/  # Account-related event handlers
│   │   │   ├── deposit/   # Deposit flow handlers
│   │   │   ├── transfer/  # Transfer flow handlers
│   │   │   └── withdraw/  # Withdraw flow handlers
│   │   ├── conversion/    # Currency conversion handlers
│   │   ├── payment/       # Payment processing handlers
│   │   └── transaction/   # Transaction-related handlers
│   ├── mapper/       # Object mapping utilities 🔄
│   ├── middleware/   # Shared middleware components 🚦
│   ├── processor/    # Business process orchestrators ⚙️
│   ├── provider/     # External service provider interfaces 🔌
│   ├── queries/      # Query pattern implementations 🔍
│   ├── registry/     # Service registry and dependency injection 📋
│   ├── repository/   # Repository interfaces & UoW 🗃️
│   ├── service/      # Application Layer: Orchestrates use cases, emits/handles events ⚙️
│   │   ├── account/  # Account service implementations
│   │   ├── auth/     # Authentication services
│   │   ├── currency/ # Currency services
│   │   └── user/     # User services
│   └── utils/        # Shared utility functions 🛠️
├── webapi/           # Presentation Layer (Web API) 🌐
│   ├── account/      # Account HTTP handlers, DTOs, webhooks, and related tests 💳
│   ├── auth/         # Authentication HTTP handlers and DTOs 🔑
│   ├── common/       # Shared web API utilities (e.g., error formatting) 🛠️
│   ├── currency/     # Currency HTTP handlers and DTOs 💱
│   ├── testutils/    # Test helpers for web API layer 🧪
│   ├── user/         # User HTTP handlers and DTOs 👤
│   └── app.go        # Fiber application setup and route registration 🚀
├── internal/         # Internal packages (not for external use) 🔒
│   └── fixtures/     # Test fixtures and mocks 🧪
├── scripts/          # Build and deployment scripts 📜
├── config/           # Configuration management ⚙️
├── go.mod            # Go module definition 📝
├── go.sum            # Go module checksums ✅
├── Makefile          # Automation scripts 🤖
├── Dockerfile        # Docker build instructions 🐳
├── docker-compose.yml# Docker Compose config 🛠️
├── .env.example      # Example environment variables 📄
├── .gitignore        # Ignore rules 🙈
├── README.md         # Project README 📖
├── ARCHITECTURE.md   # Architecture documentation 🏗️
├── CONTRIBUTING.md   # Contribution guidelines 🤝
└── vercel.json       # Vercel deployment config ☁️
```

## 🏗️ Architecture Layers

### Domain Layer (`pkg/domain/`)
- **Pure business logic** with no external dependencies
- **Value objects** like `Money` for type safety
- **Domain entities** like `Account` and `User`
- **Domain events** for event-driven architecture

### Application Layer (`pkg/service/`, `pkg/handler/`)
- **Use case orchestration** through services
- **Event handlers** for business flow processing
- **Application-specific business rules**

### Infrastructure Layer (`infra/`)
- **Database implementations** using GORM
- **External service integrations** (Stripe, currency APIs)
- **Event bus implementations**

### Presentation Layer (`webapi/`)
- **HTTP handlers** using Fiber framework
- **Request/response DTOs**
- **Authentication and middleware**

## 🎯 Key Design Principles

- **Clean Architecture:** Clear separation between layers
- **Domain-Driven Design:** Business logic encapsulated in domain layer
- **Event-Driven Architecture:** Loose coupling through domain events
- **Dependency Injection:** Services registered through registry pattern
- **Repository Pattern:** Data access abstraction
- **Unit of Work Pattern:** Transaction management

## 🧪 Testing Structure

- **Unit Tests:** Located alongside source files (`*_test.go`)
- **Integration Tests:** Test complete workflows
- **E2E Tests:** Test full business scenarios
- **Test Fixtures:** Shared test data in `internal/fixtures/`
- **Mocks:** Generated mocks for interfaces

Each directory and file is designed to support clean architecture and event-driven design. See the rest of the docs for deeper dives into each layer.
