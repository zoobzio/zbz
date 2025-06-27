package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	args := os.Args[1:] // Skip program name
	
	if len(args) == 0 {
		// No args = run all examples
		runAll()
		return
	}
	
	// Parse command
	command := strings.ToLower(args[0])
	
	switch command {
	case "json":
		runJSON()
	case "yaml":
		runYAML() 
	case "toml":
		runTOML()
	case "all":
		runAll()
	case "help", "-h", "--help":
		showHelp()
	default:
		fmt.Printf("Unknown command: %s\n\n", command)
		showHelp()
		os.Exit(1)
	}
}

func showHelp() {
	fmt.Println("Cereal Examples - Permission-based serialization demos")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  go run . [command]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  json     Run JSON scoping example (user data)")
	fmt.Println("  yaml     Run YAML scoping example (app config)")
	fmt.Println("  toml     Run TOML scoping example (server config)")
	fmt.Println("  all      Run all examples (default)")
	fmt.Println("  help     Show this help message")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  go run .           # Run all examples")
	fmt.Println("  go run . json      # Run only JSON example")
	fmt.Println("  go run . yaml      # Run only YAML example")
	fmt.Println("  go run . toml      # Run only TOML example")
}

func runJSON() {
	fmt.Println("┌──────────────────────────────────────────────────────────────┐")
	fmt.Println("│                        JSON Example                           │")
	fmt.Println("│                  User Data with Scoping                       │")
	fmt.Println("└──────────────────────────────────────────────────────────────┘")
	json()
}

func runYAML() {
	fmt.Println("┌──────────────────────────────────────────────────────────────┐")
	fmt.Println("│                        YAML Example                           │")
	fmt.Println("│              Application Config with Scoping                  │")
	fmt.Println("└──────────────────────────────────────────────────────────────┘")
	yaml()
}

func runTOML() {
	fmt.Println("┌──────────────────────────────────────────────────────────────┐")
	fmt.Println("│                        TOML Example                           │")
	fmt.Println("│               Server Config with Scoping                      │")
	fmt.Println("└──────────────────────────────────────────────────────────────┘")
	toml()
}

func runAll() {
	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║                    Cereal Examples Suite                      ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Println()
	
	runJSON()
	fmt.Println("\n\n")
	
	runYAML()
	fmt.Println("\n\n")
	
	runTOML()
	
	fmt.Println("\n\n")
	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║                    Examples Complete!                         ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
}