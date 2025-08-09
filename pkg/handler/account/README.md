# 🏦 Account Handlers

Event-driven account operation handlers following clean architecture and domain-driven design principles.

## 🏗️ Package Structure

```text
account/
├── deposit/                 # Deposit operation handlers
│   ├── handler.go           # Event handler implementation
│   ├── handler_test.go      # Handler tests
│   ├── validation.go        # Business validation
│   └── validation_test.go   # Validation tests
│
├── withdraw/                # Withdrawal operation handlers
│   ├── handler.go
│   ├── handler_test.go
│   ├── validation.go
│   └── validation_test.go
│
├── transfer/                # Transfer operation handlers
│   ├── handler.go
│   ├── handler_test.go
│   ├── validation.go
│   └── validation_test.go
│
└── common/                  # Shared components
    ├── account_query_handler.go
    ├── account_query_handler_test.go
    ├── validator.go
    └── validator_test.go
```

## 🚀 Key Features

- **Event-Driven**: Handlers respond to domain events
- **Clean Architecture**: Clear separation of concerns
- **Testable**: Comprehensive test coverage
- **Modular**: Independent operation handlers
- **Validation**: Built-in request validation

## 🧪 Testing

- Unit tests co-located with implementation
- Table-driven tests for comprehensive coverage
- Test helpers in `testutils` package
- Mock implementations using `mockery`

## 🔄 Event Flow

Each operation follows a consistent event flow:

1. **Request Event**: Operation initiated (e.g., `DepositRequested`)
2. **Validation**: Input validation and business rules
3. **Processing**: Core business logic
4. **Persistence**: State changes saved
5. **Response Event**: Operation result published

## 📚 Dependencies

- `github.com/gofiber/fiber/v2` - HTTP server
- `github.com/google/uuid` - ID generation
- `github.com/stretchr/testify` - Testing utilities

## 🏗️ Design Principles

- **Single Responsibility**: Each handler does one thing
- **Dependency Injection**: Dependencies passed explicitly
- **Immutability**: State changes through events
- **Error Handling**: Consistent error responses
