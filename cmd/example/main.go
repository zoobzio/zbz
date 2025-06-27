package main

import (
	"context"

	"zbz/cmd/example/internal/contracts"
	"zbz/api"
	"zbz/zlog"
	_ "zbz/providers/zlog-zap"  // Import zap driver
)

func main() {
	ctx := context.Background()

	_, _, err := zbz.InitTelemetry(ctx, "zbz")
	if err != nil {
		zlog.Fatal("Failed to initialize tracer", zlog.Err(err))
	}
	
	// Test depot + flux integration before starting server
	zlog.Info("Running depot + flux integration test")
	testDepotFluxIntegration()
	
	// Users can optionally add their own remark directories:
	// zbz.Remark.AddPath("docs/custom")

	e := zbz.NewEngine()

	// Set up providers first
	e.SetHTTP(&contracts.HTTPContract)
	e.SetDatabase(&contracts.PrimaryDatabase)
	e.SetAuth(&contracts.AuthContract)

	// Inject core contracts - they handle their own setup including database creation
	e.Inject(
		contracts.ContactContract,
		contracts.CompanyContract,
		contracts.FormContract,
		contracts.PropertyContract,
		contracts.FieldContract,
	)

	// Prime the engine to set up middleware and default endpoints
	e.Prime()

	e.Start(":8080")
}
