---
icon: material/link
---
# API Endpoints & Usage

The Fintech Platform exposes a comprehensive RESTful API for all its functionalities. The API design prioritizes clear resource naming, standard HTTP methods, and meaningful status codes.

## 📄 OpenAPI Specification

A detailed OpenAPI (Swagger) specification is available at [api/openapi.yaml](api/openapi.yaml). This file can be used with tools like Swagger UI to explore and test the API interactively.

## 📝 Example Requests

You can find practical examples of API requests in the [requests](requests/account.http) file, which can be executed directly using IDE extensions like the REST Client for VS Code or similar tools.

### Webhook Endpoints ⚡

- `POST /webhook/payment-status`: Receives asynchronous payment status updates from payment providers. This endpoint is called by the provider (or mock) to confirm payment completion or failure. **(Event-driven, not called by end users)**

### Asynchronous/Event-Driven Endpoints

- `POST /account/:id/deposit` and `POST /account/:id/withdraw` now initiate a transaction and return immediately. The account balance is updated only after the corresponding webhook event is received and processed.

### Authentication 🔑

- `POST /login`: Authenticates a user with their credentials (username/email and password) and returns a JSON Web Token (JWT) upon successful authentication. This token must be included in the `Authorization` header for all protected endpoints. 🔐

### User Management 👤

- `POST /user`: Registers a new user in the system. ➕
- `GET /user/:id`: Retrieves the profile details of a specific user by their ID. **(Protected)** 🔍
- `PUT /user/:id`: Updates the profile information for a specific user. **(Protected)** ✏️
- `DELETE /user/:id`: Deletes a user account from the system. **(Protected)** 🗑️

### Account Operations 💳

- `POST /account`: Creates a new financial account linked to the authenticated user. **(Protected)** 🆕
- `POST /account/:id/deposit`: Initiates a deposit of funds into the specified account. **(Protected)** ⬆️
- `POST /account/:id/withdraw`: Processes a withdrawal of funds from the specified account, subject to balance availability. **(Protected)** ⬇️
- `GET /account/:id/balance`: Fetches the current balance of the specified account. **(Protected)** 💲
- `GET /account/:id/transactions`: Retrieves a list of all transactions associated with the specified account. **(Protected)** 📜

## 🚨 Error Handling

The API follows RESTful conventions for error responses and uses consistent error handling patterns:

### HTTP Status Codes

- **200 OK**: Request successful
- **201 Created**: Resource created successfully
- **204 No Content**: Request successful, no content to return
- **400 Bad Request**: Invalid request data or validation errors
- **401 Unauthorized**: Authentication required or invalid credentials
- **403 Forbidden**: Authenticated but not authorized for the resource
- **404 Not Found**: Resource not found
- **422 Unprocessable Entity**: Business rule violations (e.g., insufficient funds)
- **429 Too Many Requests**: Rate limit exceeded
- **500 Internal Server Error**: Unexpected server error

### Error Response Format

All error responses follow the RFC 9457 Problem Details format:

```json
{
  "type": "about:blank",
  "title": "Error Title",
  "status": 404,
  "detail": "Detailed error message",
  "instance": "/user/123"
}
```

### Common Error Scenarios

- **User Not Found**: Returns 404 when attempting to update/delete a non-existent user
- **Account Not Found**: Returns 404 when accessing non-existent accounts
- **Insufficient Funds**: Returns 422 when withdrawal amount exceeds balance
- **Invalid Currency**: Returns 422 for unsupported currency codes
- **Unauthorized Access**: Returns 403 when users try to access other users' resources
- **Validation Errors**: Returns 400 with detailed field-specific error messages
