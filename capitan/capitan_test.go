package capitan

import (
	"context"
	"testing"
	"time"
)

// Test hook types
type TestHookType int

const (
	TestEvent TestHookType = iota
	AnotherEvent
)

func (t TestHookType) String() string {
	switch t {
	case TestEvent:
		return "test.event"
	case AnotherEvent:
		return "test.another"
	default:
		return "test.unknown"
	}
}

// Test data types
type TestData struct {
	Message string `json:"message"`
	Count   int    `json:"count"`
}

type AnotherData struct {
	Name  string `json:"name"`
	Value bool   `json:"value"`
}

func TestBasicHookRegistrationAndEmission(t *testing.T) {
	// Reset for clean test
	Reset()

	// Track received events
	var receivedInput TestData
	var receivedOutput TestData

	// Register input handler
	RegisterInput[TestData](TestEvent, func(data TestData) error {
		receivedInput = data
		return nil
	})

	// Register output handler
	RegisterOutput[TestData](TestEvent, func(data TestData) error {
		receivedOutput = data
		return nil
	})

	// Emit test event
	testData := TestData{
		Message: "Hello World",
		Count:   42,
	}

	err := Emit(context.Background(), TestEvent, "test-source", testData, nil)
	if err != nil {
		t.Fatalf("Failed to emit event: %v", err)
	}

	// Give handlers time to process
	time.Sleep(10 * time.Millisecond)

	// Verify input handler received event
	if receivedInput.Message != "Hello World" {
		t.Errorf("Expected input message 'Hello World', got '%s'", receivedInput.Message)
	}
	if receivedInput.Count != 42 {
		t.Errorf("Expected input count 42, got %d", receivedInput.Count)
	}

	// Verify output handler received event
	if receivedOutput.Message != "Hello World" {
		t.Errorf("Expected output message 'Hello World', got '%s'", receivedOutput.Message)
	}
	if receivedOutput.Count != 42 {
		t.Errorf("Expected output count 42, got %d", receivedOutput.Count)
	}
}

func TestTransformHandler(t *testing.T) {
	// Reset for clean test
	Reset()

	var transformedOutput AnotherData

	// Register transform handler  
	RegisterTransform[TestData, AnotherData](TestEvent, AnotherEvent, func(input TestData) (AnotherData, error) {
		// Transform TestData to AnotherData
		return AnotherData{
			Name:  input.Message,
			Value: input.Count > 0,
		}, nil
	})

	// Register output handler for transformed data
	RegisterOutput[AnotherData](AnotherEvent, func(data AnotherData) error {
		transformedOutput = data
		return nil
	})

	// Emit original event
	testData := TestData{
		Message: "Transformed Message",
		Count:   100,
	}

	err := Emit(context.Background(), TestEvent, "test-source", testData, nil)
	if err != nil {
		t.Fatalf("Failed to emit event: %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	// Verify transformation occurred
	if transformedOutput.Name != "Transformed Message" {
		t.Errorf("Expected transformed name 'Transformed Message', got '%s'", transformedOutput.Name)
	}
	if !transformedOutput.Value {
		t.Errorf("Expected transformed value true, got false")
	}
}

func TestMultipleHandlers(t *testing.T) {
	// Reset for clean test
	Reset()

	handlerCount := 0

	// Register multiple handlers for the same event
	RegisterInput[TestData](TestEvent, func(data TestData) error {
		handlerCount++
		return nil
	})

	RegisterInput[TestData](TestEvent, func(data TestData) error {
		handlerCount++
		return nil
	})

	RegisterOutput[TestData](TestEvent, func(data TestData) error {
		handlerCount++
		return nil
	})

	// Emit event
	testData := TestData{Message: "Multiple", Count: 1}
	err := Emit(context.Background(), TestEvent, "test-source", testData, nil)
	if err != nil {
		t.Fatalf("Failed to emit event: %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	// Should have called all 3 handlers
	if handlerCount != 3 {
		t.Errorf("Expected 3 handlers to be called, got %d", handlerCount)
	}
}

func TestGetStats(t *testing.T) {
	// Reset for clean test
	Reset()

	// Register some handlers
	RegisterInput[TestData](TestEvent, func(data TestData) error { return nil })
	RegisterOutput[TestData](TestEvent, func(data TestData) error { return nil })
	RegisterInput[AnotherData](AnotherEvent, func(data AnotherData) error { return nil })

	stats := GetStats()

	if stats.TotalHandlers != 3 {
		t.Errorf("Expected 3 total handlers, got %d", stats.TotalHandlers)
	}

	if stats.HookTypes["test.event"] != 2 {
		t.Errorf("Expected 2 handlers for test.event, got %d", stats.HookTypes["test.event"])
	}

	if stats.HookTypes["test.another"] != 1 {
		t.Errorf("Expected 1 handler for test.another, got %d", stats.HookTypes["test.another"])
	}
}