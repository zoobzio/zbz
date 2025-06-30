package main

import (
	"encoding/json"
	"fmt"
	"time"
	"zbz/adapters/zlog-loki"
	"zbz/adapters/zlog-metrics"
	"zbz/zlog"
	
	// Auto-hydrate with Capitan
	_ "zbz/capitan"
)

func demoAdapters() {
	fmt.Println("\n🔌 Adapter Ecosystem Demo")
	
	// Enable metrics collection
	zlog_metrics.EnableMetrics()
	
	// Enable Loki shipping (would normally point to real Loki)
	zlog_loki.PipeToLoki("http://localhost:3100", map[string]string{
		"service":     "demo-app",
		"environment": "development",
	})
	
	// Generate some logs with routing fields
	zlog.Info("User authenticated", 
		zlog.String("user_id", "alice"),
		zlog.UserScope("alice"),
		zlog.Layer("security"),
		zlog.Concern("auth"))
	
	zlog.Error("Database query failed", 
		zlog.String("query", "SELECT * FROM users"),
		zlog.Duration("duration", 2500*time.Millisecond),
		zlog.Layer("data"),
		zlog.Concern("critical"))
	
	zlog.Info("API request processed", 
		zlog.String("endpoint", "/api/users"),
		zlog.Int("status", 200),
		zlog.Duration("latency", 150*time.Millisecond),
		zlog.UserScope("bob"),
		zlog.Privacy("public"))
	
	zlog.Warn("Rate limit approaching", 
		zlog.String("client_ip", "192.168.1.100"),
		zlog.Int("current_rate", 95),
		zlog.Int("limit", 100),
		zlog.Layer("security"),
		zlog.Concern("performance"))
	
	// Give adapters time to process
	time.Sleep(100 * time.Millisecond)
	
	// Show metrics collected
	metricsData := zlog_metrics.GetMetrics()
	fmt.Println("\n📊 Collected Metrics:")
	
	metricsJSON, _ := json.MarshalIndent(metricsData, "", "  ")
	fmt.Printf("%s\n", metricsJSON)
	
	fmt.Printf("\n✅ Adapters processed %d log events\n", metricsData.TotalLogs)
	fmt.Printf("✅ Loki adapter: Routes logs by layer/concern/privacy\n")
	fmt.Printf("✅ Metrics adapter: Extracts business metrics automatically\n")
	fmt.Printf("✅ Both adapters work simultaneously from same events\n")
}