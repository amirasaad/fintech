---
icon: material/lightbulb
---

# Examples

Here are some examples demonstrating how to interact with the Fintech Platform.

## 🖥️ CLI Interaction

1. **Start the CLI:**

   ```bash
   go run cmd/cli/main.go
   ```

2. **Login (when prompted):**

   ```bash
   # Enter your username/email and password when prompted
   ```

3. **Create an account:**

   ```bash
   > create
   Account created: ID=xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx, Balance=0.00
   ```

4. **Deposit funds:**

   ```bash
   > deposit xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx 100.50
   Deposited 100.50 to account xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx. New balance: 100.50
   ```

5. **Check balance:**

   ```bash
   > balance xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
   Account xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx balance: 100.50
   ```

## 🌐 API Interaction (using `curl`)

1. **Register a new user:**

   ```bash
   curl -X POST http://localhost:3000/user \
     -H "Content-Type: application/json" \
     -d '{"username":"apiuser","email":"api@example.com","password":"apipassword"}'
   ```

2. **Login to get a JWT token:**

   ```bash
   curl -X POST http://localhost:3000/login \
     -H "Content-Type: application/json" \
     -d '{"identity":"apiuser","password":"apipassword"}'
   # Copy the token from the response for subsequent requests
   ```

3. **Create an account (using the JWT token):**

   ```bash
   curl -X POST http://localhost:3000/account \
     -H "Authorization: Bearer YOUR_JWT_TOKEN"
   ```

4. **Deposit funds into the account:**

   ```bash
   curl -X POST http://localhost:3000/account/YOUR_ACCOUNT_ID/deposit \
     -H "Content-Type: application/json" \
     -H "Authorization: Bearer YOUR_JWT_TOKEN" \
     -d '{"amount": 500.75}'
   # Note: This creates a pending transaction. The account balance will only update after the payment provider (or mock) calls the webhook endpoint to confirm payment completion.
   ```

5. **Get account balance:**

   ```bash
   curl -X GET http://localhost:3000/account/YOUR_ACCOUNT_ID/balance \
     -H "Authorization: Bearer YOUR_JWT_TOKEN"
   ```
