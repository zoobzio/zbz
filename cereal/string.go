package cereal

import (
	"fmt"
	"reflect"
	"strings"
	"unicode/utf8"
)

// StringProvider implements CerealProvider for string data
type StringProvider struct {
	config      CerealConfig
	validateUTF8 bool
}

// NewStringProvider creates a new string provider contract
func NewStringProvider(config CerealConfig) *CerealContract[string] {
	// Apply defaults
	if config.Name == "" {
		config.Name = "string"
	}
	
	provider := &StringProvider{
		config:       config,
		validateUTF8: true, // Default to UTF-8 validation
	}
	
	// Native type is string
	var nativeString string
	
	return NewContract("string", provider, nativeString, config)
}

// SetUTF8Validation enables or disables UTF-8 validation
func (s *StringProvider) SetUTF8Validation(validate bool) {
	s.validateUTF8 = validate
}

// Marshal for string provider converts data to string representation
func (s *StringProvider) Marshal(data any) ([]byte, error) {
	var str string
	
	switch v := data.(type) {
	case string:
		str = v
	case []byte:
		str = string(v)
	case fmt.Stringer:
		str = v.String()
	default:
		// Fall back to fmt.Sprintf for other types
		str = fmt.Sprintf("%v", v)
	}
	
	// Validate UTF-8 if enabled
	if s.validateUTF8 && !utf8.ValidString(str) {
		return nil, fmt.Errorf("invalid UTF-8 string")
	}
	
	return []byte(str), nil
}

// Unmarshal for string provider converts bytes to string
func (s *StringProvider) Unmarshal(data []byte, target any) error {
	str := string(data)
	
	// Validate UTF-8 if enabled
	if s.validateUTF8 && !utf8.ValidString(str) {
		return fmt.Errorf("invalid UTF-8 string")
	}
	
	targetVal := reflect.ValueOf(target)
	if targetVal.Kind() != reflect.Ptr {
		return fmt.Errorf("target must be a pointer")
	}
	
	targetElem := targetVal.Elem()
	
	switch targetElem.Kind() {
	case reflect.String:
		targetElem.SetString(str)
		return nil
	case reflect.Slice:
		// Check if it's []byte
		if targetElem.Type().Elem().Kind() == reflect.Uint8 {
			targetElem.Set(reflect.ValueOf([]byte(str)))
			return nil
		}
		return fmt.Errorf("string provider can only unmarshal to string or []byte, got %s", targetElem.Type())
	default:
		return fmt.Errorf("string provider can only unmarshal to string or []byte, got %s", targetElem.Type())
	}
}

// MarshalScoped for string data (no scoping - strings have no fields)
func (s *StringProvider) MarshalScoped(data any, userPermissions []string) ([]byte, error) {
	// Strings have no fields to scope, so this is identical to Marshal
	return s.Marshal(data)
}

// UnmarshalScoped for string data (no scoping - strings have no fields)
func (s *StringProvider) UnmarshalScoped(data []byte, target any, userPermissions []string, operation string) error {
	// Strings have no fields to scope, so this is identical to Unmarshal
	return s.Unmarshal(data, target)
}

// ContentType returns the MIME type for plain text
func (s *StringProvider) ContentType() string {
	return "text/plain; charset=utf-8"
}

// Format returns the format identifier
func (s *StringProvider) Format() string {
	return "string"
}

// SupportsBinaryData returns false (string provider is for text data)
func (s *StringProvider) SupportsBinaryData() bool {
	return false
}

// SupportsStreaming returns false (not implemented yet)
func (s *StringProvider) SupportsStreaming() bool {
	return false
}

// Close cleans up the provider (string provider has no cleanup needed)
func (s *StringProvider) Close() error {
	return nil
}

// Utility methods for string processing

// EscapeHTML escapes HTML characters in the string
func (s *StringProvider) EscapeHTML(str string) string {
	replacer := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		"\"", "&quot;",
		"'", "&#39;",
	)
	return replacer.Replace(str)
}

// UnescapeHTML unescapes HTML characters in the string
func (s *StringProvider) UnescapeHTML(str string) string {
	replacer := strings.NewReplacer(
		"&amp;", "&",
		"&lt;", "<",
		"&gt;", ">",
		"&quot;", "\"",
		"&#39;", "'",
	)
	return replacer.Replace(str)
}

// TrimSpace removes leading and trailing whitespace
func (s *StringProvider) TrimSpace(str string) string {
	return strings.TrimSpace(str)
}