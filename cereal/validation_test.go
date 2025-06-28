package cereal

import (
	"testing"
)

// Test struct with validation tags
type User struct {
	Name    string `json:"name" validate:"required,min=2,max=50"`
	Email   string `json:"email" validate:"required,email"`
	Age     int    `json:"age" validate:"min=0,max=120"`
	Website string `json:"website,omitempty" validate:"omitempty,url"`
}

func TestValidation_ValidStruct(t *testing.T) {
	user := User{
		Name:    "John Doe",
		Email:   "john@example.com",
		Age:     30,
		Website: "https://example.com",
	}

	// Should marshal successfully with valid data
	data, err := JSON.Marshal(user)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Should unmarshal successfully with valid data
	var unmarshaled User
	err = JSON.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if unmarshaled.Name != user.Name {
		t.Errorf("Expected name %s, got %s", user.Name, unmarshaled.Name)
	}
}

func TestValidation_InvalidStruct(t *testing.T) {
	user := User{
		Name:    "", // Required field missing
		Email:   "invalid-email", // Invalid email format
		Age:     -5, // Negative age
		Website: "not-a-url", // Invalid URL
	}

	// Should fail validation during marshal
	_, err := JSON.Marshal(user)
	if err == nil {
		t.Error("Expected validation error, got nil")
	}

	// Test individual validation
	if err := Validate(user); err == nil {
		t.Error("Expected validation error, got nil")
	}
}

func TestValidation_PartiallyValidStruct(t *testing.T) {
	user := User{
		Name:  "John",
		Email: "john@example.com",
		Age:   25,
		// Website omitted - should be OK since it's omitempty
	}

	// Should marshal successfully
	data, err := JSON.Marshal(user)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Should unmarshal successfully
	var unmarshaled User
	err = JSON.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
}

func TestValidation_FormatErrors(t *testing.T) {
	user := User{
		Name:  "A", // Too short
		Email: "bad-email",
		Age:   200, // Too old
	}

	err := Validate(user)
	if err == nil {
		t.Fatal("Expected validation error")
	}

	formatted := FormatValidationErrors(err)
	if len(formatted) == 0 {
		t.Error("Expected formatted errors")
	}

	// Check that we get structured error information
	for _, e := range formatted {
		if e.Field == "" || e.Tag == "" {
			t.Errorf("Expected field and tag to be set, got Field=%s, Tag=%s", e.Field, e.Tag)
		}
	}
}

func TestValidation_CustomValidator(t *testing.T) {
	// Test setting a custom validator
	originalValidator := globalValidator
	defer func() {
		globalValidator = originalValidator
	}()

	// Set a validator that always passes
	SetValidator(&mockValidator{})

	user := User{
		Name:  "", // This should normally fail validation
		Email: "invalid",
		Age:   -1,
	}

	// Should pass with mock validator
	_, err := JSON.Marshal(user)
	if err != nil {
		t.Errorf("Expected no error with mock validator, got: %v", err)
	}
}

// mockValidator always returns nil (passes validation)
type mockValidator struct{}

func (m *mockValidator) Validate(s interface{}) error {
	return nil
}

func TestValidation_NonStructTypes(t *testing.T) {
	// Test that validation skips non-struct types
	data := "not a struct"
	
	jsonData, err := JSON.Marshal(data)
	if err != nil {
		t.Errorf("Expected no error for non-struct, got: %v", err)
	}

	var result string
	err = JSON.Unmarshal(jsonData, &result)
	if err != nil {
		t.Errorf("Expected no error for non-struct, got: %v", err)
	}

	if result != data {
		t.Errorf("Expected %s, got %s", data, result)
	}
}