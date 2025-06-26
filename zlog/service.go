package zlog

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"
	
	"zbz/cereal"
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

// ZlogConfig defines provider-agnostic zlog configuration
type ZlogConfig struct {
	// Service configuration
	Name        string `yaml:"name" json:"name"`
	Level       string `yaml:"level" json:"level"`             // "debug", "info", "warn", "error", "fatal"
	Format      string `yaml:"format" json:"format"`           // "json", "console", "text"
	Development bool   `yaml:"development" json:"development"` // Enable development mode
	
	// Output configuration
	Outputs     []OutputConfig  `yaml:"outputs,omitempty" json:"outputs,omitempty"`
	OutputFile  string          `yaml:"output_file,omitempty" json:"output_file,omitempty"`
	
	// Performance settings
	BufferSize  int    `yaml:"buffer_size,omitempty" json:"buffer_size,omitempty"`
	FlushLevel  string `yaml:"flush_level,omitempty" json:"flush_level,omitempty"`
	
	// Sampling configuration (for high-volume scenarios)
	Sampling    *SamplingConfig `yaml:"sampling,omitempty" json:"sampling,omitempty"`
	
	// Provider-specific sections (each provider uses what it needs)
	Extensions  map[string]interface{} `yaml:"extensions,omitempty" json:"extensions,omitempty"`
}

// OutputConfig defines configuration for a single log output destination
type OutputConfig struct {
	Type    string `yaml:"type" json:"type"`                     // "console", "file", "syslog"
	Level   string `yaml:"level,omitempty" json:"level,omitempty"` // Override global level
	Format  string `yaml:"format,omitempty" json:"format,omitempty"` // Override global format
	Target  string `yaml:"target,omitempty" json:"target,omitempty"` // File path, syslog tag, etc.
	Options map[string]interface{} `yaml:"options,omitempty" json:"options,omitempty"` // Output-specific options
}

// SamplingConfig configures log sampling for high-volume scenarios
type SamplingConfig struct {
	Initial    int `yaml:"initial,omitempty" json:"initial,omitempty"`       // Sample first N messages per second
	Thereafter int `yaml:"thereafter,omitempty" json:"thereafter,omitempty"` // Then 1 in N thereafter per second
}

// DefaultConfig returns sensible defaults for zlog configuration
func DefaultConfig() ZlogConfig {
	return ZlogConfig{
		Name:        "app",
		Level:       "info",
		Format:      "json",
		Development: false,
		BufferSize:  1024,
		FlushLevel:  "error",
	}
}

// zZlog is the singleton service layer that orchestrates logging operations
// Like cache/hodor singletons, this manages provider abstraction + cereal serialization
type zZlog struct {
	provider     ZlogProvider          // Backend provider wrapper
	serializer   cereal.CerealProvider // Cereal handles complex field serialization
	config       ZlogConfig            // Service configuration
	contractName string                // Name of the contract that created this singleton
}

// configureFromContract initializes the singleton from a contract's registration
func configureFromContract(contractName string, provider ZlogProvider, config ZlogConfig) error {
	// Check if we need to replace existing singleton
	if zlog != nil && zlog.contractName != contractName {
		// Close old provider
		if err := zlog.provider.Close(); err != nil {
			// Log warning but continue (could use fmt.Printf for bootstrap logging)
		}
	} else if zlog != nil && zlog.contractName == contractName {
		// Same contract, no need to replace
		return nil
	}

	// Set up cereal serialization for complex field processing
	cerealConfig := cereal.DefaultConfig()
	cerealConfig.Name = "zlog-serializer"
	cerealConfig.DefaultFormat = "json" // Use JSON for structured log data
	cerealConfig.EnableCaching = true   // Cache field serialization for performance
	cerealConfig.EnableScoping = true   // Enable scoped logging for sensitive data
	
	// Create JSON provider for log field serialization
	cerealContract := cereal.NewJSONProvider(cerealConfig)
	cerealProvider := cerealContract.Provider()

	// Create service singleton
	zlog = &zZlog{
		provider:     provider,
		serializer:   cerealProvider, // Cereal handles complex field serialization
		config:       config,
		contractName: contractName,
	}

	return nil
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
			
		case AnyType:
			// Use cereal to serialize complex data structures
			processed = append(processed, z.processComplexField(field))
			
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

// processComplexField serializes complex data structures using cereal
func (z *zZlog) processComplexField(field Field) Field {
	// Use cereal to serialize complex objects to JSON strings
	serialized, err := z.serializer.Marshal(field.Value)
	if err != nil {
		// If serialization fails, fall back to string representation
		return String(field.Key+"_error", fmt.Sprintf("serialization failed: %v (value: %v)", err, field.Value))
	}
	
	// Return as string field containing JSON
	return String(field.Key, string(serialized))
}

// processComplexFieldScoped serializes complex data with field-level scoping
func (z *zZlog) processComplexFieldScoped(field Field, userPermissions []string) Field {
	// Use cereal scoped serialization to filter sensitive fields in log data
	serialized, err := z.serializer.MarshalScoped(field.Value, userPermissions)
	if err != nil {
		// If scoped serialization fails, fall back to regular serialization
		return z.processComplexField(field)
	}
	
	// Return as string field containing scoped JSON
	return String(field.Key+"_scoped", string(serialized))
}