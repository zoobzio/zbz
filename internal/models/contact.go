package models

import (
	"zbz/lib"
)

type Contact struct {
	zbz.Model
	Name    string `db:"name" json:"name" validate:"required" desc:"The name of a contact" ex:"John Doe" edit:"Owner"`
	Email   string `db:"email" json:"email" validate:"required,email" desc:"The email of a contact" ex:"john.doe@example.com" edit:"Owner"`
	Phone   string `db:"phone" json:"phone" validate:"required" desc:"The phone number of a contact" ex:"+1234567890" edit:"Owner"`
	Address string `db:"address" json:"address" validate:"required" desc:"The address of a contact" ex:"123 Main St, Anytown, USA" edit:"Owner"`
}
