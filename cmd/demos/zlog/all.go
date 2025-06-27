package zlog

import (
	"fmt"
	"strings"
	"time"
)

// AllDemos runs all zlog demos in sequence
func AllDemos() {
	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║                    ZBZ zlog Demo Suite                        ║")
	fmt.Println("║                Complete Logging Framework                      ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Println()
	
	// Demo 1: JSON structured logging
	fmt.Println("🚀 Demo 1: JSON Structured Logging")
	fmt.Println(strings.Repeat("-", 50))
	JsonDemo()
	
	fmt.Println("\n\n")
	time.Sleep(2 * time.Second) // Brief pause between demos
	
	// Demo 2: Security features
	fmt.Println("🛡️  Demo 2: Security & PII Protection")
	fmt.Println(strings.Repeat("-", 50))
	SecurityDemo()
	
	fmt.Println("\n\n")
	
	// Summary
	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║                     Demos Complete!                           ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Println("🎯 What you've seen:")
	fmt.Println("   ✅ Structured logging with multiple providers (zap)")
	fmt.Println("   ✅ Output piping to multiple destinations")
	fmt.Println("   ✅ Field processing pipeline with custom processors")
	fmt.Println("   ✅ Security plugin with encryption and PII protection") 
	fmt.Println("   ✅ Pattern-based sensitive data detection")
	fmt.Println("   ✅ Multiple PII handling modes (hash/redact/partial)")
	fmt.Println("   ✅ JSON and console output formats")
	fmt.Println()
	fmt.Println("📁 Output files created:")
	fmt.Println("   • /tmp/zlog-json-demo.log")
	fmt.Println("   • /tmp/zlog-security-demo.log")
	fmt.Println()
	fmt.Println("🔗 Next steps:")
	fmt.Println("   • Explore zbz/providers/zlog-* for more providers")
	fmt.Println("   • Check zbz/plugins/zlog-* for additional plugins")
	fmt.Println("   • Try 'zbz demo cereal' for serialization demos")
	fmt.Println()
}