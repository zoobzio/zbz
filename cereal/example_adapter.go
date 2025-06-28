package cereal

import (
	"context"
	"fmt"
	"reflect"
	"strings"
)

// Example: How an adapter would bridge cereal validation events to capitan
// This would typically be in a separate adapter package that imports both cereal and capitan

// ExampleValidationAdapter shows how to bridge validation events to external systems
type ExampleValidationAdapter struct {
	eventSink func(eventType string, data interface{}) // Represents capitan.Emit or similar
}

// NewExampleValidationAdapter creates a validation event adapter
func NewExampleValidationAdapter(eventSink func(string, interface{})) *ExampleValidationAdapter {
	adapter := &ExampleValidationAdapter{
		eventSink: eventSink,
	}
	
	// Register the adapter as the event handler
	OnEvent(adapter.handleCerealEvent)
	
	return adapter
}

// handleCerealEvent bridges cereal events to external systems
func (a *ExampleValidationAdapter) handleCerealEvent(action string, model interface{}, data map[string]interface{}) {
	// Transform cereal event to external event format
	externalEvent := map[string]interface{}{
		"source": "cereal",
		"action": action,
		"model":  model,
		"data":   data,
	}
	
	// Send to external system (e.g., capitan, logging, metrics)
	if a.eventSink != nil {
		a.eventSink("validation.executed", externalEvent)
	}
}

// Example: How to register custom validators that work with business logic
func ExampleBusinessValidators() {
	// Register a validator that could call external APIs or business rules
	RegisterValidator("account_id", func(ctx context.Context, field reflect.Value, param string) error {
		accountID := field.String()
		
		// Accept redacted pattern
		if accountID == "ACCOUNT-XXXX" {
			return nil
		}
		
		// Example business logic validation
		if len(accountID) < 8 || len(accountID) > 20 {
			return fmt.Errorf("account_id must be between 8 and 20 characters")
		}
		
		if !strings.HasPrefix(accountID, "ACCOUNT-") {
			return fmt.Errorf("account_id must start with ACCOUNT-")
		}
		
		// In a real adapter, this might call an external service:
		// return externalAccountService.ValidateAccount(ctx, accountID)
		
		return nil
	}, "ACCOUNT-XXXX")
	
	// Register a validator for custom business entities
	RegisterValidator("employee_id", func(ctx context.Context, field reflect.Value, param string) error {
		empID := field.String()
		
		// Accept redacted pattern
		if empID == "EMP-XXXXX" {
			return nil
		}
		
		// Validate format: EMP-12345
		if len(empID) != 9 {
			return fmt.Errorf("employee_id must be 9 characters long")
		}
		
		if !strings.HasPrefix(empID, "EMP-") {
			return fmt.Errorf("employee_id must start with EMP-")
		}
		
		// Check that suffix is numeric
		suffix := empID[4:]
		for _, r := range suffix {
			if r < '0' || r > '9' {
				return fmt.Errorf("employee_id suffix must be numeric")
			}
		}
		
		return nil
	}, "EMP-XXXXX")
}

// Example: How to set up custom redaction patterns for business validators
func ExampleCustomRedactionPatterns() {
	// This would be called during adapter initialization to set up
	// custom redaction patterns for business-specific validators
	
	// The adapter could register additional redaction patterns
	// that would be picked up by the existing redaction logic
}

// Usage Example:
// func init() {
//     // In an adapter package
//     adapter := NewExampleValidationAdapter(func(eventType string, data interface{}) {
//         // Bridge to capitan or other external systems
//         capitan.Emit(context.Background(), ValidationEvent, "cereal-adapter", data, nil)
//     })
//     
//     // Register business-specific validators
//     ExampleBusinessValidators()
// }