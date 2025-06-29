package core

import (
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"zbz/cereal"
	"zbz/zlog"
)

// ZbzModel wraps user types into ZBZ-native entities with standard fields and behaviors
type ZbzModel[T any] struct {
	data      T
	createdAt *time.Time
	updatedAt *time.Time
	deletedAt *time.Time // Soft delete support
	version   int64
	metadata  map[string]any
}

// ZbzModelInterface provides type-erased access to ZbzModel methods
type ZbzModelInterface interface {
	ID() string
	CreatedAt() *time.Time
	UpdatedAt() *time.Time
	DeletedAt() *time.Time
	Version() int64
	IsDeleted() bool
	SetCreatedAt(*time.Time)
	SetUpdatedAt(*time.Time)
	SetVersion(int64)
	IncrementVersion()
	SoftDelete()
	Restore()
	Metadata() map[string]any
	SetMetadata(string, any)
	Validate() error
}

// Wrap creates a ZbzModel from any user type
func Wrap[T any](data T) ZbzModel[T] {
	return ZbzModel[T]{
		data:     data,
		version:  1,
		metadata: make(map[string]any),
	}
}

// Core field accessors

// ID extracts the ID using cached type metadata or generates fallback
func (m ZbzModel[T]) ID() string {
	// Use cached type metadata to avoid repeated reflection
	t := reflect.TypeOf(m.data)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	
	metadata := getTypeMetadata(t)
	if metadata.HasIDField {
		if idValue := metadata.IDField.GetValue(m.data); idValue != nil {
			zlog.Debug("ID extracted from cached metadata",
				zlog.String("type", t.String()),
				zlog.String("field", metadata.IDField.Name),
				zlog.Any("value", idValue),
			)
			return fmt.Sprintf("%v", idValue)
		}
	}
	
	// Fallback to generated ID
	fallbackID := m.generateFallbackID()
	zlog.Debug("Using fallback ID",
		zlog.String("type", t.String()),
		zlog.String("id", fallbackID),
	)
	return fallbackID
}

// Data returns the wrapped user type
func (m ZbzModel[T]) Data() T {
	return m.data
}

// SetData updates the wrapped user type and increments version
func (m *ZbzModel[T]) SetData(data T) {
	m.data = data
	now := time.Now()
	m.updatedAt = &now
	m.version++
}

// Lifecycle field accessors

func (m ZbzModel[T]) CreatedAt() *time.Time {
	return m.createdAt
}

func (m *ZbzModel[T]) SetCreatedAt(t *time.Time) {
	m.createdAt = t
}

func (m ZbzModel[T]) UpdatedAt() *time.Time {
	return m.updatedAt
}

func (m *ZbzModel[T]) SetUpdatedAt(t *time.Time) {
	m.updatedAt = t
}

func (m ZbzModel[T]) DeletedAt() *time.Time {
	return m.deletedAt
}

func (m ZbzModel[T]) Version() int64 {
	return m.version
}

func (m *ZbzModel[T]) SetVersion(v int64) {
	m.version = v
}

func (m *ZbzModel[T]) IncrementVersion() {
	m.version++
	now := time.Now()
	m.updatedAt = &now
}

// Soft delete support

func (m ZbzModel[T]) IsDeleted() bool {
	return m.deletedAt != nil
}

func (m *ZbzModel[T]) SoftDelete() {
	now := time.Now()
	m.deletedAt = &now
	m.updatedAt = &now
	m.version++
}

func (m *ZbzModel[T]) Restore() {
	m.deletedAt = nil
	now := time.Now()
	m.updatedAt = &now
	m.version++
}

// Metadata management

func (m ZbzModel[T]) Metadata() map[string]any {
	result := make(map[string]any)
	for k, v := range m.metadata {
		result[k] = v
	}
	return result
}

func (m *ZbzModel[T]) SetMetadata(key string, value any) {
	if m.metadata == nil {
		m.metadata = make(map[string]any)
	}
	m.metadata[key] = value
}

// Validation integrates with cereal's validation system
func (m ZbzModel[T]) Validate() error {
	// Try to use cereal's validation if available
	if err := cereal.Validate(m.data); err != nil {
		zlog.Error("Validation failed",
			zlog.String("type", reflect.TypeOf(m.data).String()),
			zlog.String("error", err.Error()),
		)
		return err
	}
	
	zlog.Debug("Validation passed",
		zlog.String("type", reflect.TypeOf(m.data).String()),
		zlog.String("id", m.ID()),
	)
	return nil
}

// JSON serialization that preserves user type structure
func (m ZbzModel[T]) MarshalJSON() ([]byte, error) {
	wrapper := struct {
		ID        string         `json:"id"`
		CreatedAt *time.Time     `json:"created_at,omitempty"`
		UpdatedAt *time.Time     `json:"updated_at,omitempty"`
		DeletedAt *time.Time     `json:"deleted_at,omitempty"`
		Version   int64          `json:"version"`
		Data      T              `json:"data"`
		Metadata  map[string]any `json:"metadata,omitempty"`
	}{
		ID:        m.ID(),
		CreatedAt: m.createdAt,
		UpdatedAt: m.updatedAt,
		DeletedAt: m.deletedAt,
		Version:   m.version,
		Data:      m.data,
	}
	
	// Only include metadata if not empty
	if len(m.metadata) > 0 {
		wrapper.Metadata = m.metadata
	}
	
	return json.Marshal(wrapper)
}

// JSON deserialization
func (m *ZbzModel[T]) UnmarshalJSON(data []byte) error {
	var wrapper struct {
		ID        string         `json:"id"`
		CreatedAt *time.Time     `json:"created_at,omitempty"`
		UpdatedAt *time.Time     `json:"updated_at,omitempty"`
		DeletedAt *time.Time     `json:"deleted_at,omitempty"`
		Version   int64          `json:"version"`
		Data      T              `json:"data"`
		Metadata  map[string]any `json:"metadata,omitempty"`
	}
	
	if err := json.Unmarshal(data, &wrapper); err != nil {
		return err
	}
	
	m.data = wrapper.Data
	m.createdAt = wrapper.CreatedAt
	m.updatedAt = wrapper.UpdatedAt
	m.deletedAt = wrapper.DeletedAt
	m.version = wrapper.Version
	m.metadata = wrapper.Metadata
	
	return nil
}

// extractIDFromStruct tries to extract an ID from the struct using reflection
func (m ZbzModel[T]) extractIDFromStruct() any {
	v := reflect.ValueOf(m.data)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	
	if v.Kind() != reflect.Struct {
		return nil
	}
	
	t := v.Type()
	
	// First, look for fields with zbz:"id" tag
	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("zbz")
		if tag == "id" {
			return v.Field(i).Interface()
		}
	}
	
	// Then look for common field names
	for _, fieldName := range []string{"ID", "Id", "id", "Key", "key", "SKU", "UUID"} {
		field := v.FieldByName(fieldName)
		if field.IsValid() && field.CanInterface() {
			return field.Interface()
		}
	}
	
	return nil
}

// generateFallbackID creates a fallback ID when no ID field is found
func (m ZbzModel[T]) generateFallbackID() string {
	// Use a simple timestamp-based ID for fallback
	// In production, this would use ULID or similar
	return fmt.Sprintf("zbz_%d", time.Now().UnixNano())
}

// CoreEvent represents events emitted by core operations
type CoreEvent struct {
	Type     string      `json:"type"`     // "created", "updated", "deleted"
	Resource string      `json:"resource"` // ResourceURI string
	CoreType string      `json:"core_type"`
	Old      any         `json:"old,omitempty"`
	New      any         `json:"new,omitempty"`
	Data     any         `json:"data,omitempty"`
	UserID   string      `json:"user_id,omitempty"`
	Time     time.Time   `json:"time"`
}

// EmitEvent emits an event for this model operation (placeholder implementation)
func (m ZbzModel[T]) EmitEvent(eventType, resource, coreType string, old *ZbzModel[T]) {
	event := CoreEvent{
		Type:     eventType,
		Resource: resource,
		CoreType: coreType,
		Time:     time.Now(),
	}
	
	switch eventType {
	case "created":
		event.Data = m
	case "updated":
		event.Old = old
		event.New = m
	case "deleted":
		event.Data = m
	}
	
	// Placeholder - would emit to capitan event system
	// capitan.Emit("core."+eventType, event)
	// capitan.Emit("core."+coreType+"."+eventType, event)
}

// Helper functions for common operations

// IsNew returns true if this model has never been persisted
func (m ZbzModel[T]) IsNew() bool {
	return m.createdAt == nil
}

// Touch updates the UpdatedAt timestamp and increments version
func (m *ZbzModel[T]) Touch() {
	now := time.Now()
	m.updatedAt = &now
	m.version++
}

// Clone creates a deep copy of the model
func (m ZbzModel[T]) Clone() ZbzModel[T] {
	clone := ZbzModel[T]{
		data:    m.data, // Note: This is a shallow copy of T
		version: m.version,
	}
	
	if m.createdAt != nil {
		t := *m.createdAt
		clone.createdAt = &t
	}
	if m.updatedAt != nil {
		t := *m.updatedAt
		clone.updatedAt = &t
	}
	if m.deletedAt != nil {
		t := *m.deletedAt
		clone.deletedAt = &t
	}
	
	// Deep copy metadata
	if m.metadata != nil {
		clone.metadata = make(map[string]any)
		for k, v := range m.metadata {
			clone.metadata[k] = v
		}
	}
	
	return clone
}