package models

type User struct {
	Base
	Email string `json:"email" validate:"required,email"`
}
