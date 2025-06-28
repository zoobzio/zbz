package catalog

import (
	"time"
)

// Container wraps user models with standard system fields
// Users never see this directly - it's managed transparently by the framework
type Container[T any] struct {
	// Standard system fields
	ID        string    `json:"id" db:"id" desc:"Unique identifier"`
	CreatedAt time.Time `json:"created_at" db:"created_at" desc:"Creation timestamp"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at" desc:"Last update timestamp"`
	Version   int       `json:"version" db:"version" desc:"Optimistic locking version"`
	
	// User's actual data
	Data T `json:"data" db:"data" desc:"User model data"`
}

// NewContainer creates a new container with system field defaults
func NewContainer[T any](data T) *Container[T] {
	return &Container[T]{
		ID:        generateID(), // TODO: implement ID generation
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Version:   1,
		Data:      data,
	}
}

// GetData returns the user's data, hiding container complexity
func (c *Container[T]) GetData() T {
	return c.Data
}

// UpdateData updates the user's data and system timestamps
func (c *Container[T]) UpdateData(data T) {
	c.Data = data
	c.UpdatedAt = time.Now()
	c.Version++
}

// generateID creates a unique identifier
// TODO: Implement proper ID generation strategy
func generateID() string {
	return "temp-id-" + time.Now().Format("20060102150405")
}