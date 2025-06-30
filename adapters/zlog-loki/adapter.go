package zlog_loki

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
	"zbz/capitan"
	"zbz/zlog"
)

// LokiAdapter ships logs to Grafana Loki
type LokiAdapter struct {
	url    string
	labels map[string]string
	client *http.Client
}

// PipeToLoki enables shipping all log events to Loki
func PipeToLoki(url string, labels map[string]string) *LokiAdapter {
	adapter := &LokiAdapter{
		url:    url,
		labels: labels,
		client: &http.Client{Timeout: 5 * time.Second},
	}
	
	// Register handler for log events
	capitan.OnLogEvent(adapter.handleLogEvent)
	
	return adapter
}

// handleLogEvent processes log events and ships to Loki
func (a *LokiAdapter) handleLogEvent(event zlog.LogEvent) {
	// Build Loki stream
	stream := map[string]string{}
	
	// Add configured labels
	for k, v := range a.labels {
		stream[k] = v
	}
	
	// Add level as label
	stream["level"] = event.Level
	
	// Extract routing fields as labels for intelligent routing
	for _, field := range event.Fields {
		switch field.Type {
		case zlog.LayerType:
			stream["layer"] = field.Value.(string)
		case zlog.ConcernType:
			stream["concern"] = field.Value.(string)
		case zlog.UserScopeType:
			stream["user_scope"] = field.Value.(string)
		case zlog.PrivacyType:
			stream["privacy"] = field.Value.(string)
		}
	}
	
	// Build log line with fields
	logLine := event.Message
	for _, field := range event.Fields {
		// Skip routing fields (already in labels)
		if isRoutingField(field) {
			continue
		}
		logLine += fmt.Sprintf(" %s=%v", field.Key, field.Value)
	}
	
	// Create Loki push request
	lokiRequest := map[string]any{
		"streams": []map[string]any{
			{
				"stream": stream,
				"values": [][]string{
					{fmt.Sprintf("%d", event.Timestamp.UnixNano()), logLine},
				},
			},
		},
	}
	
	// Send to Loki
	go a.sendToLoki(lokiRequest)
}

// sendToLoki sends the request to Loki (async)
func (a *LokiAdapter) sendToLoki(request map[string]any) {
	data, err := json.Marshal(request)
	if err != nil {
		return
	}
	
	resp, err := a.client.Post(a.url+"/loki/api/v1/push", "application/json", bytes.NewBuffer(data))
	if err != nil {
		return
	}
	defer resp.Body.Close()
}

// isRoutingField checks if field is used for routing
func isRoutingField(field zlog.Field) bool {
	switch field.Type {
	case zlog.LayerType, zlog.ConcernType, zlog.UserScopeType, 
		 zlog.TenantScopeType, zlog.RouteType, zlog.PrivacyType:
		return true
	default:
		return false
	}
}