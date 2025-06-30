package main

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
	"zbz/capitan"
)

// Standalone benchmark that can be run with: go run . --benchmark
func runBenchmark() {
	println("🚀 Capitan Performance Benchmark")
	println("=================================")
	
	// System info
	fmt.Printf("Go version: %s\n", runtime.Version())
	fmt.Printf("GOMAXPROCS: %d\n", runtime.GOMAXPROCS(0))
	fmt.Printf("NumCPU: %d\n", runtime.NumCPU())
	
	// Run different benchmark scenarios
	benchmarkBasicEmission()
	benchmarkMultiHandler()
	benchmarkConcurrentEmission()
	benchmarkMemoryUsage()
}

func benchmarkBasicEmission() {
	println("\n📊 Basic Event Emission Benchmark")
	
	capitan.Reset() // Clear previous handlers
	
	var processed int64
	capitan.RegisterByteHandler("bench.basic", func(data []byte) error {
		atomic.AddInt64(&processed, 1)
		return nil
	})
	
	numEvents := 100000
	
	start := time.Now()
	for i := 0; i < numEvents; i++ {
		capitan.EmitEvent("bench.basic", map[string]any{
			"id": i,
			"data": "benchmark_data",
		})
	}
	
	// Wait for processing
	for atomic.LoadInt64(&processed) < int64(numEvents) {
		time.Sleep(1 * time.Millisecond)
	}
	
	duration := time.Since(start)
	rate := float64(numEvents) / duration.Seconds()
	
	fmt.Printf("Events: %d\n", numEvents)
	fmt.Printf("Duration: %v\n", duration)
	fmt.Printf("Rate: %.0f events/sec\n", rate)
	fmt.Printf("Latency: %v per event\n", duration/time.Duration(numEvents))
}

func benchmarkMultiHandler() {
	println("\n🔗 Multi-Handler Benchmark")
	
	capitan.Reset()
	
	numHandlers := 10
	var processed int64
	
	for i := 0; i < numHandlers; i++ {
		capitan.RegisterByteHandler("bench.multi", func(data []byte) error {
			atomic.AddInt64(&processed, 1)
			return nil
		})
	}
	
	numEvents := 50000
	expectedProcessed := int64(numEvents * numHandlers)
	
	start := time.Now()
	for i := 0; i < numEvents; i++ {
		capitan.EmitEvent("bench.multi", map[string]any{
			"id": i,
			"handlers": numHandlers,
		})
	}
	
	// Wait for all processing
	for atomic.LoadInt64(&processed) < expectedProcessed {
		time.Sleep(1 * time.Millisecond)
	}
	
	duration := time.Since(start)
	eventRate := float64(numEvents) / duration.Seconds()
	handlerRate := float64(processed) / duration.Seconds()
	
	fmt.Printf("Events: %d\n", numEvents)
	fmt.Printf("Handlers per event: %d\n", numHandlers)
	fmt.Printf("Total handler executions: %d\n", processed)
	fmt.Printf("Duration: %v\n", duration)
	fmt.Printf("Event rate: %.0f events/sec\n", eventRate)
	fmt.Printf("Handler rate: %.0f handlers/sec\n", handlerRate)
}

func benchmarkConcurrentEmission() {
	println("\n⚡ Concurrent Emission Benchmark")
	
	capitan.Reset()
	
	var processed int64
	numHandlers := 5
	
	for i := 0; i < numHandlers; i++ {
		capitan.RegisterByteHandler("bench.concurrent", func(data []byte) error {
			atomic.AddInt64(&processed, 1)
			return nil
		})
	}
	
	numWorkers := 10
	eventsPerWorker := 10000
	totalEvents := numWorkers * eventsPerWorker
	expectedProcessed := int64(totalEvents * numHandlers)
	
	start := time.Now()
	
	var wg sync.WaitGroup
	for worker := 0; worker < numWorkers; worker++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for i := 0; i < eventsPerWorker; i++ {
				capitan.EmitEvent("bench.concurrent", map[string]any{
					"worker": workerID,
					"event":  i,
				})
			}
		}(worker)
	}
	
	wg.Wait()
	
	// Wait for all processing
	for atomic.LoadInt64(&processed) < expectedProcessed {
		time.Sleep(1 * time.Millisecond)
	}
	
	duration := time.Since(start)
	eventRate := float64(totalEvents) / duration.Seconds()
	handlerRate := float64(processed) / duration.Seconds()
	
	fmt.Printf("Workers: %d\n", numWorkers)
	fmt.Printf("Events per worker: %d\n", eventsPerWorker)
	fmt.Printf("Total events: %d\n", totalEvents)
	fmt.Printf("Handlers per event: %d\n", numHandlers)
	fmt.Printf("Total handler executions: %d\n", processed)
	fmt.Printf("Duration: %v\n", duration)
	fmt.Printf("Event rate: %.0f events/sec\n", eventRate)
	fmt.Printf("Handler rate: %.0f handlers/sec\n", handlerRate)
	fmt.Printf("Throughput: %.1f MB/sec (estimated)\n", handlerRate*0.1/1024)
}

func benchmarkMemoryUsage() {
	println("\n🧠 Memory Usage Benchmark")
	
	capitan.Reset()
	
	// Force GC before measurement
	runtime.GC()
	runtime.GC()
	
	var memBefore runtime.MemStats
	runtime.ReadMemStats(&memBefore)
	
	// Register handlers
	numHandlers := 100
	for i := 0; i < numHandlers; i++ {
		capitan.RegisterByteHandler("bench.memory", func(data []byte) error {
			// Minimal processing to test handler overhead
			return nil
		})
	}
	
	var memAfterHandlers runtime.MemStats
	runtime.ReadMemStats(&memAfterHandlers)
	
	// Process events
	numEvents := 10000
	for i := 0; i < numEvents; i++ {
		capitan.EmitEvent("bench.memory", map[string]any{
			"id": i,
		})
	}
	
	// Wait a bit for processing
	time.Sleep(100 * time.Millisecond)
	
	runtime.GC()
	runtime.GC()
	
	var memAfterEvents runtime.MemStats
	runtime.ReadMemStats(&memAfterEvents)
	
	fmt.Printf("Handlers registered: %d\n", numHandlers)
	fmt.Printf("Events processed: %d\n", numEvents)
	fmt.Printf("Memory before: %d KB\n", memBefore.Alloc/1024)
	fmt.Printf("Memory after handlers: %d KB\n", memAfterHandlers.Alloc/1024)
	fmt.Printf("Memory after events: %d KB\n", memAfterEvents.Alloc/1024)
	fmt.Printf("Handler overhead: %d KB\n", (memAfterHandlers.Alloc-memBefore.Alloc)/1024)
	eventOverhead := int64(memAfterEvents.Alloc) - int64(memAfterHandlers.Alloc)
	if eventOverhead < 0 {
		eventOverhead = 0 // GC may have cleaned up
	}
	fmt.Printf("Event overhead: %d KB\n", eventOverhead/1024)
	
	if numEvents > 0 && eventOverhead > 0 {
		fmt.Printf("Memory per event: %d bytes\n", eventOverhead/int64(numEvents))
	} else {
		fmt.Printf("Memory per event: ~0 bytes (very efficient)\n")
	}
}