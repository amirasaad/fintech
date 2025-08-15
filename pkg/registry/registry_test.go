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
	entity1 := NewBaseEntity("test1", "Test Entity")
	entity1.BEActive = true
	entity1.BEMetadata = make(map[string]string)
	entity1.BECreatedAt = time.Now()
	entity1.BEUpdatedAt = time.Now()
	registry.Register(entity1.ID(), entity1)
	assert.True(registry.IsRegistered("test1"))

	// Update existing entity
	updatedEntity := NewBaseEntity("test1", "Updated Entity")
	updatedEntity.BEActive = false
	updatedEntity.BEMetadata = make(map[string]string)
	updatedEntity.BECreatedAt = entity1.CreatedAt()
	updatedEntity.BEUpdatedAt = time.Now()
	registry.Register(updatedEntity.ID(), updatedEntity)
	entity := registry.Get("test1")
	assert.Equal("Updated Entity", entity.Name())
	assert.False(entity.Active())
}

func TestRegistry_Get(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)

	registry := New()

	// Test existing entity
	entity1 := &BaseEntity{
		BEId:        "test1",
		BEName:      "Test Entity",
		BEActive:    true,
		BEMetadata:  make(map[string]string),
		BECreatedAt: time.Now(),
		BEUpdatedAt: time.Now(),
	}
	registry.Register("test1", entity1)
	entity := registry.Get("test1")
	assert.Equal("test1", entity.ID())
	assert.Equal("Test Entity", entity.Name())
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
	active1.BEActive = true
	active2 := NewBaseEntity("active2", "Active 2")
	active2.BEActive = true
	inactive1 := NewBaseEntity("inactive1", "Inactive 1")
	inactive1.BEActive = false

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
	entity.BEMetadata = make(map[string]string)
	registry.Register("test1", entity)

	// Set metadata
	assert.True(registry.SetMetadata("test1", "key1", "value1"))
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

func TestGlobalFunctions(t *testing.T) {
	assert := assert.New(t)

	// Reset global registry to ensure clean state
	globalRegistry = New()

	// Test global Register function
	entity := NewBaseEntity("global1", "Global 1")
	entity.BEActive = true
	Register("global1", entity)
	assert.True(IsRegistered("global1"))

	// Test global Get function
	retrievedEntity := Get("global1")
	assert.NotNil(retrievedEntity)
	assert.Equal("Global 1", retrievedEntity.Name())
	assert.True(retrievedEntity.Active())

	// Test global ListRegistered function
	registered := ListRegistered()
	assert.Contains(registered, "global1")

	// Test global ListActive function
	active := ListActive()
	assert.Contains(active, "global1")

	// Test global Count function
	assert.GreaterOrEqual(Count(), 1)

	// Test global metadata functions
	assert.True(SetMetadata("global1", "test_key", "test_value"))
	value, found := GetMetadata("global1", "test_key")
	assert.True(found)
	assert.Equal("test_value", value)

	// Clean up
	Unregister("global1")
	assert.False(IsRegistered("global1"))
	assert.False(IsRegistered("global1"))
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

func TestGlobalRegistry_ThreadSafety(t *testing.T) {
	assert := assert.New(t)

	// Reset global registry and add a test entity
	globalRegistry = New()
	testEntity := NewBaseEntity("test", "Test Entity")
	Register("test", testEntity)

	var wg sync.WaitGroup
	numGoroutines := 50

	// Concurrent reads
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			entity := Get("test")
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
			id := fmt.Sprintf("global%d", index)
			entity := NewBaseEntity(id, fmt.Sprintf("Global %d", index))
			Register(id, entity)
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
	entity.BEActive = true
	entity.BEMetadata = map[string]string{"key": "value"}

	// Test the entity fields and methods
	assert.Equal("test", entity.ID())
	assert.Equal("Test Entity", entity.Name())
	assert.True(entity.Active())
	assert.Equal("value", entity.Metadata()["key"])
}
