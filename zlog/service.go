package zlog

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"
)

// ZlogProvider defines the interface that providers implement
type ZlogProvider interface {
	// Core logging methods
	Info(msg string, fields []Field)
	Error(msg string, fields []Field) 
	Debug(msg string, fields []Field)
	Warn(msg string, fields []Field)
	Fatal(msg string, fields []Field)
	Close() error
}

// HodorContract represents the interface we need from hodor (to avoid import cycles)
type HodorContract interface {
	Get(key string) ([]byte, error)
	Set(key string, data []byte, ttl interface{}) error // ttl as interface{} to avoid time import
	Delete(key string) error
	Exists(key string) (bool, error)
	List(prefix string) ([]string, error)
	GetProvider() string
	Name() string
}


// zZlog is the internal service layer that handles preprocessing
type zZlog struct {
	provider ZlogProvider
	contract string // Track which contract created this service
}

// NewZlogService creates a new service with provider
func NewZlogService(provider ZlogProvider) *zZlog {
	return &zZlog{
		provider: provider,
		contract: "", // Will be set by contract when self-registering
	}
}

// Info processes fields through pipeline and calls provider
func (z *zZlog) Info(msg string, fields ...Field) {
	processedFields := z.processFields(fields)
	z.provider.Info(msg, processedFields)
}

// Error processes fields through pipeline and calls provider
func (z *zZlog) Error(msg string, fields ...Field) {
	processedFields := z.processFields(fields)
	z.provider.Error(msg, processedFields)
}

// Debug processes fields through pipeline and calls provider
func (z *zZlog) Debug(msg string, fields ...Field) {
	processedFields := z.processFields(fields)
	z.provider.Debug(msg, processedFields)
}

// Warn processes fields through pipeline and calls provider
func (z *zZlog) Warn(msg string, fields ...Field) {
	processedFields := z.processFields(fields)
	z.provider.Warn(msg, processedFields)
}

// Fatal processes fields through pipeline and calls provider
func (z *zZlog) Fatal(msg string, fields ...Field) {
	processedFields := z.processFields(fields)
	z.provider.Fatal(msg, processedFields)
}

// processFields runs fields through the preprocessing pipeline
func (z *zZlog) processFields(fields []Field) []Field {
	var processed []Field
	
	for _, field := range fields {
		switch field.Type {
		case CallDepthType:
			// Skip configuration fields - these don't become log data
			continue
			
		case SecretType:
			// Encrypt sensitive data
			processed = append(processed, z.processSecret(field))
			
		case PIIType:
			// Hash PII for compliance
			processed = append(processed, z.processPII(field))
			
		case MetricType:
			// Convert metrics to structured fields
			processed = append(processed, z.processMetric(field))
			
		case CorrelationType:
			// Extract correlation IDs from context
			correlationFields := z.processCorrelation(field)
			processed = append(processed, correlationFields...)
			
		default:
			// Regular fields pass through unchanged
			processed = append(processed, field)
		}
	}
	
	return processed
}

// processSecret encrypts sensitive data (placeholder implementation)
func (z *zZlog) processSecret(field Field) Field {
	// TODO: Implement proper encryption with customer-controlled keys
	// For now, just redact
	return String(field.Key, "***REDACTED***")
}

// processPII hashes PII data for compliance
func (z *zZlog) processPII(field Field) Field {
	value := fmt.Sprintf("%v", field.Value)
	hash := sha256.Sum256([]byte(value))
	hashedValue := fmt.Sprintf("sha256:%x", hash[:8]) // First 8 bytes for logs
	
	return String(field.Key+"_hash", hashedValue)
}

// processMetric converts metrics to structured log fields
func (z *zZlog) processMetric(field Field) Field {
	// Add metric suffix to distinguish from regular fields
	metricKey := field.Key
	if !strings.HasSuffix(metricKey, "_ms") && !strings.HasSuffix(metricKey, "_count") {
		// Try to infer metric type from value
		switch field.Value.(type) {
		case int, int64, float64:
			// Assume duration/count metric
			metricKey += "_value"
		}
	}
	
	// Convert to appropriate type
	switch v := field.Value.(type) {
	case int:
		return Int(metricKey, v)
	case int64:
		return Int64(metricKey, v)
	case float64:
		return Float64(metricKey, v)
	default:
		return String(metricKey, fmt.Sprintf("%v", v))
	}
}

// processCorrelation extracts trace/request context
func (z *zZlog) processCorrelation(field Field) []Field {
	var fields []Field
	
	switch ctx := field.Value.(type) {
	case context.Context:
		// Extract from Go context (OTEL)
		fields = append(fields, z.extractFromGoContext(ctx)...)
		
	default:
		// Try to extract from other context types (gin.Context, etc.)
		fields = append(fields, z.extractFromGenericContext(ctx)...)
	}
	
	return fields
}

// extractFromGoContext extracts trace info from context.Context
func (z *zZlog) extractFromGoContext(ctx context.Context) []Field {
	var fields []Field
	
	// TODO: Add proper OTEL integration
	// This is a placeholder that would integrate with:
	// - go.opentelemetry.io/otel/trace
	// - Extract TraceID, SpanID from span context
	
	// For now, just add a placeholder
	if ctx != nil {
		fields = append(fields, String("context_type", "go_context"))
	}
	
	return fields
}

// extractFromGenericContext extracts from framework contexts (gin, etc.)
func (z *zZlog) extractFromGenericContext(ctx any) []Field {
	var fields []Field
	
	// TODO: Add support for:
	// - *gin.Context: Extract request ID, trace context
	// - *fiber.Ctx: Fiber framework context
	// - *echo.Context: Echo framework context
	
	// Placeholder implementation
	if ctx != nil {
		fields = append(fields, String("context_type", "generic"))
	}
	
	return fields
}