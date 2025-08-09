---
icon: material/link
---
# API Endpoints & Usage

The Fintech Platform exposes a comprehensive RESTful API for all its functionalities. The API design prioritizes clear resource naming, standard HTTP methods, and meaningful status codes.

## 📄 OpenAPI Specification

A detailed OpenAPI (Swagger) specification is available at [api/openapi.yaml](api/openapi.yaml). This file can be used with tools like Swagger UI to explore and test the API interactively.

## 📝 Example Requests

You can find practical examples of API requests in the [requests](requests/account.http) file, which can be executed directly using IDE extensions like the REST Client for VS Code or similar tools.

### ⚡ Webhook Endpoints

- `POST /webhook/payment-status`: Receives asynchronous payment status updates from payment providers. This endpoint is called by the provider (or mock) to confirm payment completion or failure. **(Event-driven, not called by end users)**
  - Requires `X-Webhook-Signature` header for verification
  - Accepts JSON payload with payment status and metadata

### ⚡ Asynchronous/Event-Driven Endpoints

- `POST /account/:id/deposit`: Initiates a deposit transaction
  - Returns `202 ⚡ Accepted` immediately with a `Location` header to track status
  - Requires `amount` and `currency` in the request body
  - Example: `{"amount": 100.50, "currency": "USD"}`

- `POST /account/:id/withdraw`: Initiates a withdrawal transaction
  - Returns `202 Accepted` immediately with a `Location` header to track status
  - Requires `amount` and `currency` in the request body
  - Example: `{"amount": 50.00, "currency": "USD"}`

- `POST /account/transfer`: Initiates a transfer between accounts
  - Returns `202 ⚡ Accepted` immediately with a `Location` header
  - Requires `from_account_id`, `to_account_id`, `amount`, and `currency`
  - Example: `{"from_account_id": "uuid1", "to_account_id": "uuid2", "amount": 75.25, "currency": "USD"}`

### 🔑 Authentication

- `POST /login`: Authenticates a user with their credentials (username/email and password) and returns a JSON Web Token (JWT) upon successful authentication. This token must be included in the `Authorization` header for all protected endpoints. 🔐

### 👤 User Management

- `POST /user`: Registers a new user in the system. ➕
  - Required fields: `username`, `email`, `password`
  - Example: `{"username": "johndoe", "email": "john@example.com", "password": "secure123"}`

- `POST /login`: Authenticates a user and returns a JWT token
  - Required fields: `email` or `username`, and `password`
  - Returns: `{"token": "jwt.token.here", "expires_in": 3600}`

- `GET /user/:id`: Retrieves the profile details of a specific user by their ID. **(Protected)** 🔍
  - Requires valid JWT in `Authorization: Bearer <token>` header
  - Returns user details without sensitive information

- `PUT /user/:id`: Updates the profile information for a specific user. **(Protected)** ✏️
  - Accepts partial updates
  - Example: `{"email": "new.email@example.com"}`

- `DELETE /user/:id`: Deletes a user account from the system. **(Protected)** 🗑️
  - Requires confirmation
  - Cascades to associated accounts and transactions

### 💳 Account Operations

- `POST /account`: Creates a new financial account. **(Protected)** 🆕
  - Required fields: `currency` (3-letter ISO code)
  - Example: `{"currency": "USD"}`

- `GET /account/:id`: Retrieves account details by ID. **(Protected)** 🔍
  - Returns balance, currency, and metadata

- `GET /accounts`: Lists all accounts for the authenticated user. **(Protected)** 📋
  - Supports pagination with `limit` and `offset` query params

- `GET /account/:id/balance`: Fetches the current balance. **(Protected)** 💲
  - Returns: `{"account_id": "uuid", "balance": 100.50, "currency": "USD"}`

- `GET /account/:id/transactions`: Retrieves transaction history. **(Protected)** 📜
  - Supports filtering by date range and transaction type
  - Example: `/account/123/transactions?from=2025-01-01&to=2025-12-31`

### 💰 Transaction Operations

- `GET /transactions`: Lists all transactions for the authenticated user. **(Protected)** 📋
  - Supports filtering by account, type, and status
  - Pagination with `limit` and `offset`

- `GET /transaction/:id`: Retrieves details of a specific transaction. **(Protected)** 🔍
  - Shows full transaction details and status

### 🌐 Currency Operations

- `GET /currencies`: Lists all supported currencies
- `GET /exchange-rate`: Gets current exchange rate between two currencies
  - Parameters: `from` (required), `to` (required)
  - Example: `/exchange-rate?from=USD&to=EUR`

## 🚨 Error Handling

The API follows RESTful conventions for error responses and uses consistent error handling patterns:

### HTTP Status Codes

| Status Code | Description |
|-------------|-------------|
| **200 OK** | Request succeeded |
| **201 Created** | Resource created successfully |
| **202 Accepted** | Request accepted for processing |
| **204 No Content** | Request succeeded, no content to return |
| **400 Bad Request** | Invalid request data or validation errors |
| **401 Unauthorized** | Authentication required or invalid credentials |
| **403 Forbidden** | Authenticated but not authorized |
| **404 Not Found** | Resource not found |
| **409 Conflict** | Resource conflict (e.g., duplicate email) |
| **422 Unprocessable Entity** | Business rule violations |
| **429 Too Many Requests** | Rate limit exceeded |
| **500 Internal Server Error** | Unexpected server error |

### Error Response Format

All error responses follow the RFC 9457 Problem Details format:

```json
{
  "type": "https://example.com/errors/invalid-currency",
  "title": "Invalid Currency",
  "status": 422,
  "detail": "Currency 'XYZ' is not supported. Supported currencies: USD, EUR, GBP, JPY",
  "instance": "/account/123/deposit",
  "errors": [
    {
      "field": "currency",
      "message": "must be a valid currency code"
    }
  ]
}
```

### Common Error Scenarios

#### Authentication & Authorization (4xx)

- **401 Unauthorized**
  - Missing or invalid JWT token
  - Expired or revoked token
  - Invalid credentials on login

- **403 Forbidden**
  - User lacks required permissions
  - Attempt to access another user's resources
  - Account disabled or locked

#### Client Errors (4xx)

- **400 Bad Request**
  - Invalid JSON in request body
  - Missing required fields
  - Invalid field formats (email, UUID, etc.)
  - Invalid query parameters

- **404 Not Found**
  - User/Account/Transaction not found
  - Invalid resource ID format
  - Deleted resources

- **409 Conflict**
  - Duplicate email/username
  - Concurrent modification
  - Resource already exists

- **422 Unprocessable Entity**
  - Insufficient funds
  - Invalid currency conversion
  - Business rule violations
  - Invalid transaction amount

#### Rate Limiting (429)

- Too many requests from this IP
- Too many failed login attempts
- API quota exceeded

### Error Codes Reference

| Code | Description | HTTP Status |
|------|-------------|-------------|
| `AUTH_REQUIRED` | Authentication required | 401 |
| `INVALID_CREDENTIALS` | Invalid username/password | 401 |
| `ACCESS_DENIED` | Insufficient permissions | 403 |
| `RESOURCE_NOT_FOUND` | Requested resource not found | 404 |
| `DUPLICATE_ENTRY` | Resource already exists | 409 |
| `INSUFFICIENT_FUNDS` | Not enough balance | 422 |
| `INVALID_CURRENCY` | Unsupported currency | 422 |
| `VALIDATION_ERROR` | Request validation failed | 400 |
| `RATE_LIMIT_EXCEEDED` | Too many requests | 429 |
| `INTERNAL_ERROR` | Server error | 500 |

### Best Practices for Error Handling

1. Always check the status code first
2. Parse the error response for details
3. Display user-friendly messages based on error codes
4. Handle rate limiting with exponential backoff
5. Log full error details for debugging
6. Implement retry logic for transient errors

### Example: Handling Errors in JavaScript

```javascript
async function makeRequest(url, options = {}) {
  try {
    const response = await fetch(url, {
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${token}`
      },
      ...options
    });

    if (!response.ok) {
      const error = await response.json();
      // Handle error
      throw error;
    }

    return await response.json();
  } catch (error) {
    console.error('API request failed:', error);
    throw error;
  }
}
```
