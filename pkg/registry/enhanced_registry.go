package registry

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Enhanced provides a full-featured registry implementation
type Enhanced struct {
	config      Config
	entities    map[string]Entity
	mu          sync.RWMutex
	observers   []Observer
	validator   Validator
	cache       Cache
	persistence Persistence
	metrics     Metrics
	health      Health
	eventBus    EventBus
}

// NewEnhanced creates a new enhanced registry
func NewEnhanced(config Config) *Enhanced {
	return &Enhanced{
		config:    config,
		entities:  make(map[string]Entity),
		observers: make([]Observer, 0),
	}
}

// WithValidator sets the validator for the registry
func (r *Enhanced) WithValidator(validator Validator) *Enhanced {
	r.validator = validator
	return r
}

// WithCache sets the cache for the registry
func (r *Enhanced) WithCache(cache Cache) *Enhanced {
	r.cache = cache
	return r
}

// WithPersistence sets the persistence layer for the registry
func (r *Enhanced) WithPersistence(persistence Persistence) *Enhanced {
	r.persistence = persistence
	return r
}

// WithMetrics sets the metrics collector for the registry
func (r *Enhanced) WithMetrics(metrics Metrics) *Enhanced {
	r.metrics = metrics
	return r
}

// WithHealth sets the health checker for the registry
func (r *Enhanced) WithHealth(health Health) *Enhanced {
	r.health = health
	return r
}

// WithEventBus sets the event bus for the registry
func (r *Enhanced) WithEventBus(eventBus EventBus) *Enhanced {
	r.eventBus = eventBus
	return r
}

// Register adds or updates an entity in the registry
func (r *Enhanced) Register(ctx context.Context, entity Entity) error {
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

	// Check if this is an update
	_, exists := r.entities[entity.ID()]

	// For any entity type, create a new BaseEntity copy to ensure thread safety
	copy := NewBaseEntity(entity.ID(), entity.Name())

	// Copy the active state from the original entity
	copy.SetActive(entity.Active())

	// Copy metadata
	for k, v := range entity.Metadata() {
		copy.SetMetadata(k, v)
	}

	// Ensure the active status is reflected in metadata for backward compatibility
	copy.SetMetadata("active", strconv.FormatBool(entity.Active()))

	// Store the copy
	r.entities[copy.ID()] = copy

	// Update cache if enabled
	if r.cache != nil {
		if err := r.cache.Set(ctx, entity); err != nil {
			log.Printf("warning: failed to update cache: %v", err)
		}
	}

	// Update persistence if enabled
	if r.persistence != nil {
		if err := r.persistence.Save(ctx, r.getAllEntitiesLocked()); err != nil {
			log.Printf("warning: failed to persist registry: %v", err)
		}
	}

	// Update metrics
	if r.metrics != nil {
		r.metrics.IncrementRegistration()
		r.metrics.SetEntityCount(len(r.entities))
		r.metrics.SetActiveCount(r.countActiveLocked())
	}

	// Emit event
	if r.eventBus != nil {
		eventType := EventEntityRegistered
		if exists {
			eventType = EventEntityUpdated
		}
		if err := r.emitEvent(eventType, entity); err != nil {
			log.Printf("warning: failed to emit %s event: %v", eventType, err)
		}
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
func (r *Enhanced) Get(ctx context.Context, id string) (Entity, error) {
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
		if err := r.cache.Set(ctx, entity); err != nil {
			// Log cache set error but don't fail the operation
			log.Printf("warning: failed to update cache for entity %s: %v", entity.ID(), err)
		}
	}

	if r.metrics != nil {
		r.metrics.IncrementLookup()
	}

	return entity, nil
}

// Unregister removes an entity from the registry
func (r *Enhanced) Unregister(ctx context.Context, id string) error {
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

	// Remove from cache if available
	if r.cache != nil {
		if err := r.cache.Delete(ctx, id); err != nil {
			// Log cache delete error but don't fail the operation
			log.Printf("warning: failed to delete entity %s from cache: %v", id, err)
		}
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
		event := Event{
			Type:      EventEntityUnregistered,
			EntityID:  id,
			Timestamp: time.Now(),
		}
		if err := r.eventBus.Emit(ctx, event); err != nil {
			// Log event emission error but don't fail the operation
			log.Printf("warning: failed to emit unregister event for entity %s: %v", id, err)
		}
	}

	// Notify observers
	for _, observer := range r.observers {
		observer.OnEntityUnregistered(ctx, id)
	}

	return nil
}

// IsRegistered checks if an entity is registered
func (r *Enhanced) IsRegistered(ctx context.Context, id string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, exists := r.entities[id]
	return exists
}

// List returns all entities
func (r *Enhanced) List(ctx context.Context) ([]Entity, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entities := make([]Entity, 0, len(r.entities))
	for _, entity := range r.entities {
		entities = append(entities, entity)
	}
	return entities, nil
}

// ListActive returns all active entities
func (r *Enhanced) ListActive(ctx context.Context) ([]Entity, error) {
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
func (r *Enhanced) ListByMetadata(
	ctx context.Context,
	key, value string,
) ([]Entity, error) {
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
func (r *Enhanced) Count(ctx context.Context) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.entities), nil
}

// CountActive returns the number of active entities
func (r *Enhanced) CountActive(ctx context.Context) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.countActiveLocked(), nil
}

// countActiveLocked is a helper method that assumes the lock is already held
func (r *Enhanced) countActiveLocked() int {
	count := 0
	for _, entity := range r.entities {
		if entity.Active() {
			count++
		}
	}
	return count
}

// GetMetadata retrieves specific metadata for an entity
func (r *Enhanced) GetMetadata(ctx context.Context, id, key string) (string, error) {
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
func (r *Enhanced) SetMetadata(ctx context.Context, id, key, value string) error {
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

	// Update the entity's metadata using the proper method
	switch e := entity.(type) {
	case *BaseEntity:
		e.SetMetadata(key, value)
	default:
		// Fallback for other implementations
		metadata := entity.Metadata()
		metadata[key] = value
	}

	// Re-register the entity to update it
	return r.Register(ctx, entity)
}

// RemoveMetadata removes specific metadata from an entity
func (r *Enhanced) RemoveMetadata(ctx context.Context, id, key string) error {
	entity, err := r.Get(ctx, id)
	if err != nil {
		return err
	}

	// Remove metadata using the proper method
	switch e := entity.(type) {
	case *BaseEntity:
		e.DeleteMetadata(key)
	default:
		// Fallback for other implementations
		metadata := entity.Metadata()
		delete(metadata, key)
	}

	// Re-register the entity to update it
	return r.Register(ctx, entity)
}

// Activate activates an entity
func (r *Enhanced) Activate(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	entity, exists := r.entities[id]
	if !exists {
		return fmt.Errorf("entity not found: %s", id)
	}

	// Use the existing entity's SetActive method if it exists
	if activator, ok := entity.(interface{ SetActive(bool) }); ok {
		activator.SetActive(true)
	}

	// Also ensure the active status is set in metadata for backward compatibility
	entity.SetMetadata("active", "true")

	// Update cache if enabled
	if r.cache != nil {
		if err := r.cache.Set(ctx, entity); err != nil {
			log.Printf("warning: failed to update cache: %v", err)
		}
	}

	// Update persistence if enabled
	if r.persistence != nil {
		if err := r.persistence.Save(ctx, r.getAllEntitiesLocked()); err != nil {
			log.Printf("warning: failed to persist registry: %v", err)
		}
	}

	// Update metrics
	if r.metrics != nil {
		// No increment of registration count since it's an update
		r.metrics.SetActiveCount(r.countActiveLocked())
	}

	// Emit event
	if r.eventBus != nil {
		if err := r.emitEvent(EventEntityActivated, entity); err != nil {
			log.Printf("warning: failed to emit %s event: %v", EventEntityActivated, err)
		}
	}

	// Notify observers
	for _, observer := range r.observers {
		observer.OnEntityUpdated(ctx, entity)
	}

	return nil
}

// Deactivate deactivates an entity
func (r *Enhanced) Deactivate(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	entity, exists := r.entities[id]
	if !exists {
		return fmt.Errorf("entity not found: %s", id)
	}

	// Use the existing entity's SetActive method if it exists
	if activator, ok := entity.(interface{ SetActive(bool) }); ok {
		activator.SetActive(false)
	}

	// Also ensure the active status is set in metadata for backward compatibility
	entity.SetMetadata("active", "false")

	// Update cache if enabled
	if r.cache != nil {
		if err := r.cache.Set(ctx, entity); err != nil {
			log.Printf("warning: failed to update cache: %v", err)
		}
	}

	// Update persistence if enabled
	if r.persistence != nil {
		if err := r.persistence.Save(ctx, r.getAllEntitiesLocked()); err != nil {
			log.Printf("warning: failed to persist registry: %v", err)
		}
	}

	// Update metrics
	if r.metrics != nil {
		// No increment of registration count since it's an update
		r.metrics.SetActiveCount(r.countActiveLocked())
	}

	// Emit event
	if r.eventBus != nil {
		if err := r.emitEvent(EventEntityDeactivated, entity); err != nil {
			log.Printf("warning: failed to emit %s event: %v", EventEntityDeactivated, err)
		}
	}

	// Notify observers
	for _, observer := range r.observers {
		observer.OnEntityUpdated(ctx, entity)
	}

	return nil
}

// ...

// Search performs a simple search on entity names
func (r *Enhanced) Search(ctx context.Context, query string) ([]Entity, error) {
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
func (r *Enhanced) SearchByMetadata(
	ctx context.Context,
	metadata map[string]string,
) ([]Entity, error) {
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
func (r *Enhanced) AddObserver(observer Observer) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.observers = append(r.observers, observer)
}

// RemoveObserver removes an observer from the registry
func (r *Enhanced) RemoveObserver(observer Observer) {
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
	s = strings.ToLower(s)
	substr = strings.ToLower(substr)
	return strings.Contains(s, substr)
}

// emitEvent is a helper method to emit events to the event bus
func (r *Enhanced) emitEvent(eventType string, entity Entity) error {
	event := Event{
		Type:      eventType,
		EntityID:  entity.ID(),
		Entity:    entity,
		Timestamp: time.Now(),
	}

	// Add metadata if available
	if metadata := entity.Metadata(); len(metadata) > 0 {
		event.Metadata = make(map[string]interface{})
		for k, v := range metadata {
			event.Metadata[k] = v
		}
	}

	return r.eventBus.Emit(context.Background(), event)
}

// getAllEntitiesLocked returns a slice of all entities in the registry
// The caller must hold the write lock
func (r *Enhanced) getAllEntitiesLocked() []Entity {
	entities := make([]Entity, 0, len(r.entities))
	for _, entity := range r.entities {
		entities = append(entities, entity)
	}
	return entities
}
