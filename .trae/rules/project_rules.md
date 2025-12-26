# Project Rules

- Tabs for indentation.
- Names: funcs/methods CamelCase; vars camelCase; packages lowercase.
- Document exported symbols with GoDoc.
- Handle errors explicitly; return fiber.Error or ErrorResponseJSON.
- Logging: use slog (no fmt.Print/Println).
- Stack: Go, Fiber, GORM, JWT (jwt/v5), validator, uuid.
- Avoid new deps unless approved; reuse existing helpers.
- TDD: write failing test → minimal fix → refactor; keep strong coverage.
- Commits: Conventional Commits.
