# Fintech App

Fintech app manages accounts and financial
transactions. The system supports operations such as:

- Opening an account
- Depositing funds
- Withdrawing funds
- Checking the balance

## Getting started

run command:

```shell
go run cmd/server.go 
```

Or using docker

```shell
docker compose up -d
```

## Endpoints

### Create Account

POST /account

### Deposit to account

POST /account/{accountID}/deposit

### Withdraw from account

POST /account/{accountID}/withdraw

### List Transactions

GET /account/{accountID}/transactions

### Get Account Balance

GET /account/{accountID}/balance

see examples at [requests.http](./docs/requests.http) file for more details.

Full spec is available at [openapi.yaml](./docs/openapi.yaml) file.

## Infrastructure & Design

The app follows domain driven design all business logic can be found at [domain](./internal/domain) package.
The app uses PostgreSQL 12 as the database.
It uses [GORM](https://gorm.io/index.html) as the ORM for the database.
It uses [goFiber](https://gofiber.io/) as the web framework.
It follows conventional commit message format.

## Testing

To run test suit run the following command

```shell
go test ./...
```

## Todos

- [ ] User Authentication.
- [x] Safe thread handling to avoid transactions to execute simultaneously.
- [x] Impalement unit of work to handle transactions. This will ensure consist data when failure occurs.
- [x] Add more tests.
