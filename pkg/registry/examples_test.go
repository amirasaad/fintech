package registry

import (
	"fmt"
	"time"
)

// Example demonstrates basic registry usage
func Example() {
	// Create a new registry
	registry := New()

	// Register some entities
	registry.Register("user-1", Meta{
		Name:     "John Doe",
		Active:   true,
		Metadata: map[string]string{"email": "john@example.com", "role": "admin"},
	})

	registry.Register("user-2", Meta{
		Name:     "Jane Smith",
		Active:   true,
		Metadata: map[string]string{"email": "jane@example.com", "role": "user"},
	})

	// Get an entity
	user := registry.Get("user-1")
	fmt.Printf("User: %s (%s)\n", user.Name, user.Metadata["email"])

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
	globalRegistry = New() // This avoids pollution from other tests/examples

	// Use global registry functions
	Register("global-user", Meta{
		Name:     "Global User",
		Active:   true,
		Metadata: map[string]string{"type": "global"},
	})

	// Get from global registry
	user := Get("global-user")
	fmt.Printf("Global user: %s\n", user.Name)

	// Check registration
	fmt.Printf("Is registered: %t\n", IsRegistered("global-user"))

	// List all
	all := ListRegistered()
	fmt.Printf("Global registry count: %d\n", len(all))

	// Set metadata
	SetMetadata("global-user", "status", "active")

	// Get metadata
	status, found := GetMetadata("global-user", "status")
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

	// Register with metadata
	registry.Register("product-1", Meta{
		Name:   "Laptop",
		Active: true,
		Metadata: map[string]string{
			"category": "Electronics",
			"price":    "999.99",
			"brand":    "TechCorp",
		},
	})

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
	// Category: Electronics (found: true)
	// Price: 999.99 (found: true)
	// In stock: true (found: true)
	// Non-existent:  (found: false)
}

// Example_fourth demonstrates entity lifecycle management
func Example_fourth() {
	registry := New()

	// Register an entity
	registry.Register("entity-1", Meta{
		Name:     "Test Entity",
		Active:   true,
		Metadata: map[string]string{"version": "1.0"},
	})

	fmt.Printf("Initial count: %d\n", registry.Count())

	// Deactivate by setting Active to false
	entity := registry.Get("entity-1")
	entity.Active = false
	registry.Register("entity-1", entity)

	fmt.Printf("Active entities: %d\n", len(registry.ListActive()))

	// Reactivate
	entity.Active = true
	registry.Register("entity-1", entity)

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
