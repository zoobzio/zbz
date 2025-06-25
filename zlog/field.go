package zlog

import (
	"time"
)

// Field represents a typed key-value pair for structured logging
type Field struct {
	Key   string
	Type  FieldType
	Value any
}

// FieldType defines the type of a log field for type safety
type FieldType int

const (
	// Regular field types
	StringType FieldType = iota
	IntType
	Int64Type
	Float64Type
	BoolType
	ErrorType
	DurationType
	TimeType
	ByteStringType
	AnyType
	StringsType
	
	// Special field types (processed by pipeline before driver)
	CallDepthType FieldType = 100 + iota // Adjust call stack depth
	SecretType                            // Encrypt sensitive data
	PIIType                               // Redact/hash PII based on compliance
	MetricType                            // Inject performance metrics
	CorrelationType                       // Propagate correlation IDs
)

// Type-safe field constructors (zap-like API)
func String(key, value string) Field {
	return Field{Key: key, Type: StringType, Value: value}
}

func Int(key string, value int) Field {
	return Field{Key: key, Type: IntType, Value: value}
}

func Int64(key string, value int64) Field {
	return Field{Key: key, Type: Int64Type, Value: value}
}

func Float64(key string, value float64) Field {
	return Field{Key: key, Type: Float64Type, Value: value}
}

func Bool(key string, value bool) Field {
	return Field{Key: key, Type: BoolType, Value: value}
}

func Err(err error) Field {
	return Field{Key: "error", Type: ErrorType, Value: err}
}

func Duration(key string, value time.Duration) Field {
	return Field{Key: key, Type: DurationType, Value: value}
}

func Time(key string, value time.Time) Field {
	return Field{Key: key, Type: TimeType, Value: value}
}

func ByteString(key string, value []byte) Field {
	return Field{Key: key, Type: ByteStringType, Value: string(value)}
}

func Any(key string, value any) Field {
	return Field{Key: key, Type: AnyType, Value: value}
}

func Strings(key string, value []string) Field {
	return Field{Key: key, Type: StringsType, Value: value}
}

// Special field constructors for pipeline processing

// CallDepth adds extra skip levels to the call stack for accurate file/line reporting
func CallDepth(depth int) Field {
	return Field{Key: "_calldepth", Type: CallDepthType, Value: depth}
}

// Secret marks sensitive data for encryption in logs
func Secret(key, value string) Field {
	return Field{Key: key, Type: SecretType, Value: value}
}

// PII marks personally identifiable information for redaction/hashing
func PII(key, value string) Field {
	return Field{Key: key, Type: PIIType, Value: value}
}

// Metric injects performance metrics into logs
func Metric(key string, value any) Field {
	return Field{Key: key, Type: MetricType, Value: value}
}

// Correlation extracts trace/request context for distributed tracing
func Correlation(ctx any) Field {
	return Field{Key: "_correlation", Type: CorrelationType, Value: ctx}
}
