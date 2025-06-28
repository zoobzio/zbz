package universal

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"zbz/capitan"
)

// Instrument wraps a native client with automatic hook emission
// This ensures that even native provider operations emit capitan hooks
func Instrument(client any, providerType string) any {
	return newInstrumentedProxy(client, providerType)
}

// InstrumentedProxy wraps any client and automatically emits hooks for method calls
type InstrumentedProxy struct {
	client       any            // The original client
	clientValue  reflect.Value  // Reflect value of client
	clientType   reflect.Type   // Reflect type of client
	providerType string         // Provider type for hook context
}

// newInstrumentedProxy creates a new instrumented proxy for any client
func newInstrumentedProxy(client any, providerType string) *InstrumentedProxy {
	return &InstrumentedProxy{
		client:       client,
		clientValue:  reflect.ValueOf(client),
		clientType:   reflect.TypeOf(client),
		providerType: providerType,
	}
}

// GetNative returns the original uninstrumented client
func (p *InstrumentedProxy) GetNative() any {
	return p.client
}

// GetProvider returns the provider type
func (p *InstrumentedProxy) GetProvider() string {
	return p.providerType
}

// Method interception using reflection
// This dynamically implements any method that exists on the wrapped client

// MethodMissing handles all method calls via reflection
func (p *InstrumentedProxy) MethodMissing(methodName string, args []reflect.Value) []reflect.Value {
	start := time.Now()
	
	// Find method on client
	method := p.clientValue.MethodByName(methodName)
	if !method.IsValid() {
		// Method doesn't exist - return error
		errorType := reflect.TypeOf((*error)(nil)).Elem()
		errorValue := reflect.New(errorType).Elem()
		errorValue.Set(reflect.ValueOf(fmt.Errorf("method %s not found on %s", methodName, p.clientType)))
		return []reflect.Value{errorValue}
	}
	
	// Emit pre-operation hook
	p.emitPreHook(methodName, args)
	
	// Call original method
	results := method.Call(args)
	
	// Emit post-operation hook
	p.emitPostHook(methodName, args, results, time.Since(start))
	
	return results
}

// Hook emission methods

func (p *InstrumentedProxy) emitPreHook(methodName string, args []reflect.Value) {
	// Extract context if first arg is context.Context
	ctx := context.Background()
	if len(args) > 0 && args[0].Type().Implements(reflect.TypeOf((*context.Context)(nil)).Elem()) {
		if contextValue, ok := args[0].Interface().(context.Context); ok {
			ctx = contextValue
		}
	}
	
	// Convert args to interface slice
	argValues := make([]any, len(args))
	for i, arg := range args {
		argValues[i] = arg.Interface()
	}
	
	// Emit hook
	capitan.Emit(ctx, NativeMethodStarted, p.providerType, NativeMethodData{
		Method:       methodName,
		Provider:     p.providerType,
		Arguments:    argValues,
		Phase:        "started",
		Timestamp:    time.Now(),
	}, nil)
}

func (p *InstrumentedProxy) emitPostHook(methodName string, args []reflect.Value, results []reflect.Value, duration time.Duration) {
	// Extract context if first arg is context.Context
	ctx := context.Background()
	if len(args) > 0 && args[0].Type().Implements(reflect.TypeOf((*context.Context)(nil)).Elem()) {
		if contextValue, ok := args[0].Interface().(context.Context); ok {
			ctx = contextValue
		}
	}
	
	// Convert results to interface slice
	resultValues := make([]any, len(results))
	var errorResult error
	
	for i, result := range results {
		resultValues[i] = result.Interface()
		
		// Check if this result is an error
		if result.Type().Implements(reflect.TypeOf((*error)(nil)).Elem()) {
			if err, ok := result.Interface().(error); ok && err != nil {
				errorResult = err
			}
		}
	}
	
	// Convert args to interface slice
	argValues := make([]any, len(args))
	for i, arg := range args {
		argValues[i] = arg.Interface()
	}
	
	// Emit hook
	capitan.Emit(ctx, NativeMethodCompleted, p.providerType, NativeMethodData{
		Method:       methodName,
		Provider:     p.providerType,
		Arguments:    argValues,
		Results:      resultValues,
		Error:        errorResult,
		Duration:     duration,
		Phase:        "completed",
		Timestamp:    time.Now(),
	}, nil)
}

// Dynamic method implementation using reflection
// This allows the proxy to respond to any method call

func (p *InstrumentedProxy) methodImplementation(methodName string) func([]reflect.Value) []reflect.Value {
	return func(args []reflect.Value) []reflect.Value {
		return p.MethodMissing(methodName, args)
	}
}

// ProxyBuilder creates dynamic proxies for different client types
type ProxyBuilder struct {
	providerType string
}

// NewProxyBuilder creates a new proxy builder
func NewProxyBuilder(providerType string) *ProxyBuilder {
	return &ProxyBuilder{
		providerType: providerType,
	}
}

// BuildProxy creates a proxy that implements the same interface as the client
func (pb *ProxyBuilder) BuildProxy(client any) any {
	clientType := reflect.TypeOf(client)
	clientValue := reflect.ValueOf(client)
	
	// If client is a pointer to struct, get the struct type
	if clientType.Kind() == reflect.Ptr {
		clientType = clientType.Elem()
	}
	
	// Create proxy struct that embeds the original client
	proxyFields := []reflect.StructField{
		{
			Name: "Client",
			Type: reflect.TypeOf(client),
			Tag:  `proxy:"original"`,
		},
		{
			Name: "ProxyType",
			Type: reflect.TypeOf(""),
			Tag:  `proxy:"type"`,
		},
	}
	
	proxyType := reflect.StructOf(proxyFields)
	proxyValue := reflect.New(proxyType).Elem()
	
	// Set the embedded client
	proxyValue.FieldByName("Client").Set(clientValue)
	proxyValue.FieldByName("ProxyType").SetString(pb.providerType)
	
	return proxyValue.Addr().Interface()
}

// Common hook types for instrumented operations
const (
	NativeMethodStarted   UniversalHookType = iota + 3100
	NativeMethodCompleted
	NativeMethodError
)

// NativeMethodData contains data for native method call hooks
type NativeMethodData struct {
	Method       string        `json:"method"`        // Method name that was called
	Provider     string        `json:"provider"`      // Provider type
	Arguments    []any         `json:"arguments"`     // Method arguments
	Results      []any         `json:"results,omitempty"` // Method results
	Error        error         `json:"error,omitempty"`   // Error if method failed
	Duration     time.Duration `json:"duration"`      // Method execution duration
	Phase        string        `json:"phase"`         // "started" or "completed"
	Timestamp    time.Time     `json:"timestamp"`     // When method was called
}

// Simplified instrumentation for common patterns

// InstrumentDatabase wraps database clients with automatic query logging
func InstrumentDatabase(db any) any {
	return Instrument(db, "database")
}

// InstrumentCache wraps cache clients with automatic operation logging
func InstrumentCache(cache any) any {
	return Instrument(cache, "cache")
}

// InstrumentStorage wraps storage clients with automatic file operation logging
func InstrumentStorage(storage any) any {
	return Instrument(storage, "storage")
}

// InstrumentSearch wraps search clients with automatic query logging
func InstrumentSearch(search any) any {
	return Instrument(search, "search")
}

// Helper for providers to easily create instrumented clients

// ProviderInstrumentation provides helpers for provider authors
type ProviderInstrumentation struct {
	providerType string
	hookEmitter  HookEmitter
}

// NewProviderInstrumentation creates instrumentation helpers for a provider
func NewProviderInstrumentation(providerType string, hookEmitter HookEmitter) *ProviderInstrumentation {
	return &ProviderInstrumentation{
		providerType: providerType,
		hookEmitter:  hookEmitter,
	}
}

// WrapClient wraps a native client with instrumentation
func (pi *ProviderInstrumentation) WrapClient(client any) any {
	return Instrument(client, pi.providerType)
}

// EmitOperation emits a data operation hook
func (pi *ProviderInstrumentation) EmitOperation(ctx context.Context, operation, uri string, duration time.Duration, err error) {
	errorMsg := ""
	if err != nil {
		errorMsg = err.Error()
	}
	
	pi.hookEmitter.Emit(ctx, DataGet, pi.providerType, DataOperationData{
		Operation: operation,
		URI:       uri,
		Provider:  pi.providerType,
		Duration:  duration,
		Error:     errorMsg,
		Timestamp: time.Now(),
	}, nil)
}

// EmitMethodCall emits a native method call hook
func (pi *ProviderInstrumentation) EmitMethodCall(ctx context.Context, method string, args []any, duration time.Duration, err error) {
	pi.hookEmitter.Emit(ctx, NativeMethodCompleted, pi.providerType, NativeMethodData{
		Method:    method,
		Provider:  pi.providerType,
		Arguments: args,
		Error:     err,
		Duration:  duration,
		Phase:     "completed",
		Timestamp: time.Now(),
	}, nil)
}