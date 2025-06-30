package capitan

import (
	"encoding/json"
	"zbz/zlog"
)

// zlogEventSink implements zlog.EventSink to integrate with Capitan
type zlogEventSink struct{}

func (s *zlogEventSink) EmitLogEvent(event zlog.LogEvent) {
	// Convert zlog event to Capitan event data
	eventData := map[string]any{
		"level":     event.Level,
		"message":   event.Message,
		"fields":    event.Fields,
		"timestamp": event.Timestamp,
	}
	
	// Emit via Capitan simple API
	EmitEvent("LogEntryCreated", eventData)
}

// Auto-hydration: when Capitan is imported, zlog gets events
func init() {
	// Auto-connect zlog to Capitan event system
	zlog.SetEventSink(&zlogEventSink{})
}

// On registers a handler for log events (convenience function)
func OnLogEvent(handler func(event zlog.LogEvent)) {
	RegisterByteHandler("LogEntryCreated", func(data []byte) error {
		var eventData map[string]any
		if err := json.Unmarshal(data, &eventData); err != nil {
			return err
		}
		
		// Convert back to zlog.LogEvent
		event := zlog.LogEvent{
			Level:   eventData["level"].(string),
			Message: eventData["message"].(string),
			// Fields would need type conversion - simplified for now
		}
		
		handler(event)
		return nil
	})
}