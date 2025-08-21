package registry

// Deprecated: Use Enhanced instead
// NewEnhancedRegistry was renamed to NewEnhanced for brevity
type EnhancedRegistry = Enhanced

// Deprecated: Use NewEnhanced instead
// NewEnhancedRegistry creates a new enhanced registry
func NewEnhancedRegistry(config Config) *Enhanced {
	return NewEnhanced(config)
}

// Deprecated: Use Provider instead
// RegistryProvider is the old name for the Provider interface
type RegistryProvider = Provider

// Deprecated: Use Config instead
// RegistryConfig is the old name for the Config struct
type RegistryConfig = Config

// Deprecated: Use Entity instead
// RegistryEntity is the old name for the Entity interface
type RegistryEntity = Entity

// Deprecated: Use Cache instead
// RegistryCache is the old name for the Cache interface
type RegistryCache = Cache

// Deprecated: Use Persistence instead
// RegistryPersistence is the old name for the Persistence interface
type RegistryPersistence = Persistence

// Deprecated: Use Metrics instead
// RegistryMetrics is the old name for the Metrics interface
type RegistryMetrics = Metrics

// Deprecated: Use Health instead
// RegistryHealth is the old name for the Health interface
type RegistryHealth = Health

// Deprecated: Use EventBus instead
// RegistryEventBus is the old name for the EventBus interface
type RegistryEventBus = EventBus

// Deprecated: Use Validator instead
// RegistryValidator is the old name for the Validator interface
type RegistryValidator = Validator

// Deprecated: Use Observer instead
// RegistryObserver is the old name for the Observer interface
type RegistryObserver = Observer

// Deprecated: Use Event instead
// RegistryEvent is the old name for the Event struct
type RegistryEvent = Event

// Deprecated: Use Factory instead
// RegistryFactory is the old name for the Factory interface
type RegistryFactory = Factory

// Deprecated: Use FactoryImpl instead
// RegistryFactoryImpl is the old name for the FactoryImpl struct
type RegistryFactoryImpl = FactoryImpl

// Deprecated: Use NewFactory instead
// NewRegistryFactory is the old name for NewFactory
func NewRegistryFactory() Factory {
	return NewFactory()
}

// Deprecated: Use NewBuilder instead
// NewRegistryBuilder is the old name for NewBuilder
func NewRegistryBuilder() *Builder {
	return NewBuilder()
}
