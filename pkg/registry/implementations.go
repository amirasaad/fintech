package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

// MemoryCache implements RegistryCache using in-memory storage
type MemoryCache struct {
	cache map[string]cacheEntry
	mu    sync.RWMutex
	ttl   time.Duration
}

type cacheEntry struct {
	entity    Entity
	expiresAt time.Time
}

// NewMemoryCache creates a new memory cache
func NewMemoryCache(ttl time.Duration) *MemoryCache {
	cache := &MemoryCache{
		cache: make(map[string]cacheEntry),
		ttl:   ttl,
	}

	// Start cleanup goroutine
	go cache.cleanup()

	return cache
}

// Get retrieves an entity from cache
func (c *MemoryCache) Get(ctx context.Context, id string) (Entity, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.cache[id]
	if !exists {
		return nil, false
	}

	// Check if expired
	if time.Now().After(entry.expiresAt) {
		delete(c.cache, id)
		return nil, false
	}

	return entry.entity, true
}

// Set stores an entity in cache
func (c *MemoryCache) Set(ctx context.Context, entity Entity) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache[entity.ID()] = cacheEntry{
		entity:    entity,
		expiresAt: time.Now().Add(c.ttl),
	}

	return nil
}

// Delete removes an entity from cache
func (c *MemoryCache) Delete(ctx context.Context, id string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.cache, id)
	return nil
}

// Clear removes all entities from cache
func (c *MemoryCache) Clear(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache = make(map[string]cacheEntry)
	return nil
}

// Size returns the number of cached entities
func (c *MemoryCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.cache)
}

// cleanup removes expired entries
func (c *MemoryCache) cleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for id, entry := range c.cache {
			if now.After(entry.expiresAt) {
				delete(c.cache, id)
			}
		}
		c.mu.Unlock()
	}
}

// FilePersistence implements RegistryPersistence using file storage
type FilePersistence struct {
	filePath string
	mu       sync.Mutex
}

// NewFilePersistence creates a new file persistence layer
func NewFilePersistence(filePath string) *FilePersistence {
	return &FilePersistence{
		filePath: filePath,
	}
}

// Save persists entities to file
func (p *FilePersistence) Save(ctx context.Context, entities []Entity) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Convert entities to serializable format
	data := make([]map[string]interface{}, len(entities))
	for i, entity := range entities {
		data[i] = map[string]interface{}{
			"id":         entity.ID(),
			"name":       entity.Name(),
			"active":     entity.Active(),
			"metadata":   entity.Metadata(),
			"created_at": entity.CreatedAt(),
			"updated_at": entity.UpdatedAt(),
		}
	}

	// Marshal to JSON
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal entities: %w", err)
	}

	// Write to file
	if err := os.WriteFile(p.filePath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// Load loads entities from file
func (p *FilePersistence) Load(ctx context.Context) ([]Entity, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Check if file exists
	if _, err := os.Stat(p.filePath); os.IsNotExist(err) {
		return []Entity{}, nil
	}

	// Read file
	data, err := os.ReadFile(p.filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Unmarshal JSON
	var rawData []map[string]interface{}
	if err := json.Unmarshal(data, &rawData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal data: %w", err)
	}

	// Convert to entities
	entities := make([]Entity, len(rawData))
	for i, raw := range rawData {
		// Parse timestamps
		createdAt, _ := time.Parse(time.RFC3339, raw["created_at"].(string))
		updatedAt, _ := time.Parse(time.RFC3339, raw["updated_at"].(string))

		// Parse metadata
		metadata := make(map[string]string)
		if rawMetadata, ok := raw["metadata"].(map[string]interface{}); ok {
			for k, v := range rawMetadata {
				if str, ok := v.(string); ok {
					metadata[k] = str
				}
			}
		}

		entities[i] = &BaseEntity{
			BEId:        raw["id"].(string),
			BEName:      raw["name"].(string),
			BEActive:    raw["active"].(bool),
			BEMetadata:  metadata,
			BECreatedAt: createdAt,
			BEUpdatedAt: updatedAt,
		}
	}

	return entities, nil
}

// Delete removes the persistence file
func (p *FilePersistence) Delete(ctx context.Context, id string) error {
	// For file persistence, we don't delete individual entities
	// The entire file is rewritten on save
	return nil
}

// Clear removes the persistence file
func (p *FilePersistence) Clear(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	return os.Remove(p.filePath)
}

// SimpleMetrics implements RegistryMetrics using simple counters
type SimpleMetrics struct {
	registrations   int64
	unregistrations int64
	lookups         int64
	errors          int64
	entityCount     int
	activeCount     int
	latencies       map[string][]time.Duration
	mu              sync.RWMutex
}

// NewSimpleMetrics creates a new simple metrics collector
func NewSimpleMetrics() *SimpleMetrics {
	return &SimpleMetrics{
		latencies: make(map[string][]time.Duration),
	}
}

// IncrementRegistration increments the registration counter
func (m *SimpleMetrics) IncrementRegistration() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.registrations++
}

// IncrementUnregistration increments the unregistration counter
func (m *SimpleMetrics) IncrementUnregistration() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.unregistrations++
}

// IncrementLookup increments the lookup counter
func (m *SimpleMetrics) IncrementLookup() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lookups++
}

// IncrementError increments the error counter
func (m *SimpleMetrics) IncrementError() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.errors++
}

// SetEntityCount sets the entity count
func (m *SimpleMetrics) SetEntityCount(count int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.entityCount = count
}

// SetActiveCount sets the active entity count
func (m *SimpleMetrics) SetActiveCount(count int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.activeCount = count
}

// RecordLatency records operation latency
func (m *SimpleMetrics) RecordLatency(operation string, duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.latencies[operation] == nil {
		m.latencies[operation] = make([]time.Duration, 0)
	}
	m.latencies[operation] = append(m.latencies[operation], duration)

	// Keep only last 100 latencies per operation
	if len(m.latencies[operation]) > 100 {
		m.latencies[operation] = m.latencies[operation][len(m.latencies[operation])-100:]
	}
}

// GetStats returns current metrics statistics
func (m *SimpleMetrics) GetStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := map[string]interface{}{
		"registrations":   m.registrations,
		"unregistrations": m.unregistrations,
		"lookups":         m.lookups,
		"errors":          m.errors,
		"entity_count":    m.entityCount,
		"active_count":    m.activeCount,
		"latencies":       m.latencies,
	}

	return stats
}

// SimpleEventBus implements RegistryEventBus using in-memory event handling
type SimpleEventBus struct {
	observers []RegistryObserver
	mu        sync.RWMutex
}

// NewSimpleEventBus creates a new simple event bus
func NewSimpleEventBus() *SimpleEventBus {
	return &SimpleEventBus{
		observers: make([]RegistryObserver, 0),
	}
}

// Subscribe adds an observer to the event bus
func (b *SimpleEventBus) Subscribe(observer RegistryObserver) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.observers = append(b.observers, observer)
	return nil
}

// Unsubscribe removes an observer from the event bus
func (b *SimpleEventBus) Unsubscribe(observer RegistryObserver) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	for i, obs := range b.observers {
		if obs == observer {
			b.observers = append(b.observers[:i], b.observers[i+1:]...)
			break
		}
	}
	return nil
}

// Publish publishes an event to all observers
func (b *SimpleEventBus) Emit(ctx context.Context, event RegistryEvent) error {
	b.mu.RLock()
	observers := make([]RegistryObserver, len(b.observers))
	copy(observers, b.observers)
	b.mu.RUnlock()

	for _, observer := range observers {
		switch event.Type {
		case EventEntityRegistered:
			observer.OnEntityRegistered(ctx, event.Entity)
		case EventEntityUnregistered:
			observer.OnEntityUnregistered(ctx, event.EntityID)
		case EventEntityUpdated:
			observer.OnEntityUpdated(ctx, event.Entity)
		case EventEntityActivated:
			observer.OnEntityActivated(ctx, event.EntityID)
		case EventEntityDeactivated:
			observer.OnEntityDeactivated(ctx, event.EntityID)
		}
	}

	return nil
}

// SimpleValidator implements RegistryValidator with basic validation
type SimpleValidator struct {
	requiredMetadata  []string
	forbiddenMetadata []string
	validators        map[string]func(string) error
}

// NewSimpleValidator creates a new simple validator
func NewSimpleValidator() *SimpleValidator {
	return &SimpleValidator{
		requiredMetadata:  make([]string, 0),
		forbiddenMetadata: make([]string, 0),
		validators:        make(map[string]func(string) error),
	}
}

// WithRequiredMetadata sets required metadata fields
func (v *SimpleValidator) WithRequiredMetadata(fields []string) *SimpleValidator {
	v.requiredMetadata = fields
	return v
}

// WithForbiddenMetadata sets forbidden metadata fields
func (v *SimpleValidator) WithForbiddenMetadata(fields []string) *SimpleValidator {
	v.forbiddenMetadata = fields
	return v
}

// WithValidator adds a custom validator for a metadata field
func (v *SimpleValidator) WithValidator(field string, validator func(string) error) *SimpleValidator {
	v.validators[field] = validator
	return v
}

// Validate validates an entity
func (v *SimpleValidator) Validate(ctx context.Context, entity Entity) error {
	// Validate required fields
	if entity.ID() == "" {
		return fmt.Errorf("entity ID cannot be empty")
	}
	if entity.Name() == "" {
		return fmt.Errorf("entity name cannot be empty")
	}

	// Validate metadata
	return v.ValidateMetadata(ctx, entity.Metadata())
}

// ValidateMetadata validates entity metadata
func (v *SimpleValidator) ValidateMetadata(ctx context.Context, metadata map[string]string) error {
	// Check required metadata
	for _, required := range v.requiredMetadata {
		if _, exists := metadata[required]; !exists {
			return fmt.Errorf("required metadata field missing: %s", required)
		}
	}

	// Check forbidden metadata
	for _, forbidden := range v.forbiddenMetadata {
		if _, exists := metadata[forbidden]; exists {
			return fmt.Errorf("forbidden metadata field present: %s", forbidden)
		}
	}

	// Run custom validators
	for field, validator := range v.validators {
		if value, exists := metadata[field]; exists {
			if err := validator(value); err != nil {
				return fmt.Errorf("validation failed for field %s: %w", field, err)
			}
		}
	}

	return nil
}

// SimpleHealth implements RegistryHealth with basic health checking
type SimpleHealth struct {
	lastError error
	mu        sync.RWMutex
}

// NewSimpleHealth creates a new simple health checker
func NewSimpleHealth() *SimpleHealth {
	return &SimpleHealth{}
}

// IsHealthy checks if the registry is healthy
func (h *SimpleHealth) IsHealthy(ctx context.Context) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.lastError == nil
}

// GetHealthStatus returns the health status
func (h *SimpleHealth) GetHealthStatus(ctx context.Context) map[string]interface{} {
	h.mu.RLock()
	defer h.mu.RUnlock()

	status := map[string]interface{}{
		"healthy":   h.lastError == nil,
		"timestamp": time.Now(),
	}

	if h.lastError != nil {
		status["last_error"] = h.lastError.Error()
	}

	return status
}

// GetLastError returns the last error
func (h *SimpleHealth) GetLastError() error {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.lastError
}

// SetError sets the last error
func (h *SimpleHealth) SetError(err error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.lastError = err
}

// ClearError clears the last error
func (h *SimpleHealth) ClearError() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.lastError = nil
}
