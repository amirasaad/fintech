package registry

import (
	"sync"
)

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

	// For BaseEntity, create a copy to ensure thread safety
	if baseEntity, ok := entity.(*BaseEntity); ok {
		// Create a new BaseEntity to avoid modifying the original
		copy := *baseEntity
		copy.BEId = id // Ensure ID is set
		r.entities[id] = &copy
	} else {
		// For custom entity types, just store as is
		r.entities[id] = entity
	}
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

	// For BaseEntity, we can update the metadata directly
	if baseEntity, ok := entity.(*BaseEntity); ok {
		if baseEntity.BEMetadata == nil {
			baseEntity.BEMetadata = make(map[string]string)
		}
		baseEntity.BEMetadata[key] = value
		return true
	}

	// For custom entity types, create a new BaseEntity with updated metadata
	// This is a fallback and might not work for all custom entity types
	metadata := entity.Metadata()
	if metadata == nil {
		metadata = make(map[string]string)
	}
	metadata[key] = value

	// Create a new BaseEntity with the updated metadata
	updatedEntity := &BaseEntity{
		BEId:        entity.ID(),
		BEName:      entity.Name(),
		BEActive:    entity.Active(),
		BEMetadata:  metadata,
		BECreatedAt: entity.CreatedAt(),
		BEUpdatedAt: entity.UpdatedAt(),
	}
	r.entities[id] = updatedEntity
	return true
}

// Global registry instance for convenience
var globalRegistry = New()

// Global convenience functions
func Register(id string, entity Entity) {
	globalRegistry.Register(id, entity)
}

func Get(id string) Entity {
	return globalRegistry.Get(id)
}

func IsRegistered(id string) bool {
	entity := globalRegistry.Get(id)
	return entity != nil && entity.ID() != ""
}

func ListRegistered() []string {
	return globalRegistry.ListRegistered()
}

func ListActive() []string {
	return globalRegistry.ListActive()
}

func Unregister(id string) bool {
	return globalRegistry.Unregister(id)
}

func Count() int {
	return globalRegistry.Count()
}

func GetMetadata(id, key string) (string, bool) {
	return globalRegistry.GetMetadata(id, key)
}

func SetMetadata(id, key, value string) bool {
	return globalRegistry.SetMetadata(id, key, value)
}
