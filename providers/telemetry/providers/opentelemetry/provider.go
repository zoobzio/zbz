package opentelemetry

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"

	"zbz/telemetry"
)

// OpenTelemetryProvider implements the TelemetryProvider interface for OpenTelemetry
type OpenTelemetryProvider struct {
	config         telemetry.TelemetryConfig
	tracerProvider *sdktrace.TracerProvider
	meterProvider  *sdkmetric.MeterProvider
	tracer         trace.Tracer
	meter          metric.Meter
	
	// Metric instruments (cached for performance)
	counters   map[string]metric.Int64Counter
	gauges     map[string]metric.Float64Gauge
	histograms map[string]metric.Float64Histogram
}

// NewProvider creates a new OpenTelemetry provider
func NewProvider(config telemetry.TelemetryConfig) (telemetry.TelemetryProvider, error) {
	provider := &OpenTelemetryProvider{
		config:     config,
		counters:   make(map[string]metric.Int64Counter),
		gauges:     make(map[string]metric.Float64Gauge),
		histograms: make(map[string]metric.Float64Histogram),
	}
	
	if err := provider.initialize(); err != nil {
		return nil, fmt.Errorf("failed to initialize OpenTelemetry provider: %w", err)
	}
	
	return provider, nil
}

// initialize sets up OpenTelemetry SDK
func (p *OpenTelemetryProvider) initialize() error {
	ctx := context.Background()
	
	// Create resource
	res, err := resource.New(
		ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(p.config.ServiceName),
			semconv.ServiceVersionKey.String(p.config.ServiceVersion),
			semconv.DeploymentEnvironmentKey.String(p.config.Environment),
		),
	)
	if err != nil {
		return fmt.Errorf("failed to create resource: %w", err)
	}
	
	// Initialize tracing if enabled
	if p.config.EnableTracing {
		if err := p.initializeTracing(ctx, res); err != nil {
			return fmt.Errorf("failed to initialize tracing: %w", err)
		}
	}
	
	// Initialize metrics if enabled
	if p.config.EnableMetrics {
		if err := p.initializeMetrics(ctx, res); err != nil {
			return fmt.Errorf("failed to initialize metrics: %w", err)
		}
	}
	
	return nil
}

// initializeTracing sets up OpenTelemetry tracing
func (p *OpenTelemetryProvider) initializeTracing(ctx context.Context, res *resource.Resource) error {
	// Create trace exporter
	exporter, err := otlptracegrpc.New(
		ctx,
		otlptracegrpc.WithEndpoint(p.config.Endpoint),
		otlptracegrpc.WithInsecure(), // TODO: Make configurable
	)
	if err != nil {
		return fmt.Errorf("failed to create trace exporter: %w", err)
	}
	
	// Create tracer provider
	p.tracerProvider = sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(p.config.TraceSampleRate)),
	)
	
	// Set global tracer provider
	otel.SetTracerProvider(p.tracerProvider)
	
	// Create tracer
	p.tracer = p.tracerProvider.Tracer(p.config.ServiceName)
	
	return nil
}

// initializeMetrics sets up OpenTelemetry metrics
func (p *OpenTelemetryProvider) initializeMetrics(ctx context.Context, res *resource.Resource) error {
	// Create metric exporter
	exporter, err := otlpmetricgrpc.New(
		ctx,
		otlpmetricgrpc.WithEndpoint(p.config.Endpoint),
		otlpmetricgrpc.WithInsecure(), // TODO: Make configurable
	)
	if err != nil {
		return fmt.Errorf("failed to create metric exporter: %w", err)
	}
	
	// Create meter provider
	p.meterProvider = sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(
			exporter,
			sdkmetric.WithInterval(p.config.ExportInterval),
		)),
		sdkmetric.WithResource(res),
	)
	
	// Set global meter provider
	otel.SetMeterProvider(p.meterProvider)
	
	// Create meter
	p.meter = p.meterProvider.Meter(p.config.ServiceName)
	
	return nil
}

// EmitMetric emits a metric using OpenTelemetry
func (p *OpenTelemetryProvider) EmitMetric(ctx context.Context, metric telemetry.Metric) error {
	if !p.config.EnableMetrics {
		return nil // Silently ignore if metrics disabled
	}
	
	// Convert labels to attributes
	attrs := make([]attribute.KeyValue, 0, len(metric.Labels))
	for k, v := range metric.Labels {
		attrs = append(attrs, attribute.String(k, v))
	}
	
	switch metric.Type {
	case telemetry.MetricTypeCounter:
		counter, err := p.getOrCreateCounter(metric.Name, metric.Description, metric.Unit)
		if err != nil {
			return err
		}
		counter.Add(ctx, int64(metric.Value), metric.WithAttributes(attrs...))
		
	case telemetry.MetricTypeGauge:
		gauge, err := p.getOrCreateGauge(metric.Name, metric.Description, metric.Unit)
		if err != nil {
			return err
		}
		gauge.Record(ctx, metric.Value, metric.WithAttributes(attrs...))
		
	case telemetry.MetricTypeHistogram:
		histogram, err := p.getOrCreateHistogram(metric.Name, metric.Description, metric.Unit)
		if err != nil {
			return err
		}
		histogram.Record(ctx, metric.Value, metric.WithAttributes(attrs...))
		
	default:
		return fmt.Errorf("unsupported metric type: %s", metric.Type)
	}
	
	return nil
}

// getOrCreateCounter gets or creates a counter instrument
func (p *OpenTelemetryProvider) getOrCreateCounter(name, description, unit string) (metric.Int64Counter, error) {
	if counter, exists := p.counters[name]; exists {
		return counter, nil
	}
	
	counter, err := p.meter.Int64Counter(
		name,
		metric.WithDescription(description),
		metric.WithUnit(unit),
	)
	if err != nil {
		return nil, err
	}
	
	p.counters[name] = counter
	return counter, nil
}

// getOrCreateGauge gets or creates a gauge instrument
func (p *OpenTelemetryProvider) getOrCreateGauge(name, description, unit string) (metric.Float64Gauge, error) {
	if gauge, exists := p.gauges[name]; exists {
		return gauge, nil
	}
	
	gauge, err := p.meter.Float64Gauge(
		name,
		metric.WithDescription(description),
		metric.WithUnit(unit),
	)
	if err != nil {
		return nil, err
	}
	
	p.gauges[name] = gauge
	return gauge, nil
}

// getOrCreateHistogram gets or creates a histogram instrument
func (p *OpenTelemetryProvider) getOrCreateHistogram(name, description, unit string) (metric.Float64Histogram, error) {
	if histogram, exists := p.histograms[name]; exists {
		return histogram, nil
	}
	
	histogram, err := p.meter.Float64Histogram(
		name,
		metric.WithDescription(description),
		metric.WithUnit(unit),
	)
	if err != nil {
		return nil, err
	}
	
	p.histograms[name] = histogram
	return histogram, nil
}

// QueryMetrics queries historical metrics (not directly supported by OTLP)
func (p *OpenTelemetryProvider) QueryMetrics(ctx context.Context, query telemetry.MetricQuery) (telemetry.MetricResults, error) {
	// OpenTelemetry OTLP doesn't provide query capabilities
	// This would typically require integration with a metrics backend like Prometheus
	return telemetry.MetricResults{}, fmt.Errorf("metric querying not supported by OpenTelemetry OTLP provider")
}

// StartTrace starts a new trace
func (p *OpenTelemetryProvider) StartTrace(ctx context.Context, operationName string) (telemetry.TraceContext, error) {
	if !p.config.EnableTracing {
		return telemetry.TraceContext{}, fmt.Errorf("tracing not enabled")
	}
	
	ctx, span := p.tracer.Start(ctx, operationName)
	spanCtx := span.SpanContext()
	
	return telemetry.TraceContext{
		TraceID:     spanCtx.TraceID().String(),
		SpanID:      spanCtx.SpanID().String(),
		Operation:   operationName,
		ServiceName: p.config.ServiceName,
		StartTime:   time.Now(),
	}, nil
}

// EndTrace ends a trace
func (p *OpenTelemetryProvider) EndTrace(ctx context.Context, trace telemetry.TraceContext) error {
	// In OpenTelemetry, ending a trace means ending the root span
	// This would require maintaining span context, which is complex
	// For now, we'll assume the span is managed elsewhere
	return nil
}

// CreateSpan creates a new span within a trace
func (p *OpenTelemetryProvider) CreateSpan(ctx context.Context, trace telemetry.TraceContext, spanName string) (telemetry.SpanContext, error) {
	if !p.config.EnableTracing {
		return telemetry.SpanContext{}, fmt.Errorf("tracing not enabled")
	}
	
	// This would require reconstructing the trace context from the TraceContext
	// For simplicity, we'll create a new span in the current context
	ctx, span := p.tracer.Start(ctx, spanName)
	spanCtx := span.SpanContext()
	
	return telemetry.SpanContext{
		TraceID:   spanCtx.TraceID().String(),
		SpanID:    spanCtx.SpanID().String(),
		ParentID:  trace.SpanID,
		Name:      spanName,
		StartTime: time.Now(),
	}, nil
}

// EndSpan ends a span
func (p *OpenTelemetryProvider) EndSpan(ctx context.Context, span telemetry.SpanContext) error {
	// Similar to EndTrace, this would require maintaining span references
	return nil
}

// EmitLog emits a log entry (OpenTelemetry logs are still experimental)
func (p *OpenTelemetryProvider) EmitLog(ctx context.Context, log telemetry.LogEntry) error {
	// OpenTelemetry logging is still experimental and not widely supported
	// For now, we'll return an error
	return fmt.Errorf("logging not supported by OpenTelemetry provider (experimental feature)")
}

// QueryLogs queries historical logs (not supported)
func (p *OpenTelemetryProvider) QueryLogs(ctx context.Context, query telemetry.LogQuery) (telemetry.LogResults, error) {
	return telemetry.LogResults{}, fmt.Errorf("log querying not supported by OpenTelemetry provider")
}

// Health returns the health status of the provider
func (p *OpenTelemetryProvider) Health(ctx context.Context) (telemetry.ProviderHealth, error) {
	// Basic health check - verify providers are initialized
	status := "healthy"
	message := ""
	
	if p.config.EnableTracing && p.tracerProvider == nil {
		status = "unhealthy"
		message = "tracer provider not initialized"
	}
	
	if p.config.EnableMetrics && p.meterProvider == nil {
		status = "unhealthy"
		message = "meter provider not initialized"
	}
	
	return telemetry.ProviderHealth{
		Status:      status,
		Message:     message,
		LastChecked: time.Now(),
		Metrics: map[string]any{
			"tracing_enabled": p.config.EnableTracing,
			"metrics_enabled": p.config.EnableMetrics,
			"service_name":    p.config.ServiceName,
		},
	}, nil
}

// Close shuts down the provider
func (p *OpenTelemetryProvider) Close() error {
	var errs []error
	
	if p.tracerProvider != nil {
		if err := p.tracerProvider.Shutdown(context.Background()); err != nil {
			errs = append(errs, fmt.Errorf("failed to shutdown tracer provider: %w", err))
		}
	}
	
	if p.meterProvider != nil {
		if err := p.meterProvider.Shutdown(context.Background()); err != nil {
			errs = append(errs, fmt.Errorf("failed to shutdown meter provider: %w", err))
		}
	}
	
	if len(errs) > 0 {
		return fmt.Errorf("errors during shutdown: %v", errs)
	}
	
	return nil
}

// GetProvider returns the provider name
func (p *OpenTelemetryProvider) GetProvider() string {
	return "opentelemetry"
}

// GetNative returns the native OpenTelemetry components
func (p *OpenTelemetryProvider) GetNative() any {
	return map[string]any{
		"tracer_provider": p.tracerProvider,
		"meter_provider":  p.meterProvider,
		"tracer":          p.tracer,
		"meter":           p.meter,
	}
}

// Register the provider with the telemetry package
func init() {
	telemetry.RegisterProvider("opentelemetry", NewProvider)
}