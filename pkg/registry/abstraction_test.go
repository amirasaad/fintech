package registry

import (
	"context"
	"testing"
	"time"
)

func TestEntityInterface(t *testing.T) {
	// Test BaseEntity implementation
	entity := NewBaseEntity("test-id", "Test Entity")

	if entity.ID() != "test-id" {
		t.Errorf("Expected ID 'test-id', got '%s'", entity.ID())
	}

	if entity.Name() != "Test Entity" {
		t.Errorf("Expected name 'Test Entity', got '%s'", entity.Name())
	}

	if !entity.Active() {
		t.Error("Expected entity to be active")
	}

	metadata := entity.Metadata()
	if metadata == nil {
		t.Error("Expected metadata to be initialized")
	}

	// Test metadata operations
	metadata["key"] = "value"
	if entity.Metadata()["key"] != "value" {
		t.Error("Failed to set metadata")
	}
}

func TestEnhancedRegistry_BasicOperations(t *testing.T) {
	ctx := context.Background()
	config := RegistryConfig{
		Name:             "test-registry",
		EnableEvents:     true,
		EnableValidation: true,
		CacheSize:        10,
		CacheTTL:         time.Minute,
	}

	registry := NewEnhancedRegistry(config)

	// Test registration
	entity := NewBaseEntity("test-1", "Test Entity 1")
	err := registry.Register(ctx, entity)
	if err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	// Test retrieval
	retrieved, err := registry.Get(ctx, "test-1")
	if err != nil {
		t.Fatalf("Failed to get entity: %v", err)
	}

	if retrieved.ID() != "test-1" {
		t.Errorf("Expected ID 'test-1', got '%s'", retrieved.ID())
	}

	// Test listing
	entities, err := registry.List(ctx)
	if err != nil {
		t.Fatalf("Failed to list entities: %v", err)
	}

	if len(entities) != 1 {
		t.Errorf("Expected 1 entity, got %d", len(entities))
	}

	// Test counting
	count, err := registry.Count(ctx)
	if err != nil {
		t.Fatalf("Failed to count entities: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected count 1, got %d", count)
	}

	// Test unregistration
	err = registry.Unregister(ctx, "test-1")
	if err != nil {
		t.Fatalf("Failed to unregister entity: %v", err)
	}

	// Verify unregistration
	if registry.IsRegistered(ctx, "test-1") {
		t.Error("Entity should not be registered after unregistration")
	}
}

func TestEnhancedRegistry_WithCache(t *testing.T) {
	ctx := context.Background()
	config := RegistryConfig{
		Name:             "test-registry",
		EnableEvents:     false,
		EnableValidation: false,
	}

	registry := NewEnhancedRegistry(config)
	cache := NewMemoryCache(time.Minute)
	registry.WithCache(cache)

	// Register entity
	entity := NewBaseEntity("test-1", "Test Entity 1")
	err := registry.Register(ctx, entity)
	if err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	// Verify cache size
	if cache.Size() != 1 {
		t.Errorf("Expected cache size 1, got %d", cache.Size())
	}

	// Test cache retrieval
	cached, found := cache.Get(ctx, "test-1")
	if !found {
		t.Error("Entity should be in cache")
	}

	if cached.ID() != "test-1" {
		t.Errorf("Expected cached entity ID 'test-1', got '%s'", cached.ID())
	}

	// Test cache expiration
	shortCache := NewMemoryCache(time.Millisecond)
	registry.WithCache(shortCache)

	registry.Register(ctx, entity)
	time.Sleep(50 * time.Millisecond)

	// Force a Get to trigger possible cleanup
	_, _ = shortCache.Get(ctx, "test-1")

	if shortCache.Size() != 0 {
		t.Error("Cache should be empty after expiration")
	}
}

func TestEnhancedRegistry_WithMetrics(t *testing.T) {
	ctx := context.Background()
	config := RegistryConfig{
		Name:             "test-registry",
		EnableEvents:     false,
		EnableValidation: false,
	}

	registry := NewEnhancedRegistry(config)
	metrics := NewSimpleMetrics()
	registry.WithMetrics(metrics)

	// Perform operations
	entity := NewBaseEntity("test-1", "Test Entity 1")
	registry.Register(ctx, entity)
	registry.Get(ctx, "test-1")
	registry.Unregister(ctx, "test-1")

	// Check metrics
	stats := metrics.GetStats()

	if stats["registrations"] != int64(1) {
		t.Errorf("Expected 1 registration, got %v", stats["registrations"])
	}

	if stats["lookups"] != int64(1) {
		t.Errorf("Expected 1 lookup, got %v", stats["lookups"])
	}

	if stats["unregistrations"] != int64(1) {
		t.Errorf("Expected 1 unregistration, got %v", stats["unregistrations"])
	}
}

func TestEnhancedRegistry_WithValidation(t *testing.T) {
	ctx := context.Background()
	config := RegistryConfig{
		Name:             "test-registry",
		EnableEvents:     false,
		EnableValidation: true,
	}

	registry := NewEnhancedRegistry(config)
	validator := NewSimpleValidator()
	registry.WithValidator(validator)

	// Test valid entity
	entity := NewBaseEntity("test-1", "Test Entity 1")
	err := registry.Register(ctx, entity)
	if err != nil {
		t.Fatalf("Failed to register valid entity: %v", err)
	}

	// Test invalid entity (empty ID)
	invalidEntity := NewBaseEntity("", "Test Entity 2")
	err = registry.Register(ctx, invalidEntity)
	if err == nil {
		t.Error("Should fail to register entity with empty ID")
	}

	// Test invalid entity (empty name)
	invalidEntity2 := NewBaseEntity("test-2", "")
	err = registry.Register(ctx, invalidEntity2)
	if err == nil {
		t.Error("Should fail to register entity with empty name")
	}
}

func TestEnhancedRegistry_WithEvents(t *testing.T) {
	ctx := context.Background()
	config := RegistryConfig{
		Name:             "test-registry",
		EnableEvents:     true,
		EnableValidation: false,
	}

	registry := NewEnhancedRegistry(config)
	eventBus := NewSimpleEventBus()
	registry.WithEventBus(eventBus)

	// Create test observer
	events := make([]string, 0)
	observer := &testObserver{events: &events}
	eventBus.Subscribe(observer)

	// Perform operations
	entity := NewBaseEntity("test-1", "Test Entity 1")
	registry.Register(ctx, entity)
	registry.Unregister(ctx, "test-1")

	// Check events
	expectedEvents := []string{"registered", "unregistered"}
	if len(events) != len(expectedEvents) {
		t.Errorf("Expected %d events, got %d", len(expectedEvents), len(events))
	}

	for i, expected := range expectedEvents {
		if events[i] != expected {
			t.Errorf("Expected event '%s', got '%s'", expected, events[i])
		}
	}
}

func TestEnhancedRegistry_MetadataOperations(t *testing.T) {
	ctx := context.Background()
	config := RegistryConfig{
		Name:             "test-registry",
		EnableEvents:     false,
		EnableValidation: false,
	}

	registry := NewEnhancedRegistry(config)

	// Register entity
	entity := NewBaseEntity("test-1", "Test Entity 1")
	err := registry.Register(ctx, entity)
	if err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	// Test metadata operations
	err = registry.SetMetadata(ctx, "test-1", "key1", "value1")
	if err != nil {
		t.Fatalf("Failed to set metadata: %v", err)
	}

	value, err := registry.GetMetadata(ctx, "test-1", "key1")
	if err != nil {
		t.Fatalf("Failed to get metadata: %v", err)
	}

	if value != "value1" {
		t.Errorf("Expected metadata value 'value1', got '%s'", value)
	}

	// Test metadata removal
	err = registry.RemoveMetadata(ctx, "test-1", "key1")
	if err != nil {
		t.Fatalf("Failed to remove metadata: %v", err)
	}

	_, err = registry.GetMetadata(ctx, "test-1", "key1")
	if err == nil {
		t.Error("Should fail to get removed metadata")
	}
}

func TestEnhancedRegistry_SearchOperations(t *testing.T) {
	ctx := context.Background()
	config := RegistryConfig{
		Name:             "test-registry",
		EnableEvents:     false,
		EnableValidation: false,
	}

	registry := NewEnhancedRegistry(config)

	// Register multiple entities
	entities := []Entity{
		NewBaseEntity("test-1", "Apple Device"),
		NewBaseEntity("test-2", "Banana Fruit"),
		NewBaseEntity("test-3", "Cherry Berry"),
	}

	for _, entity := range entities {
		err := registry.Register(ctx, entity)
		if err != nil {
			t.Fatalf("Failed to register entity: %v", err)
		}
	}

	// Test search
	results, err := registry.Search(ctx, "Apple")
	if err != nil {
		t.Fatalf("Failed to search: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 search result, got %d", len(results))
	}

	// Test metadata search
	entity := NewBaseEntity("test-4", "Test Entity")
	entity.Metadata()["category"] = "fruit"
	entity.Metadata()["color"] = "red"

	err = registry.Register(ctx, entity)
	if err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	results, err = registry.SearchByMetadata(ctx, map[string]string{"category": "fruit"})
	if err != nil {
		t.Fatalf("Failed to search by metadata: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 metadata search result, got %d", len(results))
	}
}

func TestRegistryFactory(t *testing.T) {
	ctx := context.Background()
	factory := NewRegistryFactory()

	// Test basic creation
	config := RegistryConfig{
		Name:             "test-registry",
		EnableEvents:     true,
		EnableValidation: true,
		CacheSize:        10,
		CacheTTL:         time.Minute,
	}

	registry, err := factory.Create(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create registry: %v", err)
	}

	if registry == nil {
		t.Fatal("Registry should not be nil")
	}

	// Test convenience functions
	basicRegistry := NewBasicRegistry()
	if basicRegistry == nil {
		t.Fatal("Basic registry should not be nil")
	}

	monitoredRegistry := NewMonitoredRegistry("monitored")
	if monitoredRegistry == nil {
		t.Fatal("Monitored registry should not be nil")
	}
}

func TestRegistryBuilder(t *testing.T) {
	builder := NewRegistryBuilder()

	config := builder.
		WithName("test-registry").
		WithMaxEntities(1000).
		WithCache(100, time.Minute).
		WithPersistence("/tmp/test.json", time.Minute).
		WithValidation([]string{"required"}, []string{"forbidden"}).
		Build()

	if config.Name != "test-registry" {
		t.Errorf("Expected name 'test-registry', got '%s'", config.Name)
	}

	if config.MaxEntities != 1000 {
		t.Errorf("Expected max entities 1000, got %d", config.MaxEntities)
	}

	if config.CacheSize != 100 {
		t.Errorf("Expected cache size 100, got %d", config.CacheSize)
	}

	if !config.EnablePersistence {
		t.Error("Persistence should be enabled")
	}

	if len(config.RequiredMetadata) != 1 {
		t.Errorf("Expected 1 required metadata field, got %d", len(config.RequiredMetadata))
	}

	// Test registry creation from builder
	registry, err := builder.BuildRegistry()
	if err != nil {
		t.Fatalf("Failed to build registry: %v", err)
	}

	if registry == nil {
		t.Fatal("Registry should not be nil")
	}
}

func TestFilePersistence(t *testing.T) {
	ctx := context.Background()
	filePath := "/tmp/test_registry.json"

	persistence := NewFilePersistence(filePath)

	// Test save and load
	entities := []Entity{
		NewBaseEntity("test-1", "Test Entity 1"),
		NewBaseEntity("test-2", "Test Entity 2"),
	}

	err := persistence.Save(ctx, entities)
	if err != nil {
		t.Fatalf("Failed to save entities: %v", err)
	}

	loaded, err := persistence.Load(ctx)
	if err != nil {
		t.Fatalf("Failed to load entities: %v", err)
	}

	if len(loaded) != len(entities) {
		t.Errorf("Expected %d loaded entities, got %d", len(entities), len(loaded))
	}

	// Test clear
	err = persistence.Clear(ctx)
	if err != nil {
		t.Fatalf("Failed to clear persistence: %v", err)
	}

	loaded, err = persistence.Load(ctx)
	if err != nil {
		t.Fatalf("Failed to load after clear: %v", err)
	}

	if len(loaded) != 0 {
		t.Error("Should have no entities after clear")
	}
}

// testObserver implements RegistryObserver for testing
type testObserver struct {
	events *[]string
}

func (o *testObserver) OnEntityRegistered(ctx context.Context, entity Entity) {
	*o.events = append(*o.events, "registered")
}

func (o *testObserver) OnEntityUnregistered(ctx context.Context, id string) {
	*o.events = append(*o.events, "unregistered")
}

func (o *testObserver) OnEntityUpdated(ctx context.Context, entity Entity) {
	*o.events = append(*o.events, "updated")
}

func (o *testObserver) OnEntityActivated(ctx context.Context, id string) {
	*o.events = append(*o.events, "activated")
}

func (o *testObserver) OnEntityDeactivated(ctx context.Context, id string) {
	*o.events = append(*o.events, "deactivated")
}
