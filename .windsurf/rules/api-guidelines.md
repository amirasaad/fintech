---
trigger: model_decision
description: API development guidelines for HTTP handlers and endpoints
globs: webapi/*.go
---

# API Guidelines

## HTTP Handler Patterns

- Use existing `ErrorResponseJSON` utility for HTTP responses - see [webapi/utils.go](mdc:webapi/utils.go)
- Return appropriate `fiber.Error` for error handling
- Follow RESTful conventions
- Implement proper input validation
- Use structured logging for all operations

## API Structure

- Main app setup: [webapi/app.go](mdc:webapi/app.go)
- Account endpoints: [webapi/account.go](mdc:webapi/account.go)
- User endpoints: [webapi/user.go](mdc:webapi/user.go)
- Authentication: [webapi/auth.go](mdc:webapi/auth.go)
- Test utilities: [webapi/test_utils.go](mdc:webapi/test_utils.go)

## Security Guidelines

- Use JWT for authentication - see [pkg/middleware/auth.go](mdc:pkg/middleware/auth.go)
- Validate all inputs with go-playground/validator
- Never expose sensitive data in logs/responses
- Implement rate limiting
- Use middleware for auth checks

## Validation

- Validate all user inputs
- Use go-playground/validator for struct validation
- Return clear validation error messages
- Sanitize inputs to prevent injection attacks

## Error Handling

- Use `ErrorResponseJSON` for consistent error responses
- Return appropriate HTTP status codes
- Log errors with context
- Provide meaningful error messages to clients

## Performance

- Use GORM preloading for related data
- Implement proper caching strategies
- Use pagination for large datasets
- Optimize response payloads
- Monitor and log performance metrics

## Examples

- Handler pattern: [webapi/account.go](mdc:webapi/account.go)
- Auth middleware: [pkg/middleware/auth.go](mdc:pkg/middleware/auth.go)
- Error utilities: [webapi/utils.go](mdc:webapi/utils.go)
