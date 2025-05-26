package models

import (
	"time"
)

type Root struct {
	ID        string    `gorm:"primaryKey" json:"id" validate:"uidv4"`
	Name      string    `json:"name" validate:"required"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
