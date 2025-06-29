package capitan

import (
	"context"
	"time"

	"zbz/cereal"
)

// Public API for capitan hook system - uses concrete hooks, no reflection\n\n// EmitEvent is a simple wrapper for emitting events without type constraints\nfunc EmitEvent(eventType string, data map[string]any) {\n\t// Simple event emission for logging/monitoring\n\t// Uses string hook type and empty context for convenience\n\tctx := context.Background()\n\tEmit(ctx, StringHookType(eventType), \"rocco\", data, data)\n}\n\n// StringHookType allows using string as hook type\ntype StringHookType string\n\nfunc (s StringHookType) String() string {\n\treturn string(s)\n}"

// RegisterInput registers a typed input handler for a specific hook type
func RegisterInput[T any, H HookType](hookType H, handler InputHookFunc[T]) {
	concrete := &ConcreteInputHook[T]{
		hookType: hookType.String(),
		handler:  handler,
	}
	serviceManager.register(hookType.String(), concrete)
}

// RegisterOutput registers a typed output handler for a specific hook type
func RegisterOutput[T any, H HookType](hookType H, handler OutputHookFunc[T]) {
	concrete := &ConcreteOutputHook[T]{
		hookType: hookType.String(),
		handler:  handler,
	}
	serviceManager.register(hookType.String(), concrete)
}

// RegisterTransform registers a typed transform handler
func RegisterTransform[TIn, TOut any, HIn, HOut HookType](inputType HIn, outputType HOut, handler TransformHookFunc[TIn, TOut]) {
	concrete := &ConcreteTransformHook[TIn, TOut]{
		hookType:    inputType.String(),
		outputType:  outputType.String(),
		transformer: handler,
	}
	serviceManager.register(inputType.String(), concrete)
}

// Emit sends a typed event to all registered handlers
func Emit[T any, H HookType](ctx context.Context, hookType H, source string, data T, metadata map[string]any) error {
	// Create event structure
	event := TypedEvent[T]{
		Type:      hookType.String(),
		Source:    source,
		Timestamp: time.Now(),
		Data:      data,
		Context:   ctx,
		Metadata:  metadata,
	}

	// Serialize once at the boundary
	eventBytes, err := cereal.JSON.Marshal(event)
	if err != nil {
		return err
	}

	// Service layer gets pure bytes
	return serviceManager.emitBytes(hookType.String(), eventBytes)
}

// GetStats returns basic statistics about the hook system
func GetStats() HookStats {
	return serviceManager.getStats()
}

// Reset clears all handlers - useful for testing
func Reset() {
	serviceManager.reset()
}