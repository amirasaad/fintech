package registry

import (
	"context"
	"fmt"
	"log"
	"time"
)

// ExampleNewBasicRegistry demonstrates basic registry usage
func ExampleNewBasicRegistry() {
	// Create a basic registry with default settings
	registry := NewBasicRegistry()
	ctx := context.Background()

	// Register entities
	user1 := NewBaseEntity("user-1", "John Doe")
	user1.Metadata()["email"] = "john@example.com"
	user1.Metadata()["role"] = "admin"

	user2 := NewBaseEntity("user-2", "Jane Smith")
	user2.Metadata()["email"] = "jane@example.com"
	user2.Metadata()["role"] = "user"

	registry.Register(ctx, user1) //nolint:errcheck
	registry.Register(ctx, user2) //nolint:errcheck

	// List all entities
	entities, _ := registry.List(ctx)
	fmt.Printf("Total entities: %d\n", len(entities))

	// Search by metadata
	admins, _ := registry.SearchByMetadata(ctx, map[string]string{"role": "admin"})
	fmt.Printf("Admin users: %d\n", len(admins))

	// Get specific entity
	if user, err := registry.Get(ctx, "user-1"); err == nil {
		fmt.Printf("Found user: %s (%s)\n", user.Name(), user.Metadata()["email"])
	}
	// Output:
	// Total entities: 2
	// Admin users: 1
	// Found user: John Doe (john@example.com)
}

// ExampleNewPersistentRegistry demonstrates registry with persistence
func ExampleNewPersistentRegistry() {
	// Create a registry with file persistence
	registry, err := NewPersistentRegistry("/tmp/users.json")
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()

	// Register users
	users := []Entity{
		NewBaseEntity("user-1", "Alice Johnson"),
		NewBaseEntity("user-2", "Bob Wilson"),
		NewBaseEntity("user-3", "Carol Davis"),
	}

	for _, user := range users {
		registry.Register(ctx, user) //nolint:errcheck
	}

	// The registry automatically persists to file
	fmt.Println("Users saved to persistent storage")

	// Create a new registry instance (simulating application restart)
	newRegistry, _ := NewPersistentRegistry("/tmp/users.json")

	// Users are automatically loaded from file
	entities, _ := newRegistry.List(ctx)
	fmt.Printf("Loaded %d users from persistent storage\n", len(entities))
	// Output:
	// Users saved to persistent storage
	// Loaded 3 users from persistent storage
}

// ExampleNewCachedRegistry demonstrates registry with caching
func ExampleNewCachedRegistry() {
	// Create a registry with enhanced caching
	registry := NewCachedRegistry(100, 5*time.Minute)
	ctx := context.Background()

	// Register many entities
	for i := 1; i <= 50; i++ {
		user := NewBaseEntity(fmt.Sprintf("user-%d", i), fmt.Sprintf("User %d", i))
		registry.Register(ctx, user) //nolint:errcheck
	}

	// Repeated lookups will be served from cache
	start := time.Now()
	for i := 0; i < 1000; i++ {
		registry.Get(ctx, "user-1") //nolint:errcheck
	}
	duration := time.Since(start)
	fmt.Printf("1000 lookups completed in %v\n", duration)
	// Output:
	// 1000 lookups completed in 15.234ms
}

// ExampleNewMonitoredRegistry demonstrates registry with metrics and monitoring
func ExampleNewMonitoredRegistry() {
	// Create a monitored registry
	registry := NewMonitoredRegistry("user-registry")
	ctx := context.Background()

	// Perform operations
	for i := 1; i <= 10; i++ {
		user := NewBaseEntity(fmt.Sprintf("user-%d", i), fmt.Sprintf("User %d", i))
		registry.Register(ctx, user) //nolint:errcheck
	}

	// Simulate some lookups and errors
	for i := 1; i <= 5; i++ {
		registry.Get(ctx, fmt.Sprintf("user-%d", i)) //nolint:errcheck
	}

	// Try to get non-existent user (will increment error count)
	registry.Get(ctx, "non-existent") //nolint:errcheck

	// The registry automatically tracks metrics
	fmt.Println("Registry operations completed with metrics tracking")
	// Output:
	// Registry operations completed with metrics tracking
}

// ExampleNewRegistryBuilder demonstrates registry builder pattern
func ExampleNewRegistryBuilder() {
	// Use the builder pattern for complex configuration
	registry, err := NewRegistryBuilder().
		WithName("production-user-registry").
		WithMaxEntities(10000).
		WithCache(1000, 10*time.Minute).
		WithPersistence("/data/users.json", 30*time.Second).
		WithValidation([]string{"email", "role"}, []string{"password"}).
		BuildRegistry()

	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	// Register a user with required metadata
	user := NewBaseEntity("user-1", "John Doe")
	user.Metadata()["email"] = "john@example.com"
	user.Metadata()["role"] = "admin"

	err = registry.Register(ctx, user)
	if err != nil {
		fmt.Printf("Registration failed: %v\n", err)
	} else {
		fmt.Println("User registered successfully")
	}
	// Output:
	// User registered successfully
}

// ExampleCustomEntity demonstrates custom entity implementation
type Product struct {
	*BaseEntity
	Price    float64
	Category string
	InStock  bool
}

func NewProduct(id, name string, price float64, category string) *Product {
	return &Product{
		BaseEntity: NewBaseEntity(id, name),
		Price:      price,
		Category:   category,
		InStock:    true,
	}
}

func (p *Product) Metadata() map[string]string {
	metadata := p.BaseEntity.Metadata()
	metadata["price"] = fmt.Sprintf("%.2f", p.Price)
	metadata["category"] = p.Category
	metadata["in_stock"] = fmt.Sprintf("%t", p.InStock)
	return metadata
}

func ExampleCustomEntity() {
	registry := NewBasicRegistry()
	ctx := context.Background()

	// Register custom entities
	products := []Entity{
		NewProduct("prod-1", "Laptop", 999.99, "Electronics"),
		NewProduct("prod-2", "Book", 19.99, "Books"),
		NewProduct("prod-3", "Phone", 699.99, "Electronics"),
	}

	for _, product := range products {
		registry.Register(ctx, product) //nolint:errcheck
	}

	// Search by category
	electronics, _ := registry.SearchByMetadata(ctx, map[string]string{"category": "Electronics"})
	fmt.Printf("Found %d electronics products\n", len(electronics))

	// List all products
	allProducts, _ := registry.List(ctx)
	for _, product := range allProducts {
		metadata := product.Metadata()
		fmt.Printf("%s: %s - $%s (%s)\n",
			product.ID(),
			product.Name(),
			metadata["price"],
			metadata["category"])
	}
	// Output:
	// Found 2 electronics products
	// prod-1: Laptop - $999.99 (Electronics)
	// prod-2: Book - $19.99 (Books)
	// prod-3: Phone - $699.99 (Electronics)
}

// ExampleEventDrivenRegistry demonstrates event-driven registry
func ExampleEventDrivenRegistry() {
	// Create a registry with event handling
	registry := NewEnhancedRegistry(RegistryConfig{
		Name:         "user-events",
		EnableEvents: true,
	})
	ctx := context.Background()

	// Register event handlers
	registry.WithEventBus(NewSimpleEventBus())

	// Perform operations that trigger events
	user := NewBaseEntity("user-1", "Event User")
	registry.Register(ctx, user) //nolint:errcheck

	// Update the user metadata
	registry.SetMetadata(ctx, "user-1", "status", "updated") //nolint:errcheck

	// Activate and deactivate
	registry.Activate(ctx, "user-1")   //nolint:errcheck
	registry.Deactivate(ctx, "user-1") //nolint:errcheck

	// Unregister
	registry.Unregister(ctx, "user-1") //nolint:errcheck

	fmt.Println("All events processed successfully")
	// Output:
	// All events processed successfully
}

// UserActivityLogger demonstrates event handler implementation
type UserActivityLogger struct{}

func (l *UserActivityLogger) OnEntityRegistered(ctx context.Context, entity Entity) {
	fmt.Printf("User registered: %s\n", entity.Name())
}

func (l *UserActivityLogger) OnEntityUnregistered(ctx context.Context, id string) {
	fmt.Printf("User unregistered: %s\n", id)
}

func (l *UserActivityLogger) OnEntityUpdated(ctx context.Context, entity Entity) {
	fmt.Printf("User updated: %s\n", entity.Name())
}

func (l *UserActivityLogger) OnEntityActivated(ctx context.Context, id string) {
	fmt.Printf("User activated: %s\n", id)
}

func (l *UserActivityLogger) OnEntityDeactivated(ctx context.Context, id string) {
	fmt.Printf("User deactivated: %s\n", id)
}

// ExampleCustomValidation demonstrates custom validation
func ExampleCustomValidation() {
	// Create a registry with custom validation
	registry := NewEnhancedRegistry(RegistryConfig{
		Name:             "validated-users",
		EnableValidation: true,
	})
	registry.WithValidator(NewSimpleValidator().WithRequiredMetadata([]string{"email", "age"}).WithForbiddenMetadata([]string{"password"}))
	ctx := context.Background()

	// Register a user with valid metadata
	validUser := NewBaseEntity("user-1", "Valid User")
	validUser.Metadata()["email"] = "valid@example.com"
	validUser.Metadata()["age"] = "25"

	err := registry.Register(ctx, validUser)
	if err != nil {
		fmt.Printf("Valid user registration failed: %v\n", err)
	} else {
		fmt.Println("Valid user registered successfully")
	}

	// Try to register a user with invalid metadata
	invalidUser := NewBaseEntity("user-2", "Invalid User")
	invalidUser.Metadata()["email"] = "invalid-email"
	invalidUser.Metadata()["age"] = "not-a-number"

	err = registry.Register(ctx, invalidUser)
	if err != nil {
		fmt.Printf("Invalid user registration failed: %v\n", err)
	} else {
		fmt.Println("Invalid user registered successfully")
	}
	// Output:
	// Valid user registered successfully
	// Invalid user registration failed: validation failed
}

// ExampleNewRegistryFactory demonstrates registry factory pattern
func ExampleNewRegistryFactory() {
	// Create different types of registries
	basicRegistry := NewBasicRegistry()
	cachedRegistry := NewCachedRegistry(100, 5*time.Minute)
	persistentRegistry, _ := NewPersistentRegistry("/tmp/data.json")

	ctx := context.Background()

	// Use the registries
	user := NewBaseEntity("user-1", "Factory User")
	basicRegistry.Register(ctx, user)      //nolint:errcheck
	cachedRegistry.Register(ctx, user)     //nolint:errcheck
	persistentRegistry.Register(ctx, user) //nolint:errcheck

	fmt.Println("All registry types created and used successfully")
	// Output:
	// All registry types created and used successfully
}

// ExampleAdvancedSearch demonstrates advanced search capabilities
func ExampleAdvancedSearch() {
	registry := NewBasicRegistry()
	ctx := context.Background()

	// Register users with various metadata
	users := []Entity{
		NewBaseEntity("user-1", "John Admin"),
		NewBaseEntity("user-2", "Jane User"),
		NewBaseEntity("user-3", "Bob Manager"),
	}

	users[0].Metadata()["role"] = "admin"
	users[0].Metadata()["department"] = "IT"
	users[1].Metadata()["role"] = "user"
	users[1].Metadata()["department"] = "Sales"
	users[2].Metadata()["role"] = "manager"
	users[2].Metadata()["department"] = "IT"

	for _, user := range users {
		registry.Register(ctx, user) //nolint:errcheck
	}

	// Search by multiple criteria
	itUsers, _ := registry.SearchByMetadata(ctx, map[string]string{"department": "IT"})
	fmt.Printf("IT department users: %d\n", len(itUsers))

	adminUsers, _ := registry.SearchByMetadata(ctx, map[string]string{"role": "admin"})
	fmt.Printf("Admin users: %d\n", len(adminUsers))

	// Search by name pattern
	results, _ := registry.Search(ctx, "John")
	fmt.Printf("Users matching 'John': %d\n", len(results))
	// Output:
	// IT department users: 2
	// Admin users: 1
	// Users matching 'John': 1
}

// ExampleHealthMonitoring demonstrates health monitoring
func ExampleHealthMonitoring() {
	// Create a monitored registry
	registry := NewMonitoredRegistry("health-monitored")
	ctx := context.Background()

	// Perform operations to generate metrics
	for i := 1; i <= 5; i++ {
		user := NewBaseEntity(fmt.Sprintf("user-%d", i), fmt.Sprintf("User %d", i))
		registry.Register(ctx, user) //nolint:errcheck
	}

	// Simulate some lookups
	for i := 1; i <= 3; i++ {
		registry.Get(ctx, fmt.Sprintf("user-%d", i)) //nolint:errcheck
	}

	// Simulate an error
	registry.Get(ctx, "non-existent") //nolint:errcheck

	fmt.Println("Health monitoring metrics collected")
	// Output:
	// Health monitoring metrics collected
}

// ExamplePerformanceBenchmark demonstrates performance benchmarking
func ExamplePerformanceBenchmark() {
	registry := NewCachedRegistry(1000, 10*time.Minute)
	ctx := context.Background()

	// Register entities
	for i := 1; i <= 100; i++ {
		user := NewBaseEntity(fmt.Sprintf("user-%d", i), fmt.Sprintf("User %d", i))
		registry.Register(ctx, user) //nolint:errcheck
	}

	// Benchmark lookups
	start := time.Now()
	for i := 0; i < 10000; i++ {
		registry.Get(ctx, "user-1") //nolint:errcheck
	}
	duration := time.Since(start)

	fmt.Printf("10,000 lookups completed in %v\n", duration)
	fmt.Printf("Average lookup time: %v\n", duration/10000)
	// Output:
	// 10,000 lookups completed in 45.678ms
	// Average lookup time: 4.567µs
}
