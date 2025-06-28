package telemetry

import (
	"context"
	"time"

	"zbz/capitan"
)

// TelemetryHookType defines telemetry service hook types
type TelemetryHookType int

const (
	MetricEmitted TelemetryHookType = iota
	TraceStarted
	TraceEnded
	SpanCreated
	SpanEnded
	LogEmitted
	ProviderHealthChanged
	ConfigurationChanged
)

func (t TelemetryHookType) String() string {
	switch t {
	case MetricEmitted:
		return "telemetry.metric_emitted"
	case TraceStarted:
		return "telemetry.trace_started"
	case TraceEnded:
		return "telemetry.trace_ended"
	case SpanCreated:
		return "telemetry.span_created"
	case SpanEnded:
		return "telemetry.span_ended"
	case LogEmitted:
		return "telemetry.log_emitted"
	case ProviderHealthChanged:
		return "telemetry.provider_health_changed"
	case ConfigurationChanged:
		return "telemetry.configuration_changed"
	default:
		return "telemetry.unknown"
	}
}

// Telemetry Event Data Types

type MetricEmittedData struct {
	MetricName string            `json:"metric_name"`
	MetricType MetricType        `json:"metric_type"`
	Value      float64           `json:"value"`
	Labels     map[string]string `json:"labels,omitempty"`
	Provider   string            `json:"provider"`
	Timestamp  time.Time         `json:"timestamp"`
}

type TraceStartedData struct {
	TraceID       string    `json:"trace_id"`
	OperationName string    `json:"operation_name"`
	ServiceName   string    `json:"service_name"`
	Provider      string    `json:"provider"`
	StartTime     time.Time `json:"start_time"`
}

type TraceEndedData struct {
	TraceID     string        `json:"trace_id"`
	Duration    time.Duration `json:"duration"`
	SpanCount   int           `json:"span_count"`
	Provider    string        `json:"provider"`
	EndTime     time.Time     `json:"end_time"`
}

type SpanCreatedData struct {
	TraceID   string            `json:"trace_id"`
	SpanID    string            `json:"span_id"`
	ParentID  string            `json:"parent_id,omitempty"`
	Name      string            `json:"name"`
	Tags      map[string]string `json:"tags,omitempty"`
	Provider  string            `json:"provider"`
	StartTime time.Time         `json:"start_time"`
}

type SpanEndedData struct {
	TraceID  string        `json:"trace_id"`
	SpanID   string        `json:"span_id"`
	Name     string        `json:"name"`
	Duration time.Duration `json:"duration"`
	Provider string        `json:"provider"`
	EndTime  time.Time     `json:"end_time"`
}

type LogEmittedData struct {
	Level     string         `json:"level"`
	Message   string         `json:"message"`
	Fields    map[string]any `json:"fields,omitempty"`
	TraceID   string         `json:"trace_id,omitempty"`
	SpanID    string         `json:"span_id,omitempty"`
	Provider  string         `json:"provider"`
	Timestamp time.Time      `json:"timestamp"`
}

type ProviderHealthChangedData struct {
	Provider    string         `json:"provider"`
	OldStatus   string         `json:"old_status"`
	NewStatus   string         `json:"new_status"`
	Message     string         `json:"message,omitempty"`
	Metrics     map[string]any `json:"metrics,omitempty"`
	Timestamp   time.Time      `json:"timestamp"`
}

type ConfigurationChangedData struct {
	Provider   string                 `json:"provider"`
	OldConfig  map[string]any         `json:"old_config"`
	NewConfig  map[string]any         `json:"new_config"`
	Changes    []ConfigurationChange  `json:"changes"`
	Timestamp  time.Time              `json:"timestamp"`
}

type ConfigurationChange struct {
	Field    string `json:"field"`
	OldValue any    `json:"old_value"`
	NewValue any    `json:"new_value"`
}

// Hook emission helpers for the telemetry service

// emitMetricEmittedHook emits a hook when a metric is emitted
func (s *zTelemetryService) emitMetricEmittedHook(ctx context.Context, metric Metric, provider string) {
	data := MetricEmittedData{
		MetricName: metric.Name,
		MetricType: metric.Type,
		Value:      metric.Value,
		Labels:     metric.Labels,
		Provider:   provider,
		Timestamp:  metric.Timestamp,
	}
	
	capitan.Emit(ctx, MetricEmitted, "telemetry-service", data, nil)
}

// emitTraceStartedHook emits a hook when a trace is started
func (s *zTelemetryService) emitTraceStartedHook(ctx context.Context, trace TraceContext, provider string) {
	data := TraceStartedData{
		TraceID:       trace.TraceID,
		OperationName: trace.Operation,
		ServiceName:   trace.ServiceName,
		Provider:      provider,
		StartTime:     trace.StartTime,
	}
	
	capitan.Emit(ctx, TraceStarted, "telemetry-service", data, nil)
}

// emitTraceEndedHook emits a hook when a trace is ended
func (s *zTelemetryService) emitTraceEndedHook(ctx context.Context, trace TraceContext, duration time.Duration, spanCount int, provider string) {
	data := TraceEndedData{
		TraceID:   trace.TraceID,
		Duration:  duration,
		SpanCount: spanCount,
		Provider:  provider,
		EndTime:   time.Now(),
	}
	
	capitan.Emit(ctx, TraceEnded, "telemetry-service", data, nil)
}

// emitSpanCreatedHook emits a hook when a span is created
func (s *zTelemetryService) emitSpanCreatedHook(ctx context.Context, span SpanContext, provider string) {
	data := SpanCreatedData{
		TraceID:   span.TraceID,
		SpanID:    span.SpanID,
		ParentID:  span.ParentID,
		Name:      span.Name,
		Tags:      span.Tags,
		Provider:  provider,
		StartTime: span.StartTime,
	}
	
	capitan.Emit(ctx, SpanCreated, "telemetry-service", data, nil)
}

// emitSpanEndedHook emits a hook when a span is ended
func (s *zTelemetryService) emitSpanEndedHook(ctx context.Context, span SpanContext, duration time.Duration, provider string) {
	data := SpanEndedData{
		TraceID:  span.TraceID,
		SpanID:   span.SpanID,
		Name:     span.Name,
		Duration: duration,
		Provider: provider,
		EndTime:  time.Now(),
	}
	
	capitan.Emit(ctx, SpanEnded, "telemetry-service", data, nil)
}

// emitLogEmittedHook emits a hook when a log is emitted
func (s *zTelemetryService) emitLogEmittedHook(ctx context.Context, log LogEntry, provider string) {
	data := LogEmittedData{
		Level:     log.Level,
		Message:   log.Message,
		Fields:    log.Fields,
		TraceID:   log.TraceID,
		SpanID:    log.SpanID,
		Provider:  provider,
		Timestamp: log.Timestamp,
	}
	
	capitan.Emit(ctx, LogEmitted, "telemetry-service", data, nil)
}

// emitProviderHealthChangedHook emits a hook when provider health changes
func (s *zTelemetryService) emitProviderHealthChangedHook(ctx context.Context, provider, oldStatus, newStatus, message string, metrics map[string]any) {
	data := ProviderHealthChangedData{
		Provider:  provider,
		OldStatus: oldStatus,
		NewStatus: newStatus,
		Message:   message,
		Metrics:   metrics,
		Timestamp: time.Now(),
	}
	
	capitan.Emit(ctx, ProviderHealthChanged, "telemetry-service", data, nil)
}

// emitConfigurationChangedHook emits a hook when configuration changes
func (s *zTelemetryService) emitConfigurationChangedHook(ctx context.Context, provider string, oldConfig, newConfig TelemetryConfig) {
	changes := s.calculateConfigurationChanges(oldConfig, newConfig)
	
	data := ConfigurationChangedData{
		Provider:  provider,
		OldConfig: s.configToMap(oldConfig),
		NewConfig: s.configToMap(newConfig),
		Changes:   changes,
		Timestamp: time.Now(),
	}
	
	capitan.Emit(ctx, ConfigurationChanged, "telemetry-service", data, nil)
}

// calculateConfigurationChanges computes the differences between two configurations
func (s *zTelemetryService) calculateConfigurationChanges(old, new TelemetryConfig) []ConfigurationChange {
	var changes []ConfigurationChange
	
	if old.ServiceName != new.ServiceName {
		changes = append(changes, ConfigurationChange{
			Field:    "service_name",
			OldValue: old.ServiceName,
			NewValue: new.ServiceName,
		})
	}
	
	if old.Environment != new.Environment {
		changes = append(changes, ConfigurationChange{
			Field:    "environment",
			OldValue: old.Environment,
			NewValue: new.Environment,
		})
	}
	
	if old.EnableMetrics != new.EnableMetrics {
		changes = append(changes, ConfigurationChange{
			Field:    "enable_metrics",
			OldValue: old.EnableMetrics,
			NewValue: new.EnableMetrics,
		})
	}
	
	if old.EnableTracing != new.EnableTracing {
		changes = append(changes, ConfigurationChange{
			Field:    "enable_tracing",
			OldValue: old.EnableTracing,
			NewValue: new.EnableTracing,
		})
	}
	
	if old.EnableLogging != new.EnableLogging {
		changes = append(changes, ConfigurationChange{
			Field:    "enable_logging",
			OldValue: old.EnableLogging,
			NewValue: new.EnableLogging,
		})
	}
	
	if old.TraceSampleRate != new.TraceSampleRate {
		changes = append(changes, ConfigurationChange{
			Field:    "trace_sample_rate",
			OldValue: old.TraceSampleRate,
			NewValue: new.TraceSampleRate,
		})
	}
	
	return changes
}

// configToMap converts a TelemetryConfig to a map for serialization
func (s *zTelemetryService) configToMap(config TelemetryConfig) map[string]any {
	return map[string]any{
		"provider_key":        config.ProviderKey,
		"provider_type":       config.ProviderType,
		"service_name":        config.ServiceName,
		"service_version":     config.ServiceVersion,
		"environment":         config.Environment,
		"enable_metrics":      config.EnableMetrics,
		"enable_tracing":      config.EnableTracing,
		"enable_logging":      config.EnableLogging,
		"auto_emit_metrics":   config.AutoEmitMetrics,
		"trace_sample_rate":   config.TraceSampleRate,
		"metric_sample_rate":  config.MetricSampleRate,
		"export_interval":     config.ExportInterval.String(),
		"export_timeout":      config.ExportTimeout.String(),
		"resource_attributes": config.ResourceAttributes,
	}
}