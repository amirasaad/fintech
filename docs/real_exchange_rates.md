# Real Exchange Rates Integration

## Overview

The fintech application now supports real-time exchange rates through integration with external APIs. This feature provides accurate currency conversion for multi-currency transactions while maintaining high availability through caching and fallback mechanisms.

## Architecture

### Components

1. **Exchange Rate Providers** - External API integrations
2. **Exchange Rate Service** - Orchestrates providers and caching
3. **Real Currency Converter** - Implements the domain interface
4. **Cache Layer** - In-memory caching for performance
5. **Configuration System** - Environment-based settings

### Design Principles

- **High Availability** - Multiple providers with fallback
- **Performance** - Intelligent caching with TTL
- **Reliability** - Health checks and error handling
- **Security** - API key management and rate limiting
- **Observability** - Comprehensive logging

## Configuration

### Environment Variables

```bash
# ExchangeRate API Configuration
EXCHANGE_RATE_API_KEY=your_api_key_here
EXCHANGE_RATE_API_URL=https://api.exchangerate-api.com/v4/latest

# Cache Settings
EXCHANGE_RATE_CACHE_TTL=15m
EXCHANGE_RATE_FALLBACK_TTL=1h

# HTTP Settings
EXCHANGE_RATE_HTTP_TIMEOUT=10s
EXCHANGE_RATE_MAX_RETRIES=3

# Rate Limiting
EXCHANGE_RATE_REQUESTS_PER_MINUTE=60
EXCHANGE_RATE_BURST_SIZE=10

# Fallback Settings
EXCHANGE_RATE_ENABLE_FALLBACK=true
```

### Default Values

- **Cache TTL**: 15 minutes
- **HTTP Timeout**: 10 seconds
- **Max Retries**: 3
- **Requests per Minute**: 60
- **Burst Size**: 10
- **Fallback Enabled**: true

## Supported Providers

### ExchangeRate API (Primary)

- **URL**: <https://api.exchangerate-api.com/v4/latest>
- **Authentication**: API Key (optional for free tier)
- **Rate Limits**: 1000 requests/month (free), 100,000 requests/month (paid)
- **Update Frequency**: Daily
- **Coverage**: 170+ currencies

### Fallback Provider

When external APIs are unavailable, the system falls back to a stub converter with predefined rates:

```go
rates := map[string]map[string]float64{
    "USD": {"EUR": 0.84, "GBP": 0.76, "JPY": 0.0027},
    "EUR": {"USD": 1.19, "GBP": 0.90, "JPY": 0.0024},
    "GBP": {"USD": 1.32, "EUR": 1.11, "JPY": 0.0024},
    "JPY": {"USD": 0.0027, "EUR": 0.0024, "GBP": 0.0024},
}
```

## Usage Examples

### Basic Currency Conversion

```go
// Get a single exchange rate
rate, err := exchangeRateService.GetRate("USD", "EUR")
if err != nil {
    // Handle error
}
fmt.Printf("1 USD = %.4f EUR\n", rate.Rate)

// Convert an amount
converter := NewRealCurrencyConverter(exchangeRateService, fallback, logger)
conversion, err := converter.Convert(100.0, "USD", "EUR")
if err != nil {
    // Handle error
}
fmt.Printf("100 USD = %.2f EUR (rate: %.4f)\n", 
    conversion.ConvertedAmount, conversion.ConversionRate)
```

### Multiple Currency Conversion

```go
// Get rates for multiple currencies
rates, err := exchangeRateService.GetRates("USD", []string{"EUR", "GBP", "JPY"})
if err != nil {
    // Handle error
}

for currency, rate := range rates {
    fmt.Printf("USD to %s: %.4f\n", currency, rate.Rate)
}
```

## API Response Examples

### Successful Conversion

```json
{
  "status": 200,
  "message": "Deposit successful (converted)",
  "data": {
    "transaction": {
      "id": "123e4567-e89b-12d3-a456-426614174000",
      "user_id": "user-uuid",
      "account_id": "account-uuid",
      "amount": 120.0,
      "balance": 220.0,
      "created_at": "2024-05-01T12:00:00Z",
      "currency": "USD",
      "original_amount": 100.0,
      "original_currency": "EUR",
      "conversion_rate": 1.2
    },
    "original_amount": 100.0,
    "original_currency": "EUR",
    "converted_amount": 120.0,
    "converted_currency": "USD",
    "conversion_rate": 1.2
  }
}
```

### Error Response (Service Unavailable)

```json
{
  "status": 503,
  "message": "Exchange rate service unavailable",
  "data": null
}
```

## Error Handling

### Error Types

- `ErrExchangeRateUnavailable` - All providers are down
- `ErrUnsupportedCurrencyPair` - Currency pair not supported
- `ErrExchangeRateExpired` - Cached rate has expired
- `ErrExchangeRateInvalid` - Invalid rate received from provider

### Fallback Strategy

1. **Cache First** - Check for valid cached rates
2. **Provider Chain** - Try providers in order of preference
3. **Stub Fallback** - Use predefined rates if all else fails
4. **Error Response** - Return appropriate error if no fallback available

## Performance Considerations

### Caching Strategy

- **TTL-based**: Rates cached for 15 minutes by default
- **Automatic Cleanup**: Expired entries removed every 5 minutes
- **Memory Efficient**: Thread-safe in-memory storage

### Rate Limiting

- **Per-minute limits**: Configurable request limits
- **Burst protection**: Prevents API abuse
- **Provider-specific**: Different limits per provider

### Health Checks

- **Provider Monitoring**: Regular health checks
- **Automatic Failover**: Unhealthy providers skipped
- **Recovery**: Providers retried on next request

## Monitoring and Logging

### Log Levels

- **INFO**: Successful conversions and provider status
- **WARN**: Provider failures and fallback usage
- **ERROR**: Critical failures and configuration issues
- **DEBUG**: Detailed request/response information

### Key Metrics

- Conversion success rate
- Provider response times
- Cache hit ratio
- Fallback usage frequency
- Error rates by provider

## Security Considerations

### API Key Management

- **Environment Variables**: Secure storage of API keys
- **Masked Logging**: API keys partially masked in logs
- **Rotation Support**: Easy key rotation through config

### Data Validation

- **Rate Validation**: Ensure rates are positive and finite
- **Currency Validation**: Validate ISO 4217 codes
- **TTL Validation**: Ensure reasonable expiration times

## Deployment

### Production Setup

1. **API Key Configuration**:

   ```bash
   export EXCHANGE_RATE_API_KEY="your_production_key"
   ```

2. **Cache Configuration**:

   ```bash
   export EXCHANGE_RATE_CACHE_TTL="10m"
   export EXCHANGE_RATE_FALLBACK_TTL="30m"
   ```

3. **Rate Limiting**:

   ```bash
   export EXCHANGE_RATE_REQUESTS_PER_MINUTE="100"
   export EXCHANGE_RATE_BURST_SIZE="20"
   ```

### Development Setup

For development without API keys:

```bash
# No API key needed - will use fallback
export EXCHANGE_RATE_ENABLE_FALLBACK=true
```

## Testing

### Unit Tests

```bash
go test ./pkg/infrastructure -v
```

### Integration Tests

```bash
# Test with real API (requires API key)
EXCHANGE_RATE_API_KEY=your_key go test ./pkg/infrastructure -v -tags=integration
```

### Mock Testing

```go
// Use mock providers for testing
mockProvider := &MockExchangeRateProvider{}
mockProvider.On("GetRate", "USD", "EUR").Return(&provider.ExchangeRate{
    Rate: 0.85,
    // ... other fields
}, nil)

service := NewExchangeRateService([]provider.ExchangeRateProvider{mockProvider}, cache, logger)
```

## Future Enhancements

### Planned Features

1. **Redis Cache** - Distributed caching for multi-instance deployments
2. **Additional Providers** - Support for more exchange rate APIs
3. **Historical Rates** - Store and retrieve historical exchange rates
4. **Rate Alerts** - Notifications for significant rate changes
5. **Analytics Dashboard** - Monitor conversion patterns and costs

### Provider Integration

Easy to add new providers by implementing the `provider.ExchangeRateProvider` interface:

```go
type MyProvider struct {
    // Provider-specific fields
}

func (p *MyProvider) GetRate(from, to string) (*provider.ExchangeRate, error) {
    // Implementation
}

func (p *MyProvider) Name() string {
    return "my-provider"
}

func (p *MyProvider) IsHealthy() bool {
    // Health check implementation
}
```

## Troubleshooting

### Common Issues

1. **API Key Issues**:
   - Verify API key is correct
   - Check API key permissions
   - Ensure key is not expired

2. **Rate Limiting**:
   - Monitor request counts
   - Adjust rate limits if needed
   - Consider upgrading API plan

3. **Cache Issues**:
   - Check cache TTL settings
   - Monitor memory usage
   - Verify cache cleanup is working

4. **Provider Failures**:
   - Check provider health status
   - Review error logs
   - Verify network connectivity

### Debug Mode

Enable debug logging for detailed troubleshooting:

```go
logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelDebug,
}))
```
