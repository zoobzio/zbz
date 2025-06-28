package capitan

import (
	"context"
	"time"
)

// HookType is the base type for all hook type enums
type HookType interface {
	~int
	String() string
}

// TypedEvent represents a type-safe event with generic payload
type TypedEvent[T any] struct {
	Type      string          `json:"type"`
	Source    string          `json:"source"`
	Timestamp time.Time       `json:"timestamp"`
	Data      T               `json:"data"`
	Context   context.Context `json:"-"`
	Metadata  map[string]any  `json:"metadata,omitempty"`
}