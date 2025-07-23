package registry

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// EnhancedRegistry provides a full-featured registry implementation
type EnhancedRegistry struct {
	config      RegistryConfig
	entities    map[string]Entity
	mu          sync.RWMutex
	observers   []RegistryObserver
	validator   RegistryValidator
	cache       RegistryCache
	persistence RegistryPersistence
	metrics     RegistryMetrics
	health      RegistryHealth
	eventBus    RegistryEventBus
}

// NewEnhancedRegistry creates a new enhanced registry
func NewEnhancedRegistry(config RegistryConfig) *EnhancedRegistry {
	return &EnhancedRegistry{
		config:    config,
		entities:  make(map[string]Entity),
		observers: make([]RegistryObserver, 0),
	}
}

// WithValidator sets the validator for the registry
func (r *EnhancedRegistry) WithValidator(validator RegistryValidator) *EnhancedRegistry {
	r.validator = validator
	return r
}

// WithCache sets the cache for the registry
func (r *EnhancedRegistry) WithCache(cache RegistryCache) *EnhancedRegistry {
	r.cache = cache
	return r
}

// WithPersistence sets the persistence layer for the registry
func (r *EnhancedRegistry) WithPersistence(persistence RegistryPersistence) *EnhancedRegistry {
	r.persistence = persistence
	return r
}

// WithMetrics sets the metrics collector for the registry
func (r *EnhancedRegistry) WithMetrics(metrics RegistryMetrics) *EnhancedRegistry {
	r.metrics = metrics
	return r
}

// WithHealth sets the health checker for the registry
func (r *EnhancedRegistry) WithHealth(health RegistryHealth) *EnhancedRegistry {
	r.health = health
	return r
}

// WithEventBus sets the event bus for the registry
func (r *EnhancedRegistry) WithEventBus(eventBus RegistryEventBus) *EnhancedRegistry {
	r.eventBus = eventBus
	return r
}

// Register adds or updates an entity in the registry
func (r *EnhancedRegistry) Register(ctx context.Context, entity Entity) error {
	start := time.Now()
	defer func() {
		if r.metrics != nil {
			r.metrics.RecordLatency("register", time.Since(start))
		}
	}()

	// Validate entity if validator is set
	if r.validator != nil {
		if err := r.validator.Validate(ctx, entity); err != nil {
			if r.metrics != nil {
				r.metrics.IncrementError()
			}
			return fmt.Errorf("validation failed: %w", err)
		}
	}

	// Check max entities limit
	if r.config.MaxEntities > 0 {
		r.mu.RLock()
		currentCount := len(r.entities)
		r.mu.RUnlock()
		if currentCount >= r.config.MaxEntities {
			if r.metrics != nil {
				r.metrics.IncrementError()
			}
			return fmt.Errorf("registry is full (max entities: %d)", r.config.MaxEntities)
		}
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if entity already exists
	_, exists := r.entities[entity.ID()]
	r.entities[entity.ID()] = entity

	// Update cache if available
	if r.cache != nil {
		r.cache.Set(ctx, entity) //nolint:errcheck
	}

	// Update metrics
	if r.metrics != nil {
		r.metrics.IncrementRegistration()
		r.metrics.SetEntityCount(len(r.entities))
		if entity.Active() {
			activeCount := r.countActiveLocked()
			r.metrics.SetActiveCount(activeCount)
		}
	}

	// Publish event
	if r.eventBus != nil {
		eventType := EventEntityRegistered
		if exists {
			eventType = EventEntityUpdated
		}
		event := RegistryEvent{
			Type:      eventType,
			EntityID:  entity.ID(),
			Entity:    entity,
			Timestamp: time.Now(),
		}
		r.eventBus.Emit(ctx, event) //nolint:errcheck
	}

	// Notify observers
	for _, observer := range r.observers {
		if exists {
			observer.OnEntityUpdated(ctx, entity)
		} else {
			observer.OnEntityRegistered(ctx, entity)
		}
	}

	return nil
}

// Get retrieves an entity by ID
func (r *EnhancedRegistry) Get(ctx context.Context, id string) (Entity, error) {
	start := time.Now()
	defer func() {
		if r.metrics != nil {
			r.metrics.RecordLatency("get", time.Since(start))
		}
	}()

	// Try cache first
	if r.cache != nil {
		if entity, found := r.cache.Get(ctx, id); found {
			if r.metrics != nil {
				r.metrics.IncrementLookup()
			}
			return entity, nil
		}
	}

	r.mu.RLock()
	entity, exists := r.entities[id]
	r.mu.RUnlock()

	if !exists {
		if r.metrics != nil {
			r.metrics.IncrementError()
		}
		return nil, fmt.Errorf("entity not found: %s", id)
	}

	// Update cache
	if r.cache != nil {
		r.cache.Set(ctx, entity) //nolint:errcheck
	}

	if r.metrics != nil {
		r.metrics.IncrementLookup()
	}

	return entity, nil
}

// Unregister removes an entity from the registry
func (r *EnhancedRegistry) Unregister(ctx context.Context, id string) error {
	start := time.Now()
	defer func() {
		if r.metrics != nil {
			r.metrics.RecordLatency("unregister", time.Since(start))
		}
	}()

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.entities[id]; !exists {
		if r.metrics != nil {
			r.metrics.IncrementError()
		}
		return fmt.Errorf("entity not found: %s", id)
	}

	delete(r.entities, id)

	// Remove from cache
	if r.cache != nil {
		r.cache.Delete(ctx, id) //nolint:errcheck
	}

	// Update metrics
	if r.metrics != nil {
		r.metrics.IncrementUnregistration()
		r.metrics.SetEntityCount(len(r.entities))
		activeCount := r.countActiveLocked()
		r.metrics.SetActiveCount(activeCount)
	}

	// Publish event
	if r.eventBus != nil {
		event := RegistryEvent{
			Type:      EventEntityUnregistered,
			EntityID:  id,
			Timestamp: time.Now(),
		}
		r.eventBus.Emit(ctx, event) //nolint:errcheck
	}

	// Notify observers
	for _, observer := range r.observers {
		observer.OnEntityUnregistered(ctx, id)
	}

	return nil
}

// IsRegistered checks if an entity is registered
func (r *EnhancedRegistry) IsRegistered(ctx context.Context, id string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, exists := r.entities[id]
	return exists
}

// List returns all entities
func (r *EnhancedRegistry) List(ctx context.Context) ([]Entity, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entities := make([]Entity, 0, len(r.entities))
	for _, entity := range r.entities {
		entities = append(entities, entity)
	}
	return entities, nil
}

// ListActive returns all active entities
func (r *EnhancedRegistry) ListActive(ctx context.Context) ([]Entity, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entities := make([]Entity, 0)
	for _, entity := range r.entities {
		if entity.Active() {
			entities = append(entities, entity)
		}
	}
	return entities, nil
}

// ListByMetadata returns entities with specific metadata
func (r *EnhancedRegistry) ListByMetadata(ctx context.Context, key, value string) ([]Entity, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entities := make([]Entity, 0)
	for _, entity := range r.entities {
		if metadata := entity.Metadata(); metadata != nil {
			if val, exists := metadata[key]; exists && val == value {
				entities = append(entities, entity)
			}
		}
	}
	return entities, nil
}

// Count returns the total number of entities
func (r *EnhancedRegistry) Count(ctx context.Context) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.entities), nil
}

// CountActive returns the number of active entities
func (r *EnhancedRegistry) CountActive(ctx context.Context) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.countActiveLocked(), nil
}

// countActiveLocked is a helper method that assumes the lock is already held
func (r *EnhancedRegistry) countActiveLocked() int {
	count := 0
	for _, entity := range r.entities {
		if entity.Active() {
			count++
		}
	}
	return count
}

// GetMetadata retrieves specific metadata for an entity
func (r *EnhancedRegistry) GetMetadata(ctx context.Context, id, key string) (string, error) {
	entity, err := r.Get(ctx, id)
	if err != nil {
		return "", err
	}

	metadata := entity.Metadata()
	if val, exists := metadata[key]; exists {
		return val, nil
	}

	return "", fmt.Errorf("metadata key not found: %s", key)
}

// SetMetadata sets specific metadata for an entity
func (r *EnhancedRegistry) SetMetadata(ctx context.Context, id, key, value string) error {
	entity, err := r.Get(ctx, id)
	if err != nil {
		return err
	}

	// Validate metadata if validator is set
	if r.validator != nil {
		metadata := entity.Metadata()
		metadata[key] = value
		if err := r.validator.ValidateMetadata(ctx, metadata); err != nil {
			return fmt.Errorf("metadata validation failed: %w", err)
		}
	}

	// Update the entity's metadata
	metadata := entity.Metadata()
	metadata[key] = value

	// Re-register the entity to update it
	return r.Register(ctx, entity)
}

// RemoveMetadata removes specific metadata from an entity
func (r *EnhancedRegistry) RemoveMetadata(ctx context.Context, id, key string) error {
	entity, err := r.Get(ctx, id)
	if err != nil {
		return err
	}

	metadata := entity.Metadata()
	delete(metadata, key)

	// Re-register the entity to update it
	return r.Register(ctx, entity)
}

// Activate activates an entity
func (r *EnhancedRegistry) Activate(ctx context.Context, id string) error {
	entity, err := r.Get(ctx, id)
	if err != nil {
		return err
	}

	// Create a new entity with active status
	// Note: This is a simplified approach - in a real implementation,
	// you might want to make the Entity interface mutable or use a different approach
	baseEntity := &BaseEntity{
		BEId:        entity.ID(),
		BEName:      entity.Name(),
		BEActive:    true,
		BEMetadata:  entity.Metadata(),
		BECreatedAt: entity.CreatedAt(),
		BEUpdatedAt: entity.UpdatedAt(),
	}

	return r.Register(ctx, baseEntity)
}

// Deactivate deactivates an entity
func (r *EnhancedRegistry) Deactivate(ctx context.Context, id string) error {
	entity, err := r.Get(ctx, id)
	if err != nil {
		return err
	}

	// Create a new entity with inactive status
	baseEntity := &BaseEntity{
		BEId:        entity.ID(),
		BEName:      entity.Name(),
		BEActive:    false,
		BEMetadata:  entity.Metadata(),
		BECreatedAt: entity.CreatedAt(),
		BEUpdatedAt: entity.UpdatedAt(),
	}

	return r.Register(ctx, baseEntity)
}

// Search performs a simple search on entity names
func (r *EnhancedRegistry) Search(ctx context.Context, query string) ([]Entity, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entities := make([]Entity, 0)
	for _, entity := range r.entities {
		if contains(entity.Name(), query) {
			entities = append(entities, entity)
		}
	}
	return entities, nil
}

// SearchByMetadata searches entities by metadata
func (r *EnhancedRegistry) SearchByMetadata(ctx context.Context, metadata map[string]string) ([]Entity, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entities := make([]Entity, 0)
	for _, entity := range r.entities {
		entityMetadata := entity.Metadata()
		matches := true
		for key, value := range metadata {
			if val, exists := entityMetadata[key]; !exists || val != value {
				matches = false
				break
			}
		}
		if matches {
			entities = append(entities, entity)
		}
	}
	return entities, nil
}

// AddObserver adds an observer to the registry
func (r *EnhancedRegistry) AddObserver(observer RegistryObserver) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.observers = append(r.observers, observer)
}

// RemoveObserver removes an observer from the registry
func (r *EnhancedRegistry) RemoveObserver(observer RegistryObserver) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i, obs := range r.observers {
		if obs == observer {
			r.observers = append(r.observers[:i], r.observers[i+1:]...)
			break
		}
	}
}

// contains is a helper function for string search
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) && (s[:len(substr)] == substr ||
			s[len(s)-len(substr):] == substr ||
			containsSubstring(s, substr))))
}

// containsSubstring is a helper function for substring search
func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
