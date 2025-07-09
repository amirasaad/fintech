package registry

import (
	"context"
	"fmt"
	"log"
	"time"
)

// Example 1: Basic Registry Usage
func ExampleBasicRegistry() {
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

	registry.Register(ctx, user1)
	registry.Register(ctx, user2)

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
}

// Example 2: Registry with Persistence
func ExamplePersistentRegistry() {
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
		registry.Register(ctx, user)
	}

	// The registry automatically persists to file
	fmt.Println("Users saved to persistent storage")

	// Create a new registry instance (simulating application restart)
	newRegistry, _ := NewPersistentRegistry("/tmp/users.json")

	// Users are automatically loaded from file
	entities, _ := newRegistry.List(ctx)
	fmt.Printf("Loaded %d users from persistent storage\n", len(entities))
}

// Example 3: Registry with Caching
func ExampleCachedRegistry() {
	// Create a registry with enhanced caching
	registry := NewCachedRegistry(100, 5*time.Minute)
	ctx := context.Background()

	// Register many entities
	for i := 1; i <= 50; i++ {
		user := NewBaseEntity(fmt.Sprintf("user-%d", i), fmt.Sprintf("User %d", i))
		registry.Register(ctx, user)
	}

	// Repeated lookups will be served from cache
	start := time.Now()
	for i := 0; i < 1000; i++ {
		registry.Get(ctx, "user-1") //nolint:errcheck
	}
	duration := time.Since(start)
	fmt.Printf("1000 lookups completed in %v\n", duration)
}

// Example 4: Registry with Metrics and Monitoring
func ExampleMonitoredRegistry() {
	// Create a monitored registry
	registry := NewMonitoredRegistry("user-registry")
	ctx := context.Background()

	// Perform operations
	for i := 1; i <= 10; i++ {
		user := NewBaseEntity(fmt.Sprintf("user-%d", i), fmt.Sprintf("User %d", i))
		registry.Register(ctx, user)
	}

	// Simulate some lookups and errors
	for i := 1; i <= 5; i++ {
		registry.Get(ctx, fmt.Sprintf("user-%d", i))
	}

	// Try to get non-existent user (will increment error count)
	registry.Get(ctx, "non-existent")

	// The registry automatically tracks metrics
	fmt.Println("Registry operations completed with metrics tracking")
}

// Example 5: Registry Builder Pattern
func ExampleRegistryBuilder() {
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
}

// Example 6: Custom Entity Implementation
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
		registry.Register(ctx, product)
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
}

// Example 7: Event-Driven Registry
func ExampleEventDrivenRegistry() {
	// Create registry with events
	registry := NewBasicRegistry()
	ctx := context.Background()

	// Create event observer
	observer := &UserActivityLogger{}

	// Subscribe to events (in a real implementation, you'd get the event bus from the registry)
	fmt.Println("Event-driven registry example:")
	fmt.Println("(In a real implementation, events would be automatically triggered)")

	// Register users (would trigger events)
	users := []Entity{
		NewBaseEntity("user-1", "Alice"),
		NewBaseEntity("user-2", "Bob"),
	}

	for _, user := range users {
		registry.Register(ctx, user)
		// Simulate event notification
		observer.OnEntityRegistered(ctx, user)
	}

	// Unregister a user (would trigger events)
	registry.Unregister(ctx, "user-1") //nolint:errcheck
	observer.OnEntityUnregistered(ctx, "user-1")
}

// UserActivityLogger implements RegistryObserver
type UserActivityLogger struct{}

func (l *UserActivityLogger) OnEntityRegistered(ctx context.Context, entity Entity) {
	fmt.Printf("USER REGISTERED: %s (%s)\n", entity.Name(), entity.ID())
}

func (l *UserActivityLogger) OnEntityUnregistered(ctx context.Context, id string) {
	fmt.Printf("USER UNREGISTERED: %s\n", id)
}

func (l *UserActivityLogger) OnEntityUpdated(ctx context.Context, entity Entity) {
	fmt.Printf("USER UPDATED: %s (%s)\n", entity.Name(), entity.ID())
}

func (l *UserActivityLogger) OnEntityActivated(ctx context.Context, id string) {
	fmt.Printf("USER ACTIVATED: %s\n", id)
}

func (l *UserActivityLogger) OnEntityDeactivated(ctx context.Context, id string) {
	fmt.Printf("USER DEACTIVATED: %s\n", id)
}

// Example 8: Registry with Custom Validation
func ExampleCustomValidation() {
	// Create validator with custom rules
	validator := NewSimpleValidator().
		WithRequiredMetadata([]string{"email", "age"}).
		WithForbiddenMetadata([]string{"password", "ssn"}).
		WithValidator("email", validateEmail).
		WithValidator("age", validateAge)

	// Create registry with custom validator
	config := RegistryConfig{
		Name:             "validated-registry",
		EnableValidation: true,
	}
	registry := NewEnhancedRegistry(config).WithValidator(validator)
	ctx := context.Background()

	// Test valid user
	validUser := NewBaseEntity("user-1", "John Doe")
	validUser.Metadata()["email"] = "john@example.com"
	validUser.Metadata()["age"] = "25"

	err := registry.Register(ctx, validUser)
	if err != nil {
		fmt.Printf("Valid user registration failed: %v\n", err)
	} else {
		fmt.Println("Valid user registered successfully")
	}

	// Test invalid user (missing required metadata)
	invalidUser := NewBaseEntity("user-2", "Jane Smith")
	invalidUser.Metadata()["email"] = "jane@example.com"
	// Missing age

	err = registry.Register(ctx, invalidUser)
	if err != nil {
		fmt.Printf("Invalid user correctly rejected: %v\n", err)
	}

	// Test user with forbidden metadata
	forbiddenUser := NewBaseEntity("user-3", "Bob Wilson")
	forbiddenUser.Metadata()["email"] = "bob@example.com"
	forbiddenUser.Metadata()["age"] = "30"
	forbiddenUser.Metadata()["password"] = "secret123" // Forbidden

	err = registry.Register(ctx, forbiddenUser)
	if err != nil {
		fmt.Printf("User with forbidden metadata correctly rejected: %v\n", err)
	}
}

// Validation functions
func validateEmail(email string) error {
	if len(email) == 0 {
		return fmt.Errorf("email cannot be empty")
	}
	if len(email) < 5 {
		return fmt.Errorf("email too short")
	}
	return nil
}

func validateAge(age string) error {
	if len(age) == 0 {
		return fmt.Errorf("age cannot be empty")
	}
	// In a real implementation, you'd parse and validate the age
	return nil
}

// Example 9: Registry Factory Patterns
func ExampleRegistryFactory() {
	ctx := context.Background()
	factory := NewRegistryFactory()

	// Production registry
	prodConfig := RegistryConfig{
		Name:              "production-users",
		MaxEntities:       10000,
		EnableEvents:      true,
		EnableValidation:  true,
		CacheSize:         1000,
		CacheTTL:          5 * time.Minute,
		EnablePersistence: true,
		PersistencePath:   "/data/users.json",
		AutoSaveInterval:  30 * time.Second,
	}
	prodRegistry, err := factory.Create(ctx, prodConfig)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Production registry created with persistence and monitoring")

	// Development registry
	devConfig := RegistryConfig{
		Name:             "development-users",
		MaxEntities:      1000,
		EnableEvents:     true,
		EnableValidation: true,
		CacheSize:        100,
		CacheTTL:         time.Minute,
	}
	devRegistry, err := factory.Create(ctx, devConfig)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Development registry created with metrics and events")

	// Test registry
	testConfig := RegistryConfig{
		Name:             "test-users",
		MaxEntities:      1000,
		EnableEvents:     false,
		EnableValidation: true,
		CacheSize:        100,
		CacheTTL:         time.Minute,
	}
	testRegistry, err := factory.Create(ctx, testConfig)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Test registry created with minimal features")

	// Use the registries
	user := NewBaseEntity("test-user", "Test User")
	prodRegistry.Register(ctx, user) //nolint:errcheck
	devRegistry.Register(ctx, user)  //nolint:errcheck
	testRegistry.Register(ctx, user) //nolint:errcheck

	fmt.Println("User registered in all three registry types")
}

// Example 10: Advanced Search and Filtering
func ExampleAdvancedSearch() {
	registry := NewBasicRegistry()
	ctx := context.Background()

	// Register users with various metadata
	users := []struct {
		id       string
		name     string
		email    string
		role     string
		location string
		active   bool
	}{
		{"user-1", "Alice Johnson", "alice@company.com", "admin", "NYC", true},
		{"user-2", "Bob Smith", "bob@company.com", "user", "LA", true},
		{"user-3", "Carol Davis", "carol@company.com", "manager", "NYC", false},
		{"user-4", "David Wilson", "david@company.com", "user", "Chicago", true},
		{"user-5", "Eve Brown", "eve@company.com", "admin", "LA", true},
	}

	for _, u := range users {
		user := NewBaseEntity(u.id, u.name)
		user.Metadata()["email"] = u.email
		user.Metadata()["role"] = u.role
		user.Metadata()["location"] = u.location
		if !u.active {
			user.Metadata()["active"] = "false"
		}
		registry.Register(ctx, user)
	}

	// Search by name
	aliceResults, _ := registry.Search(ctx, "Alice")
	fmt.Printf("Found %d users with 'Alice' in name\n", len(aliceResults))

	// Search by role
	admins, _ := registry.SearchByMetadata(ctx, map[string]string{"role": "admin"})
	fmt.Printf("Found %d admin users\n", len(admins))

	// Search by location
	nycUsers, _ := registry.SearchByMetadata(ctx, map[string]string{"location": "NYC"})
	fmt.Printf("Found %d users in NYC\n", len(nycUsers))

	// List active users
	activeUsers, _ := registry.ListActive(ctx)
	fmt.Printf("Found %d active users\n", len(activeUsers))

	// Count total users
	totalCount, _ := registry.Count(ctx)
	fmt.Printf("Total users: %d\n", totalCount)
}

// Example 11: Registry with Health Monitoring
func ExampleHealthMonitoring() {
	// Create registry with health monitoring
	config := RegistryConfig{
		Name:             "health-monitored-registry",
		EnableEvents:     true,
		EnableValidation: true,
	}

	registry := NewEnhancedRegistry(config)
	health := NewSimpleHealth()
	registry.WithHealth(health)

	ctx := context.Background()

	// Check health status
	if health.IsHealthy(ctx) {
		fmt.Println("Registry is healthy")
	}

	// Simulate an error
	health.SetError(fmt.Errorf("simulated error"))

	if !health.IsHealthy(ctx) {
		fmt.Println("Registry is unhealthy")
		status := health.GetHealthStatus(ctx)
		fmt.Printf("Health status: %+v\n", status)
	}

	// Clear error
	health.ClearError()
	if health.IsHealthy(ctx) {
		fmt.Println("Registry is healthy again")
	}
}

// Example 12: Registry Performance Benchmark
func ExamplePerformanceBenchmark() {
	// Create different registry configurations for comparison
	configs := []struct {
		name   string
		config RegistryConfig
	}{
		{
			name: "Basic Registry",
			config: RegistryConfig{
				Name:             "basic",
				EnableEvents:     false,
				EnableValidation: false,
			},
		},
		{
			name: "Cached Registry",
			config: RegistryConfig{
				Name:             "cached",
				EnableEvents:     false,
				EnableValidation: false,
				CacheSize:        1000,
				CacheTTL:         time.Minute,
			},
		},
		{
			name: "Full Featured Registry",
			config: RegistryConfig{
				Name:             "full",
				EnableEvents:     true,
				EnableValidation: true,
				CacheSize:        1000,
				CacheTTL:         time.Minute,
			},
		},
	}

	ctx := context.Background()

	for _, cfg := range configs {
		registry := NewEnhancedRegistry(cfg.config)

		// Add cache if configured
		if cfg.config.CacheSize > 0 {
			registry.WithCache(NewMemoryCache(cfg.config.CacheTTL))
		}

		// Benchmark registration
		start := time.Now()
		for i := 1; i <= 1000; i++ {
			user := NewBaseEntity(fmt.Sprintf("user-%d", i), fmt.Sprintf("User %d", i))
			registry.Register(ctx, user)
		}
		registerTime := time.Since(start)

		// Benchmark lookups
		start = time.Now()
		for i := 1; i <= 1000; i++ {
			registry.Get(ctx, fmt.Sprintf("user-%d", i))
		}
		lookupTime := time.Since(start)

		fmt.Printf("%s:\n", cfg.name)
		fmt.Printf("  1000 registrations: %v\n", registerTime)
		fmt.Printf("  1000 lookups: %v\n", lookupTime)
		fmt.Printf("  Average registration: %v\n", registerTime/1000)
		fmt.Printf("  Average lookup: %v\n", lookupTime/1000)
		fmt.Println()
	}
}
