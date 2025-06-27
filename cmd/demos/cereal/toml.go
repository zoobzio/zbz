package cereal

import (
	"fmt"
	"strings"
	"time"

	"zbz/cereal"
)

// TomlDemo demonstrates cereal TOML scoping with server configuration
func TomlDemo() {
	fmt.Println("📦 ZBZ Framework cereal TOML Scoping Demo")
	fmt.Println(strings.Repeat("=", 50))
	
	// Step 1: Define server configuration structure
	fmt.Println("\n🖥️  Step 1: Server configuration with scoping")
	
	type TLSConfig struct {
		Enabled  bool   `toml:"enabled" scope:"public"`
		CertFile string `toml:"cert_file" scope:"private"`
		KeyFile  string `toml:"key_file" scope:"admin"`
		MinTLS   string `toml:"min_version" scope:"public"`
	}
	
	type HTTPConfig struct {
		Host         string        `toml:"host" scope:"public"`
		Port         int           `toml:"port" scope:"public"`
		ReadTimeout  time.Duration `toml:"read_timeout" scope:"private"`
		WriteTimeout time.Duration `toml:"write_timeout" scope:"private"`
		TLS          TLSConfig     `toml:"tls"`
	}
	
	type LogConfig struct {
		Level      string `toml:"level" scope:"private"`
		Format     string `toml:"format" scope:"public"`
		OutputFile string `toml:"output_file" scope:"private"`
		Rotation   bool   `toml:"rotation" scope:"private"`
	}
	
	type AuthConfig struct {
		Provider     string        `toml:"provider" scope:"public"`
		JWTSecret    string        `toml:"jwt_secret" scope:"admin"`
		TokenExpiry  time.Duration `toml:"token_expiry" scope:"private"`
		RefreshExpiry time.Duration `toml:"refresh_expiry" scope:"private"`
		OIDC         struct {
			ClientID     string `toml:"client_id" scope:"private"`
			ClientSecret string `toml:"client_secret" scope:"admin"`
			Issuer       string `toml:"issuer" scope:"public"`
		} `toml:"oidc"`
	}
	
	type ServerConfig struct {
		Name         string            `toml:"name" scope:"public"`
		Environment  string            `toml:"environment" scope:"public"`
		Version      string            `toml:"version" scope:"public"`
		HTTP         HTTPConfig        `toml:"http"`
		Logging      LogConfig         `toml:"logging"`
		Auth         AuthConfig        `toml:"auth"`
		Features     map[string]bool   `toml:"features" scope:"private"`
		Maintenance  bool              `toml:"maintenance_mode" scope:"private"`
		AdminSecret  string            `toml:"admin_secret" scope:"admin"`
	}
	
	config := ServerConfig{
		Name:        "zbz-api-server",
		Environment: "production",
		Version:     "1.2.3",
		HTTP: HTTPConfig{
			Host:         "0.0.0.0",
			Port:         8080,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			TLS: TLSConfig{
				Enabled:  true,
				CertFile: "/etc/ssl/certs/api.crt",
				KeyFile:  "/etc/ssl/private/api.key",
				MinTLS:   "1.2",
			},
		},
		Logging: LogConfig{
			Level:      "info",
			Format:     "json",
			OutputFile: "/var/log/zbz-api.log",
			Rotation:   true,
		},
		Auth: AuthConfig{
			Provider:      "oidc",
			JWTSecret:     "super-secret-jwt-signing-key-production",
			TokenExpiry:   1 * time.Hour,
			RefreshExpiry: 24 * time.Hour,
		},
		Features: map[string]bool{
			"rate_limiting":    true,
			"request_logging":  true,
			"health_checks":    true,
			"metrics":          true,
			"debug_endpoints":  false,
		},
		Maintenance:  false,
		AdminSecret:  "admin-override-secret-key-production",
	}
	
	// Set OIDC config
	config.Auth.OIDC.ClientID = "zbz-api-production"
	config.Auth.OIDC.ClientSecret = "oidc-client-secret-production"
	config.Auth.OIDC.Issuer = "https://auth.example.com"
	
	// Step 2: Public scope (service discovery, health checks)
	fmt.Println("\n🌐 Step 2: Public scope - for service discovery")
	publicTOML, err := cereal.TOML.Marshal(config, "public")
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		return
	}
	fmt.Printf("Public TOML:\n%s\n", string(publicTOML))
	
	// Step 3: Private scope (operations team, monitoring)
	fmt.Println("\n🔒 Step 3: Private scope - for operations and monitoring")
	privateTOML, err := cereal.TOML.Marshal(config, "private")
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		return
	}
	fmt.Printf("Private TOML:\n%s\n", string(privateTOML))
	
	// Step 4: Admin scope (complete server configuration)
	fmt.Println("\n👑 Step 4: Admin scope - complete server config")
	adminTOML, err := cereal.TOML.Marshal(config, "admin")
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		return
	}
	fmt.Printf("Admin TOML:\n%s\n", string(adminTOML))
	
	// Step 5: Multi-server deployment
	fmt.Println("\n🏢 Step 5: Multi-server deployment configuration")
	
	servers := map[string]ServerConfig{
		"web-01": {
			Name: "zbz-web-01", Environment: "production",
			HTTP: HTTPConfig{Host: "10.0.1.10", Port: 8080},
		},
		"web-02": {
			Name: "zbz-web-02", Environment: "production", 
			HTTP: HTTPConfig{Host: "10.0.1.11", Port: 8080},
		},
		"api-01": {
			Name: "zbz-api-01", Environment: "production",
			HTTP: HTTPConfig{Host: "10.0.2.10", Port: 9090},
		},
	}
	
	for name, srv := range servers {
		fmt.Printf("\n   %s (public info):\n", name)
		srvTOML, err := cereal.TOML.Marshal(srv, "public")
		if err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			continue
		}
		fmt.Printf("%s", string(srvTOML))
	}
	
	// Step 6: Configuration loading and validation
	fmt.Println("\n✅ Step 6: Configuration loading with scope validation")
	
	// Try to load admin secrets with limited scope
	var opsConfig ServerConfig
	err = cereal.TOML.Unmarshal(adminTOML, &opsConfig, "private")
	if err != nil {
		fmt.Printf("✅ Correctly blocked admin secrets from ops scope: %v\n", err)
	}
	
	// Load with full admin access
	var fullConfig ServerConfig
	err = cereal.TOML.Unmarshal(adminTOML, &fullConfig, "admin")
	if err != nil {
		fmt.Printf("❌ Error loading full config: %v\n", err)
	} else {
		fmt.Printf("✅ Successfully loaded complete server configuration\n")
		fmt.Printf("   Server: %s (%s)\n", fullConfig.Name, fullConfig.Environment)
		fmt.Printf("   HTTP: %s:%d (TLS: %v)\n", fullConfig.HTTP.Host, fullConfig.HTTP.Port, fullConfig.HTTP.TLS.Enabled)
		fmt.Printf("   Auth: %s provider\n", fullConfig.Auth.Provider)
	}
	
	fmt.Println("\n✅ TOML scoping demo complete!")
	fmt.Println("🖥️  Server configuration secured by access level")
}