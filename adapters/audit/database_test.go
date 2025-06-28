package audit

import (
	"context"
	"testing"
	"time"

	"zbz/capitan"
	"zbz/database"
)

func TestDatabaseAuditAdapter(t *testing.T) {
	// For this simple test, we'll just verify the adapter connects without error
	// In a real environment, this would connect to actual audit systems
	
	err := ConnectDatabaseToAudit()
	if err != nil {
		t.Fatalf("Failed to connect database to audit: %v", err)
	}
	
	// Simulate a database event to make sure handlers are registered
	ctx := context.Background()
	
	createData := database.RecordCreatedData{
		TableName: "users",
		RecordID:  "123",
		Data:      map[string]any{"name": "John", "email": "john@example.com"},
	}
	
	// This should not error if the handler is properly registered
	err = capitan.Emit(ctx, database.RecordCreated, "database-service", createData, nil)
	if err != nil {
		t.Fatalf("Failed to emit create event: %v", err)
	}
	
	// Verify the capitan system is tracking our handlers
	stats := capitan.GetStats()
	
	// We should have at least some handlers registered
	if stats.TotalHandlers == 0 {
		t.Error("Expected some handlers to be registered")
	}
	
	// Check that database hook types are present
	foundDBHooks := false
	for hookType := range stats.HookTypes {
		if len(hookType) > 8 && hookType[:8] == "database" {
			foundDBHooks = true
			break
		}
	}
	
	if !foundDBHooks {
		t.Error("Expected database hook types to be registered")
	}
}

func TestAuditEventStructure(t *testing.T) {
	// Test that audit events have correct structure
	auditEvent := AuditEvent{
		Action:    "CREATE",
		Table:     "test_table",
		RecordID:  "test_id",
		Timestamp: time.Now(),
		Changes:   map[string]any{"field": "value"},
	}
	
	if auditEvent.Action != "CREATE" {
		t.Errorf("Expected action 'CREATE', got '%s'", auditEvent.Action)
	}
	
	if auditEvent.Table != "test_table" {
		t.Errorf("Expected table 'test_table', got '%s'", auditEvent.Table)
	}
	
	if auditEvent.RecordID != "test_id" {
		t.Errorf("Expected record ID 'test_id', got '%v'", auditEvent.RecordID)
	}
	
	if auditEvent.Changes == nil {
		t.Error("Expected changes to be non-nil")
	}
}