package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"zbz/cmd/tests/universal"
)

var testCmd = &cobra.Command{
	Use:   "test",
	Short: "Run framework ecosystem tests",
	Long: `Run comprehensive ecosystem tests for ZBZ framework components.

Available test suites:
- universal: Universal data access pattern validation across all providers`,
}

// Universal test commands
var universalTestCmd = &cobra.Command{
	Use:   "universal",
	Short: "Run universal data access ecosystem tests",
	Long:  "Validate universal data access patterns work correctly across all providers.",
}

var universalComplianceTestCmd = &cobra.Command{
	Use:   "compliance",
	Short: "Universal interface compliance test",
	Run: func(cmd *cobra.Command, args []string) {
		universal.ComplianceTest()
	},
}

var universalOrchestrationTestCmd = &cobra.Command{
	Use:   "orchestration",
	Short: "Provider orchestration test",
	Run: func(cmd *cobra.Command, args []string) {
		universal.OrchestrationTest()
	},
}

var universalIntegrationTestCmd = &cobra.Command{
	Use:   "integration",
	Short: "Cross-provider integration test",
	Run: func(cmd *cobra.Command, args []string) {
		universal.IntegrationTest()
	},
}

var universalHooksTestCmd = &cobra.Command{
	Use:   "hooks",
	Short: "Hook ecosystem test",
	Run: func(cmd *cobra.Command, args []string) {
		universal.HooksTest()
	},
}

var universalPerformanceTestCmd = &cobra.Command{
	Use:   "performance",
	Short: "Performance and load test",
	Run: func(cmd *cobra.Command, args []string) {
		universal.PerformanceTest()
	},
}

var universalAllTestCmd = &cobra.Command{
	Use:   "all",
	Short: "Run all universal ecosystem tests",
	Run: func(cmd *cobra.Command, args []string) {
		universal.AllTests()
	},
}

func initTestCommands() {
	// Add test command to root
	rootCmd.AddCommand(testCmd)
	
	// Build test command tree
	testCmd.AddCommand(universalTestCmd)
	
	// Universal test subcommands
	universalTestCmd.AddCommand(universalComplianceTestCmd)
	universalTestCmd.AddCommand(universalOrchestrationTestCmd)
	universalTestCmd.AddCommand(universalIntegrationTestCmd)
	universalTestCmd.AddCommand(universalHooksTestCmd)
	universalTestCmd.AddCommand(universalPerformanceTestCmd)
	universalTestCmd.AddCommand(universalAllTestCmd)
	
	// Default behavior - if no subcommand, show help
	testCmd.Run = func(cmd *cobra.Command, args []string) {
		cmd.Help()
	}
	
	universalTestCmd.Run = func(cmd *cobra.Command, args []string) {
		fmt.Println("🧪 ZBZ universal data access tests available:")
		fmt.Println("  zbz test universal compliance     - Universal interface compliance test")
		fmt.Println("  zbz test universal orchestration  - Provider orchestration test")
		fmt.Println("  zbz test universal integration    - Cross-provider integration test")
		fmt.Println("  zbz test universal hooks          - Hook ecosystem test")
		fmt.Println("  zbz test universal performance    - Performance and load test")
		fmt.Println("  zbz test universal all            - Run all universal ecosystem tests")
		fmt.Println()
		fmt.Println("Run 'zbz test universal [command] --help' for more information.")
	}
}