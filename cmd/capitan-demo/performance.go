package main

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
	"zbz/capitan"
)

func performanceDemo() {
	println("\n5. ⚡ Performance Demo")
	println("   (High-throughput event processing)")
	
	// Performance counters
	var (
		eventsEmitted   int64
		eventsProcessed int64
		errors          int64
	)
	
	// Register high-performance handlers
	numHandlers := 5
	for i := 0; i < numHandlers; i++ {
		handlerID := i
		capitan.RegisterByteHandler("perf.test", func(data []byte) error {
			atomic.AddInt64(&eventsProcessed, 1)
			
			// Simulate some work (parsing, validation, etc.)
			if len(data) > 0 {
				// Success
				return nil
			}
			
			atomic.AddInt64(&errors, 1)
			return fmt.Errorf("handler %d: empty data", handlerID)
		})
	}
	
	// Performance test parameters
	numEvents := 10000
	concurrency := 10
	
	println("   📊 Test parameters:")
	fmt.Printf("     Events: %d\n", numEvents)
	fmt.Printf("     Handlers: %d\n", numHandlers)
	fmt.Printf("     Concurrency: %d\n", concurrency)
	
	// Run performance test
	start := time.Now()
	
	var wg sync.WaitGroup
	eventsPerWorker := numEvents / concurrency
	
	for worker := 0; worker < concurrency; worker++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			
			for i := 0; i < eventsPerWorker; i++ {
				capitan.EmitEvent("perf.test", map[string]any{
					"worker_id": workerID,
					"event_id":  i,
					"timestamp": time.Now().Unix(),
					"data":      fmt.Sprintf("test_data_%d_%d", workerID, i),
				})
				atomic.AddInt64(&eventsEmitted, 1)
			}
		}(worker)
	}
	
	wg.Wait()
	
	// Give time for all events to process
	time.Sleep(100 * time.Millisecond)
	
	duration := time.Since(start)
	
	// Performance results
	println("   📈 Performance Results:")
	fmt.Printf("     Duration: %v\n", duration)
	fmt.Printf("     Events emitted: %d\n", eventsEmitted)
	fmt.Printf("     Events processed: %d\n", eventsProcessed)
	fmt.Printf("     Expected total: %d (events × handlers)\n", eventsEmitted*int64(numHandlers))
	fmt.Printf("     Errors: %d\n", errors)
	
	if duration > 0 {
		emitRate := float64(eventsEmitted) / duration.Seconds()
		processRate := float64(eventsProcessed) / duration.Seconds()
		
		fmt.Printf("     Emit rate: %.0f events/sec\n", emitRate)
		fmt.Printf("     Process rate: %.0f handlers/sec\n", processRate)
		fmt.Printf("     Throughput: %.0f MB/sec (estimated)\n", processRate*0.1) // ~100 bytes per event
	}
	
	// Show system statistics
	stats := capitan.GetStats()
	println("   🔍 System Statistics:")
	fmt.Printf("     Total handler registrations: %d\n", stats.TotalHandlers)
	fmt.Printf("     Event types: %d\n", len(stats.HookTypes))
	for eventType, count := range stats.HookTypes {
		fmt.Printf("       %s: %d handlers\n", eventType, count)
	}
}

func memoryDemo() {
	println("\n6. 🧠 Memory Efficiency Demo")
	println("   (Zero-allocation event processing)")
	
	// This would normally use runtime.ReadMemStats for real memory testing
	// For demo purposes, we'll show the concept
	
	var processed int64
	
	// Register efficient byte handler (no allocation in handler)
	capitan.RegisterByteHandler("memory.test", func(data []byte) error {
		// Direct byte processing - no JSON parsing, no allocations
		if len(data) > 10 {
			atomic.AddInt64(&processed, 1)
		}
		return nil
	})
	
	println("   Processing 1000 events with zero-allocation handlers...")
	
	start := time.Now()
	
	for i := 0; i < 1000; i++ {
		// EmitEvent does allocate for JSON marshaling, but the service layer
		// and handlers can be zero-allocation if designed carefully
		capitan.EmitEvent("memory.test", map[string]any{
			"id":   i,
			"data": "test_data_for_memory_efficiency",
		})
	}
	
	time.Sleep(50 * time.Millisecond)
	duration := time.Since(start)
	
	fmt.Printf("   ✅ Processed %d events in %v\n", processed, duration)
	fmt.Printf("   💡 Key: Service layer uses byte slices, minimal allocations\n")
	fmt.Printf("   💡 Adapters can be zero-allocation if they avoid JSON parsing\n")
}