package main

import (
	"zbz/internal/models"
	"zbz/lib"

	"net/http"
	"time"
)

func main() {
	e := zbz.NewEngine()

	zbz.NewCore(
		e,
		&zbz.CoreOperation{
			Model: &zbz.CoreModel{
				Name:        "User",
				Description: "A user in the system.",
			},
			Create: &zbz.HTTPOperation{
				Name:        "Create User",
				Description: "Endpoint to create a new user.",
				Method:      "POST",
				Path:        "/v1/users",
				Tag:         "User",
				Response: &zbz.HTTPResponse{
					Status: http.StatusCreated,
					Ref:    "User",
					Errors: []int{
						http.StatusBadRequest,
						http.StatusUnauthorized,
						http.StatusForbidden,
					},
				},
				Auth: true,
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
		},
		&models.User{
			ID:        "123e4567-e89b-12d3-a456-426614174000",
			Name:      "John Doe",
			Email:     "john.doe@example.com",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	)

	e.Start()
}
