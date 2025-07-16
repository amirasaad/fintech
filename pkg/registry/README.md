# 🗃️ Registry Package

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

---

## 🧰 Usage Examples

### 🧪 Registering an Entity

```go
user := registry.NewBaseEntity("user-1", "John Doe")
user.Metadata()["email"] = "john@example.com"
err := registry.Register(ctx, user)
```

### 🧪 Custom Registry with Caching & Persistence

```go
reg, err := registry.NewRegistryBuilder().
    WithName("prod-reg").
    WithCache(1000, 10*time.Minute).
    WithPersistence("/data/entities.json", 30*time.Second).
    BuildRegistry()
```

---

## 🏅 Best Practices

- Use property-style getters (e.g., `Name()`, not `GetName()`)
- Prefer registry interfaces for dependency inversion
- Leverage event bus and observer for decoupled side effects
- Use the builder for complex configuration

---

## 📄 References

- [Full documentation](https://github.com/amirasaad/fintech/blob/main/pkg/registry/README.md)
- [All abstractions](https://github.com/amirasaad/fintech/blob/main/pkg/registry/interface.go)
- [Usage patterns](https://github.com/amirasaad/fintech/blob/main/pkg/registry/examples_test.go)
