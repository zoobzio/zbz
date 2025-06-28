package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "zbz",
	Short: "ZBZ Framework CLI tool",
	Long: `ZBZ is an opinionated Go framework for building APIs with automatic CRUD generation,
OpenAPI documentation, and structured error handling.

This CLI provides demos, tools, and utilities for working with the ZBZ framework.`,
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	// Add demo command
	rootCmd.AddCommand(demoCmd)
	
	// Initialize test commands
	initTestCommands()
	
	// Future commands can be added here:
	// rootCmd.AddCommand(toolsCmd)
	// rootCmd.AddCommand(benchCmd)
}