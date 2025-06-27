package cereal

import (
	"fmt"
	"strings"
	"time"
)

// AllDemos runs all cereal demos in sequence
func AllDemos() {
	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║                   ZBZ cereal Demo Suite                       ║")
	fmt.Println("║              Permission-Based Serialization                   ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Println()
	
	// Demo 1: JSON user data scoping
	fmt.Println("📦 Demo 1: JSON User Data Scoping")
	fmt.Println(strings.Repeat("-", 50))
	JsonDemo()
	
	fmt.Println("\n\n")
	time.Sleep(2 * time.Second) // Brief pause between demos
	
	// Demo 2: YAML application configuration
	fmt.Println("⚙️  Demo 2: YAML Application Configuration")
	fmt.Println(strings.Repeat("-", 50))
	YamlDemo()
	
	fmt.Println("\n\n")
	time.Sleep(2 * time.Second)
	
	// Demo 3: TOML server configuration
	fmt.Println("🖥️  Demo 3: TOML Server Configuration")
	fmt.Println(strings.Repeat("-", 50))
	TomlDemo()
	
	fmt.Println("\n\n")
	
	// Summary
	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
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
	fmt.Println("   ✅ Real-world use cases for each format")
	fmt.Println()
	fmt.Println("💡 Use cases demonstrated:")
	fmt.Println("   • User profiles with privacy levels")
	fmt.Println("   • Application config with environment separation")
	fmt.Println("   • Server config with security-sensitive data")
	fmt.Println("   • Multi-environment and multi-server deployments")
	fmt.Println()
	fmt.Println("🔗 Next steps:")
	fmt.Println("   • Explore zbz/cereal for more scoping examples")
	fmt.Println("   • Try 'zbz demo zlog' for logging demonstrations")
	fmt.Println("   • Check individual format tests in zbz/cereal/tests/")
	fmt.Println()
}