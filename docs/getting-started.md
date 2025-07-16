---
icon: material/rocket
---

# Getting Started

These instructions will guide you through setting up and running the Fintech Platform on your local machine for development and testing.

## 🛠️ Prerequisites

- **Go:** Version 1.22 or higher. Download from [golang.org/dl](https://golang.org/dl/). 🐹
- **Docker & Docker Compose:** For PostgreSQL and running the app in containers. [docker.com](https://www.docker.com/get-started) 🐳
- **PostgreSQL Client (Optional):** Tools like `psql` or GUI clients (e.g., DBeaver, pgAdmin) 🐘

## ⬇️ Installation

1. **Clone the repository:**

   ```bash
   git clone https://github.com/amirasaad/fintech.git
   cd fintech
   ```

2. **Set up Environment Variables:**

   ```bash
   cp .env_sample .env
   # Edit .env as needed (see .env_sample for options)
   ```

   At a minimum, set a strong value for `AUTH_JWT_SECRET` in `.env`.

## ▶️ Running the Application

### Using Docker Compose (Recommended)

```bash
docker compose up --build -d
```

- The app will be at `http://localhost:3000`.
- PostgreSQL at port `5432`.

### Running Locally (without Docker for Go app)

1. Start PostgreSQL with Docker Compose:

   ```bash
   docker compose up db -d
   ```

2. Run the Go app:

   ```bash
   go run cmd/server/main.go
   ```

### Running the CLI

```bash
go run cmd/cli/main.go
```

## 🗄️ Migrations

- Create a new migration:

  ```bash
  make migrate-create
  ```

- Apply all migrations:

  ```bash
  make migrate-up
  ```

- Revert last migration:

  ```bash
  make migrate-down
  ```

- See `internal/migrations/` for migration files.

## 💡 Tips

- The app loads env vars from `.env` (via `godotenv`).
- For payment/webhook testing, use the mock provider or call the webhook endpoint manually.
- See [docs/index.md](index.md) for navigation and more guides.
