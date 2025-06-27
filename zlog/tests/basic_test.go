package tests

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"zbz/zlog"
)

func TestBasicLogging(t *testing.T) {
	// Capture log output
	var buf bytes.Buffer
	zlog.Pipe(&buf)
	defer zlog.Clear()

	// Test basic logging
	zlog.Info("Test message",
		zlog.String("key", "value"),
		zlog.Int("number", 42))

	// Give async pipe handlers time to write
	time.Sleep(50 * time.Millisecond)

	output := buf.String()
	if !strings.Contains(output, "Test message") {
		t.Errorf("Expected 'Test message' in output, got: %s", output)
	}
	if !strings.Contains(output, "value") {
		t.Errorf("Expected 'value' in output, got: %s", output)
	}
}

func TestFieldProcessors(t *testing.T) {
	// Register a test processor
	processed := false
	zlog.Process(zlog.StringType, func(field zlog.Field) []zlog.Field {
		if field.Key == "test_field" {
			processed = true
			return []zlog.Field{zlog.String(field.Key, "PROCESSED_"+field.Value.(string))}
		}
		return []zlog.Field{field}
	})

	// Capture log output
	var buf bytes.Buffer
	zlog.Pipe(&buf)
	defer zlog.Clear()

	// Log with the test field
	zlog.Info("Processing test", zlog.String("test_field", "original"))

	// Give async pipe handlers time to write
	time.Sleep(50 * time.Millisecond)

	if !processed {
		t.Error("Processor was not called")
	}

	output := buf.String()
	if !strings.Contains(output, "PROCESSED_original") {
		t.Errorf("Expected 'PROCESSED_original' in output, got: %s", output)
	}
}

func TestMultiplePipes(t *testing.T) {
	// Set up multiple output destinations
	var buf1, buf2 bytes.Buffer
	zlog.Pipe(&buf1)
	zlog.Pipe(&buf2)
	defer zlog.Clear()

	// Log a message
	zlog.Info("Multi-pipe test", zlog.String("destination", "both"))

	// Give async pipe handlers time to write
	time.Sleep(50 * time.Millisecond)

	// Both buffers should contain the message
	output1 := buf1.String()
	output2 := buf2.String()

	if !strings.Contains(output1, "Multi-pipe test") {
		t.Errorf("Buffer 1 missing message: %s", output1)
	}
	if !strings.Contains(output2, "Multi-pipe test") {
		t.Errorf("Buffer 2 missing message: %s", output2)
	}
}

func TestAllFieldTypes(t *testing.T) {
	var buf bytes.Buffer
	zlog.Pipe(&buf)
	defer zlog.Clear()

	// Test all field types
	zlog.Info("Field types test",
		zlog.String("string", "test"),
		zlog.Int("int", 123),
		zlog.Bool("bool", true),
		zlog.Float64("float", 3.14),
		zlog.Any("any", map[string]string{"key": "value"}))

	// Give async pipe handlers time to write
	time.Sleep(50 * time.Millisecond)

	output := buf.String()
	
	// Check that all values appear in output
	expectedValues := []string{"test", "123", "true", "3.14", "value"}
	for _, expected := range expectedValues {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected '%s' in output, got: %s", expected, output)
		}
	}
}