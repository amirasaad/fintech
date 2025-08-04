package registry

import (
	"fmt"
	"sync"
	"testing"

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
	registry.Register("test1", Meta{Name: "Test Entity", Active: true})
	assert.True(registry.IsRegistered("test1"))

	// Update existing entity
	registry.Register("test1", Meta{Name: "Updated Entity", Active: false})
	entity := registry.Get("test1")
	assert.Equal("Updated Entity", entity.Name)
	assert.False(entity.Active)
}

func TestRegistry_Get(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)

	registry := New()

	// Test existing entity
	registry.Register("test1", Meta{Name: "Test Entity", Active: true})
	entity := registry.Get("test1")
	assert.Equal("test1", entity.ID)
	assert.Equal("Test Entity", entity.Name)
	assert.True(entity.Active)

	// Test unknown entity returns empty meta
	unknown := registry.Get("unknown")
	assert.Equal("unknown", unknown.ID)
	assert.False(unknown.Active)
}

func TestRegistry_IsRegistered(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)

	registry := New()

	assert.False(registry.IsRegistered("test1"))
	registry.Register("test1", Meta{Name: "Test"})
	assert.True(registry.IsRegistered("test1"))
}

func TestRegistry_ListRegistered(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)

	registry := New()
	registered := registry.ListRegistered()
	assert.Empty(registered)

	registry.Register("test1", Meta{Name: "Test 1"})
	registry.Register("test2", Meta{Name: "Test 2"})
	registered = registry.ListRegistered()

	assert.Contains(registered, "test1")
	assert.Contains(registered, "test2")
	assert.Len(registered, 2)
}

func TestRegistry_ListActive(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)

	registry := New()
	registry.Register("active1", Meta{Name: "Active 1", Active: true})
	registry.Register("active2", Meta{Name: "Active 2", Active: true})
	registry.Register("inactive1", Meta{Name: "Inactive 1", Active: false})

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
	registry.Register("test1", Meta{Name: "Test"})

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

	registry.Register("test1", Meta{Name: "Test 1"})
	assert.Equal(1, registry.Count())

	registry.Register("test2", Meta{Name: "Test 2"})
	assert.Equal(2, registry.Count())

	registry.Unregister("test1")
	assert.Equal(1, registry.Count())
}

func TestRegistry_Metadata(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)

	registry := New()
	registry.Register("test1", Meta{Name: "Test"})

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

	// Set metadata for non-existent entity
	assert.False(registry.SetMetadata("nonexistent", "key", "value"))
}

func TestGlobalFunctions(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)

	// Reset global registry to ensure clean state
	globalRegistry = New()

	// Test global Register function
	Register("global1", Meta{Name: "Global 1", Active: true})
	assert.True(IsRegistered("global1"))

	// Test global Get function
	entity := Get("global1")
	assert.Equal("Global 1", entity.Name)
	assert.True(entity.Active)

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

	// Test global Unregister function
	assert.True(Unregister("global1"))
	assert.False(IsRegistered("global1"))
}

func TestRegistry_ThreadSafety(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)

	registry := New()
	var wg sync.WaitGroup
	numGoroutines := 100

	// Concurrent reads
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			entity := registry.Get("test")
			assert.Equal("test", entity.ID)
		}()
	}

	// Concurrent writes
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(index int) {
			defer wg.Done()
			id := fmt.Sprintf("test%d", index)
			registry.Register(id, Meta{Name: fmt.Sprintf("Test %d", index)})
		}(i)
	}

	wg.Wait()
	// If we reach here without race conditions, the test passes
}

func TestGlobalRegistry_ThreadSafety(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)

	var wg sync.WaitGroup
	numGoroutines := 50

	// Concurrent reads
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			entity := Get("test")
			assert.Equal("test", entity.ID)
		}()
	}

	// Concurrent writes
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(index int) {
			defer wg.Done()
			id := fmt.Sprintf("global%d", index)
			Register(id, Meta{Name: fmt.Sprintf("Global %d", index)})
		}(i)
	}

	wg.Wait()
	// If we reach here without race conditions, the test passes
}

func TestMeta_Structure(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)

	meta := Meta{
		ID:       "test",
		Name:     "Test Entity",
		Active:   true,
		Metadata: map[string]string{"key": "value"},
	}

	assert.Equal("test", meta.ID)
	assert.Equal("Test Entity", meta.Name)
	assert.True(meta.Active)
	assert.Equal("value", meta.Metadata["key"])
}
