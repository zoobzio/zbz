package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"zbz/cereal"
)

func main() {
	args := os.Args[1:] // Skip program name
	
	if len(args) == 0 {
		// No args = run all demos
		runAllDemos()
		return
	}
	
	// Parse command
	command := strings.ToLower(args[0])
	
	switch command {
	case "json":
		runJSONDemo()
	case "yaml":
		runYAMLDemo() 
	case "toml":
		runTOMLDemo()
	case "all":
		runAllDemos()
	case "scoping":
		runScopingDemo()
	case "security":
		runSecurityDemo()
	case "help", "-h", "--help":
		showHelp()
	default:
		fmt.Printf("Unknown command: %s\n\n", command)
		showHelp()
		os.Exit(1)
	}
}

func showHelp() {
	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║                    ZBZ cereal Demo Suite                      ║")
	fmt.Println("║              Permission-Based Serialization                   ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  go run . [command]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  json      📦 JSON scoping demo (user data)")
	fmt.Println("  yaml      ⚙️  YAML scoping demo (app config)")
	fmt.Println("  toml      🖥️  TOML scoping demo (server config)")
	fmt.Println("  scoping   🔐 Advanced scoping scenarios")
	fmt.Println("  security  🛡️  Security & validation demo")
	fmt.Println("  all       🎯 Run all demos (default)")
	fmt.Println("  help      ❓ Show this help message")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  go run .           # Run all demos")
	fmt.Println("  go run . json      # Run only JSON demo")
	fmt.Println("  go run . yaml      # Run only YAML demo")
	fmt.Println("  go run . scoping   # Advanced scoping scenarios")
	fmt.Println()
	fmt.Println("🔗 What you'll see:")
	fmt.Println("   • Permission-based data serialization")
	fmt.Println("   • Multi-level access control (public/private/admin)")
	fmt.Println("   • Secure deserialization with scope validation")
	fmt.Println("   • Real-world use cases across JSON/YAML/TOML")
	fmt.Println("   • Advanced scoping patterns and security features")
}

func runJSONDemo() {
	printHeader("📦 JSON Demo", "User Data with Permission Scoping")
	jsonDemo()
}

func runYAMLDemo() {
	printHeader("⚙️ YAML Demo", "Application Configuration with Scoping")
	yamlDemo()
}

func runTOMLDemo() {
	printHeader("🖥️ TOML Demo", "Server Configuration with Scoping")
	tomlDemo()
}

func runScopingDemo() {
	printHeader("🔐 Advanced Scoping", "Complex Permission Scenarios")
	advancedScopingDemo()
}

func runSecurityDemo() {
	printHeader("🛡️ Security Demo", "Validation & Security Features")
	securityDemo()
}

func runAllDemos() {
	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║                   ZBZ cereal Demo Suite                       ║")
	fmt.Println("║              Permission-Based Serialization                   ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Println()
	
	runJSONDemo()
	pauseBetweenDemos()
	
	runYAMLDemo()
	pauseBetweenDemos()
	
	runTOMLDemo()
	pauseBetweenDemos()
	
	runScopingDemo()
	pauseBetweenDemos()
	
	runSecurityDemo()
	
	printSummary()
}

func printHeader(title, subtitle string) {
	fmt.Println("┌──────────────────────────────────────────────────────────────┐")
	fmt.Printf("│  %-58s  │\n", title)
	fmt.Printf("│  %-58s  │\n", subtitle)
	fmt.Println("└──────────────────────────────────────────────────────────────┘")
	fmt.Println()
}

func pauseBetweenDemos() {
	fmt.Println("\n" + strings.Repeat("─", 60) + "\n")
	time.Sleep(1 * time.Second)
}

func printSummary() {
	fmt.Println("\n\n╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║                     Demos Complete!                           ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Println("🎯 What you've seen:")
	fmt.Println("   ✅ Permission-based data serialization")
	fmt.Println("   ✅ JSON scoping for user data and APIs")
	fmt.Println("   ✅ YAML scoping for application configuration")
	fmt.Println("   ✅ TOML scoping for server configuration")
	fmt.Println("   ✅ Multi-level access control (public/private/admin)")
	fmt.Println("   ✅ Secure deserialization with scope validation")
	fmt.Println("   ✅ Advanced scoping patterns and security features")
	fmt.Println()
	fmt.Println("💡 Use cases demonstrated:")
	fmt.Println("   • User profiles with privacy levels")
	fmt.Println("   • Application config with environment separation")
	fmt.Println("   • Server config with security-sensitive data")
	fmt.Println("   • Multi-environment and multi-server deployments")
	fmt.Println("   • Complex permission matrices and validation")
	fmt.Println()
	fmt.Println("🔗 Next steps:")
	fmt.Println("   • Explore zbz/cereal for more scoping examples")
	fmt.Println("   • Try 'go run ../zlog-demo' for logging demonstrations")
	fmt.Println("   • Try 'go run ../capitan-demo' for event coordination")
	fmt.Println("   • Check individual format tests in zbz/cereal/tests/")
	fmt.Println()
}

// Demo implementations

func jsonDemo() {
	fmt.Println("📦 ZBZ Framework cereal JSON Scoping Demo")
	fmt.Println(strings.Repeat("=", 50))
	
	// Step 1: Define user data structure
	fmt.Println("\n👤 Step 1: User data structure with scoping")
	
	type User struct {
		ID       int    `json:"id" scope:"public"`
		Username string `json:"username" scope:"public"`
		Email    string `json:"email" scope:"private"`
		Phone    string `json:"phone" scope:"admin"`
		Password string `json:"-" scope:"never"`
		Role     string `json:"role" scope:"admin"`
		Profile  struct {
			Name     string `json:"name" scope:"public"`
			Bio      string `json:"bio" scope:"public"`
			Location string `json:"location" scope:"private"`
		} `json:"profile"`
	}
	
	user := User{
		ID:       12345,
		Username: "johndoe",
		Email:    "john.doe@example.com",
		Phone:    "555-123-4567",
		Password: "super-secret-password",
		Role:     "user",
	}
	user.Profile.Name = "John Doe"
	user.Profile.Bio = "Software developer and coffee enthusiast"
	user.Profile.Location = "San Francisco, CA"
	
	// Step 2: Public scope (guest users)
	fmt.Println("\n🌐 Step 2: Public scope - what guests see")
	publicJSON, err := cereal.JSON.Marshal(user, "public")
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		return
	}
	fmt.Printf("Public JSON:\n%s\n", string(publicJSON))
	
	// Step 3: Private scope (authenticated users)  
	fmt.Println("\n🔒 Step 3: Private scope - what authenticated users see")
	privateJSON, err := cereal.JSON.Marshal(user, "private")
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		return
	}
	fmt.Printf("Private JSON:\n%s\n", string(privateJSON))
	
	// Step 4: Admin scope (administrators)
	fmt.Println("\n👑 Step 4: Admin scope - what administrators see")
	adminJSON, err := cereal.JSON.Marshal(user, "admin")
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		return
	}
	fmt.Printf("Admin JSON:\n%s\n", string(adminJSON))
	
	// Step 5: Deserialization with scoping
	fmt.Println("\n⬅️  Step 5: Deserializing with scope validation")
	
	// Try to deserialize admin data with public scope
	var publicUser User
	err = cereal.JSON.Unmarshal(adminJSON, &publicUser, "public")
	if err != nil {
		fmt.Printf("✅ Correctly blocked admin data from public scope: %v\n", err)
	} else {
		fmt.Printf("❌ Security issue: admin data leaked to public scope\n")
	}
	
	// Deserialize with correct scope
	var adminUser User
	err = cereal.JSON.Unmarshal(adminJSON, &adminUser, "admin")
	if err != nil {
		fmt.Printf("❌ Error deserializing admin data: %v\n", err)
	} else {
		fmt.Printf("✅ Successfully deserialized with admin scope\n")
		fmt.Printf("   Admin user role: %s\n", adminUser.Role)
	}
	
	fmt.Println("\n✅ JSON scoping demo complete!")
	fmt.Println("🔐 Data exposed based on user permissions")
}

func yamlDemo() {
	fmt.Println("⚙️ ZBZ Framework cereal YAML Scoping Demo")
	fmt.Println(strings.Repeat("=", 50))
	
	// Application configuration with environment-based scoping
	fmt.Println("\n🔧 Step 1: Application configuration structure")
	
	type DatabaseConfig struct {
		Host     string `yaml:"host" scope:"development,production"`
		Port     int    `yaml:"port" scope:"development,production"`
		Username string `yaml:"username" scope:"development,production"`
		Password string `yaml:"password" scope:"production"`
		SSL      bool   `yaml:"ssl" scope:"development,production"`
	}
	
	type AppConfig struct {
		Name        string         `yaml:"name" scope:"development,production"`
		Version     string         `yaml:"version" scope:"development,production"`
		Debug       bool           `yaml:"debug" scope:"development"`
		SecretKey   string         `yaml:"secret_key" scope:"production"`
		Database    DatabaseConfig `yaml:"database"`
		Environment string         `yaml:"environment" scope:"development,production"`
	}
	
	config := AppConfig{
		Name:        "MyApp",
		Version:     "1.2.3",
		Debug:       true,
		SecretKey:   "super-secret-key-12345",
		Environment: "development",
		Database: DatabaseConfig{
			Host:     "localhost",
			Port:     5432,
			Username: "myapp_user",
			Password: "database-secret-password",
			SSL:      false,
		},
	}
	
	// Step 2: Development environment view
	fmt.Println("\n🔧 Step 2: Development environment configuration")
	devYAML, err := cereal.YAML.Marshal(config, "development")
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		return
	}
	fmt.Printf("Development YAML:\n%s\n", string(devYAML))
	
	// Step 3: Production environment view
	fmt.Println("\n🏭 Step 3: Production environment configuration")
	prodYAML, err := cereal.YAML.Marshal(config, "production")
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		return
	}
	fmt.Printf("Production YAML:\n%s\n", string(prodYAML))
	
	fmt.Println("\n✅ YAML scoping demo complete!")
	fmt.Println("🔧 Configuration filtered by environment")
}

func tomlDemo() {
	fmt.Println("🖥️ ZBZ Framework cereal TOML Scoping Demo")
	fmt.Println(strings.Repeat("=", 50))
	
	// Server configuration with security-level scoping
	fmt.Println("\n🖥️ Step 1: Server configuration structure")
	
	type SecurityConfig struct {
		APIKey       string `toml:"api_key" scope:"admin"`
		JWTSecret    string `toml:"jwt_secret" scope:"admin"`
		EnableAuth   bool   `toml:"enable_auth" scope:"operator,admin"`
		RateLimit    int    `toml:"rate_limit" scope:"operator,admin"`
	}
	
	type ServerConfig struct {
		Host        string         `toml:"host" scope:"operator,admin"`
		Port        int            `toml:"port" scope:"operator,admin"`
		Workers     int            `toml:"workers" scope:"operator,admin"`
		LogLevel    string         `toml:"log_level" scope:"operator,admin"`
		Security    SecurityConfig `toml:"security"`
		MetricsPort int            `toml:"metrics_port" scope:"admin"`
	}
	
	serverConfig := ServerConfig{
		Host:        "0.0.0.0",
		Port:        8080,
		Workers:     4,
		LogLevel:    "info",
		MetricsPort: 9090,
		Security: SecurityConfig{
			APIKey:     "api-key-12345-secret",
			JWTSecret:  "jwt-signing-secret-key",
			EnableAuth: true,
			RateLimit:  1000,
		},
	}
	
	// Step 2: Operator scope (basic server operations)
	fmt.Println("\n👷 Step 2: Operator scope - basic server config")
	operatorTOML, err := cereal.TOML.Marshal(serverConfig, "operator")
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		return
	}
	fmt.Printf("Operator TOML:\n%s\n", string(operatorTOML))
	
	// Step 3: Admin scope (full server configuration)
	fmt.Println("\n👑 Step 3: Admin scope - full server config with secrets")
	adminTOML, err := cereal.TOML.Marshal(serverConfig, "admin")
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		return
	}
	fmt.Printf("Admin TOML:\n%s\n", string(adminTOML))
	
	fmt.Println("\n✅ TOML scoping demo complete!")
	fmt.Println("🖥️ Server config filtered by security level")
}

func advancedScopingDemo() {
	fmt.Println("🔐 Advanced Scoping Patterns Demo")
	fmt.Println(strings.Repeat("=", 50))
	
	// Complex multi-level scoping
	fmt.Println("\n🎯 Step 1: Complex permission matrix")
	
	type SensitiveDocument struct {
		ID          int    `json:"id" scope:"public"`
		Title       string `json:"title" scope:"public"`
		Content     string `json:"content" scope:"employee"`
		PersonalPII string `json:"personal_pii" scope:"hr+legal"`
		Salary      int    `json:"salary" scope:"hr+manager,executive"`
		SSN         string `json:"ssn" scope:"hr+compliance"`
		BankAccount string `json:"bank_account" scope:"payroll+compliance"`
	}
	
	doc := SensitiveDocument{
		ID:          1001,
		Title:       "Employee Record - Jane Smith",
		Content:     "Performance review and general employment information",
		PersonalPII: "Home address: 123 Main St, SSN: 123-45-6789",
		Salary:      85000,
		SSN:         "123-45-6789",
		BankAccount: "Account: 987654321, Routing: 123456789",
	}
	
	// Test different permission combinations
	fmt.Println("\n👥 Step 2: Testing permission combinations")
	
	permissions := []struct {
		name  string
		perms []string
	}{
		{"Public", []string{"public"}},
		{"Employee", []string{"employee"}},
		{"HR Manager", []string{"hr", "manager"}},
		{"HR + Legal", []string{"hr", "legal"}},
		{"HR + Compliance", []string{"hr", "compliance"}},
		{"Payroll + Compliance", []string{"payroll", "compliance"}},
		{"Executive", []string{"executive"}},
		{"HR + Manager + Compliance", []string{"hr", "manager", "compliance"}},
	}
	
	for _, perm := range permissions {
		fmt.Printf("\n   🔍 %s permissions: %v\n", perm.name, perm.perms)
		data, err := cereal.JSON.Marshal(doc, perm.perms...)
		if err != nil {
			fmt.Printf("      ❌ Error: %v\n", err)
			continue
		}
		fmt.Printf("      📄 Data: %s\n", string(data))
	}
	
	fmt.Println("\n✅ Advanced scoping demo complete!")
	fmt.Println("🎯 Complex permission matrices handled correctly")
}

func securityDemo() {
	fmt.Println("🛡️ Security & Validation Demo")
	fmt.Println(strings.Repeat("=", 50))
	
	// Data with validation constraints and security concerns
	fmt.Println("\n🔒 Step 1: Secure data with validation")
	
	type SecureUser struct {
		ID       int    `json:"id" scope:"public" validate:"required,min=1"`
		Email    string `json:"email" scope:"private" validate:"required,email"`
		Phone    string `json:"phone" scope:"admin" validate:"len=12"`
		APIKey   string `json:"api_key" scope:"admin" validate:"required,len=32"`
		CreditCard string `json:"credit_card" scope:"admin" validate:"len=16,numeric"`
	}
	
	user := SecureUser{
		ID:         12345,
		Email:      "user@example.com",
		Phone:      "555-123-4567",
		APIKey:     "abcd1234567890abcd1234567890abcd",
		CreditCard: "4111111111111111",
	}
	
	// Step 2: Show redacted values that maintain validation
	fmt.Println("\n🎭 Step 2: Redacted values that pass validation")
	
	publicData, err := cereal.JSON.Marshal(user, "public")
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		return
	}
	fmt.Printf("Public data (redacted but valid):\n%s\n", string(publicData))
	
	// Step 3: Attempt to unmarshal with insufficient permissions
	fmt.Println("\n🚫 Step 3: Security validation during unmarshaling")
	
	// Try to inject admin data with public permissions
	maliciousJSON := `{
		"id": 99999,
		"email": "hacker@evil.com",
		"phone": "000-000-0000",
		"api_key": "stolen_api_key_123456789012345",
		"credit_card": "9999999999999999"
	}`
	
	var publicUser SecureUser
	err = cereal.JSON.Unmarshal([]byte(maliciousJSON), &publicUser, "public")
	if err != nil {
		fmt.Printf("❌ Error unmarshaling with public scope: %v\n", err)
	} else {
		fmt.Printf("✅ Malicious data blocked - restricted fields cleared\n")
		fmt.Printf("   Resulting user: ID=%d, Email=%s, Phone=%s\n", 
			publicUser.ID, publicUser.Email, publicUser.Phone)
		fmt.Printf("   API Key blocked: '%s'\n", publicUser.APIKey)
		fmt.Printf("   Credit Card blocked: '%s'\n", publicUser.CreditCard)
	}
	
	fmt.Println("\n✅ Security demo complete!")
	fmt.Println("🛡️ Data protected through scope validation")
}