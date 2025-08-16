package registry

import (
	"errors"
	"sync"
	"time"
)

// BaseEntity provides a thread-safe implementation of the EntityCore interface.
// It's designed to be embedded in domain-specific entity types to provide common
// functionality like ID management, metadata, and lifecycle tracking.
//
// BaseEntity implements all core interfaces (Identifiable, Named, ActiveStatusChecker,
// MetadataController, Timestamped) and the composite Entity interface.
// All methods are safe for concurrent access.
type BaseEntity struct {
	mu        sync.RWMutex
	id        string
	name      string
	active    bool
	metadata  map[string]string
	createdAt time.Time
	updatedAt time.Time

	// Backward compatibility fields
	// Deprecated: Use ID() and Name() instead
	BEId string
	// Deprecated: Use Name() and SetName() instead
	BEName string
	// Deprecated: Use IsActive() and SetActive() instead
	BEActive bool
	// Deprecated: Use Metadata() and SetMetadata() instead
	BEMetadata map[string]string
	// Deprecated: Use CreatedAt() instead
	BECreatedAt time.Time
	// Deprecated: Use UpdatedAt() instead
	BEUpdatedAt time.Time
}

// NewBaseEntity creates a new BaseEntity with the specified ID and name.
// Both ID and name must be non-empty strings.
//
// This function returns a concrete *BaseEntity type. If you need the Entity interface,
// the return value can be assigned to an Entity variable.
func NewBaseEntity(id, name string) *BaseEntity {
	now := time.Now().UTC()
	return &BaseEntity{
		id:        id,
		name:      name,
		active:    true,
		metadata:  make(map[string]string),
		createdAt: now,
		updatedAt: now,
		// Backward compatibility
		BEId:       id,
		BEName:     name,
		BEActive:   true,
		BEMetadata: make(map[string]string),
	}
}

// ID returns the entity's unique identifier.
// This method is safe for concurrent access.
func (e *BaseEntity) ID() string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	// Backward compatibility
	if e.id == "" && e.BEId != "" {
		e.id = e.BEId
	}
	return e.id
}

// SetID updates the entity's ID and updates the updated timestamp.
// This method is safe for concurrent access.
func (e *BaseEntity) SetID(id string) error {
	if id == "" {
		return errors.New("id cannot be empty")
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	e.id = id
	// Backward compatibility
	e.BEId = id
	e.updatedAt = time.Now().UTC()
	return nil
}

// Name returns the entity's name.
// This method is safe for concurrent access.
func (e *BaseEntity) Name() string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	// Backward compatibility
	if e.name == "" && e.BEName != "" {
		e.name = e.BEName
	}
	return e.name
}

// SetName updates the entity's name and updates the updated timestamp.
// This method is safe for concurrent access.
func (e *BaseEntity) SetName(name string) error {
	if name == "" {
		return errors.New("name cannot be empty")
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	e.name = name
	// Backward compatibility
	e.BEName = name
	e.updatedAt = time.Now().UTC()
	return nil
}

// IsActive returns whether the entity is currently active.
// This method is safe for concurrent access.
func (e *BaseEntity) IsActive() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	// Backward compatibility
	if !e.active && e.BEActive {
		e.active = e.BEActive
	}
	return e.active || e.BEActive
}

// SetActive updates the entity's active status and updates the updated timestamp.
// This method is safe for concurrent access.
func (e *BaseEntity) SetActive(active bool) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.active != active {
		e.active = active
		// Backward compatibility
		e.BEActive = active
		e.updatedAt = time.Now().UTC()
	}
}

// Metadata returns a copy of the entity's metadata.
// This method is safe for concurrent access and returns a defensive copy.
func (e *BaseEntity) Metadata() map[string]string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// Backward compatibility - initialize from BEMetadata if needed
	if e.metadata == nil && e.BEMetadata != nil {
		e.metadata = make(map[string]string, len(e.BEMetadata))
		for k, v := range e.BEMetadata {
			e.metadata[k] = v
		}
	}

	if e.metadata == nil {
		return nil
	}

	// Return a copy to prevent external modification
	result := make(map[string]string, len(e.metadata))
	for k, v := range e.metadata {
		result[k] = v
	}
	return result
}

// SetMetadata sets a metadata key-value pair and updates the updated timestamp.
// This method is safe for concurrent access.
func (e *BaseEntity) SetMetadata(key, value string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.metadata == nil {
		e.metadata = make(map[string]string)
	}

	e.metadata[key] = value
	// Backward compatibility
	if e.BEMetadata == nil {
		e.BEMetadata = make(map[string]string)
	}
	e.BEMetadata[key] = value
	e.updatedAt = time.Now().UTC()
}

// DeleteMetadata removes a metadata key and updates the updated timestamp.
// This method is safe for concurrent access.
func (e *BaseEntity) DeleteMetadata(key string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.metadata != nil {
		delete(e.metadata, key)
	}

	// Backward compatibility
	if e.BEMetadata != nil {
		delete(e.BEMetadata, key)
	}

	e.updatedAt = time.Now().UTC()
}

// ClearMetadata removes all metadata and updates the updated timestamp.
// This method is safe for concurrent access.
func (e *BaseEntity) ClearMetadata() {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.metadata = make(map[string]string)
	// Backward compatibility
	e.BEMetadata = make(map[string]string)
	e.updatedAt = time.Now().UTC()
}

// CreatedAt returns when the entity was created.
// This method is safe for concurrent access.
func (e *BaseEntity) CreatedAt() time.Time {
	e.mu.RLock()
	defer e.mu.RUnlock()

	return e.createdAt
}

// UpdatedAt returns when the entity was last updated.
// This method is safe for concurrent access.
func (e *BaseEntity) UpdatedAt() time.Time {
	e.mu.RLock()
	defer e.mu.RUnlock()

	return e.updatedAt
}
