package main

import (
	"fmt"
	"log"
	"zbz/cereal"
)

// Config represents application configuration with permission-scoped fields
type Config struct {
	AppName      string   `yaml:"app_name"`                                        // No scope = always visible
	Version      string   `yaml:"version" scope:"public"`                          // Public info
	Environment  string   `yaml:"environment" scope:"read"`                        // Read permission
	DatabaseURL  string   `yaml:"database_url" scope:"admin"`                      // Admin only
	APIKeys      []string `yaml:"api_keys" scope:"admin+security"`                 // Admin AND security
	SecretKey    string   `yaml:"secret_key" scope:"admin+security"`               // Admin AND security
	DebugMode    bool     `yaml:"debug_mode" scope:"developer"`                    // Developer only
	AuditWebhook string   `yaml:"audit_webhook" scope:"compliance,admin+security"` // compliance OR (admin+security)
}

func yaml() {
	// Create a config with all fields populated
	config := Config{
		AppName:      "MyApp",
		Version:      "1.0.0",
		Environment:  "production",
		DatabaseURL:  "postgres://user:pass@localhost/myapp",
		APIKeys:      []string{"key-123", "key-456"},
		SecretKey:    "super-secret-key",
		DebugMode:    false,
		AuditWebhook: "https://audit.example.com/webhook",
	}

	fmt.Println("=== YAML Serialization with Scoping ===\n")

	// Example 1: No permissions - only unscoped fields
	fmt.Println("1. No permissions (public API):")
	data, _ := cereal.YAML.Marshal(config)
	fmt.Printf("%s\n", string(data))

	// Example 2: Public permission
	fmt.Println("2. Public permission:")
	data, _ = cereal.YAML.Marshal(config, "public")
	fmt.Printf("%s\n", string(data))

	// Example 3: Read permission
	fmt.Println("3. Read permission (monitoring service):")
	data, _ = cereal.YAML.Marshal(config, "public", "read")
	fmt.Printf("%s\n", string(data))

	// Example 4: Admin without security
	fmt.Println("4. Admin permission (but no security clearance):")
	data, _ = cereal.YAML.Marshal(config, "public", "read", "admin")
	fmt.Printf("%s\n", string(data))

	// Example 5: Admin with security
	fmt.Println("5. Admin + Security permissions:")
	data, _ = cereal.YAML.Marshal(config, "public", "read", "admin", "security")
	fmt.Printf("%s\n", string(data))

	// Example 6: Developer access
	fmt.Println("6. Developer access:")
	data, _ = cereal.YAML.Marshal(config, "public", "read", "developer")
	fmt.Printf("%s\n", string(data))

	// Example 7: Compliance access
	fmt.Println("7. Compliance auditor access:")
	data, _ = cereal.YAML.Marshal(config, "public", "compliance")
	fmt.Printf("%s\n", string(data))

	fmt.Println("=== YAML Deserialization with Scoping ===\n")

	// Example 8: Unmarshaling config with limited permissions
	yamlConfig := `app_name: HackedApp
version: "2.0.0-evil"
environment: "hacked"
database_url: "postgres://hacker:stolen@evil.com/db"
api_keys:
  - "stolen-key-1"
  - "stolen-key-2"
secret_key: "compromised-secret"
debug_mode: true
audit_webhook: "https://evil.com/steal-data"`

	var newConfig Config

	fmt.Println("8. Unmarshaling potentially malicious YAML with 'read' permission:")
	fmt.Printf("Input:\n%s\n", yamlConfig)

	err := cereal.YAML.Unmarshal([]byte(yamlConfig), &newConfig, "public", "read")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("\nResult:\n")
	fmt.Printf("  AppName: %s\n", newConfig.AppName)
	fmt.Printf("  Version: %s\n", newConfig.Version)
	fmt.Printf("  Environment: %s\n", newConfig.Environment)
	fmt.Printf("  DatabaseURL: %s (should be empty)\n", newConfig.DatabaseURL)
	fmt.Printf("  APIKeys: %v (should be empty)\n", newConfig.APIKeys)
	fmt.Printf("  SecretKey: %s (should be empty)\n", newConfig.SecretKey)
	fmt.Println("\nNote: Sensitive fields were automatically filtered out")

	// Example 9: Configuration migration example
	fmt.Println("\n9. Configuration migration (different permission levels):")

	// Export for public documentation
	publicYAML, _ := cereal.YAML.Marshal(config, "public")
	fmt.Println("Public documentation version:")
	fmt.Printf("%s\n", string(publicYAML))

	// Export for developers (includes debug settings)
	devYAML, _ := cereal.YAML.Marshal(config, "public", "read", "developer")
	fmt.Println("Developer version:")
	fmt.Printf("%s\n", string(devYAML))

	// Full admin export (for secure backup)
	adminYAML, _ := cereal.YAML.Marshal(config, "public", "read", "admin", "security")
	fmt.Println("Admin backup version:")
	fmt.Printf("%s", string(adminYAML))
}
