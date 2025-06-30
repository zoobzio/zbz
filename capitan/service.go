package capitan

import (
	"sync"
)

// ServiceManager provides byte-based event processing with no reflection
type ServiceManager struct {
	mu       sync.RWMutex
	handlers map[string][]ByteHandler // Just interfaces that take []byte
	stats    HookStats
}

// Global service manager instance - only deals with bytes
var serviceManager = &ServiceManager{
	handlers: make(map[string][]ByteHandler),
	stats:    HookStats{HookTypes: make(map[string]int)},
}

// register adds a concrete hook to the service layer
func (s *ServiceManager) register(hookType string, handler ByteHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.handlers[hookType] = append(s.handlers[hookType], handler)
	s.stats.HookTypes[hookType]++
	s.stats.TotalHandlers++
}

// emitBytes sends bytes to all registered handlers for a hook type
func (s *ServiceManager) emitBytes(hookType string, eventBytes []byte) error {
	s.mu.RLock()
	handlers := make([]ByteHandler, len(s.handlers[hookType]))
	copy(handlers, s.handlers[hookType])
	s.mu.RUnlock()

	// Execute all handlers for this hook type
	for _, handler := range handlers {
		if err := handler.Handle(eventBytes); err != nil {
			// Silent failure to avoid circular logging
			_ = err
		}
	}

	return nil
}

// getStats returns current statistics about registered hooks
func (s *ServiceManager) getStats() HookStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Return a copy to avoid race conditions
	stats := HookStats{
		HookTypes:     make(map[string]int),
		TotalHandlers: s.stats.TotalHandlers,
	}

	for hookType, count := range s.stats.HookTypes {
		stats.HookTypes[hookType] = count
	}

	return stats
}

// reset clears all handlers - useful for testing
func (s *ServiceManager) reset() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.handlers = make(map[string][]ByteHandler)
	s.stats = HookStats{HookTypes: make(map[string]int)}
}

// HookStats provides information about registered hooks
type HookStats struct {
	HookTypes     map[string]int `json:"hook_types"`
	TotalHandlers int            `json:"total_handlers"`
}