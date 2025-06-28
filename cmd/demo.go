package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"zbz/cmd/demos/cereal"
	"zbz/cmd/demos/universal"
	"zbz/cmd/demos/zlog"
)

var demoCmd = &cobra.Command{
	Use:   "demo",
	Short: "Run framework component demos",
	Long: `Run interactive demos for various ZBZ framework components.

Available demos:
- universal: Universal data access pattern across all providers
- zlog: Structured logging with plugins and security features  
- cereal: Permission-based serialization demos`,
}

// Zlog demo commands
var zlogCmd = &cobra.Command{
	Use:   "zlog",
	Short: "Run zlog (logging) demos",
	Long:  "Demonstrate zlog's features including structured logging, security plugins, and output piping.",
}

var zlogJsonCmd = &cobra.Command{
	Use:   "json",
	Short: "JSON logging demo with zap provider",
	Run: func(cmd *cobra.Command, args []string) {
		zlog.JsonDemo()
	},
}

var zlogSecurityCmd = &cobra.Command{
	Use:   "security",
	Short: "Security plugin demo with encryption and PII protection",
	Run: func(cmd *cobra.Command, args []string) {
		zlog.SecurityDemo()
	},
}

var zlogAllCmd = &cobra.Command{
	Use:   "all",
	Short: "Run all zlog demos",
	Run: func(cmd *cobra.Command, args []string) {
		zlog.AllDemos()
	},
}

// Cereal demo commands  
var cerealCmd = &cobra.Command{
	Use:   "cereal",
	Short: "Run cereal (serialization) demos",
	Long:  "Demonstrate cereal's permission-based serialization features.",
}

var cerealJsonCmd = &cobra.Command{
	Use:   "json",
	Short: "JSON scoping demo with user data",
	Run: func(cmd *cobra.Command, args []string) {
		cereal.JsonDemo()
	},
}

var cerealYamlCmd = &cobra.Command{
	Use:   "yaml", 
	Short: "YAML scoping demo with app config",
	Run: func(cmd *cobra.Command, args []string) {
		cereal.YamlDemo()
	},
}

var cerealTomlCmd = &cobra.Command{
	Use:   "toml",
	Short: "TOML scoping demo with server config", 
	Run: func(cmd *cobra.Command, args []string) {
		cereal.TomlDemo()
	},
}

var cerealAllCmd = &cobra.Command{
	Use:   "all",
	Short: "Run all cereal demos",
	Run: func(cmd *cobra.Command, args []string) {
		cereal.AllDemos()
	},
}

// Universal demo commands
var universalCmd = &cobra.Command{
	Use:   "universal",
	Short: "Run universal data access demos",
	Long:  "Demonstrate universal data access patterns that work across any provider.",
}

var universalBasicCmd = &cobra.Command{
	Use:   "basic",
	Short: "Basic universal interface demo",
	Run: func(cmd *cobra.Command, args []string) {
		universal.BasicDemo()
	},
}

var universalOrchestrationCmd = &cobra.Command{
	Use:   "orchestration",
	Short: "Provider orchestration demo with native features",
	Run: func(cmd *cobra.Command, args []string) {
		universal.OrchestrationDemo()
	},
}

var universalCrossProviderCmd = &cobra.Command{
	Use:   "cross-provider",
	Short: "Cross-provider data flow demo",
	Run: func(cmd *cobra.Command, args []string) {
		universal.CrossProviderDemo()
	},
}

var universalRealTimeCmd = &cobra.Command{
	Use:   "real-time",
	Short: "Real-time synchronization demo",
	Run: func(cmd *cobra.Command, args []string) {
		universal.RealTimeDemo()
	},
}

var universalHooksCmd = &cobra.Command{
	Use:   "hooks",
	Short: "Hook ecosystem demo",
	Run: func(cmd *cobra.Command, args []string) {
		universal.HooksDemo()
	},
}

var universalAllCmd = &cobra.Command{
	Use:   "all",
	Short: "Run all universal demos",
	Run: func(cmd *cobra.Command, args []string) {
		universal.AllDemos()
	},
}

func init() {
	// Build command tree
	demoCmd.AddCommand(universalCmd)
	demoCmd.AddCommand(zlogCmd)
	demoCmd.AddCommand(cerealCmd)
	
	// Zlog subcommands
	zlogCmd.AddCommand(zlogJsonCmd)
	zlogCmd.AddCommand(zlogSecurityCmd)
	zlogCmd.AddCommand(zlogAllCmd)
	
	// Cereal subcommands
	cerealCmd.AddCommand(cerealJsonCmd)
	cerealCmd.AddCommand(cerealYamlCmd)
	cerealCmd.AddCommand(cerealTomlCmd)
	cerealCmd.AddCommand(cerealAllCmd)
	
	// Universal subcommands
	universalCmd.AddCommand(universalBasicCmd)
	universalCmd.AddCommand(universalOrchestrationCmd)
	universalCmd.AddCommand(universalCrossProviderCmd)
	universalCmd.AddCommand(universalRealTimeCmd)
	universalCmd.AddCommand(universalHooksCmd)
	universalCmd.AddCommand(universalAllCmd)
	
	// Default behavior - if no subcommand, show help
	demoCmd.Run = func(cmd *cobra.Command, args []string) {
		cmd.Help()
	}
	
	zlogCmd.Run = func(cmd *cobra.Command, args []string) {
		fmt.Println("🚀 ZBZ zlog demos available:")
		fmt.Println("  zbz demo zlog json       - JSON logging with zap provider")
		fmt.Println("  zbz demo zlog security   - Security plugins and encryption")
		fmt.Println("  zbz demo zlog all        - Run all zlog demos")
		fmt.Println()
		fmt.Println("Run 'zbz demo zlog [command] --help' for more information.")
	}
	
	cerealCmd.Run = func(cmd *cobra.Command, args []string) {
		fmt.Println("📦 ZBZ cereal demos available:")
		fmt.Println("  zbz demo cereal json     - JSON scoping with user data")
		fmt.Println("  zbz demo cereal yaml     - YAML scoping with app config")
		fmt.Println("  zbz demo cereal toml     - TOML scoping with server config")
		fmt.Println("  zbz demo cereal all      - Run all cereal demos")
		fmt.Println()
		fmt.Println("Run 'zbz demo cereal [command] --help' for more information.")
	}
	
	universalCmd.Run = func(cmd *cobra.Command, args []string) {
		fmt.Println("🌟 ZBZ universal data access demos available:")
		fmt.Println("  zbz demo universal basic           - Basic universal interface pattern")
		fmt.Println("  zbz demo universal orchestration   - Provider orchestration with native features")
		fmt.Println("  zbz demo universal cross-provider  - Data flow between different providers")
		fmt.Println("  zbz demo universal real-time       - Real-time synchronization via subscriptions")
		fmt.Println("  zbz demo universal hooks           - Hook ecosystem and observability")
		fmt.Println("  zbz demo universal all             - Run all universal demos")
		fmt.Println()
		fmt.Println("Run 'zbz demo universal [command] --help' for more information.")
	}
}