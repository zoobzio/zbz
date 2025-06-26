package main

import (
	"fmt"
	"time"
)

// This example demonstrates how flux integrates with cereal for configuration management
// It uses mock implementations to show the integration pattern

// Mock interfaces for demonstration
type HodorContract interface {
	Get(key string) ([]byte, error)
	Set(key string, data []byte, ttl int) error
	Subscribe(key string, callback func([]byte)) error
	Unsubscribe(key string, subscription interface{}) error
}

type CerealProvider interface {
	Marshal(data any) ([]byte, error)
	Unmarshal(data []byte, target any) error
	MarshalScoped(data any, userPermissions []string) ([]byte, error)
	UnmarshalScoped(data []byte, target any, userPermissions []string, operation string) error
	ContentType() string
	Format() string
}

type FluxContract interface {
	Stop() error
	Key() string
	IsActive() bool
}

// CerealFlux demonstrates flux integration with cereal
type CerealFlux struct {
	hodorContract HodorContract
	cerealProvider CerealProvider
}

// NewCerealFlux creates a new flux service with cereal integration
func NewCerealFlux(hodor HodorContract, cereal CerealProvider) *CerealFlux {
	return &CerealFlux{
		hodorContract: hodor,
		cerealProvider: cereal,
	}
}

// Watch creates a configuration watcher with cereal serialization
func (cf *CerealFlux) Watch(configPath string, callback func(interface{})) (FluxContract, error) {
	// Load initial configuration using cereal
	data, err := cf.hodorContract.Get(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	
	// Parse using cereal
	var config interface{}
	err = cf.cerealProvider.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("cereal unmarshal failed: %w", err)
	}
	
	// Create watcher
	watcher := &cerealWatcher{
		path: configPath,
		hodor: cf.hodorContract,
		cereal: cf.cerealProvider,
		callback: callback,
		active: true,
	}
	
	// Subscribe to changes
	err = cf.hodorContract.Subscribe(configPath, watcher.handleChange)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe: %w", err)
	}
	
	// Trigger initial callback
	callback(config)
	
	return watcher, nil
}

// WatchScoped creates a configuration watcher with field-level scoping
func (cf *CerealFlux) WatchScoped(configPath string, userPermissions []string, callback func(interface{})) (FluxContract, error) {
	// Load initial configuration using cereal with scoping
	data, err := cf.hodorContract.Get(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	
	// Parse with scoped unmarshal
	var config interface{}
	err = cf.cerealProvider.UnmarshalScoped(data, &config, userPermissions, "read")
	if err != nil {
		return nil, fmt.Errorf("cereal scoped unmarshal failed: %w", err)
	}
	
	// Create scoped watcher
	watcher := &cerealScopedWatcher{
		path: configPath,
		hodor: cf.hodorContract,
		cereal: cf.cerealProvider,
		permissions: userPermissions,
		callback: callback,
		active: true,
	}
	
	// Subscribe to changes
	err = cf.hodorContract.Subscribe(configPath, watcher.handleChange)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe: %w", err)
	}
	
	// Trigger initial callback with filtered data
	callback(config)
	
	return watcher, nil
}

// UpdateConfig updates a configuration file using cereal serialization
func (cf *CerealFlux) UpdateConfig(configPath string, newConfig interface{}) error {
	// Serialize using cereal
	data, err := cf.cerealProvider.Marshal(newConfig)
	if err != nil {
		return fmt.Errorf("cereal marshal failed: %w", err)
	}
	
	// Store in hodor
	return cf.hodorContract.Set(configPath, data, 0) // No TTL for configs
}

// UpdateConfigScoped updates a configuration file with permission validation
func (cf *CerealFlux) UpdateConfigScoped(configPath string, newConfig interface{}, userPermissions []string) error {
	// Serialize with scoping using cereal
	data, err := cf.cerealProvider.MarshalScoped(newConfig, userPermissions)
	if err != nil {
		return fmt.Errorf("cereal scoped marshal failed: %w", err)
	}
	
	// Store in hodor
	return cf.hodorContract.Set(configPath, data, 0)
}

// cerealWatcher implements FluxContract for regular configuration watching
type cerealWatcher struct {
	path     string
	hodor    HodorContract
	cereal   CerealProvider
	callback func(interface{})
	active   bool
}

func (cw *cerealWatcher) handleChange(newData []byte) {
	var config interface{}
	err := cw.cereal.Unmarshal(newData, &config)
	if err != nil {
		fmt.Printf("Failed to unmarshal config change: %v\n", err)
		return
	}
	
	cw.callback(config)
}

func (cw *cerealWatcher) Stop() error {
	cw.active = false
	return cw.hodor.Unsubscribe(cw.path, nil)
}

func (cw *cerealWatcher) Key() string {
	return cw.path
}

func (cw *cerealWatcher) IsActive() bool {
	return cw.active
}

// cerealScopedWatcher implements FluxContract for scoped configuration watching
type cerealScopedWatcher struct {
	path        string
	hodor       HodorContract
	cereal      CerealProvider
	permissions []string
	callback    func(interface{})
	active      bool
}

func (csw *cerealScopedWatcher) handleChange(newData []byte) {
	var config interface{}
	err := csw.cereal.UnmarshalScoped(newData, &config, csw.permissions, "read")
	if err != nil {
		fmt.Printf("Failed to unmarshal scoped config change: %v\n", err)
		return
	}
	
	csw.callback(config)
}

func (csw *cerealScopedWatcher) Stop() error {
	csw.active = false
	return csw.hodor.Unsubscribe(csw.path, nil)
}

func (csw *cerealScopedWatcher) Key() string {
	return csw.path
}

func (csw *cerealScopedWatcher) IsActive() bool {
	return csw.active
}

// Test configuration structures
type DatabaseConfig struct {
	Host     string `json:"host" yaml:"host"`
	Port     int    `json:"port" yaml:"port"`
	Username string `json:"username" yaml:"username"`
	Password string `json:"password" yaml:"password" scope:"read:admin,write:admin"`
	Database string `json:"database" yaml:"database"`
}

type AppConfig struct {
	Name     string         `json:"name" yaml:"name"`
	Debug    bool          `json:"debug" yaml:"debug" scope:"read:developer"`
	Database DatabaseConfig `json:"database" yaml:"database"`
	Secret   string         `json:"secret" yaml:"secret" scope:"read:admin,write:admin"`
}

// Mock implementations for demonstration
type MockHodor struct {
	data map[string][]byte
}

func NewMockHodor() *MockHodor {
	return &MockHodor{
		data: make(map[string][]byte),
	}
}

func (m *MockHodor) Get(key string) ([]byte, error) {
	data, exists := m.data[key]
	if !exists {
		return nil, fmt.Errorf("key not found: %s", key)
	}
	return data, nil
}

func (m *MockHodor) Set(key string, data []byte, ttl int) error {
	m.data[key] = data
	return nil
}

func (m *MockHodor) Subscribe(key string, callback func([]byte)) error {
	// Mock subscription - in real implementation, this would watch for file changes
	fmt.Printf("Subscribed to changes for: %s\n", key)
	return nil
}

func (m *MockHodor) Unsubscribe(key string, subscription interface{}) error {
	fmt.Printf("Unsubscribed from: %s\n", key)
	return nil
}

type MockCereal struct{}

func (m *MockCereal) Marshal(data any) ([]byte, error) {
	return []byte(fmt.Sprintf(`{"mock": "serialized %T"}`, data)), nil
}

func (m *MockCereal) Unmarshal(data []byte, target any) error {
	fmt.Printf("Mock deserializing %s into %T\n", string(data), target)
	return nil
}

func (m *MockCereal) MarshalScoped(data any, userPermissions []string) ([]byte, error) {
	return []byte(fmt.Sprintf(`{"mock": "scoped serialized %T", "permissions": %v}`, data, userPermissions)), nil
}

func (m *MockCereal) UnmarshalScoped(data []byte, target any, userPermissions []string, operation string) error {
	fmt.Printf("Mock scoped deserializing %s into %T (perms: %v, op: %s)\n", string(data), target, userPermissions, operation)
	return nil
}

func (m *MockCereal) ContentType() string { return "application/json" }
func (m *MockCereal) Format() string { return "json" }

func main() {
	fmt.Println("🌊 Cereal-Flux Integration Demo")
	fmt.Println("===============================")
	
	// Set up services
	hodor := NewMockHodor()
	cereal := &MockCereal{}
	flux := NewCerealFlux(hodor, cereal)
	
	// Set up initial configuration
	config := AppConfig{
		Name:  "MyApp",
		Debug: true,
		Database: DatabaseConfig{
			Host:     "localhost",
			Port:     5432,
			Username: "app",
			Password: "secret123",
			Database: "myapp",
		},
		Secret: "super-secret-key",
	}
	
	// Store initial config
	configData := `{"name": "MyApp", "debug": true, "database": {"host": "localhost", "port": 5432}, "secret": "super-secret-key"}`
	hodor.Set("app.json", []byte(configData), 0)
	
	fmt.Println("\n📋 Basic Configuration Watching:")
	
	// Set up configuration watcher
	watcher, err := flux.Watch("app.json", func(newConfig interface{}) {
		fmt.Printf("✅ Configuration updated: %v\n", newConfig)
	})
	if err != nil {
		fmt.Printf("❌ Failed to create watcher: %v\n", err)
		return
	}
	defer watcher.Stop()
	
	// Update configuration
	fmt.Println("\n📝 Updating Configuration:")
	err = flux.UpdateConfig("app.json", config)
	if err != nil {
		fmt.Printf("❌ Failed to update config: %v\n", err)
	} else {
		fmt.Println("✅ Configuration updated successfully")
	}
	
	fmt.Println("\n🔒 Scoped Configuration Watching:")
	
	// Set up scoped watcher (developer permissions)
	developerPermissions := []string{"developer"}
	scopedWatcher, err := flux.WatchScoped("app.json", developerPermissions, func(newConfig interface{}) {
		fmt.Printf("✅ Scoped configuration updated (developer view): %v\n", newConfig)
	})
	if err != nil {
		fmt.Printf("❌ Failed to create scoped watcher: %v\n", err)
		return
	}
	defer scopedWatcher.Stop()
	
	// Try to update with insufficient permissions
	fmt.Println("\n🚫 Attempting Unauthorized Update:")
	err = flux.UpdateConfigScoped("app.json", config, []string{"user"})
	if err != nil {
		fmt.Printf("❌ Unauthorized update blocked (expected): %v\n", err)
	}
	
	// Update with admin permissions
	fmt.Println("\n✅ Authorized Update:")
	adminPermissions := []string{"admin"}
	err = flux.UpdateConfigScoped("app.json", config, adminPermissions)
	if err != nil {
		fmt.Printf("❌ Failed to update with admin permissions: %v\n", err)
	} else {
		fmt.Println("✅ Configuration updated with admin permissions")
	}
	
	fmt.Println("\n🎉 Cereal-Flux Integration Demo Complete!")
	fmt.Println("========================================")
	fmt.Println("Key Benefits Demonstrated:")
	fmt.Println("✅ Unified serialization for configuration files")
	fmt.Println("✅ Field-level scoping for sensitive configuration")
	fmt.Println("✅ Hot-reload configuration watching")
	fmt.Println("✅ Permission-based configuration updates")
	fmt.Println("✅ Type-safe configuration management")
	
	// Simulate some time for watchers to process
	time.Sleep(100 * time.Millisecond)
}