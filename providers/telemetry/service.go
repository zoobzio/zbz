package telemetry

import (
	"context"
	"fmt"
	"sync"
	"time"

	"zbz/capitan"
	"zbz/universal"
)

// Global singleton service instance (following zlog pattern)
var (
	service *zTelemetryService
	mutex   sync.RWMutex
)

// zTelemetryService is the singleton telemetry service
type zTelemetryService struct {
	providers map[string]TelemetryProvider
	config    TelemetryConfig
	mutex     sync.RWMutex
}

// register initializes the telemetry service with a provider (internal function)
func register(providerFunc TelemetryProviderFunction, config TelemetryConfig) error {
	mutex.Lock()
	defer mutex.Unlock()

	provider, err := providerFunc(config)
	if err != nil {
		return fmt.Errorf("failed to create telemetry provider: %w", err)
	}

	if service == nil {
		service = &zTelemetryService{
			providers: make(map[string]TelemetryProvider),
			config:    config,
		}
	}

	// Register provider with key (default to "default" if not specified)
	key := config.ProviderKey
	if key == "" {
		key = "default"
	}

	service.providers[key] = provider

	// Auto-subscribe to capitan hooks for automatic telemetry emission
	if config.AutoEmitMetrics {
		service.setupAutoEmission()
	}

	return nil
}

// setupAutoEmission configures automatic metric emission from capitan hooks
func (s *zTelemetryService) setupAutoEmission() {
	// Subscribe to database hooks for automatic database metrics
	capitan.Subscribe(DatabaseRecordCreated, func(data DatabaseRecordCreatedData) {
		s.emitCounterMetric("database.record.created", 1, map[string]string{
			"table": data.TableName,
		})
	})

	capitan.Subscribe(DatabaseRecordUpdated, func(data DatabaseRecordUpdatedData) {
		s.emitCounterMetric("database.record.updated", 1, map[string]string{
			"table": data.TableName,
		})
	})

	capitan.Subscribe(DatabaseQueryExecuted, func(data DatabaseQueryExecutedData) {
		s.emitCounterMetric("database.query.executed", 1, map[string]string{
			"query_uri": data.QueryURI,
		})
		s.emitHistogramMetric("database.query.duration", float64(data.Duration.Milliseconds()), map[string]string{
			"query_uri": data.QueryURI,
		})
	})

	// Subscribe to HTTP hooks for automatic HTTP metrics
	capitan.Subscribe(HTTPRequestReceived, func(data HTTPRequestReceivedData) {
		s.emitCounterMetric("http.request.received", 1, map[string]string{
			"method": data.Method,
			"path":   data.Path,
		})
	})

	capitan.Subscribe(HTTPResponseSent, func(data HTTPResponseSentData) {
		s.emitCounterMetric("http.response.sent", 1, map[string]string{
			"method":      data.Method,
			"path":        data.Path,
			"status_code": fmt.Sprintf("%d", data.StatusCode),
		})
		s.emitHistogramMetric("http.request.duration", float64(data.Duration.Milliseconds()), map[string]string{
			"method": data.Method,
			"path":   data.Path,
		})
	})
}

// emitCounterMetric emits a counter metric to all providers
func (s *zTelemetryService) emitCounterMetric(name string, value float64, labels map[string]string) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	metric := Metric{
		Name:      name,
		Type:      MetricTypeCounter,
		Value:     value,
		Labels:    labels,
		Timestamp: time.Now(),
	}

	for _, provider := range s.providers {
		go provider.EmitMetric(context.Background(), metric)
	}
}

// emitHistogramMetric emits a histogram metric to all providers
func (s *zTelemetryService) emitHistogramMetric(name string, value float64, labels map[string]string) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	metric := Metric{
		Name:      name,
		Type:      MetricTypeHistogram,
		Value:     value,
		Labels:    labels,
		Timestamp: time.Now(),
	}

	for _, provider := range s.providers {
		go provider.EmitMetric(context.Background(), metric)
	}
}

// getProvider returns a provider by key
func (s *zTelemetryService) getProvider(key string) (TelemetryProvider, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if key == "" {
		key = "default"
	}

	provider, exists := s.providers[key]
	if !exists {
		return nil, fmt.Errorf("telemetry provider '%s' not found", key)
	}

	return provider, nil
}

// Execute performs telemetry operations via OperationURI
func (s *zTelemetryService) Execute(operation universal.OperationURI, params any) (any, error) {
	if operation.Service() != "telemetry" {
		return nil, fmt.Errorf("invalid service '%s' for telemetry operation", operation.Service())
	}

	// Route based on operation category
	switch operation.Category() {
	case "metrics":
		return s.executeMetricOperation(operation.Operation(), params)
	case "traces":
		return s.executeTraceOperation(operation.Operation(), params)
	case "logs":
		return s.executeLogOperation(operation.Operation(), params)
	default:
		return nil, fmt.Errorf("unsupported telemetry operation category: %s", operation.Category())
	}
}

// executeMetricOperation handles metric operations
func (s *zTelemetryService) executeMetricOperation(operation string, params any) (any, error) {
	paramsMap, ok := params.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("params must be map[string]any for metric operations")
	}

	switch operation {
	case "emit":
		metric, err := s.parseMetricFromParams(paramsMap)
		if err != nil {
			return nil, err
		}
		return nil, s.emitMetricToProvider(metric, paramsMap["provider"].(string))

	case "query":
		return s.queryMetrics(paramsMap)

	default:
		return nil, fmt.Errorf("unsupported metric operation: %s", operation)
	}
}

// executeTraceOperation handles trace operations
func (s *zTelemetryService) executeTraceOperation(operation string, params any) (any, error) {
	paramsMap, ok := params.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("params must be map[string]any for trace operations")
	}

	switch operation {
	case "start":
		return s.startTrace(paramsMap)
	case "end":
		return nil, s.endTrace(paramsMap)
	case "span":
		return s.createSpan(paramsMap)
	default:
		return nil, fmt.Errorf("unsupported trace operation: %s", operation)
	}
}

// executeLogOperation handles log operations
func (s *zTelemetryService) executeLogOperation(operation string, params any) (any, error) {
	paramsMap, ok := params.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("params must be map[string]any for log operations")
	}

	switch operation {
	case "structured":
		return nil, s.emitStructuredLog(paramsMap)
	case "query":
		return s.queryLogs(paramsMap)
	default:
		return nil, fmt.Errorf("unsupported log operation: %s", operation)
	}
}

// Helper methods for operation execution
func (s *zTelemetryService) parseMetricFromParams(params map[string]any) (Metric, error) {
	return Metric{
		Name:      params["name"].(string),
		Type:      MetricType(params["type"].(string)),
		Value:     params["value"].(float64),
		Labels:    params["labels"].(map[string]string),
		Timestamp: time.Now(),
	}, nil
}

func (s *zTelemetryService) emitMetricToProvider(metric Metric, providerKey string) error {
	provider, err := s.getProvider(providerKey)
	if err != nil {
		return err
	}
	return provider.EmitMetric(context.Background(), metric)
}

func (s *zTelemetryService) queryMetrics(params map[string]any) (any, error) {
	providerKey := params["provider"].(string)
	provider, err := s.getProvider(providerKey)
	if err != nil {
		return nil, err
	}
	
	query := MetricQuery{
		MetricName: params["metric_name"].(string),
		StartTime:  params["start_time"].(time.Time),
		EndTime:    params["end_time"].(time.Time),
		Labels:     params["labels"].(map[string]string),
	}
	
	return provider.QueryMetrics(context.Background(), query)
}

func (s *zTelemetryService) startTrace(params map[string]any) (TraceContext, error) {
	providerKey := params["provider"].(string)
	provider, err := s.getProvider(providerKey)
	if err != nil {
		return TraceContext{}, err
	}
	
	return provider.StartTrace(context.Background(), params["operation_name"].(string))
}

func (s *zTelemetryService) endTrace(params map[string]any) error {
	providerKey := params["provider"].(string)
	provider, err := s.getProvider(providerKey)
	if err != nil {
		return err
	}
	
	traceCtx := params["trace_context"].(TraceContext)
	return provider.EndTrace(context.Background(), traceCtx)
}

func (s *zTelemetryService) createSpan(params map[string]any) (SpanContext, error) {
	providerKey := params["provider"].(string)
	provider, err := s.getProvider(providerKey)
	if err != nil {
		return SpanContext{}, err
	}
	
	traceCtx := params["trace_context"].(TraceContext)
	spanName := params["span_name"].(string)
	
	return provider.CreateSpan(context.Background(), traceCtx, spanName)
}

func (s *zTelemetryService) emitStructuredLog(params map[string]any) error {
	providerKey := params["provider"].(string)
	provider, err := s.getProvider(providerKey)
	if err != nil {
		return err
	}
	
	logEntry := LogEntry{
		Level:     params["level"].(string),
		Message:   params["message"].(string),
		Fields:    params["fields"].(map[string]any),
		Timestamp: time.Now(),
	}
	
	return provider.EmitLog(context.Background(), logEntry)
}

func (s *zTelemetryService) queryLogs(params map[string]any) (any, error) {
	providerKey := params["provider"].(string)
	provider, err := s.getProvider(providerKey)
	if err != nil {
		return nil, err
	}
	
	query := LogQuery{
		Level:     params["level"].(string),
		StartTime: params["start_time"].(time.Time),
		EndTime:   params["end_time"].(time.Time),
		Fields:    params["fields"].(map[string]string),
	}
	
	return provider.QueryLogs(context.Background(), query)
}

// Config returns the current telemetry configuration
func (s *zTelemetryService) Config() TelemetryConfig {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.config
}

// Close shuts down all telemetry providers
func (s *zTelemetryService) Close() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	for _, provider := range s.providers {
		if err := provider.Close(); err != nil {
			return err
		}
	}

	service = nil
	return nil
}

// Hook type definitions for capitan integration
type TelemetryHookType int

const (
	DatabaseRecordCreated TelemetryHookType = iota
	DatabaseRecordUpdated
	DatabaseQueryExecuted
	HTTPRequestReceived
	HTTPResponseSent
)

func (t TelemetryHookType) String() string {
	switch t {
	case DatabaseRecordCreated:
		return "database.record.created"
	case DatabaseRecordUpdated:
		return "database.record.updated"
	case DatabaseQueryExecuted:
		return "database.query.executed"
	case HTTPRequestReceived:
		return "http.request.received"
	case HTTPResponseSent:
		return "http.response.sent"
	default:
		return "telemetry.unknown"
	}
}

// Hook data types
type DatabaseRecordCreatedData struct {
	TableName string `json:"table_name"`
	RecordID  any    `json:"record_id"`
}

type DatabaseRecordUpdatedData struct {
	TableName string `json:"table_name"`
	RecordID  any    `json:"record_id"`
}

type DatabaseQueryExecutedData struct {
	QueryURI string        `json:"query_uri"`
	Duration time.Duration `json:"duration"`
}

type HTTPRequestReceivedData struct {
	Method string `json:"method"`
	Path   string `json:"path"`
}

type HTTPResponseSentData struct {
	Method     string        `json:"method"`
	Path       string        `json:"path"`
	StatusCode int           `json:"status_code"`
	Duration   time.Duration `json:"duration"`
}