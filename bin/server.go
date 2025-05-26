package main

import (
	"zbz/database/models"
	"zbz/lib"
	"zbz/lib/controllers"
)

func main() {
	e := zbz.NewEngine()

	controllers.NewAuthController("/v1/auth", e)

	controllers.NewCoreController[models.Contact]("/v1/contacts", e)
	controllers.NewCoreController[models.Company]("/v1/companies", e)
	controllers.NewCoreController[models.Field]("/v1/fields", e)
	controllers.NewCoreController[models.Property]("/v1/properties", e)

	e.Start()
}
