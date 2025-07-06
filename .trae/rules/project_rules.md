# Project Rules

## 1. Code Style

* **Indentation:** Use tabs for indentation.
* **Naming Conventions:**
  * Functions and methods: `CamelCase` (e.g., `CreateAccount`, `GetBalance`).
  * Variables: `camelCase` (e.g., `uowFactory`, `request`).
  * Package names: `lowercase`.
* **Comments:** Use comments for documentation, especially for exported functions and complex logic. Follow GoDoc conventions.
* **Error Handling:** Explicitly handle errors, returning appropriate `fiber.Error` or using `ErrorResponseJSON` for HTTP responses.
* **Logging:** Use "slog/log" for structured logging.
* **Git Commit Messages:** Use Conventional Commits.

## 2. Languages and Frameworks

* **Primary Language:** Go
* **Web Framework:** Fiber (github.com/gofiber/fiber/v2)
* **ORM:** GORM (gorm.io/gorm)
* **Authentication:** JWT (github.com/golang-jwt/jwt/v5)
* **Validation:** go-playground/validator (github.com/go-playground/validator)
* **UUID Generation:** github.com/google/uuid

## 3. API Restrictions

* Avoid introducing new external dependencies unless explicitly approved.
* Prioritize using existing project utilities and helper functions (e.g., `ErrorResponseJSON`).
* Do not use `fmt.Print` or `fmt.Println` for logging.

## 4. Development Workflow

* **Test-Driven Development (TDD):** All new features and bug fixes should follow a TDD approach:
    1. Write a failing test that defines the desired behavior or reproduces the bug.
    2. Write the minimum amount of code necessary to make the test pass.
    3. Refactor the code to improve its design, readability, and maintainability, ensuring all tests still pass.
* **Unit Tests:** Write comprehensive unit tests for all new or modified code. Tests should be placed alongside the code they test (e.g., `_test.go` files).
* **Test Coverage:** Strive for high test coverage, especially for core business logic and critical paths.
