# Dynamic Currency System

## Overview

The fintech application supports dynamic currency management, allowing currencies to be added, updated, and configured at runtime without requiring code changes or system restarts.

## Architecture

### 1. Currency Registry (`pkg/currency/`)

The currency registry is the core component that manages currency metadata:

```go
type Meta struct {
    Decimals int    // Number of decimal places (e.g., 2 for USD, 8 for BTC)
    Symbol   string // Currency symbol (e.g., "$", "₿", "€")
}

type Registry struct {
    currencies map[string]Meta
    mu         sync.RWMutex  // Thread-safe access
}
```

### 2. Global Registry

A global registry instance provides convenience functions:

```go
// Global convenience functions
currency.Get("USD")           // Get currency metadata
currency.Register("BTC", meta) // Add/update currency
currency.IsSupported("EUR")   // Check if currency is supported
currency.ListSupported()      // Get all supported currencies
```

## Dynamic Currency Features

### 1. Runtime Currency Registration

Add new currencies without restarting the application:

```go
// Register cryptocurrencies
currency.Register("BTC", currency.Meta{Decimals: 8, Symbol: "₿"})
currency.Register("ETH", currency.Meta{Decimals: 18, Symbol: "Ξ"})

// Register new fiat currencies
currency.Register("BRL", currency.Meta{Decimals: 2, Symbol: "R$"})
```

### 2. Currency Updates

Update existing currency configurations:

```go
// Update USD to support 3 decimal places for micro-transactions
currency.Register("USD", currency.Meta{Decimals: 3, Symbol: "$"})

// Update JPY to support decimal places
currency.Register("JPY", currency.Meta{Decimals: 2, Symbol: "¥"})
```

### 3. Multi-Tenant Support

Different tenants can have different currency configurations:

```go
// Tenant A: Traditional banking
tenantARegistry := currency.NewRegistry()
tenantARegistry.Register("USD", currency.Meta{Decimals: 2, Symbol: "$"})
tenantARegistry.Register("EUR", currency.Meta{Decimals: 2, Symbol: "€"})

// Tenant B: Cryptocurrency exchange
tenantBRegistry := currency.NewRegistry()
tenantBRegistry.Register("BTC", currency.Meta{Decimals: 8, Symbol: "₿"})
tenantBRegistry.Register("ETH", currency.Meta{Decimals: 18, Symbol: "Ξ"})
```

### 4. Graceful Fallback

Unknown currencies return default metadata:

```go
// Unknown currency returns default configuration
unknownInfo := currency.Get("UNKNOWN_CURRENCY")
// Returns: Meta{Decimals: 2, Symbol: "UNKNOWN_CURRENCY"}
```

## Domain Integration

### 1. Account Creation with Dynamic Currencies

```go
// Create accounts with any registered currency
btcAccount, err := domain.NewAccountWithCurrency(userID, "BTC")
ethAccount, err := domain.NewAccountWithCurrency(userID, "ETH")
```

### 2. Money Operations with Dynamic Currencies

```go
// Create money objects with dynamic currencies
btcMoney, err := domain.NewMoney(0.001, "BTC")  // 0.001 BTC
ethMoney, err := domain.NewMoney(0.5, "ETH")    // 0.5 ETH

// Perform operations
_, err = btcAccount.Deposit(userID, btcMoney)
balance, err := btcAccount.GetBalance(userID)
```

### 3. Precision Handling

Each currency maintains its own precision rules:

```go
// USD: 2 decimal places
usdMoney, _ := domain.NewMoney(100.99, "USD")     // Valid
usdMoney, _ := domain.NewMoney(100.999, "USD")    // Error: too many decimals

// JPY: 0 decimal places  
jpyMoney, _ := domain.NewMoney(1000, "JPY")       // Valid
jpyMoney, _ := domain.NewMoney(1000.5, "JPY")     // Error: decimals not allowed

// BTC: 8 decimal places
btcMoney, _ := domain.NewMoney(0.00000001, "BTC") // Valid: 1 satoshi
```

## Real-World Use Cases

### 1. Cryptocurrency Exchange

```go
// Register cryptocurrencies as they become available
currency.Register("BTC", currency.Meta{Decimals: 8, Symbol: "₿"})
currency.Register("ETH", currency.Meta{Decimals: 18, Symbol: "Ξ"})
currency.Register("USDT", currency.Meta{Decimals: 6, Symbol: "₮"})
currency.Register("ADA", currency.Meta{Decimals: 6, Symbol: "₳"})

// Create accounts for each cryptocurrency
btcAccount := domain.NewAccountWithCurrency(userID, "BTC")
ethAccount := domain.NewAccountWithCurrency(userID, "ETH")
```

### 2. International Banking

```go
// Support new national currencies
currency.Register("TRY", currency.Meta{Decimals: 2, Symbol: "₺"}) // Turkish Lira
currency.Register("INR", currency.Meta{Decimals: 2, Symbol: "₹"}) // Indian Rupee
currency.Register("BRL", currency.Meta{Decimals: 2, Symbol: "R$"}) // Brazilian Real
```

### 3. Micro-Transaction Support

```go
// Update USD to support micro-transactions
currency.Register("USD", currency.Meta{Decimals: 3, Symbol: "$"})

// Now supports amounts like $0.001
microMoney, _ := domain.NewMoney(0.001, "USD")
```

### 4. Currency Migration

```go
// Scenario: Migrating from 2 to 3 decimal places for USD
// Step 1: Update the currency configuration
currency.Register("USD", currency.Meta{Decimals: 3, Symbol: "$"})

// Step 2: Existing accounts continue to work
// Step 3: New operations use the updated configuration
newMoney, _ := domain.NewMoney(100.999, "USD") // Now valid
```

## Configuration Management

### 1. Database-Driven Configuration

```go
// Load currencies from database
func LoadCurrenciesFromDB() {
    currencies := db.GetCurrencies()
    for _, curr := range currencies {
        currency.Register(curr.Code, currency.Meta{
            Decimals: curr.Decimals,
            Symbol:   curr.Symbol,
        })
    }
}
```

### 2. Configuration File Support

```go
// Load currencies from config file
func LoadCurrenciesFromConfig(filename string) error {
    data, err := os.ReadFile(filename)
    if err != nil {
        return err
    }
    
    var config struct {
        Currencies map[string]currency.Meta `json:"currencies"`
    }
    
    if err := json.Unmarshal(data, &config); err != nil {
        return err
    }
    
    for code, meta := range config.Currencies {
        currency.Register(code, meta)
    }
    
    return nil
}
```

### 3. Hot Reloading

```go
// Reload currency configuration without restart
func HotReloadCurrencies() {
    // Clear existing currencies (be careful with this in production)
    // Reload from source
    LoadCurrenciesFromDB()
    log.Println("Currency configuration reloaded")
}
```

## Thread Safety

The currency registry is fully thread-safe:

```go
// Concurrent reads are safe
go func() {
    info := currency.Get("USD")
    // Use info...
}()

// Concurrent writes are safe
go func() {
    currency.Register("NEW", currency.Meta{Decimals: 2, Symbol: "N"})
}()
```

## Error Handling

### 1. Invalid Currency Codes

```go
// Invalid currency format
account, err := domain.NewAccountWithCurrency(userID, "INVALID_CODE")
if err != nil {
    // err == domain.ErrInvalidCurrencyCode
}
```

### 2. Precision Validation

```go
// Too many decimal places
money, err := domain.NewMoney(100.999, "USD")
if err != nil {
    // err contains precision error message
}
```

### 3. Currency Mismatch

```go
// Cannot add different currencies
usdMoney, _ := domain.NewMoney(100, "USD")
eurMoney, _ := domain.NewMoney(100, "EUR")
sum, err := usdMoney.Add(eurMoney)
if err != nil {
    // err == domain.ErrInvalidCurrencyCode
}
```

## Performance Considerations

### 1. Registry Lookups

- Currency lookups are O(1) hash map operations
- Thread-safe with minimal lock contention
- Global registry uses read-write mutex for optimal performance

### 2. Memory Usage

- Each currency metadata is small (~24 bytes)
- Registry grows linearly with number of currencies
- Default currencies are pre-loaded for fast access

### 3. Concurrency

- Read operations use read locks (shared access)
- Write operations use write locks (exclusive access)
- Designed for high read-to-write ratios

## Best Practices

### 1. Currency Registration

```go
// Register currencies early in application startup
func init() {
    // Register default currencies
    currency.Register("USD", currency.Meta{Decimals: 2, Symbol: "$"})
    currency.Register("EUR", currency.Meta{Decimals: 2, Symbol: "€"})
    
    // Load additional currencies from configuration
    LoadCurrenciesFromConfig("currencies.json")
}
```

### 2. Validation

```go
// Always validate currency codes before use
if !currency.IsSupported(currencyCode) {
    return fmt.Errorf("unsupported currency: %s", currencyCode)
}
```

### 3. Error Handling

```go
// Handle currency-related errors gracefully
money, err := domain.NewMoney(amount, currencyCode)
if err != nil {
    // Log error and return appropriate response
    return fmt.Errorf("invalid money amount: %w", err)
}
```

## Testing

### 1. Unit Tests

```go
func TestDynamicCurrency(t *testing.T) {
    // Register test currency
    currency.Register("TEST", currency.Meta{Decimals: 2, Symbol: "T"})
    
    // Test account creation
    account, err := domain.NewAccountWithCurrency(userID, "TEST")
    assert.NoError(t, err)
    
    // Test money operations
    money, err := domain.NewMoney(100.50, "TEST")
    assert.NoError(t, err)
    
    // Test account operations
    _, err = account.Deposit(userID, money)
    assert.NoError(t, err)
}
```

### 2. Integration Tests

```go
func TestCurrencyHotReload(t *testing.T) {
    // Test initial state
    money1, _ := domain.NewMoney(100.99, "USD")
    
    // Update currency configuration
    currency.Register("USD", currency.Meta{Decimals: 3, Symbol: "$"})
    
    // Test new configuration
    money2, _ := domain.NewMoney(100.999, "USD")
    assert.NotEqual(t, money1.String(), money2.String())
}
```

## Conclusion

The dynamic currency system provides:

1. **Flexibility**: Add currencies without code changes
2. **Scalability**: Support for unlimited currencies
3. **Thread Safety**: Safe concurrent access
4. **Performance**: Fast lookups and minimal overhead
5. **Maintainability**: Clean separation of concerns
6. **Extensibility**: Easy to add new features

This system enables the fintech application to adapt to changing business requirements and support new currencies as they emerge in the market. 