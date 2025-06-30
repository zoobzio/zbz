package zlog_metrics

import (
	"sync"
	"time"
	"zbz/capitan"
	"zbz/zlog"
)

// MetricsCollector collects metrics from log events
type MetricsCollector struct {
	logCounts    map[string]int64  // Count by level
	errorCounts  map[string]int64  // Count by error type
	userActivity map[string]int64  // Count by user
	latencies    []time.Duration   // Latency metrics
	mu           sync.RWMutex
}

var collector *MetricsCollector

// EnableMetrics starts collecting metrics from log events
func EnableMetrics() *MetricsCollector {
	collector = &MetricsCollector{
		logCounts:    make(map[string]int64),
		errorCounts:  make(map[string]int64),
		userActivity: make(map[string]int64),
		latencies:    make([]time.Duration, 0),
	}
	
	// Register handler for log events
	capitan.OnLogEvent(collector.handleLogEvent)
	
	return collector
}

// handleLogEvent processes log events for metrics
func (c *MetricsCollector) handleLogEvent(event zlog.LogEvent) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	// Count by log level
	c.logCounts[event.Level]++
	
	// Extract business metrics from fields
	for _, field := range event.Fields {
		switch field.Type {
		case zlog.UserScopeType:
			// Track user activity
			userID := field.Value.(string)
			c.userActivity[userID]++
			
		case zlog.DurationType:
			// Collect latency metrics
			if duration, ok := field.Value.(time.Duration); ok {
				c.latencies = append(c.latencies, duration)
			}
			
		case zlog.ErrorType:
			// Count error types
			if err, ok := field.Value.(error); ok {
				c.errorCounts[err.Error()]++
			}
		}
	}
	
	// Special handling for error-level logs
	if event.Level == "ERROR" {
		c.errorCounts["total_errors"]++
	}
}

// GetMetrics returns current metrics snapshot
func GetMetrics() MetricsSnapshot {
	if collector == nil {
		return MetricsSnapshot{}
	}
	
	collector.mu.RLock()
	defer collector.mu.RUnlock()
	
	// Calculate latency percentiles
	var avgLatency time.Duration
	if len(collector.latencies) > 0 {
		var total time.Duration
		for _, lat := range collector.latencies {
			total += lat
		}
		avgLatency = total / time.Duration(len(collector.latencies))
	}
	
	return MetricsSnapshot{
		LogCounts:     copyMap(collector.logCounts),
		ErrorCounts:   copyMap(collector.errorCounts),
		UserActivity:  copyMap(collector.userActivity),
		AverageLatency: avgLatency,
		TotalLogs:     sumMap(collector.logCounts),
	}
}

// MetricsSnapshot represents metrics at a point in time
type MetricsSnapshot struct {
	LogCounts      map[string]int64 `json:"log_counts"`
	ErrorCounts    map[string]int64 `json:"error_counts"`
	UserActivity   map[string]int64 `json:"user_activity"`
	AverageLatency time.Duration    `json:"average_latency"`
	TotalLogs      int64            `json:"total_logs"`
}

// Helper functions
func copyMap(original map[string]int64) map[string]int64 {
	copy := make(map[string]int64)
	for k, v := range original {
		copy[k] = v
	}
	return copy
}

func sumMap(m map[string]int64) int64 {
	var total int64
	for _, v := range m {
		total += v
	}
	return total
}