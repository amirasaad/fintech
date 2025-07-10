package registry

import (
	"context"
	"time"
)

// Entity represents any entity that can be registered
type Entity interface {
	ID() string
	Name() string
	Active() bool
	Metadata() map[string]string
	CreatedAt() time.Time
	UpdatedAt() time.Time
}

// RegistryProvider defines the interface for registry implementations
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

// RegistryObserver defines the interface for registry event observers
type RegistryObserver interface {
	OnEntityRegistered(ctx context.Context, entity Entity)
	OnEntityUnregistered(ctx context.Context, id string)
	OnEntityUpdated(ctx context.Context, entity Entity)
	OnEntityActivated(ctx context.Context, id string)
	OnEntityDeactivated(ctx context.Context, id string)
}

// RegistryEvent represents a registry event
type RegistryEvent struct {
	Type      string                 `json:"type"`
	EntityID  string                 `json:"entity_id"`
	Entity    Entity                 `json:"entity,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// RegistryEventType constants
const (
	EventEntityRegistered   = "entity_registered"
	EventEntityUnregistered = "entity_unregistered"
	EventEntityUpdated      = "entity_updated"
	EventEntityActivated    = "entity_activated"
	EventEntityDeactivated  = "entity_deactivated"
)

// RegistryEventBus defines the interface for registry event handling
type RegistryEventBus interface {
	Subscribe(observer RegistryObserver) error
	Unsubscribe(observer RegistryObserver) error
	Publish(ctx context.Context, event RegistryEvent) error
}

// RegistryConfig holds configuration for registry implementations
type RegistryConfig struct {
	// General settings
	Name             string        `json:"name"`
	MaxEntities      int           `json:"max_entities"`
	DefaultTTL       time.Duration `json:"default_ttl"`
	EnableEvents     bool          `json:"enable_events"`
	EnableValidation bool          `json:"enable_validation"`

	// Performance settings
	CacheSize         int           `json:"cache_size"`
	CacheTTL          time.Duration `json:"cache_ttl"`
	EnableCompression bool          `json:"enable_compression"`

	// Persistence settings
	EnablePersistence bool          `json:"enable_persistence"`
	PersistencePath   string        `json:"persistence_path"`
	AutoSaveInterval  time.Duration `json:"auto_save_interval"`

	// Validation settings
	RequiredMetadata   []string                      `json:"required_metadata"`
	ForbiddenMetadata  []string                      `json:"forbidden_metadata"`
	MetadataValidators map[string]func(string) error `json:"-"`
}

// RegistryValidator defines the interface for entity validation
type RegistryValidator interface {
	Validate(ctx context.Context, entity Entity) error
	ValidateMetadata(ctx context.Context, metadata map[string]string) error
}

// RegistryPersistence defines the interface for registry persistence
type RegistryPersistence interface {
	Save(ctx context.Context, entities []Entity) error
	Load(ctx context.Context) ([]Entity, error)
	Delete(ctx context.Context, id string) error
	Clear(ctx context.Context) error
}

// RegistryCache defines the interface for registry caching
type RegistryCache interface {
	Get(ctx context.Context, id string) (Entity, bool)
	Set(ctx context.Context, entity Entity) error
	Delete(ctx context.Context, id string) error
	Clear(ctx context.Context) error
	Size() int
}

// RegistryMetrics defines the interface for registry metrics
type RegistryMetrics interface {
	IncrementRegistration()
	IncrementUnregistration()
	IncrementLookup()
	IncrementError()
	SetEntityCount(count int)
	SetActiveCount(count int)
	RecordLatency(operation string, duration time.Duration)
}

// RegistryHealth defines the interface for registry health checks
type RegistryHealth interface {
	IsHealthy(ctx context.Context) bool
	GetHealthStatus(ctx context.Context) map[string]interface{}
	GetLastError() error
}

// RegistryFactory defines the interface for creating registry instances
type RegistryFactory interface {
	Create(ctx context.Context, config RegistryConfig) (RegistryProvider, error)
	CreateWithPersistence(ctx context.Context, config RegistryConfig, persistence RegistryPersistence) (RegistryProvider, error)
	CreateWithCache(ctx context.Context, config RegistryConfig, cache RegistryCache) (RegistryProvider, error)
	CreateWithMetrics(ctx context.Context, config RegistryConfig, metrics RegistryMetrics) (RegistryProvider, error)
}

// BaseEntity provides a default implementation of the Entity interface
type BaseEntity struct {
	BEId        string            `json:"id"`
	BEName      string            `json:"name"`
	BEActive    bool              `json:"active"`
	BEMetadata  map[string]string `json:"metadata,omitempty"`
	BECreatedAt time.Time         `json:"created_at"`
	BEUpdatedAt time.Time         `json:"updated_at"`
}

// Property-style getter methods to implement Entity interface
func (e *BaseEntity) ID() string   { return e.BEId }
func (e *BaseEntity) Name() string { return e.BEName }
func (e *BaseEntity) Active() bool { return e.BEActive }
func (e *BaseEntity) Metadata() map[string]string {
	if e.BEMetadata == nil {
		e.BEMetadata = make(map[string]string)
	}
	return e.BEMetadata
}
func (e *BaseEntity) CreatedAt() time.Time { return e.BECreatedAt }
func (e *BaseEntity) UpdatedAt() time.Time { return e.BEUpdatedAt }

// Add a compile-time check to ensure BaseEntity implements the Entity interface.
var _ Entity = (*BaseEntity)(nil)

// NewBaseEntity creates a new base entity
func NewBaseEntity(id, name string) *BaseEntity {
	now := time.Now()
	return &BaseEntity{
		BEId:        id,
		BEName:      name,
		BEActive:    true,
		BEMetadata:  make(map[string]string),
		BECreatedAt: now,
		BEUpdatedAt: now,
	}
}

// RegistryBuilder provides a fluent interface for building registry configurations
type RegistryBuilder struct {
	config RegistryConfig
}

// NewRegistryBuilder creates a new registry builder
func NewRegistryBuilder() *RegistryBuilder {
	return &RegistryBuilder{
		config: RegistryConfig{
			EnableEvents:     true,
			EnableValidation: true,
			CacheSize:        1000,
			CacheTTL:         5 * time.Minute,
		},
	}
}

// WithName sets the registry name
func (b *RegistryBuilder) WithName(name string) *RegistryBuilder {
	b.config.Name = name
	return b
}

// WithMaxEntities sets the maximum number of entities
func (b *RegistryBuilder) WithMaxEntities(max int) *RegistryBuilder {
	b.config.MaxEntities = max
	return b
}

// WithDefaultTTL sets the default TTL for entities
func (b *RegistryBuilder) WithDefaultTTL(ttl time.Duration) *RegistryBuilder {
	b.config.DefaultTTL = ttl
	return b
}

// WithCache sets cache configuration
func (b *RegistryBuilder) WithCache(size int, ttl time.Duration) *RegistryBuilder {
	b.config.CacheSize = size
	b.config.CacheTTL = ttl
	return b
}

// WithPersistence enables persistence with the given path
func (b *RegistryBuilder) WithPersistence(path string, interval time.Duration) *RegistryBuilder {
	b.config.EnablePersistence = true
	b.config.PersistencePath = path
	b.config.AutoSaveInterval = interval
	return b
}

// WithValidation sets validation configuration
func (b *RegistryBuilder) WithValidation(required, forbidden []string) *RegistryBuilder {
	b.config.RequiredMetadata = required
	b.config.ForbiddenMetadata = forbidden
	return b
}

// Build returns the built configuration
func (b *RegistryBuilder) Build() RegistryConfig {
	return b.config
}
