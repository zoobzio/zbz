package capitan

import "encoding/json"

// EmitEvent is a simple wrapper for emitting events
func EmitEvent(eventType string, data map[string]any) {
	// Convert to JSON bytes and emit
	if eventBytes, err := json.Marshal(data); err == nil {
		serviceManager.emitBytes(eventType, eventBytes)
	}
}

// RegisterByteHandler registers a simple byte handler for an event type
func RegisterByteHandler(eventType string, handler func([]byte) error) {
	bh := &simpleByteHandler{fn: handler}
	serviceManager.register(eventType, bh)
}

// simpleByteHandler wraps a function to implement ByteHandler
type simpleByteHandler struct {
	fn func([]byte) error
}

func (s *simpleByteHandler) Handle(data []byte) error {
	return s.fn(data)
}