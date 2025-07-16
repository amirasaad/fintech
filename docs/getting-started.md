# 🚀 Getting Started

Welcome to the Fintech Platform! This guide will help you set up, run, and develop the project locally or in production.

---

## 🗝️ Environment Setup

1. **Copy the sample file:**

   ```bash
   cp .env_sample .env
   ```

2. **Edit `.env` and fill in required values:**
   - Set a strong value for `JWT_SECRET_KEY` (required for authentication).
   - If using Stripe or Exchange Rate APIs, add your API keys.

**Key variables:**

- `APP_PORT` — Port for the app (default: 3000)
- `APP_HOST` — Host (default: localhost)
- `DATABASE_URL` — PostgreSQL connection string
- `REDIS_URL` — Redis connection string
- `JWT_SECRET_KEY` — 🔑 **Required!** Use a long, random string
- `EXCHANGE_RATE_API_KEY` — (Optional) For real exchange rates
- `PAYMENT_PROVIDER_STRIP_API__KEY` — (Optional) For Stripe integration

All other values have sensible defaults for local development. See `.env_sample` for the full list.

---

## 🗄️ Database & Redis

### PostgreSQL Database

- Used for all persistent data.
- Docker Compose starts a PostgreSQL container for you.
- Connection in `.env`:

  ```bash
  DATABASE_URL=postgres://postgres:password@localhost:5432/fintech?sslmode=disable
  ```

- **Default credentials:**
  - User: `postgres`
  - Password: `password`
  - DB: `fintech`
  - Host: `localhost`
  - Port: `5432`
- Connect with any PostgreSQL client (DBeaver, pgAdmin, psql, etc.).

### Redis (Optional, for Caching/Rate Limiting)

- Used for caching and rate limiting.
- Docker Compose starts a Redis container for you.
- Connection in `.env`:

  ```bash
  REDIS_URL=redis://localhost:6379/0
  ```

- **Default credentials:**
  - Host: `localhost`
  - Port: `6379`
  - DB: `0`
  - No password by default

### Docker Compose Services

When you run:

```bash
docker compose up --build -d
```

it will start:

- The Fintech app (on <http://localhost:3000>)
- PostgreSQL (on `localhost:5432`)
- Redis (on `localhost:6379`)

### Troubleshooting

- **PostgreSQL not connecting?**
  - Make sure Docker is running.
  - Check that `DATABASE_URL` matches your local setup.
  - Use `docker compose logs db` to see DB logs.
- **Redis not connecting?**
  - Make sure Docker is running.
  - Check that `REDIS_URL` matches your local setup.
  - Use `docker compose logs redis` to see Redis logs.

### Production Tips

- For production, update `DATABASE_URL` and `REDIS_URL` with your real credentials and hosts.
- Never use the default passwords in production!

---

## 🧰 Prerequisites

- **Go:** v1.24.4 or higher ([download](https://golang.org/dl/))
- **Docker & Docker Compose:** For database and local dev ([download](https://www.docker.com/get-started))
- **PostgreSQL Client (optional):** For DB inspection (e.g., DBeaver, pgAdmin)

---

## 🏗️ Local Development

### 1. Clone the Repository

```bash
git clone https://github.com/amirasaad/fintech.git
cd fintech
```

### 2. Set Up Environment Variables

Copy the sample file and edit as needed:

```bash
cp .env_sample .env
```

Set a strong value for `AUTH_JWT_SECRET` in `.env`.

---

### 3. Run with Docker Compose (Recommended)

This starts the app and PostgreSQL:

```bash
docker compose up --build -d
```

- App: <http://localhost:3000>
- DB:  localhost:5432

---

### 4. Run Go App Locally (with Docker DB)

Start only the DB:

```bash
docker compose up db -d
```

Run the server:

```bash
go run cmd/server/main.go
```

---

### 5. Run the CLI

```bash
go run cmd/cli/main.go
```

Follow prompts to log in and use commands like `create`, `deposit`, `withdraw`, `balance`, `logout`, `exit`.

---

## 🧪 Testing

Run all tests:

```bash
go test -v ./...
```

Generate coverage report:

```bash
make cov_report
```

---

## 🛠️ Makefile Targets

- `make migrate-create` — Create a new DB migration
- `make migrate-up` — Apply all pending migrations
- `make migrate-down` — Revert the last migration
- `make cov_report` — Generate coverage report

---

## 🗃️ Database Migrations

Migrations are managed with `golang-migrate`.

- Create: `make migrate-create`
- Apply:  `make migrate-up`
- Revert: `make migrate-down`

---

## ☁️ Vercel Deployment

- The project supports serverless deployment on Vercel.
- See `vercel.json` for config.
- Deploy via Vercel dashboard or CLI.

---

## 🛟 Troubleshooting

- **Ports in use?** Stop other services using 3000/5432.
- **DB connection errors?** Check `.env` and Docker status.
- **JWT errors?** Ensure `JWT_SECRET_KEY` is set.
- **Migrations fail?** Use `make migrate-down` or fix dirty DB state.

---

Happy coding! 🎉
