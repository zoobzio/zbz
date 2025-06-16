package zbz

import (
	"time"

	"github.com/docker/distribution/uuid"
)

// BaseModel is an interface that defines a method for creating a new model with default values.
type BaseModel interface {
	NewModelDefaults() *Model
}

// Model is a base model that includes common fields such as ID, CreatedAt, and UpdatedAt.
type Model struct {
	ID        string    `db:"id" json:"id" validate:"required,uuidv4" desc:"A unique identifier" ex:"123e4567-e89b-12d3-a456-426614174000"`
	CreatedAt time.Time `db:"created_at" json:"createdAt" validate:"required" desc:"The time the user was created" ex:"2023-10-01T12:00:00Z"`
	UpdatedAt time.Time `db:"updated_at" json:"updatedAt" validate:"required" desc:"The time the user was last updated" ex:"2023-10-01T12:00:00Z"`
}

// New creates a new instance of Model with a unique ID and current timestamps.
func (m Model) NewModelDefaults() *Model {
	return &Model{
		ID:        uuid.Generate().String(),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
}
