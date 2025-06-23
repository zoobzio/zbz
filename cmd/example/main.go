package main

import (
	"context"

	"zbz/cmd/example/internal/contracts"
	"zbz/lib"
	"zbz/shared/logger"
)

func main() {
	ctx := context.Background()

	_, _, err := zbz.InitTelemetry(ctx, "zbz")
	if err != nil {
		logger.Log.Fatal("Failed to initialize tracer", logger.Err(err))
	}

	e := zbz.NewEngine()

	// Set up providers first
	e.SetHTTP(&contracts.HTTPContract)
	e.SetDatabase(&contracts.PrimaryDatabase)

	// Inject core contracts - they handle their own setup including database creation
	e.Inject(
		contracts.ContactContract,
		contracts.CompanyContract,
		contracts.FormContract,
		contracts.PropertyContract,
		contracts.FieldContract,
	)

	e.Start(":8080")
}
