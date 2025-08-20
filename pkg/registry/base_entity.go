package registry

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"sync"
	"time"
)

// / BaseEntity provides a thread-safe default implementation of core entity interfaces.
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

	// Create a map to hold the JSON representation with core fields
	data := map[string]any{
		"id":         e.id,
		"name":       e.name,
		"active":     e.active,
		"created_at": e.createdAt.Format(time.RFC3339Nano),
		"updated_at": e.updatedAt.Format(time.RFC3339Nano),
	}

	// Include metadata as a separate object, not at the root level
	if len(e.metadata) > 0 {
		// Create a copy of the metadata to avoid concurrent access issues
		metadataCopy := make(map[string]string, len(e.metadata))
		maps.Copy(metadataCopy, e.metadata)
		// Only include the full metadata object, not individual fields at root
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
			maps.Copy(e.metadata, metadata)
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
	_ Identity             = (*BaseEntity)(nil)
	_ Named                = (*BaseEntity)(nil)
	_ ActivationController = (*BaseEntity)(nil)
	_ MetadataController   = (*BaseEntity)(nil)
	_ Timestamped          = (*BaseEntity)(nil)
	_ MetadataReader       = (*BaseEntity)(nil)
	_ MetadataWriter       = (*BaseEntity)(nil)
	_ MetadataRemover      = (*BaseEntity)(nil)
	_ MetadataClearer      = (*BaseEntity)(nil)
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
	_ MetadataRemover      = (*BaseEntity)(nil)
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

	maps.Copy(e.metadata, metadata)
	// Backward compatibility
	if e.BEMetadata == nil {
		e.BEMetadata = make(map[string]string, len(metadata))
	}
	maps.Copy(e.BEMetadata, metadata)
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
