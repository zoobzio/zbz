package main

import (
	"fmt"
	"log"
	"zbz/cereal"
)

// ServerConfig represents server configuration with permission-scoped fields
type ServerConfig struct {
	Name            string   `toml:"name"`                                      // No scope = always visible
	Version         string   `toml:"version" scope:"public"`                   // Public info
	Port            int      `toml:"port" scope:"read"`                        // Read permission
	Host            string   `toml:"host" scope:"read"`                        // Read permission
	DatabaseURL     string   `toml:"database_url" scope:"admin"`               // Admin only
	JWTSecret       string   `toml:"jwt_secret" scope:"admin+security"`        // Admin AND security
	APIKeys         []string `toml:"api_keys" scope:"admin+security"`          // Admin AND security
	RedisURL        string   `toml:"redis_url" scope:"admin"`                  // Admin only
	LogLevel        string   `toml:"log_level" scope:"developer"`              // Developer only
	DebugMode       bool     `toml:"debug_mode" scope:"developer"`             // Developer only
	MetricsEndpoint string   `toml:"metrics_endpoint" scope:"monitoring,admin+security"` // monitoring OR (admin+security)
}

func toml() {
	// Create a server config with all fields populated
	config := ServerConfig{
		Name:            "MyAPI",
		Version:         "2.1.0",
		Port:            8080,
		Host:            "0.0.0.0",
		DatabaseURL:     "postgres://user:pass@db:5432/myapi",
		JWTSecret:       "super-secret-jwt-key",
		APIKeys:         []string{"key-abc123", "key-def456"},
		RedisURL:        "redis://cache:6379",
		LogLevel:        "debug",
		DebugMode:       true,
		MetricsEndpoint: "http://metrics.internal:9090/push",
	}

	fmt.Println("=== TOML Serialization with Scoping ===\n")

	// Example 1: No permissions - only unscoped fields
	fmt.Println("1. No permissions (public documentation):")
	data, _ := cereal.TOML.Marshal(config)
	fmt.Printf("%s\n", string(data))

	// Example 2: Public permission
	fmt.Println("2. Public + read permissions (status page):")
	data, _ = cereal.TOML.Marshal(config, "public", "read")
	fmt.Printf("%s\n", string(data))

	// Example 3: Admin access without security
	fmt.Println("3. Admin permission (but no security clearance):")
	data, _ = cereal.TOML.Marshal(config, "public", "read", "admin")
	fmt.Printf("%s\n", string(data))

	// Example 4: Admin with security
	fmt.Println("4. Admin + Security permissions (full secrets):")
	data, _ = cereal.TOML.Marshal(config, "public", "read", "admin", "security")
	fmt.Printf("%s\n", string(data))

	// Example 5: Developer access
	fmt.Println("5. Developer access (debug info):")
	data, _ = cereal.TOML.Marshal(config, "public", "read", "developer")
	fmt.Printf("%s\n", string(data))

	// Example 6: Monitoring access
	fmt.Println("6. Monitoring service access:")
	data, _ = cereal.TOML.Marshal(config, "public", "monitoring")
	fmt.Printf("%s\n", string(data))

	fmt.Println("=== TOML Deserialization with Scoping ===\n")

	// Example 7: Configuration injection attack
	maliciousTOML := `name = "HackedAPI"
version = "99.9.9-malicious"
port = 9999
host = "0.0.0.0"
database_url = "postgres://hacker:stolen@evil.com:5432/stolen_db"
jwt_secret = "compromised-key"
api_keys = ["hacker-key-1", "hacker-key-2"]
redis_url = "redis://evil.cache:6379"
log_level = "trace"
debug_mode = true
metrics_endpoint = "http://evil.com/exfiltrate"`

	var newConfig ServerConfig
	
	fmt.Println("7. Config injection with 'read' permission only:")
	fmt.Printf("Input TOML:\n%s\n", maliciousTOML)
	
	err := cereal.TOML.Unmarshal([]byte(maliciousTOML), &newConfig, "public", "read")
	if err != nil {
		log.Fatal(err)
	}
	
	fmt.Printf("\nResult after scoping:\n")
	fmt.Printf("  Name: %s\n", newConfig.Name)
	fmt.Printf("  Version: %s\n", newConfig.Version)
	fmt.Printf("  Port: %d\n", newConfig.Port)
	fmt.Printf("  Host: %s\n", newConfig.Host)
	fmt.Printf("  DatabaseURL: %s (should be empty)\n", newConfig.DatabaseURL)
	fmt.Printf("  JWTSecret: %s (should be empty)\n", newConfig.JWTSecret)
	fmt.Printf("  APIKeys: %v (should be empty)\n", newConfig.APIKeys)
	fmt.Printf("  LogLevel: %s (should be empty)\n", newConfig.LogLevel)
	fmt.Printf("  DebugMode: %t (should be false)\n", newConfig.DebugMode)
	fmt.Println("\nNote: Sensitive config fields were automatically filtered out")

	// Example 8: Configuration deployment scenarios
	fmt.Println("\n8. Configuration deployment scenarios:")
	
	// Production deployment (admin gets secrets)
	prodConfig, _ := cereal.TOML.Marshal(config, "public", "read", "admin", "security")
	fmt.Println("Production deployment config (admin+security):")
	fmt.Printf("%s\n", string(prodConfig))
	
	// Developer environment (no production secrets)
	devConfig, _ := cereal.TOML.Marshal(config, "public", "read", "developer")
	fmt.Println("Developer environment config:")
	fmt.Printf("%s\n", string(devConfig))
	
	// Monitoring setup (only needs metrics endpoint)
	monitorConfig, _ := cereal.TOML.Marshal(config, "public", "monitoring")
	fmt.Println("Monitoring service config:")
	fmt.Printf("%s", string(monitorConfig))
}