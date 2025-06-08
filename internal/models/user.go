package models

import (
	"time"
)

type User struct {
	ID        string    `gorm:"primaryKey" json:"id" validate:"uidv4" desc:"A unique identifier" ex:"123e4567-e89b-12d3-a456-426614174000"`
	Name      string    `json:"name" validate:"required" desc:"The name of the user" ex:"John Doe"`
	Email     string    `json:"email" validate:"required,email" desc:"The email of the user" ex:"john.doe@example.com"`
	CreatedAt time.Time `json:"createdAt" validate:"required" desc:"The creation time of the user" ex:"2023-10-01T12:00:00Z"`
	UpdatedAt time.Time `json:"updatedAt" validate:"required" desc:"The last update time of the user" ex:"2023-10-01T12:00:00Z"`
}
