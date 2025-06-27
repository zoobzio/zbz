package cereal

import (
	"fmt"
	"strings"

	"zbz/cereal"
)

// YamlDemo demonstrates cereal YAML scoping with application config
func YamlDemo() {
	fmt.Println("📦 ZBZ Framework cereal YAML Scoping Demo")
	fmt.Println(strings.Repeat("=", 50))
	
	// Step 1: Define application configuration structure
	fmt.Println("\n⚙️  Step 1: Application configuration with scoping")
	
	type DatabaseConfig struct {
		Host     string `yaml:"host" scope:"public"`
		Port     int    `yaml:"port" scope:"public"`
		Name     string `yaml:"database" scope:"public"`
		Username string `yaml:"username" scope:"private"`
		Password string `yaml:"password" scope:"admin"`
		SSL      bool   `yaml:"ssl" scope:"public"`
	}
	
	type RedisConfig struct {
		Host     string `yaml:"host" scope:"public"`
		Port     int    `yaml:"port" scope:"public"`
		Password string `yaml:"password" scope:"admin"`
		DB       int    `yaml:"db" scope:"private"`
	}
	
	type AppConfig struct {
		Name        string          `yaml:"name" scope:"public"`
		Version     string          `yaml:"version" scope:"public"`
		Environment string          `yaml:"environment" scope:"public"`
		Debug       bool            `yaml:"debug" scope:"private"`
		SecretKey   string          `yaml:"secret_key" scope:"admin"`
		Database    DatabaseConfig  `yaml:"database"`
		Redis       RedisConfig     `yaml:"redis"`
		Features    map[string]bool `yaml:"features" scope:"private"`
	}
	
	config := AppConfig{
		Name:        "zbz-api-server",
		Version:     "v1.2.3",
		Environment: "production",
		Debug:       false,
		SecretKey:   "super-secret-jwt-signing-key-12345",
		Database: DatabaseConfig{
			Host:     "db.example.com",
			Port:     5432,
			Name:     "zbz_production",
			Username: "zbz_user",
			Password: "database-password-secret",
			SSL:      true,
		},
		Redis: RedisConfig{
			Host:     "redis.example.com",
			Port:     6379,
			Password: "redis-auth-secret",
			DB:       0,
		},
		Features: map[string]bool{
			"auth_enabled":     true,
			"rate_limiting":    true,
			"debug_endpoints":  false,
			"analytics":        true,
		},
	}
	
	// Step 2: Public scope (status endpoints, health checks)
	fmt.Println("\n🌐 Step 2: Public scope - safe for status pages")
	publicYAML, err := cereal.YAML.Marshal(config, "public")
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		return
	}
	fmt.Printf("Public YAML:\n%s\n", string(publicYAML))
	
	// Step 3: Private scope (developers, operations team)
	fmt.Println("\n🔒 Step 3: Private scope - for developers and ops")
	privateYAML, err := cereal.YAML.Marshal(config, "private")
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		return
	}
	fmt.Printf("Private YAML:\n%s\n", string(privateYAML))
	
	// Step 4: Admin scope (full access including secrets)
	fmt.Println("\n👑 Step 4: Admin scope - complete configuration")
	adminYAML, err := cereal.YAML.Marshal(config, "admin")
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		return
	}
	fmt.Printf("Admin YAML:\n%s\n", string(adminYAML))
	
	// Step 5: Environment-specific configs
	fmt.Println("\n🌍 Step 5: Multiple environment configurations")
	
	environments := map[string]AppConfig{
		"development": {
			Name: "zbz-api-dev", Environment: "development", Debug: true,
			Database: DatabaseConfig{Host: "localhost", Port: 5432, Name: "zbz_dev"},
		},
		"staging": {
			Name: "zbz-api-staging", Environment: "staging", Debug: false,
			Database: DatabaseConfig{Host: "staging-db.internal", Port: 5432, Name: "zbz_staging"},
		},
		"production": config,
	}
	
	for env, cfg := range environments {
		fmt.Printf("\n   %s environment (public view):\n", env)
		envYAML, err := cereal.YAML.Marshal(cfg, "public")
		if err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			continue
		}
		fmt.Printf("%s", string(envYAML))
	}
	
	// Step 6: Configuration validation
	fmt.Println("\n✅ Step 6: Configuration deserialization and validation")
	
	// Try to load admin config with limited scope
	var limitedConfig AppConfig
	err = cereal.YAML.Unmarshal(adminYAML, &limitedConfig, "private")
	if err != nil {
		fmt.Printf("✅ Correctly blocked admin secrets from private scope: %v\n", err)
	}
	
	// Load with correct scope
	var fullConfig AppConfig
	err = cereal.YAML.Unmarshal(adminYAML, &fullConfig, "admin")
	if err != nil {
		fmt.Printf("❌ Error loading admin config: %v\n", err)
	} else {
		fmt.Printf("✅ Successfully loaded full configuration\n")
		fmt.Printf("   App: %s v%s\n", fullConfig.Name, fullConfig.Version)
		fmt.Printf("   Database: %s@%s:%d\n", fullConfig.Database.Username, fullConfig.Database.Host, fullConfig.Database.Port)
	}
	
	fmt.Println("\n✅ YAML scoping demo complete!")
	fmt.Println("⚙️  Configuration safely scoped by access level")
}