package models

import (
	"time"
	zbz "zbz/lib"
)

type User struct {
	zbz.Model
	//ID        string    `json:"id" validate:"required,uuidv4" desc:"A unique identifier" ex:"123e4567-e89b-12d3-a456-426614174000"`
	Name      string    `json:"name" validate:"required" desc:"The name of the user" edit:"Owner" ex:"John Doe"`
	Email     string    `json:"email" validate:"required,email" desc:"The email of the user" edit:"Owner" ex:"john.doe@example.com"`
	CreatedAt time.Time `json:"createdAt" validate:"required" desc:"The time the user was created" ex:"2023-10-01T12:00:00Z"`
	UpdatedAt time.Time `json:"updatedAt" validate:"required" desc:"The time the user was last updated" ex:"2023-10-01T12:00:00Z"`
}
