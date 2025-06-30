package main

import (
	"fmt"
	"os"
	"time"
	"zbz/capitan"
	"zbz/zlog"
)

func main() {
	// Check for benchmark flag
	if len(os.Args) > 1 && os.Args[1] == "--benchmark" {
		runBenchmark()
		return
	}
	
	println("🏴‍☠️ Capitan Event System Demo")
	println("================================")
	println("💡 Run with --benchmark for performance testing")
	
	// Configure logging
	zlog.Configure(zlog.Config{
		Level:  zlog.INFO,
		Format: "console",
	})
	
	// Show auto-hydration (capitan automatically receives zlog events)
	autoHydrationDemo()
	
	// Show direct event emission
	directEventDemo()
	
	// Show adapter ecosystem
	adapterEcosystemDemo()
	
	// Show event transformation
	transformationDemo()
	
	// Show typed events
	typedEventDemo()
	
	// Show performance characteristics
	performanceDemo()
	
	// Show memory efficiency
	memoryDemo()
	
	println("\n✅ Capitan Demo Complete!")
	println("🚀 Ready for enterprise-scale event coordination")
}

func autoHydrationDemo() {
	println("\n1. 🔄 Auto-Hydration Demo")
	println("   (Capitan automatically receives zlog events)")
	
	// Register a simple log event handler
	capitan.RegisterByteHandler("LogEntryCreated", func(data []byte) error {
		fmt.Printf("   📊 Capitan received log event: %d bytes\n", len(data))
		return nil
	})
	
	// Just log normally - capitan will automatically receive the events
	zlog.Info("User login successful", 
		zlog.String("user_id", "alice"),
		zlog.String("ip", "192.168.1.100"))
		
	zlog.Warn("Rate limit approaching",
		zlog.String("service", "api"),
		zlog.Int("current_requests", 95))
	
	time.Sleep(50 * time.Millisecond) // Let events process
	
	stats := capitan.GetStats()
	fmt.Printf("   📈 Events processed: %d handlers for %d event types\n", 
		stats.TotalHandlers, len(stats.HookTypes))
}

func directEventDemo() {
	println("\n2. 📡 Direct Event Emission Demo")
	
	// Register handlers for custom events
	capitan.RegisterByteHandler("user.created", func(data []byte) error {
		fmt.Printf("   👤 User creation handler received: %s\n", string(data))
		return nil
	})
	
	capitan.RegisterByteHandler("user.created", func(data []byte) error {
		fmt.Printf("   📧 Welcome email handler triggered\n")
		return nil
	})
	
	// Emit custom events
	capitan.EmitEvent("user.created", map[string]any{
		"user_id": "bob",
		"email":   "bob@example.com",
		"plan":    "enterprise",
	})
	
	time.Sleep(50 * time.Millisecond)
}