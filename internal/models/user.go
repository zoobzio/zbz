package models

import (
	zbz "zbz/lib"
)

type User struct {
	zbz.Model
	Name  string `db:"name" json:"name" validate:"required" desc:"The name of the user" edit:"Owner" ex:"John Doe"`
	Email string `db:"email" json:"email" validate:"required,email" desc:"The email of the user" edit:"Owner" ex:"john.doe@example.com"`
}
