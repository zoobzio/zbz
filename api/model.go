package zbz

import "time"

// BaseModel is a generic interface that can be used to define models with common fields.
type BaseModel interface {
	GetModel() Model
}

// Model is a base model that includes common fields such as ID, CreatedAt, and UpdatedAt.
type Model struct {
	ID        string    `db:"id" json:"id" validate:"required,uuid4" desc:"A unique identifier" ex:"123e4567-e89b-12d3-a456-426614174000"`
	CreatedAt time.Time `db:"created_at" json:"createdAt" validate:"required" desc:"The time the user was created" ex:"2023-10-01T12:00:00Z"`
	UpdatedAt time.Time `db:"updated_at" json:"updatedAt" validate:"required" desc:"The time the user was last updated" ex:"2023-10-01T12:00:00Z"`
}

// GetModel returns the model instance (implements BaseModel interface)
func (m Model) GetModel() Model {
	return m
}
