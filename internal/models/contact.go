package models

type Contact struct {
	Base
	Email string `json:"email" validate:"required,email"`
}
