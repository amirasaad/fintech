package registry

import (
	"sync"
)

// Meta represents generic metadata that can be associated with any entity
type Meta struct {
	// Generic fields that can be used by any registry
	ID       string            `json:"id"`
	Name     string            `json:"name"`
	Active   bool              `json:"active"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// Registry is a generic, thread-safe registry for managing any type of entity
type Registry struct {
	entities map[string]Meta
	mu       sync.RWMutex
}

// New creates a new empty registry
func New() *Registry {
	return &Registry{
		entities: make(map[string]Meta),
	}
}

// Register adds or updates an entity in the registry
func (r *Registry) Register(id string, meta Meta) {
	r.mu.Lock()
	defer r.mu.Unlock()
	meta.ID = id // Ensure ID is set
	r.entities[id] = meta
}

// Get returns entity metadata for the given ID
// Returns empty Meta if the entity is not found
func (r *Registry) Get(id string) Meta {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if meta, exists := r.entities[id]; exists {
		return meta
	}

	// Return empty meta for unknown entities
	return Meta{ID: id, Active: false}
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

	ids := make([]string, 0)
	for id, meta := range r.entities {
		if meta.Active {
			ids = append(ids, id)
		}
	}
	return ids
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

	if meta, exists := r.entities[id]; exists {
		if meta.Metadata != nil {
			value, found := meta.Metadata[key]
			return value, found
		}
	}
	return "", false
}

// SetMetadata sets a specific metadata value for an entity
func (r *Registry) SetMetadata(id, key, value string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if meta, exists := r.entities[id]; exists {
		if meta.Metadata == nil {
			meta.Metadata = make(map[string]string)
		}
		meta.Metadata[key] = value
		r.entities[id] = meta
		return true
	}
	return false
}

// Global registry instance for convenience
var globalRegistry = New()

// Global convenience functions
func Register(id string, meta Meta) {
	globalRegistry.Register(id, meta)
}

func Get(id string) Meta {
	return globalRegistry.Get(id)
}

func IsRegistered(id string) bool {
	return globalRegistry.IsRegistered(id)
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
