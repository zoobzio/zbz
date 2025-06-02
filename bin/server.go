package main

import (
	"zbz/internal/models"
	"zbz/lib"
)

func main() {
	e := zbz.NewEngine()
	e.Prime()

	zbz.NewCore[models.User](e, &zbz.CoreOperation{
		Create: &zbz.HTTPOperation{
			Name:        "Create User",
			Description: "Endpoint to create a new user.",
			Method:      "POST",
			Path:        "/v1/users",
			Tag:         "User",
			Auth:        true,
		},
		Read: &zbz.HTTPOperation{
			Name:        "Get User",
			Description: "Endpoint to get a user by ID.",
			Method:      "GET",
			Path:        "/v1/users/:id",
			Tag:         "User",
			Auth:        true,
		},
		Update: &zbz.HTTPOperation{
			Name:        "Update User",
			Description: "Endpoint to update a user by ID.",
			Method:      "PUT",
			Path:        "/v1/users/:id",
			Tag:         "User",
			Auth:        true,
		},
		Delete: &zbz.HTTPOperation{
			Name:        "Delete User",
			Description: "Endpoint to delete a user by ID.",
			Method:      "DELETE",
			Path:        "/v1/users/:id",
			Tag:         "User",
			Auth:        true,
		},
	})

	e.Start()
}
