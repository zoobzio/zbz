package telemetry

import (
	"context"
	"fmt"

	"zbz/universal"
)

// Public API functions that delegate to the singleton service (like database pattern)

// Register creates telemetry service from provider function and config (main API)
// Example: telemetry.Register(opentelemetry.NewProvider, config)
func Register(providerFunc TelemetryProviderFunction, config TelemetryConfig) error {
	return register(providerFunc, config)
}

// Collection creates a typed telemetry contract from the singleton service
// Example: telemetry.Collection[Metric]("metrics")
func Collection[T any](name string, telemetryType TelemetryType) TelemetryContract[T] {
	if service == nil {
		panic("telemetry not configured - call telemetry.Register() first")
	}

	return &zTelemetryContract[T]{
		collectionName: name,
		providerKey:    "default",
		service:        service,
		collectionType: telemetryType,
	}
}

// CollectionWithProvider creates a typed telemetry contract with specific provider
// Example: telemetry.CollectionWithProvider[Metric]("metrics", "prometheus", TelemetryTypeMetric)
func CollectionWithProvider[T any](name, providerKey string, telemetryType TelemetryType) TelemetryContract[T] {
	if service == nil {
		panic("telemetry not configured - call telemetry.Register() first")
	}

	return &zTelemetryContract[T]{
		collectionName: name,
		providerKey:    providerKey,
		service:        service,
		collectionType: telemetryType,
	}
}

// Metrics creates a metrics collection contract
// Example: telemetry.Metrics[Metric]("application")
func Metrics[T any](name string) TelemetryContract[T] {
	return Collection[T](name, TelemetryTypeMetric)
}

// Traces creates a traces collection contract
// Example: telemetry.Traces[TraceContext]("requests")
func Traces[T any](name string) TelemetryContract[T] {
	return Collection[T](name, TelemetryTypeTrace)
}

// Logs creates a logs collection contract
// Example: telemetry.Logs[LogEntry]("application")
func Logs[T any](name string) TelemetryContract[T] {
	return Collection[T](name, TelemetryTypeLog)
}

// URI-based operations

// Execute runs a telemetry operation via URI resolution
// Example: telemetry.Execute("telemetry://metrics/emit", params)
func Execute(operationURI string, params map[string]any) (any, error) {
	if service == nil {
		return nil, ErrNotConfigured
	}
	
	opURI, err := universal.ParseOperationURI(operationURI)
	if err != nil {
		return nil, fmt.Errorf("invalid operation URI: %w", err)
	}
	
	return service.Execute(opURI, params)
}

// Quick emit functions for common operations

// EmitMetric emits a metric using the default provider
func EmitMetric(ctx context.Context, metric Metric) error {
	if service == nil {
		return ErrNotConfigured
	}
	
	provider, err := service.getProvider("default")
	if err != nil {
		return err
	}
	
	return provider.EmitMetric(ctx, metric)
}

// EmitCounter emits a counter metric with convenience parameters
func EmitCounter(ctx context.Context, name string, value float64, labels map[string]string) error {
	metric := Metric{
		Name:   name,
		Type:   MetricTypeCounter,
		Value:  value,
		Labels: labels,
	}
	return EmitMetric(ctx, metric)
}

// EmitGauge emits a gauge metric with convenience parameters
func EmitGauge(ctx context.Context, name string, value float64, labels map[string]string) error {
	metric := Metric{
		Name:   name,
		Type:   MetricTypeGauge,
		Value:  value,
		Labels: labels,
	}
	return EmitMetric(ctx, metric)
}

// EmitHistogram emits a histogram metric with convenience parameters
func EmitHistogram(ctx context.Context, name string, value float64, labels map[string]string) error {
	metric := Metric{
		Name:   name,
		Type:   MetricTypeHistogram,
		Value:  value,
		Labels: labels,
	}
	return EmitMetric(ctx, metric)
}

// EmitLog emits a log entry using the default provider
func EmitLog(ctx context.Context, log LogEntry) error {
	if service == nil {
		return ErrNotConfigured
	}
	
	provider, err := service.getProvider("default")
	if err != nil {
		return err
	}
	
	return provider.EmitLog(ctx, log)
}

// EmitStructuredLog emits a structured log with convenience parameters
func EmitStructuredLog(ctx context.Context, level, message string, fields map[string]any) error {
	log := LogEntry{
		Level:   level,
		Message: message,
		Fields:  fields,
	}
	return EmitLog(ctx, log)
}

// Tracing operations

// StartTrace starts a new trace using the default provider
func StartTrace(ctx context.Context, operationName string) (TraceContext, error) {
	if service == nil {
		return TraceContext{}, ErrNotConfigured
	}
	
	provider, err := service.getProvider("default")
	if err != nil {
		return TraceContext{}, err
	}
	
	return provider.StartTrace(ctx, operationName)
}

// CreateSpan creates a span within an existing trace
func CreateSpan(ctx context.Context, trace TraceContext, spanName string) (SpanContext, error) {
	if service == nil {
		return SpanContext{}, ErrNotConfigured
	}
	
	provider, err := service.getProvider("default")
	if err != nil {
		return SpanContext{}, err
	}
	
	return provider.CreateSpan(ctx, trace, spanName)
}

// EndTrace ends a trace
func EndTrace(ctx context.Context, trace TraceContext) error {
	if service == nil {
		return ErrNotConfigured
	}
	
	provider, err := service.getProvider("default")
	if err != nil {
		return err
	}
	
	return provider.EndTrace(ctx, trace)
}

// EndSpan ends a span
func EndSpan(ctx context.Context, span SpanContext) error {
	if service == nil {
		return ErrNotConfigured
	}
	
	provider, err := service.getProvider("default")
	if err != nil {
		return err
	}
	
	return provider.EndSpan(ctx, span)
}

// Query operations

// QueryMetrics queries historical metrics
func QueryMetrics(ctx context.Context, query MetricQuery) (MetricResults, error) {
	if service == nil {
		return MetricResults{}, ErrNotConfigured
	}
	
	provider, err := service.getProvider("default")
	if err != nil {
		return MetricResults{}, err
	}
	
	return provider.QueryMetrics(ctx, query)
}

// QueryLogs queries historical logs
func QueryLogs(ctx context.Context, query LogQuery) (LogResults, error) {
	if service == nil {
		return LogResults{}, ErrNotConfigured
	}
	
	provider, err := service.getProvider("default")
	if err != nil {
		return LogResults{}, err
	}
	
	return provider.QueryLogs(ctx, query)
}

// Health and management operations

// Health returns the health status of all telemetry providers
func Health(ctx context.Context) (map[string]ProviderHealth, error) {
	if service == nil {
		return nil, ErrNotConfigured
	}
	
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	
	health := make(map[string]ProviderHealth)
	for key, provider := range service.providers {
		providerHealth, err := provider.Health(ctx)
		if err != nil {
			providerHealth = ProviderHealth{
				Status:      "unhealthy",
				Message:     err.Error(),
				LastChecked: time.Now(),
			}
		}
		health[key] = providerHealth
	}
	
	return health, nil
}

// Config returns the current telemetry configuration
func Config() TelemetryConfig {
	if service == nil {
		return TelemetryConfig{}
	}
	return service.Config()
}

// Provider returns a specific telemetry provider by key
func Provider(key string) (TelemetryProvider, error) {
	if service == nil {
		return nil, ErrNotConfigured
	}
	return service.getProvider(key)
}

// DefaultProvider returns the default telemetry provider
func DefaultProvider() (TelemetryProvider, error) {
	return Provider("default")
}

// Close shuts down the telemetry service
func Close() error {
	if service == nil {
		return nil
	}
	return service.Close()
}

// Utility functions

// IsConfigured returns true if the telemetry service has been configured
func IsConfigured() bool {
	return service != nil
}

// WithProvider returns an operation URI with specific provider
func WithProvider(baseURI, providerKey string) string {
	return fmt.Sprintf("%s?provider=%s", baseURI, providerKey)
}

// Convenience constructors for common metrics

// NewCounterMetric creates a new counter metric
func NewCounterMetric(name string, value float64, labels map[string]string) Metric {
	return Metric{
		Name:   name,
		Type:   MetricTypeCounter,
		Value:  value,
		Labels: labels,
	}
}

// NewGaugeMetric creates a new gauge metric
func NewGaugeMetric(name string, value float64, labels map[string]string) Metric {
	return Metric{
		Name:   name,
		Type:   MetricTypeGauge,
		Value:  value,
		Labels: labels,
	}
}

// NewHistogramMetric creates a new histogram metric
func NewHistogramMetric(name string, value float64, labels map[string]string) Metric {
	return Metric{
		Name:   name,
		Type:   MetricTypeHistogram,
		Value:  value,
		Labels: labels,
	}
}

// NewLogEntry creates a new structured log entry
func NewLogEntry(level, message string, fields map[string]any) LogEntry {
	return LogEntry{
		Level:   level,
		Message: message,
		Fields:  fields,
	}
}

// Batch operations for performance

// EmitMetricBatch emits multiple metrics in a single operation
func EmitMetricBatch(ctx context.Context, metrics []Metric) error {
	if service == nil {
		return ErrNotConfigured
	}
	
	provider, err := service.getProvider("default")
	if err != nil {
		return err
	}
	
	// Emit each metric (providers can optimize internally)
	for _, metric := range metrics {
		if err := provider.EmitMetric(ctx, metric); err != nil {
			return err
		}
	}
	
	return nil
}

// EmitLogBatch emits multiple log entries in a single operation
func EmitLogBatch(ctx context.Context, logs []LogEntry) error {
	if service == nil {
		return ErrNotConfigured
	}
	
	provider, err := service.getProvider("default")
	if err != nil {
		return err
	}
	
	// Emit each log (providers can optimize internally)
	for _, log := range logs {
		if err := provider.EmitLog(ctx, log); err != nil {
			return err
		}
	}
	
	return nil
}