package main

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"
	"zbz/capitan"
)

// Simulated adapter ecosystem
type MetricsAdapter struct {
	userSignups int64
	apiCalls    int64
	errors      int64
	mu          sync.RWMutex
}

type AnalyticsAdapter struct {
	events []AnalyticsEvent
	mu     sync.RWMutex
}

type AnalyticsEvent struct {
	EventType string    `json:"event_type"`
	UserID    string    `json:"user_id,omitempty"`
	Timestamp time.Time `json:"timestamp"`
	Data      any       `json:"data"`
}

type AuditAdapter struct {
	auditLogs []AuditLog
	mu        sync.RWMutex
}

type AuditLog struct {
	Action    string    `json:"action"`
	UserID    string    `json:"user_id,omitempty"`
	Timestamp time.Time `json:"timestamp"`
	Details   any       `json:"details"`
}

// Global adapter instances
var (
	metricsAdapter   = &MetricsAdapter{}
	analyticsAdapter = &AnalyticsAdapter{}
	auditAdapter     = &AuditAdapter{}
)

func adapterEcosystemDemo() {
	println("\n3. 🔗 Adapter Ecosystem Demo")
	println("   (Multiple adapters processing same events)")
	
	// Register metrics adapter
	capitan.RegisterByteHandler("user.created", func(data []byte) error {
		metricsAdapter.mu.Lock()
		metricsAdapter.userSignups++
		metricsAdapter.mu.Unlock()
		fmt.Printf("   📊 Metrics: User signups now at %d\n", metricsAdapter.userSignups)
		return nil
	})
	
	capitan.RegisterByteHandler("api.call", func(data []byte) error {
		metricsAdapter.mu.Lock()
		metricsAdapter.apiCalls++
		metricsAdapter.mu.Unlock()
		fmt.Printf("   📊 Metrics: API calls now at %d\n", metricsAdapter.apiCalls)
		return nil
	})
	
	// Register analytics adapter
	capitan.RegisterByteHandler("user.created", func(data []byte) error {
		var eventData map[string]any
		json.Unmarshal(data, &eventData)
		
		analyticsAdapter.mu.Lock()
		analyticsAdapter.events = append(analyticsAdapter.events, AnalyticsEvent{
			EventType: "user_signup",
			UserID:    getStringFromData(eventData, "user_id"),
			Timestamp: time.Now(),
			Data:      eventData,
		})
		analyticsAdapter.mu.Unlock()
		
		fmt.Printf("   📈 Analytics: Recorded user signup event\n")
		return nil
	})
	
	// Register audit adapter
	capitan.RegisterByteHandler("user.created", func(data []byte) error {
		var eventData map[string]any
		json.Unmarshal(data, &eventData)
		
		auditAdapter.mu.Lock()
		auditAdapter.auditLogs = append(auditAdapter.auditLogs, AuditLog{
			Action:    "USER_CREATED",
			UserID:    getStringFromData(eventData, "user_id"),
			Timestamp: time.Now(),
			Details:   eventData,
		})
		auditAdapter.mu.Unlock()
		
		fmt.Printf("   🔍 Audit: Logged user creation for compliance\n")
		return nil
	})
	
	// Generate events that trigger all adapters
	println("   Generating events...")
	
	for i := 0; i < 3; i++ {
		capitan.EmitEvent("user.created", map[string]any{
			"user_id": fmt.Sprintf("user_%d", i+1),
			"email":   fmt.Sprintf("user%d@example.com", i+1),
			"plan":    "basic",
		})
		
		capitan.EmitEvent("api.call", map[string]any{
			"endpoint": "/api/users",
			"method":   "POST",
			"user_id":  fmt.Sprintf("user_%d", i+1),
		})
		
		time.Sleep(10 * time.Millisecond)
	}
	
	time.Sleep(100 * time.Millisecond)
	
	// Show adapter results
	println("   📊 Final adapter states:")
	fmt.Printf("     Metrics: %d signups, %d API calls\n", 
		metricsAdapter.userSignups, metricsAdapter.apiCalls)
	fmt.Printf("     Analytics: %d events tracked\n", 
		len(analyticsAdapter.events))
	fmt.Printf("     Audit: %d compliance logs\n", 
		len(auditAdapter.auditLogs))
}

func getStringFromData(data map[string]any, key string) string {
	if val, ok := data[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}