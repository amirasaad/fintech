package registry

import (
	"context"
	"fmt"
	"time"
)

// RegistryFactoryImpl implements RegistryFactory
type RegistryFactoryImpl struct{}

// NewRegistryFactory creates a new registry factory
func NewRegistryFactory() RegistryFactory {
	return &RegistryFactoryImpl{}
}

// Create creates a basic registry with the given configuration
func (f *RegistryFactoryImpl) Create(ctx context.Context, config RegistryConfig) (RegistryProvider, error) {
	registry := NewEnhancedRegistry(config)

	// Add default implementations if not provided
	if config.EnableValidation {
		registry.WithValidator(NewSimpleValidator())
	}

	if config.CacheSize > 0 {
		registry.WithCache(NewMemoryCache(config.CacheTTL))
	}

	if config.EnableEvents {
		registry.WithEventBus(NewSimpleEventBus())
	}

	return registry, nil
}

// CreateWithPersistence creates a registry with persistence
func (f *RegistryFactoryImpl) CreateWithPersistence(ctx context.Context, config RegistryConfig, persistence RegistryPersistence) (RegistryProvider, error) {
	registry := NewEnhancedRegistry(config)

	// Add persistence
	registry.WithPersistence(persistence)

	// Load existing entities
	if entities, err := persistence.Load(ctx); err == nil {
		for _, entity := range entities {
			if err := registry.Register(ctx, entity); err != nil {
				return nil, fmt.Errorf("failed to load entity %s: %w", entity.ID(), err)
			}
		}
	}

	// Add other default implementations
	if config.EnableValidation {
		registry.WithValidator(NewSimpleValidator())
	}

	if config.CacheSize > 0 {
		registry.WithCache(NewMemoryCache(config.CacheTTL))
	}

	if config.EnableEvents {
		registry.WithEventBus(NewSimpleEventBus())
	}

	return registry, nil
}

// CreateWithCache creates a registry with custom cache
func (f *RegistryFactoryImpl) CreateWithCache(ctx context.Context, config RegistryConfig, cache RegistryCache) (RegistryProvider, error) {
	registry := NewEnhancedRegistry(config)

	// Add custom cache
	registry.WithCache(cache)

	// Add other default implementations
	if config.EnableValidation {
		registry.WithValidator(NewSimpleValidator())
	}

	if config.EnableEvents {
		registry.WithEventBus(NewSimpleEventBus())
	}

	return registry, nil
}

// CreateWithMetrics creates a registry with metrics
func (f *RegistryFactoryImpl) CreateWithMetrics(ctx context.Context, config RegistryConfig, metrics RegistryMetrics) (RegistryProvider, error) {
	registry := NewEnhancedRegistry(config)

	// Add metrics
	registry.WithMetrics(metrics)

	// Add other default implementations
	if config.EnableValidation {
		registry.WithValidator(NewSimpleValidator())
	}

	if config.CacheSize > 0 {
		registry.WithCache(NewMemoryCache(config.CacheTTL))
	}

	if config.EnableEvents {
		registry.WithEventBus(NewSimpleEventBus())
	}

	return registry, nil
}

// CreateFullFeatured creates a registry with all features enabled
func (f *RegistryFactoryImpl) CreateFullFeatured(ctx context.Context, config RegistryConfig) (RegistryProvider, error) {
	registry := NewEnhancedRegistry(config)

	// Add all implementations
	registry.WithValidator(NewSimpleValidator())
	registry.WithCache(NewMemoryCache(config.CacheTTL))
	registry.WithMetrics(NewSimpleMetrics())
	registry.WithHealth(NewSimpleHealth())
	registry.WithEventBus(NewSimpleEventBus())

	// Add persistence if enabled
	if config.EnablePersistence {
		persistence := NewFilePersistence(config.PersistencePath)
		registry.WithPersistence(persistence)

		// Load existing entities
		if entities, err := persistence.Load(ctx); err == nil {
			for _, entity := range entities {
				if err := registry.Register(ctx, entity); err != nil {
					return nil, fmt.Errorf("failed to load entity %s: %w", entity.ID(), err)
				}
			}
		}
	}

	return registry, nil
}

// CreateForTesting creates a registry optimized for testing
func (f *RegistryFactoryImpl) CreateForTesting(ctx context.Context) (RegistryProvider, error) {
	config := RegistryConfig{
		Name:             "test-registry",
		MaxEntities:      1000,
		EnableEvents:     false,
		EnableValidation: true,
		CacheSize:        100,
		CacheTTL:         time.Minute,
	}

	registry := NewEnhancedRegistry(config)
	registry.WithValidator(NewSimpleValidator())

	return registry, nil
}

// CreateForProduction creates a registry optimized for production use
func (f *RegistryFactoryImpl) CreateForProduction(ctx context.Context, name string, persistencePath string) (RegistryProvider, error) {
	config := RegistryConfig{
		Name:              name,
		MaxEntities:       10000,
		EnableEvents:      true,
		EnableValidation:  true,
		CacheSize:         1000,
		CacheTTL:          5 * time.Minute,
		EnablePersistence: true,
		PersistencePath:   persistencePath,
		AutoSaveInterval:  30 * time.Second,
	}

	return f.CreateFullFeatured(ctx, config)
}

// CreateForDevelopment creates a registry optimized for development
func (f *RegistryFactoryImpl) CreateForDevelopment(ctx context.Context, name string) (RegistryProvider, error) {
	config := RegistryConfig{
		Name:             name,
		MaxEntities:      1000,
		EnableEvents:     true,
		EnableValidation: true,
		CacheSize:        100,
		CacheTTL:         time.Minute,
	}

	registry := NewEnhancedRegistry(config)
	registry.WithValidator(NewSimpleValidator())
	registry.WithMetrics(NewSimpleMetrics())
	registry.WithEventBus(NewSimpleEventBus())

	return registry, nil
}

// Convenience functions for common registry creation patterns

// NewBasicRegistry creates a basic registry with default settings
func NewBasicRegistry() RegistryProvider {
	factory := NewRegistryFactory()
	config := RegistryConfig{
		Name:             "basic-registry",
		EnableEvents:     true,
		EnableValidation: true,
		CacheSize:        100,
		CacheTTL:         time.Minute,
	}

	registry, _ := factory.Create(context.Background(), config)
	return registry
}

// NewPersistentRegistry creates a registry with file persistence
func NewPersistentRegistry(filePath string) (RegistryProvider, error) {
	factory := NewRegistryFactory()
	config := RegistryConfig{
		Name:              "persistent-registry",
		EnableEvents:      true,
		EnableValidation:  true,
		CacheSize:         100,
		CacheTTL:          time.Minute,
		EnablePersistence: true,
		PersistencePath:   filePath,
		AutoSaveInterval:  time.Minute,
	}

	persistence := NewFilePersistence(filePath)
	return factory.CreateWithPersistence(context.Background(), config, persistence)
}

// NewCachedRegistry creates a registry with enhanced caching
func NewCachedRegistry(cacheSize int, cacheTTL time.Duration) RegistryProvider {
	factory := NewRegistryFactory()
	config := RegistryConfig{
		Name:             "cached-registry",
		EnableEvents:     true,
		EnableValidation: true,
		CacheSize:        cacheSize,
		CacheTTL:         cacheTTL,
	}

	registry, _ := factory.Create(context.Background(), config)
	return registry
}

// NewMonitoredRegistry creates a registry with metrics and monitoring
func NewMonitoredRegistry(name string) RegistryProvider {
	factory := NewRegistryFactory()
	config := RegistryConfig{
		Name:             name,
		EnableEvents:     true,
		EnableValidation: true,
		CacheSize:        100,
		CacheTTL:         time.Minute,
	}

	metrics := NewSimpleMetrics()
	registry, _ := factory.CreateWithMetrics(context.Background(), config, metrics)
	return registry
}

// BuildRegistry creates a registry with the built configuration
func (b *RegistryBuilder) BuildRegistry() (RegistryProvider, error) {
	factory := NewRegistryFactory()
	return factory.Create(context.Background(), b.Build())
}
