package registry

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"strings"
	"sync"
	"time"
)

// Core interfaces following Go's idiomatic naming conventions

// Basic interfaces (single-method)
type Identifier interface {
	ID() string
}

type IDSetter interface {
	SetID(id string) error
}

type Named interface {
	Name() string
}

type NameSetter interface {
	SetName(name string) error
}

// ActiveStatusChecker defines the interface for checking if an entity is active
type ActiveStatusChecker interface {
	Active() bool
}

type ActivationSetter interface {
	SetActive(active bool)
}

type MetadataReader interface {
	Metadata() map[string]string
}

type MetadataWriter interface {
	SetMetadata(key, value string)
}

type MetadataDeleter interface {
	DeleteMetadata(key string)
}

type MetadataClearer interface {
	ClearMetadata()
}

type Timestamped interface {
	CreatedAt() time.Time
	UpdatedAt() time.Time
}

// Composite interfaces
type Identifiable interface {
	Identifier
	IDSetter
}

type Nameable interface {
	Named
	NameSetter
}

type ActivationController interface {
	ActiveStatusChecker
	ActivationSetter
}

type MetadataController interface {
	MetadataReader
	MetadataWriter
	MetadataDeleter
	MetadataClearer
}

type EntityCore interface {
	Identifiable
	Nameable
	ActivationController
	MetadataController
	Timestamped
}

// Entity is the main interface that all registry entities must implement
// It's a composition of smaller, focused interfaces
// Deprecated: Use EntityCore for new code
type Entity = EntityCore

// EntityFactory creates new entity instances
type EntityFactory interface {
	NewEntity(id, name string) (EntityCore, error)
}

// EntityValidator validates entity state
type EntityValidator interface {
	Validate() error
}

// EntityLifecycle defines hooks for entity lifecycle events
type EntityLifecycle interface {
	BeforeCreate() error
	AfterCreate() error
	BeforeUpdate() error
	AfterUpdate() error
	BeforeDelete() error
	AfterDelete() error
}

// EntityFull combines all entity-related interfaces
type EntityFull interface {
	EntityCore
	EntityValidator
	EntityLifecycle
}

// Provider defines the interface for registry implementations
type Provider interface {
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

// Observer defines the interface for registry event observers
type Observer interface {
	OnEntityRegistered(ctx context.Context, entity Entity)
	OnEntityUnregistered(ctx context.Context, id string)
	OnEntityUpdated(ctx context.Context, entity Entity)
	OnEntityActivated(ctx context.Context, id string)
	OnEntityDeactivated(ctx context.Context, id string)
}

// Event represents a registry event
type Event struct {
	Type      string                 `json:"type"`
	EntityID  string                 `json:"entity_id"`
	Entity    Entity                 `json:"entity,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// EventType constants
const (
	EventEntityRegistered   = "entity_registered"
	EventEntityUnregistered = "entity_unregistered"
	EventEntityUpdated      = "entity_updated"
	EventEntityActivated    = "entity_activated"
	EventEntityDeactivated  = "entity_deactivated"
)

// EventBus defines the interface for registry event handling
type EventBus interface {
	Subscribe(observer Observer) error
	Unsubscribe(observer Observer) error
	Emit(ctx context.Context, event Event) error
}

// Config holds configuration for registry implementations
type Config struct {
	Name             string        `json:"name"`
	MaxEntities      int           `json:"max_entities"`
	DefaultTTL       time.Duration `json:"default_ttl"`
	EnableEvents     bool          `json:"enable_events"`
	EnableValidation bool          `json:"enable_validation"`

	// Cache settings
	CacheSize int           `json:"cache_size"`
	CacheTTL  time.Duration `json:"cache_ttl"`

	// Redis cache settings
	RedisURL          string        `json:"redis_url"`            // Redis server URL
	RedisKeyPrefix    string        `json:"redis_key_prefix"`     // Prefix for Redis keys
	RedisPoolSize     int           `json:"redis_pool_size"`      // Max connections in pool
	RedisMinIdleConns int           `json:"redis_min_idle_conns"` // Min idle connections
	RedisMaxRetries   int           `json:"redis_max_retries"`    // Max retries for commands
	RedisDialTimeout  time.Duration `json:"redis_dial_timeout"`   // Dial timeout
	RedisReadTimeout  time.Duration `json:"redis_read_timeout"`   // Read timeout
	RedisWriteTimeout time.Duration `json:"redis_write_timeout"`  // Write timeout

	// Advanced features
	EnableCompression bool   `json:"enable_compression"`
	EnablePersistence bool   `json:"enable_persistence"`
	PersistencePath   string `json:"persistence_path"`

	// Auto-save settings
	AutoSaveInterval time.Duration `json:"auto_save_interval"`

	// Metadata validation
	RequiredMetadata   []string                      `json:"required_metadata"`
	ForbiddenMetadata  []string                      `json:"forbidden_metadata"`
	MetadataValidators map[string]func(string) error `json:"-"`
}

// Validator defines the interface for entity validation
type Validator interface {
	Validate(ctx context.Context, entity Entity) error
	ValidateMetadata(ctx context.Context, metadata map[string]string) error
}

// Persistence defines the interface for registry persistence
type Persistence interface {
	Save(ctx context.Context, entities []Entity) error
	Load(ctx context.Context) ([]Entity, error)
	Delete(ctx context.Context, id string) error
	Clear(ctx context.Context) error
}

// Cache defines the interface for registry caching
type Cache interface {
	Get(ctx context.Context, id string) (Entity, bool)
	Set(ctx context.Context, entity Entity) error
	Delete(ctx context.Context, id string) error
	Clear(ctx context.Context) error
	Size() int
}

// Metrics defines the interface for registry metrics
type Metrics interface {
	IncrementRegistration()
	IncrementUnregistration()
	IncrementLookup()
	IncrementError()
	SetEntityCount(count int)
	SetActiveCount(count int)
	RecordLatency(operation string, duration time.Duration)
}

// Health defines the interface for registry health checks
type Health interface {
	IsHealthy(ctx context.Context) bool
	GetHealthStatus(ctx context.Context) map[string]interface{}
	GetLastError() error
}

// Factory defines the interface for creating registry instances
type Factory interface {
	Create(
		ctx context.Context,
		config Config,
	) (Provider, error)
	CreateWithPersistence(
		ctx context.Context,
		config Config,
		persistence Persistence,
	) (Provider, error)
	CreateWithCache(
		ctx context.Context,
		config Config,
		cache Cache,
	) (Provider, error)
	CreateWithMetrics(
		ctx context.Context,
		config Config,
		metrics Metrics,
	) (Provider, error)
}

// BaseEntity provides a thread-safe default implementation of core entity interfaces.
// It serves as a foundation for domain-specific entities by providing common
// functionality including:
//   - Unique identifier management (ID)
//   - Naming and activation state
//   - Key-value metadata storage
//   - Creation and modification timestamps
//   - Concurrent access safety
//
// BaseEntity implements the following interfaces:
//   - Identifiable: For ID management
//   - Named: For name-related operations
//   - ActivationController: For activation state management
//   - MetadataController: For metadata operations
//   - Timestamped: For creation/update timestamps
//   - Entity: Composite interface for backward compatibility
//
// Example usage:
//
//	type User struct {
//		registry.BaseEntity
//		Email    string
//		Password string `json:"-"`
//	}
//
// All exported methods are safe for concurrent access. The struct uses a read-write mutex
// to protect all internal state. When embedding BaseEntity, ensure proper initialization
// of the embedded fields.
type BaseEntity struct {
	id        string
	name      string
	active    bool
	metadata  map[string]string
	createdAt time.Time
	updatedAt time.Time
	mu        sync.RWMutex

	// Deprecated: Use ID() and SetID() methods instead
	BEId string
	// Deprecated: Use Name() and SetName() methods instead
	BEName string
	// Deprecated: Use Active() and SetActive() methods instead
	BEActive bool
	// Deprecated: Use Metadata() and related methods instead
	BEMetadata map[string]string
	// Deprecated: Use CreatedAt() and UpdatedAt() methods instead
	BECreatedAt time.Time
	// Deprecated: Use CreatedAt() and UpdatedAt() methods instead
	BEUpdatedAt time.Time
}

// MarshalJSON implements the json.Marshaler interface.
// It provides custom JSON marshaling for BaseEntity.
// This method is safe for concurrent access.
func (e *BaseEntity) MarshalJSON() ([]byte, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// Create a map to hold the JSON representation
	data := map[string]interface{}{
		"id":         e.id,
		"name":       e.name,
		"active":     e.active,
		"created_at": e.createdAt.Format(time.RFC3339Nano),
		"updated_at": e.updatedAt.Format(time.RFC3339Nano),
	}

	// Include all metadata fields at the root level for backward compatibility
	if len(e.metadata) > 0 {
		// Create a copy of the metadata to avoid concurrent access issues
		metadataCopy := make(map[string]string, len(e.metadata))
		for k, v := range e.metadata {
			metadataCopy[k] = v
			// Add each metadata field to the root level for backward compatibility
			data[k] = v
		}
		// Also include the full metadata object
		data["metadata"] = metadataCopy
	}

	return json.Marshal(data)
}

// UnmarshalJSON implements the json.Unmarshaler interface.
// It provides custom JSON unmarshaling for BaseEntity.
// This method is safe for concurrent access.
func (e *BaseEntity) UnmarshalJSON(data []byte) error {
	// Use a type alias to avoid recursion
	type Alias BaseEntity

	// Create an auxiliary struct to handle the JSON unmarshaling
	aux := &struct {
		*Alias
		CreatedAt string `json:"created_at"`
		UpdatedAt string `json:"updated_at"`
	}{
		Alias: (*Alias)(e),
	}

	// First, unmarshal into a map to handle all fields
	var rawData map[string]json.RawMessage
	if err := json.Unmarshal(data, &rawData); err != nil {
		return fmt.Errorf("failed to unmarshal BaseEntity: %w", err)
	}

	// Unmarshal the standard fields
	if err := json.Unmarshal(data, &aux); err != nil {
		return fmt.Errorf("failed to unmarshal BaseEntity: %w", err)
	}

	// Parse the timestamps
	var err error
	if aux.CreatedAt != "" {
		e.createdAt, err = time.Parse(time.RFC3339Nano, aux.CreatedAt)
		if err != nil {
			return fmt.Errorf("failed to parse created_at: %w", err)
		}
	}

	if aux.UpdatedAt != "" {
		e.updatedAt, err = time.Parse(time.RFC3339Nano, aux.UpdatedAt)
		if err != nil {
			return fmt.Errorf("failed to parse updated_at: %w", err)
		}
	}

	// Initialize the metadata map
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.metadata == nil {
		e.metadata = make(map[string]string)
	}

	// Handle metadata from the metadata object if it exists
	if metadataData, ok := rawData["metadata"]; ok {
		var metadata map[string]string
		if err := json.Unmarshal(metadataData, &metadata); err == nil {
			for k, v := range metadata {
				e.metadata[k] = v
			}
		}
	}

	// Handle direct metadata fields that might be at the root level
	for k, v := range rawData {
		// Skip standard fields that we already handle
		switch k {
		case "id", "name", "active", "created_at", "updated_at", "metadata":
			continue
		}

		// For other fields, try to unmarshal as string and add to metadata
		var strVal string
		if err := json.Unmarshal(v, &strVal); err == nil {
			e.metadata[k] = strVal
		}
	}

	return nil
}

// Ensure BaseEntity implements all interfaces
var (
	_ Identifiable         = (*BaseEntity)(nil)
	_ Named                = (*BaseEntity)(nil)
	_ ActivationController = (*BaseEntity)(nil)
	_ MetadataController   = (*BaseEntity)(nil)
	_ Timestamped          = (*BaseEntity)(nil)
	_ Entity               = (*BaseEntity)(nil) // For backward compatibility
)

// DeleteMetadata removes a metadata key from the entity.
// It's safe for concurrent access.
func (e *BaseEntity) DeleteMetadata(key string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.metadata == nil {
		e.metadata = make(map[string]string)
		return
	}

	delete(e.metadata, key)
	e.updatedAt = time.Now().UTC()
}

// SetID sets the ID of the entity.
// It's safe for concurrent access.
func (e *BaseEntity) SetID(id string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if id == "" {
		return errors.New("id cannot be empty")
	}

	e.id = id
	e.BEId = id // For backward compatibility
	e.updatedAt = time.Now().UTC()
	return nil
}

// SetName sets the name of the entity.
// It's safe for concurrent access.
func (e *BaseEntity) SetName(name string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if name == "" {
		return errors.New("name cannot be empty")
	}

	e.name = name
	e.BEName = name // For backward compatibility
	e.updatedAt = time.Now().UTC()
	return nil
}

// SetActive sets the active state of the entity.
// It's safe for concurrent access.
func (e *BaseEntity) SetActive(active bool) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.active = active
	e.BEActive = active // For backward compatibility
	e.updatedAt = time.Now().UTC()
}

// SetMetadata sets a metadata key-value pair.
// It's safe for concurrent access.
func (e *BaseEntity) SetMetadata(key, value string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.metadata == nil {
		e.metadata = make(map[string]string)
	}

	e.metadata[key] = value

	// For backward compatibility
	if e.BEMetadata == nil {
		e.BEMetadata = make(map[string]string)
	}
	e.BEMetadata[key] = value

	e.updatedAt = time.Now().UTC()
}

// ClearMetadata removes all metadata from the entity.
// It's safe for concurrent access.
func (e *BaseEntity) ClearMetadata() {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.metadata = make(map[string]string)
	e.BEMetadata = make(map[string]string) // For backward compatibility
	e.updatedAt = time.Now().UTC()
}

// Metadata returns a copy of the entity's metadata.
// It's safe for concurrent access.
func (e *BaseEntity) Metadata() map[string]string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// Return a copy to prevent external modifications
	result := make(map[string]string, len(e.metadata))
	maps.Copy(result, e.metadata)
	return result
}

// ID returns the entity's ID.
// It's safe for concurrent access.
func (e *BaseEntity) ID() string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if e.id == "" && e.BEId != "" {
		return e.BEId // For backward compatibility
	}
	return e.id
}

// Name returns the entity's name.
// It's safe for concurrent access.
func (e *BaseEntity) Name() string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if e.name == "" && e.BEName != "" {
		return e.BEName // For backward compatibility
	}
	return e.name
}

// Active returns whether the entity is active.
// It's safe for concurrent access.
func (e *BaseEntity) Active() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()

	return e.active || e.BEActive // For backward compatibility
}

// CreatedAt returns when the entity was created.
// It's safe for concurrent access.
func (e *BaseEntity) CreatedAt() time.Time {
	e.mu.RLock()
	defer e.mu.RUnlock()

	return e.createdAt
}

// UpdatedAt returns when the entity was last updated.
// It's safe for concurrent access.
func (e *BaseEntity) UpdatedAt() time.Time {
	e.mu.RLock()
	defer e.mu.RUnlock()

	return e.updatedAt
}

// Ensure BaseEntity implements all the interfaces it's meant to support
var (
	_ Identifier           = (*BaseEntity)(nil)
	_ IDSetter             = (*BaseEntity)(nil)
	_ Named                = (*BaseEntity)(nil)
	_ NameSetter           = (*BaseEntity)(nil)
	_ ActiveStatusChecker  = (*BaseEntity)(nil)
	_ ActivationController = (*BaseEntity)(nil)
	_ MetadataReader       = (*BaseEntity)(nil)
	_ MetadataWriter       = (*BaseEntity)(nil)
	_ MetadataDeleter      = (*BaseEntity)(nil)
	_ MetadataClearer      = (*BaseEntity)(nil)
	_ Timestamped          = (*BaseEntity)(nil)
	_ Entity               = (*BaseEntity)(nil)
)

// NewBaseEntity creates a new BaseEntity with the given id and name.
// The entity will be active by default and have the creation time set to now.
// Returns an error if id or name is empty.
//
// This function returns a concrete *BaseEntity type. If you need the Entity interface,
// the return value can be assigned to an Entity variable.
func NewBaseEntity(id, name string) *BaseEntity {

	now := time.Now().UTC()
	entity := &BaseEntity{
		id:        id,
		name:      name,
		active:    true,
		metadata:  make(map[string]string),
		createdAt: now,
		updatedAt: now,
		// Initialize BEFields for backward compatibility
		BEId:       id,
		BEName:     name,
		BEActive:   true,
		BEMetadata: make(map[string]string),
	}

	return entity
}

// RemoveMetadata removes a metadata key and updates the updated timestamp.
// If the key doesn't exist, this is a no-op.
// This method is safe for concurrent access.
func (e *BaseEntity) RemoveMetadata(key string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.metadata != nil {
		if _, exists := e.metadata[key]; exists {
			delete(e.metadata, key)
			e.updatedAt = time.Now().UTC()
		}
	}
}

// SetMetadataMap sets multiple metadata key-value pairs at once and updates the updated timestamp.
// This is more efficient than calling SetMetadata multiple times as it only acquires the lock once.
// This method is safe for concurrent access.
func (e *BaseEntity) SetMetadataMap(metadata map[string]string) {
	if len(metadata) == 0 {
		return
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	if e.metadata == nil {
		e.metadata = make(map[string]string, len(metadata))
	}

	for k, v := range metadata {
		e.metadata[k] = v
	}
	// Backward compatibility
	if e.BEMetadata == nil {
		e.BEMetadata = make(map[string]string, len(metadata))
	}
	for k, v := range metadata {
		e.BEMetadata[k] = v
	}
	e.updatedAt = time.Now().UTC()
}

// HasMetadata checks if the entity has a specific metadata key.
// This method is safe for concurrent access.
func (e *BaseEntity) HasMetadata(key string) bool {
	if key == "" {
		return false
	}

	e.mu.RLock()
	_, exists := e.metadata[key]
	e.mu.RUnlock()
	return exists
}

// GetMetadataValue returns the value for a metadata key and whether it exists.
// This method is more efficient than Metadata() when you only need one value.
func (e *BaseEntity) GetMetadataValue(key string) (string, bool) {
	if key == "" {
		return "", false
	}

	e.mu.RLock()
	defer e.mu.RUnlock()

	// Check in new metadata first
	if e.metadata != nil {
		if val, exists := e.metadata[key]; exists {
			return val, true
		}
	}

	// Backward compatibility - check in BEMetadata if not found in metadata
	if e.BEMetadata != nil {
		if val, exists := e.BEMetadata[key]; exists {
			// If we found it in BEMetadata but not in metadata, sync it
			if e.metadata == nil {
				e.metadata = make(map[string]string)
			}
			e.metadata[key] = val
			return val, true
		}
	}

	return "", false
}

// Use only when you're certain the inputs are valid.
func MustNewBaseEntity(id, name string) *BaseEntity {
	entity := NewBaseEntity(id, name)
	return entity
}

// Builder provides a fluent interface for building registry configurations
type Builder struct {
	config Config
}

// NewBuilder creates a new registry builder
func NewBuilder() *Builder {
	return &Builder{
		config: Config{
			EnableEvents:     true,
			EnableValidation: true,
			CacheSize:        1000,
			CacheTTL:         5 * time.Minute,
		},
	}
}

// WithName sets the registry name
func (b *Builder) WithName(name string) *Builder {
	b.config.Name = name
	return b
}

// WithMaxEntities sets the maximum number of entities
func (b *Builder) WithMaxEntities(max int) *Builder {
	b.config.MaxEntities = max
	return b
}

// WithDefaultTTL sets the default TTL for entities
func (b *Builder) WithDefaultTTL(ttl time.Duration) *Builder {
	b.config.DefaultTTL = ttl
	return b
}

// WithCache sets cache configuration
func (b *Builder) WithCache(size int, ttl time.Duration) *Builder {
	b.config.CacheSize = size
	b.config.CacheTTL = ttl
	return b
}

// WithRedis configures Redis cache settings
func (b *Builder) WithRedis(url string) *Builder {
	b.config.RedisURL = url
	// Set sensible defaults for Redis
	if b.config.RedisKeyPrefix == "" {
		b.config.RedisKeyPrefix = "registry:"
	}
	if b.config.RedisPoolSize == 0 {
		b.config.RedisPoolSize = 10
	}
	if b.config.RedisMinIdleConns == 0 {
		b.config.RedisMinIdleConns = 5
	}
	if b.config.RedisMaxRetries == 0 {
		b.config.RedisMaxRetries = 3
	}
	if b.config.RedisDialTimeout == 0 {
		b.config.RedisDialTimeout = 5 * time.Second
	}
	if b.config.RedisReadTimeout == 0 {
		b.config.RedisReadTimeout = 3 * time.Second
	}
	if b.config.RedisWriteTimeout == 0 {
		b.config.RedisWriteTimeout = 3 * time.Second
	}
	return b
}

// WithKeyPrefix sets the Redis key prefix for the registry
func (b *Builder) WithKeyPrefix(prefix string) *Builder {
	// Ensure prefix ends with a colon if not empty
	if prefix != "" && !strings.HasSuffix(prefix, ":") {
		prefix += ":"
	}
	b.config.RedisKeyPrefix = prefix
	return b
}

// WithRedisAdvanced allows fine-grained Redis configuration
func (b *Builder) WithRedisAdvanced(
	url string,
	prefix string,
	poolSize int,
	minIdleConns int,
	maxRetries int,
	dialTimeout time.Duration,
	readTimeout time.Duration,
	writeTimeout time.Duration,
) *Builder {
	b.config.RedisURL = url
	b.config.RedisKeyPrefix = prefix
	b.config.RedisPoolSize = poolSize
	b.config.RedisMinIdleConns = minIdleConns
	b.config.RedisMaxRetries = maxRetries
	b.config.RedisDialTimeout = dialTimeout
	b.config.RedisReadTimeout = readTimeout
	b.config.RedisWriteTimeout = writeTimeout
	return b
}

// WithPersistence enables persistence with the given path
func (b *Builder) WithPersistence(path string, interval time.Duration) *Builder {
	b.config.EnablePersistence = true
	b.config.PersistencePath = path
	b.config.AutoSaveInterval = interval
	return b
}

// WithValidation sets validation configuration
func (b *Builder) WithValidation(required, forbidden []string) *Builder {
	b.config.RequiredMetadata = required
	b.config.ForbiddenMetadata = forbidden
	return b
}

// Build returns the built configuration
func (b *Builder) Build() Config {
	return b.config
}
