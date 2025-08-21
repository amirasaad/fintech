package registry

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewRegistry(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)

	registry := New()
	assert.NotNil(registry)
	assert.Equal(0, registry.Count())
}

func TestRegistry_Register(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)

	registry := New()

	// Register new entity
	entity1 := NewBaseEntity("test1", "Test Entity 1")
	entity1.SetActive(true)
	registry.Register(entity1.ID(), entity1)
	assert.True(registry.IsRegistered("test1"))

	// Verify entity fields
	if entity1.ID() != "test1" || entity1.Name() != "Test Entity 1" || !entity1.Active() {
		t.Errorf("Entity fields not set correctly")
	}

	// Verify entity status
	if !entity1.Active() {
		t.Error("Entity should be active")
	}

	// Update existing entity
	updatedEntity := NewBaseEntity("test1", "Updated Entity")
	updatedEntity.SetActive(false)
	registry.Register(updatedEntity.ID(), updatedEntity)
	entity := registry.Get("test1")
	assert.Equal("test1", entity.ID())
	assert.Equal("Updated Entity", entity.Name())
	assert.False(entity.Active())
}

func TestRegistry_Get(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)

	registry := New()

	// Create a test entity
	entity1 := NewBaseEntity("test1", "Test Entity 1")
	entity1.SetActive(true)
	entity1.SetMetadata("key1", "value1")
	registry.Register("test1", entity1)
	entity := registry.Get("test1")
	assert.Equal("test1", entity.ID())
	assert.Equal("Test Entity 1", entity.Name())
	assert.True(entity.Active())

	// Test unknown entity returns nil
	unknown := registry.Get("nonexistent")
	assert.Nil(unknown)
}

func TestRegistry_IsRegistered(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)

	registry := New()

	assert.False(registry.IsRegistered("test1"))
	entity := &BaseEntity{
		BEId:        "test1",
		BEName:      "Test Entity",
		BEActive:    true,
		BEMetadata:  make(map[string]string),
		BECreatedAt: time.Now(),
		BEUpdatedAt: time.Now(),
	}
	registry.Register("test1", entity)
	assert.True(registry.IsRegistered("test1"))
	registry.Unregister("test1")
	assert.False(registry.IsRegistered("test1"))
}

func TestRegistry_ListRegistered(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)

	registry := New()
	registered := registry.ListRegistered()
	assert.Empty(registered)

	test1 := NewBaseEntity("test1", "Test 1")
	test2 := NewBaseEntity("test2", "Test 2")
	registry.Register("test1", test1)
	registry.Register("test2", test2)
	registered = registry.ListRegistered()

	assert.Contains(registered, "test1")
	assert.Contains(registered, "test2")
	assert.Len(registered, 2)
}

func TestRegistry_ListActive(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)

	registry := New()
	active1 := NewBaseEntity("active1", "Active 1")
	active1.SetActive(true)
	active2 := NewBaseEntity("active2", "Active 2")
	active2.SetActive(true)
	inactive1 := NewBaseEntity("inactive1", "Inactive 1")
	inactive1.SetActive(false)

	registry.Register("active1", active1)
	registry.Register("active2", active2)
	registry.Register("inactive1", inactive1)

	active := registry.ListActive()
	assert.Contains(active, "active1")
	assert.Contains(active, "active2")
	assert.NotContains(active, "inactive1")
	assert.Len(active, 2)
}

func TestRegistry_Unregister(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)

	registry := New()
	testEntity := NewBaseEntity("test1", "Test")
	registry.Register("test1", testEntity)

	assert.True(registry.IsRegistered("test1"))
	assert.True(registry.Unregister("test1"))
	assert.False(registry.IsRegistered("test1"))
	assert.False(registry.Unregister("test1")) // Already unregistered
}

func TestRegistry_Count(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)

	registry := New()
	assert.Equal(0, registry.Count())

	test1 := NewBaseEntity("test1", "Test 1")
	registry.Register("test1", test1)
	assert.Equal(1, registry.Count())

	test2 := NewBaseEntity("test2", "Test 2")
	registry.Register("test2", test2)
	assert.Equal(2, registry.Count())

	registry.Unregister("test1")
	assert.Equal(1, registry.Count())
}

func TestRegistry_Metadata(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)

	registry := New()
	entity := NewBaseEntity("test1", "Test")
	entity.SetActive(true)
	entity.SetMetadata("key1", "value1")
	registry.Register("test1", entity)

	// Set metadata
	assert.True(registry.SetMetadata("test1", "key2", "value2"))

	// Get metadata
	value, found := registry.GetMetadata("test1", "key1")
	assert.True(found)
	assert.Equal("value1", value)

	value, found = registry.GetMetadata("test1", "key2")
	assert.True(found)
	assert.Equal("value2", value)

	// Get non-existent metadata
	value, found = registry.GetMetadata("test1", "nonexistent")
	assert.False(found)
	assert.Empty(value)

	// Test with non-existent entity
	value, found = registry.GetMetadata("nonexistent", "key1")
	assert.False(found)
	assert.Empty(value)
}

func TestRegistryOperations(t *testing.T) {
	// Create a new registry instance
	r := New()
	const testID = "test-entity"
	entity := &BaseEntity{
		BEId:     testID,
		BEName:   "Test Entity",
		BEActive: true,
	}

	// Register
	r.Register(testID, entity)

	// Verify registration
	if !r.IsRegistered(testID) {
		t.Error("Expected entity to be registered")
	}

	// Get
	retrieved := r.Get(testID)
	if retrieved == nil || retrieved.ID() != testID {
		t.Error("Failed to retrieve registered entity")
	}

	// List
	registered := r.ListRegistered()
	if len(registered) != 1 || registered[0] != testID {
		t.Error("ListRegistered failed")
	}

	// ListActive
	active := r.ListActive()
	if len(active) != 1 || active[0] != testID {
		t.Error("ListActive failed")
	}

	// Count
	if count := r.Count(); count != 1 {
		t.Errorf("Expected count 1, got %d", count)
	}

	// Metadata
	r.SetMetadata(testID, "key", "value")
	// Get fresh instance after metadata update
	retrieved = r.Get(testID)
	val, ok := retrieved.Metadata()["key"]
	if !ok || val != "value" {
		t.Error("Metadata operations failed - could not get metadata after set")
	}

	// Test metadata removal
	r.RemoveMetadata(testID, "key")
	// Get fresh instance after metadata removal
	retrieved = r.Get(testID)
	_, ok = retrieved.Metadata()["key"]
	if ok {
		t.Error("Metadata operations failed - metadata not removed")
	}

	// Unregister
	if !r.Unregister(testID) {
		t.Error("Failed to unregister entity")
	}
	if r.IsRegistered(testID) {
		t.Error("Entity still registered after unregister")
	}
}

func TestRegistry_ThreadSafety(t *testing.T) {
	assert := assert.New(t)

	registry := New()
	var wg sync.WaitGroup
	numGoroutines := 100

	// Add a test entity
	testEntity := NewBaseEntity("test", "Test Entity")
	registry.Register("test", testEntity)

	// Concurrent reads
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			entity := registry.Get("test")
			if entity != nil {
				assert.Equal("test", entity.ID())
			}
		}()
	}

	// Concurrent writes
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(index int) {
			defer wg.Done()
			id := fmt.Sprintf("test%d", index)
			entity := NewBaseEntity(id, fmt.Sprintf("Test %d", index))
			registry.Register(id, entity)
		}(i)
	}

	wg.Wait()
	// If we reach here without race conditions, the test passes
}

func TestRegistry_ConcurrentAccess(t *testing.T) {
	assert := assert.New(t)
	registry := New()

	// Add initial test entity
	testEntity := NewBaseEntity("test", "Test Entity")
	registry.Register("test", testEntity)

	var wg sync.WaitGroup
	numGoroutines := 50

	// Concurrent reads
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			entity := registry.Get("test")
			if entity != nil {
				assert.Equal("test", entity.ID())
			}
		}()
	}

	// Concurrent writes
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(index int) {
			defer wg.Done()
			id := fmt.Sprintf("entity%d", index)
			entity := NewBaseEntity(id, fmt.Sprintf("Entity %d", index))
			registry.Register(id, entity)
		}(i)
	}

	wg.Wait()
	// If we reach here without race conditions, the test passes
}

func TestMeta_Structure(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)

	// Create a new BaseEntity
	entity := NewBaseEntity("test", "Test Entity")
	entity.SetActive(true)
	entity.SetMetadata("key", "value")

	// Test the entity fields and methods
	assert.Equal("test", entity.ID())
	assert.Equal("Test Entity", entity.Name())
	assert.True(entity.Active())

	// Get a fresh copy of metadata to ensure thread safety
	metadata := entity.Metadata()
	assert.Equal("value", metadata["key"])
}
