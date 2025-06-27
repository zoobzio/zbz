package zlog

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
)

// Private concrete logger service layer
var zlog *zZlog

// zZlog is the singleton service layer that orchestrates logging operations
// Like cache/depot singletons, this manages provider abstraction + piping + field processing
type zZlog struct {
	config   Config   // Service configuration
	provider Provider // Backend provider wrapper
	output   *writer  // Pipeline writer that handles output to multiple destinations
	
	// Plugin system
	processors map[FieldType][]Processor // Type-specific processors
	mu         sync.RWMutex              // Protect concurrent access
}

// Register a new config/provider
func (z *zZlog) Register(config Config, provider Provider) {
	z.config = config
	z.provider = provider
}

// Process registers a field processor for a specific field type
func (z *zZlog) Process(fieldType FieldType, processor Processor) {
	z.mu.Lock()
	defer z.mu.Unlock()
	
	if z.processors == nil {
		z.processors = make(map[FieldType][]Processor)
	}
	
	z.processors[fieldType] = append(z.processors[fieldType], processor)
}

// ProcessFields runs fields through the preprocessing pipeline
func (z *zZlog) ProcessFields(fields []Field) []Field {
	var processed []Field

	for _, field := range fields {
		// First check for registered processors
		z.mu.RLock()
		processors, hasProcessors := z.processors[field.Type]
		z.mu.RUnlock()
		
		if hasProcessors && len(processors) > 0 {
			// Run all registered processors for this type
			fieldResults := []Field{field}
			for _, processor := range processors {
				var nextResults []Field
				for _, f := range fieldResults {
					nextResults = append(nextResults, processor(f)...)
				}
				fieldResults = nextResults
			}
			processed = append(processed, fieldResults...)
			continue
		}
		
		// Fall back to built-in processors
		switch field.Type {
		case SecretType:
			// Built-in: Redact sensitive data (no encryption without keys)
			processed = append(processed, z.processSecret(field))

		case PIIType:
			// Built-in: Hash PII for compliance
			processed = append(processed, z.processPII(field))

		case MetricType:
			// Built-in: Convert metrics to structured fields
			processed = append(processed, z.processMetric(field))

		case CorrelationType:
			// Built-in: Extract correlation IDs from context
			correlationFields := z.processCorrelation(field)
			processed = append(processed, correlationFields...)

		case AnyType:
			// Pass complex types directly to the provider
			processed = append(processed, field)

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

// Service methods for output piping

// Pipe adds an io.Writer to receive copies of all log output
func (z *zZlog) Pipe(w io.Writer) {
	z.output.mu.Lock()
	defer z.output.mu.Unlock()
	z.output.writers = append(z.output.writers, w)
}

// ClearAllPipes removes all registered pipe writers
func (z *zZlog) ClearAllPipes() {
	z.output.mu.Lock()
	defer z.output.mu.Unlock()
	z.output.writers = z.output.writers[:0]
}

// Writer returns the writer for providers to use
func (z *zZlog) Writer() io.Writer {
	return z.output
}

// SetOriginalOutput changes where the original output goes (default: stdout)
func (z *zZlog) SetOriginalOutput(w io.Writer) {
	z.output.mu.Lock()
	defer z.output.mu.Unlock()
	z.output.original = w
}

// init sets up a default simple logger
func init() {
	// Create a simple console logger as the default
	// This will be replaced when any provider function is called
	provider := newSimpleProvider()
	output := &writer{
		original: os.Stdout,
		writers:  []io.Writer{},
	}
	zlog = &zZlog{
		config:     DefaultConfig(),
		provider:   provider,
		output:     output,
		processors: make(map[FieldType][]Processor),
	}
}
