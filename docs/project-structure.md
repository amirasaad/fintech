# 🗂️ Project Structure

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

Each directory and file is designed to support clean architecture and event-driven design. See the rest of the docs for deeper dives into each layer.
