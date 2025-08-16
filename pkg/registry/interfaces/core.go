package registry

import "time"

// Identifier defines the interface for getting an entity's ID
type Identifier interface {
	ID() string
}

// IDSetter defines the interface for setting an entity's ID
type IDSetter interface {
	SetID(id string) error
}

// Named defines the interface for getting an entity's name
type Named interface {
	Name() string
}

// NameSetter defines the interface for setting an entity's name
type NameSetter interface {
	SetName(name string) error
}

// ActiveStatusChecker defines the interface for checking if an entity is active
type ActiveStatusChecker interface {
	IsActive() bool
}

// ActivationSetter defines the interface for setting an entity's active status
type ActivationSetter interface {
	SetActive(active bool)
}

// MetadataReader defines the interface for reading metadata
type MetadataReader interface {
	Metadata() map[string]string
}

// MetadataWriter defines the interface for writing metadata
type MetadataWriter interface {
	SetMetadata(key, value string)
}

// MetadataDeleter defines the interface for deleting metadata
type MetadataDeleter interface {
	DeleteMetadata(key string)
}

// MetadataClearer defines the interface for clearing all metadata
type MetadataClearer interface {
	ClearMetadata()
}

// Timestamped defines the interface for timestamp-related operations
type Timestamped interface {
	CreatedAt() time.Time
	UpdatedAt() time.Time
}

// Identifiable combines identifier interfaces
type Identifiable interface {
	Identifier
	IDSetter
}

// Nameable combines name-related interfaces
type Nameable interface {
	Named
	NameSetter
}

// ActivationController combines activation-related interfaces
type ActivationController interface {
	ActiveStatusChecker
	ActivationSetter
}

// MetadataController combines metadata-related interfaces
type MetadataController interface {
	MetadataReader
	MetadataWriter
	MetadataDeleter
	MetadataClearer
}

// EntityCore represents the minimal set of methods required for an entity
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
