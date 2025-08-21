package registry

import (
	"fmt"
	"time"
)

// ExampleRegistry demonstrates basic registry usage
func ExampleRegistry() {
	// Create a new registry instance
	registry := New()

	// Create entities using NewBaseEntity and set properties using methods
	user1 := NewBaseEntity("user-1", "John Doe")
	user1.SetActive(true)
	user1.SetMetadata("email", "john@example.com")
	user1.SetMetadata("role", "admin")

	user2 := NewBaseEntity("user-2", "Jane Smith")
	user2.SetActive(true)
	user2.SetMetadata("email", "jane@example.com")
	user2.SetMetadata("role", "user")

	// Register entities
	registry.Register(user1.ID(), user1)
	registry.Register(user2.ID(), user2)

	// Get an entity
	user := registry.Get("user-1")
	if user != nil {
		fmt.Printf("User: %s (%s)\n", user.Name(), user.Metadata()["email"])
	}

	// Check if registered
	fmt.Printf("User-1 registered: %t\n", registry.IsRegistered("user-1"))

	// List all registered entities
	registered := registry.ListRegistered()
	fmt.Printf("Total registered: %d\n", len(registered))

	// List active entities
	active := registry.ListActive()
	fmt.Printf("Active entities: %d\n", len(active))

	// Get metadata
	email, found := registry.GetMetadata("user-1", "email")
	fmt.Printf("Email found: %t, value: %s\n", found, email)

	// Set metadata
	registry.SetMetadata("user-1", "last_login", time.Now().Format(time.RFC3339))

	// Count entities
	fmt.Printf("Total count: %d\n", registry.Count())

	// Unregister an entity
	removed := registry.Unregister("user-2")
	fmt.Printf("User-2 removed: %t\n", removed)

	// Output:
	// User: John Doe (john@example.com)
	// User-1 registered: true
	// Total registered: 2
	// Active entities: 2
	// Email found: true, value: john@example.com
	// Total count: 2
	// User-2 removed: true
}

// Example_second demonstrates global registry functions
func Example_second() {
	// Reset the global registry to ensure a clean state for the example
	registry := New() // This avoids pollution from other tests/examples

	// Use global registry functions
	globalUser := NewBaseEntity("global-user", "Global User")
	// Using setter methods (recommended)
	globalUser.SetActive(true)
	globalUser.SetMetadata("email", "user@example.com")
	globalUser.SetMetadata("role", "admin")
	registry.Register(globalUser.ID(), globalUser)

	// Get from global registry
	user := registry.Get("global-user")
	if user != nil {
		fmt.Printf("Global user: %s\n", user.Name())
	}

	// Check registration
	fmt.Printf("Is registered: %t\n", registry.IsRegistered("global-user"))

	// List all
	all := registry.ListRegistered()
	fmt.Printf("Global registry count: %d\n", len(all))

	// Set metadata
	registry.SetMetadata("global-user", "status", "active")

	// Get metadata
	status, found := registry.GetMetadata("global-user", "status")
	fmt.Printf("Status: %s (found: %t)\n", status, found)

	// Output:
	// Global user: Global User
	// Is registered: true
	// Global registry count: 1
	// Status: active (found: true)
}

// Example_third demonstrates metadata operations
func Example_third() {
	registry := New()

	// Create and register with metadata
	product := NewBaseEntity("product-1", "Laptop")
	// Using setter methods (recommended)
	product.SetActive(true)
	product.SetMetadata("category", "electronics")
	product.SetMetadata("price", "99.99")
	registry.Register("product-1", product)

	// Get specific metadata
	category, found := registry.GetMetadata("product-1", "category")
	fmt.Printf("Category: %s (found: %t)\n", category, found)

	price, found := registry.GetMetadata("product-1", "price")
	fmt.Printf("Price: %s (found: %t)\n", price, found)

	// Set new metadata
	registry.SetMetadata("product-1", "in_stock", "true")
	registry.SetMetadata("product-1", "warranty", "2 years")

	// Get updated metadata
	stock, found := registry.GetMetadata("product-1", "in_stock")
	fmt.Printf("In stock: %s (found: %t)\n", stock, found)

	// Try to get non-existent metadata
	nonExistent, found := registry.GetMetadata("product-1", "non_existent")
	fmt.Printf("Non-existent: %s (found: %t)\n", nonExistent, found)

	// Output:
	// Category: electronics (found: true)
	// Price: 99.99 (found: true)
	// In stock: true (found: true)
	// Non-existent:  (found: false)
}

// Example_fourth demonstrates entity lifecycle management
func Example_fourth() {
	registry := New()

	// Create and register an entity
	entity := NewBaseEntity("entity-1", "Test Entity")
	entity.SetActive(true)
	entity.SetMetadata("version", "1.0")
	registry.Register("entity-1", entity)

	fmt.Printf("Initial count: %d\n", registry.Count())

	// Deactivate
	entityFromRegistry := registry.Get("entity-1")
	if be, ok := entityFromRegistry.(*BaseEntity); ok {
		be.SetActive(false)
		registry.Register("entity-1", be)
	}

	fmt.Printf("Active entities: %d\n", len(registry.ListActive()))

	// Reactivate
	entityFromRegistry = registry.Get("entity-1")
	if be, ok := entityFromRegistry.(*BaseEntity); ok {
		be.SetActive(true)
		registry.Register("entity-1", be)
	}

	fmt.Printf("Active entities after reactivation: %d\n", len(registry.ListActive()))

	// Unregister
	removed := registry.Unregister("entity-1")
	fmt.Printf("Removed: %t, Final count: %d\n", removed, registry.Count())

	// Output:
	// Initial count: 1
	// Active entities: 0
	// Active entities after reactivation: 1
	// Removed: true, Final count: 0
}
