package main

import (
	"zbz/internal/models"
	"zbz/lib"
)

func main() {
	e := zbz.NewEngine()

	user := zbz.NewCore[models.User]("A model representing a `User` in the system.")
	contact := zbz.NewCore[models.Contact]("A model representing a `Contact` in the system.")
	company := zbz.NewCore[models.Company]("A model representing a `Company` in the system.")
	form := zbz.NewCore[models.Form]("A model representing a `Form` in the system.")
	property := zbz.NewCore[models.Property]("A model representing a `Property` in the system.")
	field := zbz.NewCore[models.Field]("A model representing a `Field` in the system.")

	e.Inject(
		user,
		contact,
		company,
		form,
		property,
		field,
	)

	e.Start()
}
