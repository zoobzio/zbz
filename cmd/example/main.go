package main

import (
	"context"

	"go.uber.org/zap"
	"zbz/cmd/example/internal/models"
	"zbz/lib"
)

func main() {
	ctx := context.Background()

	_, _, err := zbz.InitTelemetry(ctx, "zbz")
	if err != nil {
		zbz.Log.Fatal("Failed to initialize tracer", zap.Error(err))
	}

	e := zbz.NewEngine()

	contact := zbz.NewCore[models.Contact]("A model representing a `Contact` in the system.")
	company := zbz.NewCore[models.Company]("A model representing a `Company` in the system.")
	form := zbz.NewCore[models.Form]("A model representing a `Form` in the system.")
	property := zbz.NewCore[models.Property]("A model representing a `Property` in the system.")
	field := zbz.NewCore[models.Field]("A model representing a `Field` in the system.")

	e.Inject(
		contact.Contract(
			"Contact Management - Store and manage contact information including names, email addresses, phone numbers, and physical addresses. Contacts can be associated with companies and forms.",
		),
		company.Contract(
			"Company Management - Manage company profiles and organizational data. Companies serve as containers for contacts and can be associated with multiple forms and business processes.",
		),
		form.Contract(
			"Form Builder & Management - Create, configure, and manage dynamic forms with custom fields. Forms are the core building blocks for data collection and can contain multiple field types.",
		),
		property.Contract(
			"Property Value Storage - Store dynamic property values for form submissions. Properties link fields to their actual data values, enabling flexible data storage for any form configuration.",
		),
		field.Contract(
			"Field Definition & Types - Define form field schemas including field types, validation rules, and display properties. Fields serve as templates that define the structure of form data.",
		),
	)

	e.Start()
}
