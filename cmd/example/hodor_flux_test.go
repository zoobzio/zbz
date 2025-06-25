package main

import (
	"encoding/json"
	"fmt"
	"time"

	"zbz/flux"
	"zbz/hodor"
	"zbz/zlog"
)

// TestConfig represents a simple configuration structure
type TestConfig struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Debug   bool   `json:"debug"`
}

// testHodorFluxIntegration demonstrates the new hodor + flux reactive architecture
func testHodorFluxIntegration() {
	zlog.Info("Starting hodor + flux integration test")

	// Create a memory hodor contract
	contract := hodor.NewMemory(nil)
	
	// Register the contract with hodor service
	err := contract.Register("test-storage")
	if err != nil {
		zlog.Error("Failed to register hodor contract", zlog.Err(err))
		return
	}
	defer contract.Unregister()

	// Create initial test configuration
	initialConfig := TestConfig{
		Name:    "test-app",
		Version: "1.0.0",
		Debug:   false,
	}

	// Store initial config in hodor
	configJSON, _ := json.Marshal(initialConfig)
	err = contract.Set("config.json", configJSON, 0)
	if err != nil {
		zlog.Error("Failed to store initial config", zlog.Err(err))
		return
	}

	// Set up reactive watcher using flux.Sync with hodor contract
	configWatcher, err := flux.Sync[TestConfig](
		contract,
		"config.json",
		func(old, new TestConfig) {
			zlog.Info("Configuration changed",
				zlog.String("old_name", old.Name),
				zlog.String("new_name", new.Name),
				zlog.String("old_version", old.Version),
				zlog.String("new_version", new.Version),
				zlog.Bool("old_debug", old.Debug),
				zlog.Bool("new_debug", new.Debug))
		},
	)
	if err != nil {
		zlog.Error("Failed to create config watcher", zlog.Err(err))
		return
	}
	defer configWatcher.Dismiss()

	zlog.Info("Created reactive config watcher")

	// Simulate configuration updates to test reactivity
	time.Sleep(100 * time.Millisecond) // Allow initial callback to complete

	// Update 1: Change version
	updatedConfig1 := TestConfig{
		Name:    "test-app",
		Version: "1.1.0",
		Debug:   false,
	}
	configJSON1, _ := json.Marshal(updatedConfig1)
	err = contract.Set("config.json", configJSON1, 0)
	if err != nil {
		zlog.Error("Failed to update config", zlog.Err(err))
		return
	}

	time.Sleep(100 * time.Millisecond) // Allow callback to process

	// Update 2: Enable debug mode
	updatedConfig2 := TestConfig{
		Name:    "test-app",
		Version: "1.1.0",
		Debug:   true,
	}
	configJSON2, _ := json.Marshal(updatedConfig2)
	err = contract.Set("config.json", configJSON2, 0)
	if err != nil {
		zlog.Error("Failed to update config", zlog.Err(err))
		return
	}

	time.Sleep(100 * time.Millisecond) // Allow callback to process

	// Test flux.Get for one-time retrieval
	currentConfig, err := flux.Get[TestConfig](contract, "config.json")
	if err != nil {
		zlog.Error("Failed to get current config", zlog.Err(err))
		return
	}

	zlog.Info("Current configuration retrieved",
		zlog.String("name", currentConfig.Name),
		zlog.String("version", currentConfig.Version),
		zlog.Bool("debug", currentConfig.Debug))

	// Test watcher control methods
	zlog.Info("Testing watcher pause/resume")
	
	err = configWatcher.Pause()
	if err != nil {
		zlog.Error("Failed to pause watcher", zlog.Err(err))
		return
	}

	// This update should not trigger callback (paused)
	pausedConfig := TestConfig{
		Name:    "paused-app",
		Version: "2.0.0",
		Debug:   false,
	}
	pausedJSON, _ := json.Marshal(pausedConfig)
	err = contract.Set("config.json", pausedJSON, 0)
	if err != nil {
		zlog.Error("Failed to update config while paused", zlog.Err(err))
		return
	}

	time.Sleep(100 * time.Millisecond)

	// Resume watcher
	err = configWatcher.Resume()
	if err != nil {
		zlog.Error("Failed to resume watcher", zlog.Err(err))
		return
	}

	// This update should trigger callback (resumed)
	resumedConfig := TestConfig{
		Name:    "resumed-app",
		Version: "2.1.0",
		Debug:   true,
	}
	resumedJSON, _ := json.Marshal(resumedConfig)
	err = contract.Set("config.json", resumedJSON, 0)
	if err != nil {
		zlog.Error("Failed to update config after resume", zlog.Err(err))
		return
	}

	time.Sleep(100 * time.Millisecond)

	// Test resolve method (current value without callback)
	resolved, err := configWatcher.Resolve()
	if err != nil {
		zlog.Error("Failed to resolve current value", zlog.Err(err))
		return
	}

	if resolvedConfig, ok := resolved.(TestConfig); ok {
		zlog.Info("Resolved current configuration",
			zlog.String("name", resolvedConfig.Name),
			zlog.String("version", resolvedConfig.Version),
			zlog.Bool("debug", resolvedConfig.Debug))
	}

	// Display watcher status
	zlog.Info("Watcher status check",
		zlog.Bool("is_active", configWatcher.IsActive()),
		zlog.Bool("is_paused", configWatcher.IsPaused()),
		zlog.Bool("is_recovering", configWatcher.IsRecovering()))

	zlog.Info("Hodor + flux integration test completed successfully")
}

func init() {
	// Register test function to be called from main
	fmt.Println("Hodor + Flux integration test available")
}