package registry

import (
	"errors"
	"maps"
	"sync"
	"time"
)

var ErrNotFound = errors.New("entity not found")

// Registry is a thread-safe registry for managing entities that implement the Entity interface
type Registry struct {
	entities map[string]Entity
	mu       sync.RWMutex
}

// New creates a new empty registry
func New() *Registry {
	return &Registry{
		entities: make(map[string]Entity),
	}
}

// Register adds or updates an entity in the registry
func (r *Registry) Register(id string, entity Entity) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Create a new BaseEntity with the same values
	entityCopy := &BaseEntity{
		id:        entity.ID(),
		name:      entity.Name(),
		active:    entity.Active(),
		metadata:  make(map[string]string),
		createdAt: entity.CreatedAt(),
		updatedAt: time.Now(),
	}

	// Copy metadata
	maps.Copy(entityCopy.metadata, entity.Metadata())

	r.entities[id] = entityCopy
}

// Get returns the entity for the given ID
// Returns nil if the entity is not found
func (r *Registry) Get(id string) Entity {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if entity, exists := r.entities[id]; exists {
		return entity
	}

	// Return nil for unknown entities
	return nil
}

// IsRegistered checks if an entity ID is registered
func (r *Registry) IsRegistered(id string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, exists := r.entities[id]
	return exists
}

// ListRegistered returns a list of all registered entity IDs
func (r *Registry) ListRegistered() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ids := make([]string, 0, len(r.entities))
	for id := range r.entities {
		ids = append(ids, id)
	}
	return ids
}

// ListActive returns a list of all active entity IDs
func (r *Registry) ListActive() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var active []string
	for id, entity := range r.entities {
		if entity.Active() {
			active = append(active, id)
		}
	}
	return active
}

// Unregister removes an entity from the registry
func (r *Registry) Unregister(id string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.entities[id]; exists {
		delete(r.entities, id)
		return true
	}
	return false
}

// Count returns the total number of registered entities
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.entities)
}

// GetMetadata returns a specific metadata value for an entity
func (r *Registry) GetMetadata(id, key string) (string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if entity, exists := r.entities[id]; exists {
		metadata := entity.Metadata()
		if metadata != nil {
			value, found := metadata[key]
			return value, found
		}
	}
	return "", false
}

// SetMetadata sets a specific metadata value for an entity
func (r *Registry) SetMetadata(id, key, value string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	entity, exists := r.entities[id]
	if !exists {
		return false
	}

	// copy the metadata to avoid data races
	metadata := make(map[string]string)
	for k, v := range entity.Metadata() {
		metadata[k] = v
	}

	metadata[key] = value

	// Create a new BaseEntity with the updated metadata
	updatedEntity := &BaseEntity{
		id:        entity.ID(),
		name:      entity.Name(),
		active:    entity.Active(),
		metadata:  metadata,
		createdAt: entity.CreatedAt(),
		updatedAt: entity.UpdatedAt(),
	}
	r.entities[id] = updatedEntity
	return true
}

// RemoveMetadata removes a specific metadata key from an entity
func (r *Registry) RemoveMetadata(id, key string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	entity, exists := r.entities[id]
	if !exists {
		return false
	}

	// copy the metadata to avoid data races
	metadata := make(map[string]string)
	for k, v := range entity.Metadata() {
		metadata[k] = v
	}

	// Check if the key exists before trying to delete
	if _, exists := metadata[key]; !exists {
		return false
	}

	delete(metadata, key)

	// Create a new BaseEntity with the updated metadata
	updatedEntity := &BaseEntity{
		id:        entity.ID(),
		name:      entity.Name(),
		active:    entity.Active(),
		metadata:  metadata,
		createdAt: entity.CreatedAt(),
		updatedAt: time.Now(),
	}
	r.entities[id] = updatedEntity
	return true
}

// Create and manage registry instances explicitly in your application code.
