package telemetry

import (
	"context"
	"time"
)

// TelemetryProvider defines the universal interface for telemetry backends
// Compatible with OpenTelemetry, Prometheus, DataDog, New Relic, etc.
type TelemetryProvider interface {
	// Metrics operations
	EmitMetric(ctx context.Context, metric Metric) error
	QueryMetrics(ctx context.Context, query MetricQuery) (MetricResults, error)
	
	// Tracing operations
	StartTrace(ctx context.Context, operationName string) (TraceContext, error)
	EndTrace(ctx context.Context, trace TraceContext) error
	CreateSpan(ctx context.Context, trace TraceContext, spanName string) (SpanContext, error)
	EndSpan(ctx context.Context, span SpanContext) error
	
	// Logging operations
	EmitLog(ctx context.Context, log LogEntry) error
	QueryLogs(ctx context.Context, query LogQuery) (LogResults, error)
	
	// Health and lifecycle
	Health(ctx context.Context) (ProviderHealth, error)
	Close() error
	
	// Provider metadata
	GetProvider() string
	GetNative() any // Provider-specific client (*otel.TracerProvider, *prometheus.Registry)
}

// Core telemetry types

// MetricType defines the type of metric
type MetricType string

const (
	MetricTypeCounter   MetricType = "counter"
	MetricTypeGauge     MetricType = "gauge"
	MetricTypeHistogram MetricType = "histogram"
	MetricTypeSummary   MetricType = "summary"
)

// Metric represents a telemetry metric
type Metric struct {
	Name        string            `json:"name"`
	Type        MetricType        `json:"type"`
	Value       float64           `json:"value"`
	Labels      map[string]string `json:"labels,omitempty"`
	Description string            `json:"description,omitempty"`
	Unit        string            `json:"unit,omitempty"`
	Timestamp   time.Time         `json:"timestamp"`
}

// MetricQuery represents a query for historical metrics
type MetricQuery struct {
	MetricName string            `json:"metric_name"`
	StartTime  time.Time         `json:"start_time"`
	EndTime    time.Time         `json:"end_time"`
	Labels     map[string]string `json:"labels,omitempty"`
	Aggregation string           `json:"aggregation,omitempty"` // "sum", "avg", "max", "min"
	Step       time.Duration     `json:"step,omitempty"`        // Time resolution
}

// MetricResults represents the results of a metric query
type MetricResults struct {
	MetricName string              `json:"metric_name"`
	DataPoints []MetricDataPoint   `json:"data_points"`
	Labels     map[string]string   `json:"labels,omitempty"`
	Metadata   map[string]any      `json:"metadata,omitempty"`
}

// MetricDataPoint represents a single metric data point
type MetricDataPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
}

// TraceContext represents an active trace
type TraceContext struct {
	TraceID     string            `json:"trace_id"`
	SpanID      string            `json:"span_id"`
	ParentID    string            `json:"parent_id,omitempty"`
	Baggage     map[string]string `json:"baggage,omitempty"`
	StartTime   time.Time         `json:"start_time"`
	Operation   string            `json:"operation"`
	ServiceName string            `json:"service_name"`
}

// SpanContext represents an active span within a trace
type SpanContext struct {
	TraceID   string            `json:"trace_id"`
	SpanID    string            `json:"span_id"`
	ParentID  string            `json:"parent_id,omitempty"`
	Name      string            `json:"name"`
	Tags      map[string]string `json:"tags,omitempty"`
	StartTime time.Time         `json:"start_time"`
}

// LogEntry represents a structured log entry
type LogEntry struct {
	Level     string         `json:"level"`     // "debug", "info", "warn", "error"
	Message   string         `json:"message"`
	Fields    map[string]any `json:"fields,omitempty"`
	TraceID   string         `json:"trace_id,omitempty"`
	SpanID    string         `json:"span_id,omitempty"`
	Timestamp time.Time      `json:"timestamp"`
	Service   string         `json:"service,omitempty"`
}

// LogQuery represents a query for historical logs
type LogQuery struct {
	Level     string            `json:"level,omitempty"`
	StartTime time.Time         `json:"start_time"`
	EndTime   time.Time         `json:"end_time"`
	Fields    map[string]string `json:"fields,omitempty"`
	TraceID   string            `json:"trace_id,omitempty"`
	Limit     int               `json:"limit,omitempty"`
}

// LogResults represents the results of a log query
type LogResults struct {
	Entries  []LogEntry     `json:"entries"`
	Total    int64          `json:"total"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// ProviderHealth represents the health status of a telemetry provider
type ProviderHealth struct {
	Status      string         `json:"status"`      // "healthy", "degraded", "unhealthy"
	Message     string         `json:"message,omitempty"`
	LastChecked time.Time      `json:"last_checked"`
	Metrics     map[string]any `json:"metrics,omitempty"`
}

// TelemetryProviderFunction creates a telemetry provider instance
type TelemetryProviderFunction func(config TelemetryConfig) (TelemetryProvider, error)

// TelemetryConfig defines provider-agnostic telemetry configuration
type TelemetryConfig struct {
	// Provider settings
	ProviderKey     string `json:"provider_key,omitempty"`     // "default", "metrics", "traces"
	ProviderType    string `json:"provider_type"`              // "opentelemetry", "prometheus", "datadog"
	
	// Connection settings
	Endpoint        string `json:"endpoint"`                   // OTLP endpoint, Prometheus gateway, etc.
	APIKey          string `json:"api_key,omitempty"`          // For DataDog, New Relic, etc.
	SecretKey       string `json:"secret_key,omitempty"`       // For DataDog, New Relic, etc.
	Insecure        bool   `json:"insecure,omitempty"`         // Skip TLS verification
	
	// Service identification
	ServiceName     string `json:"service_name"`
	ServiceVersion  string `json:"service_version,omitempty"`
	Environment     string `json:"environment,omitempty"`      // "dev", "staging", "prod"
	
	// Feature flags
	EnableMetrics   bool   `json:"enable_metrics"`
	EnableTracing   bool   `json:"enable_tracing"`
	EnableLogging   bool   `json:"enable_logging"`
	AutoEmitMetrics bool   `json:"auto_emit_metrics"`          // Auto-emit from capitan hooks
	
	// Performance settings
	BatchSize       int           `json:"batch_size,omitempty"`       // Batch size for exports
	ExportInterval  time.Duration `json:"export_interval,omitempty"`  // How often to export
	ExportTimeout   time.Duration `json:"export_timeout,omitempty"`   // Export timeout
	
	// Sampling settings
	TraceSampleRate   float64 `json:"trace_sample_rate,omitempty"`   // 0.0 to 1.0
	MetricSampleRate  float64 `json:"metric_sample_rate,omitempty"`  // 0.0 to 1.0
	
	// Resource attributes
	ResourceAttributes map[string]string `json:"resource_attributes,omitempty"`
}

// DefaultConfig returns sensible defaults for telemetry configuration
func DefaultConfig() TelemetryConfig {
	return TelemetryConfig{
		ProviderKey:      "default",
		ServiceName:      "zbz-service",
		Environment:      "development",
		EnableMetrics:    true,
		EnableTracing:    true,
		EnableLogging:    true,
		AutoEmitMetrics:  true,
		BatchSize:        100,
		ExportInterval:   5 * time.Second,
		ExportTimeout:    30 * time.Second,
		TraceSampleRate:  1.0,
		MetricSampleRate: 1.0,
	}
}

// Provider registry for dynamic telemetry providers
var providerRegistry = make(map[string]TelemetryProviderFunction)

// RegisterProvider registers a telemetry provider factory
func RegisterProvider(name string, factory TelemetryProviderFunction) {
	providerRegistry[name] = factory
}

// NewProvider creates a provider instance by name
func NewProvider(name string, config TelemetryConfig) (TelemetryProvider, error) {
	factory, exists := providerRegistry[name]
	if !exists {
		return nil, fmt.Errorf("unknown telemetry provider: %s", name)
	}
	return factory(config)
}

// ListProviders returns all registered provider names
func ListProviders() []string {
	providers := make([]string, 0, len(providerRegistry))
	for name := range providerRegistry {
		providers = append(providers, name)
	}
	return providers
}

// Common telemetry errors
var (
	ErrProviderNotFound    = fmt.Errorf("telemetry provider not found")
	ErrInvalidMetricType   = fmt.Errorf("invalid metric type")
	ErrInvalidTimeRange    = fmt.Errorf("invalid time range")
	ErrProviderUnavailable = fmt.Errorf("telemetry provider unavailable")
	ErrNotConfigured       = fmt.Errorf("telemetry not configured")
)

// TelemetryError represents telemetry-specific errors
type TelemetryError struct {
	Code     string
	Message  string
	Provider string
	Cause    error
}

func (e *TelemetryError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (provider: %s): %v", 
			e.Code, e.Message, e.Provider, e.Cause)
	}
	return fmt.Sprintf("%s: %s (provider: %s)", 
		e.Code, e.Message, e.Provider)
}

func (e *TelemetryError) Unwrap() error {
	return e.Cause
}