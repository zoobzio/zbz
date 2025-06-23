package main

import (
	"context"

	"go.uber.org/zap"
	"zbz/cmd/example/internal/contracts"
	"zbz/lib"
)

func main() {
	ctx := context.Background()

	_, _, err := zbz.InitTelemetry(ctx, "zbz")
	if err != nil {
		zbz.Log.Fatal("Failed to initialize tracer", zap.Error(err))
	}

	e := zbz.NewEngine()

	// Inject contracts directly - they handle their own setup including database creation
	e.Inject(
		contracts.UserContract,
		contracts.ContactContract,
		contracts.CompanyContract,
		contracts.FormContract,
		contracts.PropertyContract,
		contracts.FieldContract,
	)

	e.Start()
}
