# Registry System

A flexible, extensible registry system for managing entities with support for caching, persistence, validation, events, and metrics.

## Features

- **Flexible Entity Management**: Register, retrieve, update, and unregister entities
- **Caching**: Built-in memory caching with TTL support
- **Persistence**: File-based persistence with automatic save/load
- **Validation**: Customizable validation rules for entities and metadata
- **Events**: Event-driven architecture with observer pattern
- **Metrics**: Built-in metrics collection and monitoring
- **Health Monitoring**: Health status tracking and error reporting
- **Search**: Full-text search and metadata-based filtering
- **Lifecycle Management**: Activate/deactivate entities
- **Thread Safety**: Concurrent access support with proper locking

## Architecture

The registry system follows clean architecture principles with clear separation of concerns:

```
┌─────────────────────────────────────────────────────────────┐
│                    Registry System                          │
├─────────────────────────────────────────────────────────────┤
│  Interface Layer (Abstractions)                             │
│  ├── Entity                                                 │
│  ├── RegistryProvider                                       │
│  ├── RegistryObserver                                       │
│  ├── RegistryEventBus                                       │
│  ├── RegistryValidator                                      │
│  ├── RegistryPersistence                                    │
│  ├── RegistryCache                                          │
│  ├── RegistryMetrics                                        │
│  └── RegistryHealth                                         │
├─────────────────────────────────────────────────────────────┤
│  Implementation Layer                                       │
│  ├── EnhancedRegistry                                       │
│  ├── BaseEntity                                             │
│  ├── MemoryCache                                            │
│  ├── FilePersistence                                        │
│  ├── SimpleMetrics                                          │
│  ├── SimpleEventBus                                         │
│  ├── SimpleValidator                                        │
│  └── SimpleHealth                                           │
├─────────────────────────────────────────────────────────────┤
│  Factory Layer                                              │
│  ├── RegistryFactory                                        │
│  ├── RegistryBuilder                                        │
│  └── Convenience Functions                                  │
└─────────────────────────────────────────────────────────────┘
```

## Quick Start

### Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "time"
    "github.com/amirasaad/fintech/pkg/registry"
)

func main() {
    // Create a basic registry
    registry := registry.NewBasicRegistry()
    ctx := context.Background()

    // Register an entity
    user := registry.NewBaseEntity("user-1", "John Doe")
    user.GetMetadata()["email"] = "john@example.com"
    
    err := registry.Register(ctx, user)
    if err != nil {
        panic(err)
    }

    // Retrieve the entity
    retrieved, err := registry.Get(ctx, "user-1")
    if err != nil {
        panic(err)
    }

    fmt.Printf("Found user: %s\n", retrieved.GetName())
}
```

### Advanced Configuration

```go
// Use the builder pattern for complex configuration
registry, err := registry.NewRegistryBuilder().
    WithName("production-registry").
    WithMaxEntities(10000).
    WithCache(1000, 10*time.Minute).
    WithPersistence("/data/entities.json", 30*time.Second).
    WithValidation([]string{"email", "role"}, []string{"password"}).
    BuildRegistry()

if err != nil {
    panic(err)
}
```

### Custom Entity Implementation

```go
type Product struct {
    *registry.BaseEntity
    Price    float64
    Category string
    InStock  bool
}

func NewProduct(id, name string, price float64, category string) *Product {
    return &Product{
        BaseEntity: registry.NewBaseEntity(id, name),
        Price:      price,
        Category:   category,
        InStock:    true,
    }
}

func (p *Product) GetMetadata() map[string]string {
    metadata := p.BaseEntity.GetMetadata()
    metadata["price"] = fmt.Sprintf("%.2f", p.Price)
    metadata["category"] = p.Category
    metadata["in_stock"] = fmt.Sprintf("%t", p.InStock)
    return metadata
}
```

## Core Concepts

### Entity Interface

All entities must implement the `Entity` interface:

```go
type Entity interface {
    GetID() string
    GetName() string
    IsActive() bool
    GetMetadata() map[string]string
    GetCreatedAt() time.Time
    GetUpdatedAt() time.Time
}
```

### Registry Provider

The main interface for registry operations:

```go
type RegistryProvider interface {
    // Core operations
    Register(ctx context.Context, entity Entity) error
    Get(ctx context.Context, id string) (Entity, error)
    Unregister(ctx context.Context, id string) error
    IsRegistered(ctx context.Context, id string) bool
    
    // Listing operations
    List(ctx context.Context) ([]Entity, error)
    ListActive(ctx context.Context) ([]Entity, error)
    ListByMetadata(ctx context.Context, key, value string) ([]Entity, error)
    
    // Counting operations
    Count(ctx context.Context) (int, error)
    CountActive(ctx context.Context) (int, error)
    
    // Metadata operations
    GetMetadata(ctx context.Context, id, key string) (string, error)
    SetMetadata(ctx context.Context, id, key, value string) error
    RemoveMetadata(ctx context.Context, id, key string) error
    
    // Lifecycle operations
    Activate(ctx context.Context, id string) error
    Deactivate(ctx context.Context, id string) error
    
    // Search operations
    Search(ctx context.Context, query string) ([]Entity, error)
    SearchByMetadata(ctx context.Context, metadata map[string]string) ([]Entity, error)
}
```

## Configuration

### RegistryConfig

```go
type RegistryConfig struct {
    // General settings
    Name                string        `json:"name"`
    MaxEntities         int           `json:"max_entities"`
    DefaultTTL          time.Duration `json:"default_ttl"`
    EnableEvents        bool          `json:"enable_events"`
    EnableValidation    bool          `json:"enable_validation"`
    
    // Performance settings
    CacheSize           int           `json:"cache_size"`
    CacheTTL            time.Duration `json:"cache_ttl"`
    EnableCompression   bool          `json:"enable_compression"`
    
    // Persistence settings
    EnablePersistence   bool          `json:"enable_persistence"`
    PersistencePath     string        `json:"persistence_path"`
    AutoSaveInterval    time.Duration `json:"auto_save_interval"`
    
    // Validation settings
    RequiredMetadata    []string      `json:"required_metadata"`
    ForbiddenMetadata   []string      `json:"forbidden_metadata"`
    MetadataValidators  map[string]func(string) error `json:"-"`
}
```

## Usage Patterns

### 1. Basic Registry

```go
registry := registry.NewBasicRegistry()
```

### 2. Cached Registry

```go
registry := registry.NewCachedRegistry(1000, 5*time.Minute)
```

### 3. Persistent Registry

```go
registry, err := registry.NewPersistentRegistry("/data/entities.json")
```

### 4. Monitored Registry

```go
registry := registry.NewMonitoredRegistry("my-registry")
```

### 5. Custom Configuration

```go
config := registry.RegistryConfig{
    Name:             "custom-registry",
    MaxEntities:      5000,
    EnableEvents:     true,
    EnableValidation: true,
    CacheSize:        500,
    CacheTTL:         5 * time.Minute,
}

registry := registry.NewEnhancedRegistry(config)
```

## Advanced Features

### Event Handling

```go
// Create an observer
type MyObserver struct{}

func (o *MyObserver) OnEntityRegistered(ctx context.Context, entity registry.Entity) {
    fmt.Printf("Entity registered: %s\n", entity.GetName())
}

func (o *MyObserver) OnEntityUnregistered(ctx context.Context, id string) {
    fmt.Printf("Entity unregistered: %s\n", id)
}

// Subscribe to events
observer := &MyObserver{}
// In a real implementation, you'd subscribe to the event bus
```

### Custom Validation

```go
validator := registry.NewSimpleValidator().
    WithRequiredMetadata([]string{"email", "age"}).
    WithForbiddenMetadata([]string{"password"}).
    WithValidator("email", validateEmail).
    WithValidator("age", validateAge)

registry.WithValidator(validator)
```

### Metrics Collection

```go
metrics := registry.NewSimpleMetrics()
registry.WithMetrics(metrics)

// Later, get statistics
stats := metrics.GetStats()
fmt.Printf("Registrations: %d\n", stats["registrations"])
fmt.Printf("Lookups: %d\n", stats["lookups"])
```

### Health Monitoring

```go
health := registry.NewSimpleHealth()
registry.WithHealth(health)

// Check health status
if health.IsHealthy(ctx) {
    fmt.Println("Registry is healthy")
} else {
    status := health.GetHealthStatus(ctx)
    fmt.Printf("Health status: %+v\n", status)
}
```

## Performance Considerations

### Caching Strategy

- Use appropriate cache sizes based on entity count
- Set reasonable TTL values to balance memory usage and performance
- Monitor cache hit rates through metrics

### Persistence Strategy

- Use auto-save intervals appropriate for your use case
- Consider compression for large datasets
- Implement backup strategies for critical data

### Concurrency

- The registry is thread-safe for concurrent access
- Use appropriate context timeouts for operations
- Consider connection pooling for external dependencies

## Best Practices

### 1. Entity Design

- Keep entities lightweight and focused
- Use metadata for flexible attributes
- Implement proper validation rules

### 2. Configuration

- Use the builder pattern for complex configurations
- Set appropriate limits for your use case
- Enable features only when needed

### 3. Error Handling

- Always check returned errors
- Implement proper logging for debugging
- Use health monitoring for production systems

### 4. Performance

- Monitor metrics regularly
- Use caching appropriately
- Consider persistence requirements

### 5. Security

- Validate all inputs
- Use appropriate access controls
- Sanitize metadata values

## Examples

See the `examples.go` file for comprehensive examples including:

- Basic registry usage
- Persistent storage
- Caching strategies
- Event-driven patterns
- Custom validation
- Factory patterns
- Advanced search
- Health monitoring
- Performance benchmarking

## Currency-Specific Example

See the `currency_example.go` file for a practical example using the registry for currency management, including:

- Currency entity implementation
- Currency-specific validation
- Search and filtering
- Lifecycle management
- Performance testing
- Event handling

## Testing

The registry system includes comprehensive tests covering:

- Basic operations
- Caching behavior
- Validation rules
- Event handling
- Persistence operations
- Performance characteristics
- Error conditions

Run tests with:

```bash
go test ./pkg/registry/... -v
```

## Contributing

When contributing to the registry system:

1. Follow the established patterns
2. Add tests for new features
3. Update documentation
4. Consider performance implications
5. Maintain backward compatibility

## License

This registry system is part of the fintech project and follows the same licensing terms.
