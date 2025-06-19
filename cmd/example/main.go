package main

import (
	"context"

	"zbz/cmd/example/internal/models"
	"zbz/lib"
)

func main() {
	ctx := context.Background()

	_, _, err := zbz.InitTelemetry(ctx, "zbz")
	if err != nil {
		zbz.Log.Fatalw("Failed to initialize tracer", "error", err)
	}

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
