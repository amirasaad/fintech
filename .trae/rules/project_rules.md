# Project Rules for Trae AI Agent

These guidelines define the conventions and practices to be followed by the AI agent for the `fintech` project.

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
