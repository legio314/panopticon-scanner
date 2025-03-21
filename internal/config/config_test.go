// internal/config/config_test.go
package config

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadConfig(t *testing.T) {
	// Create a temporary directory for the test
	tempDir, err := ioutil.TempDir("", "config-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test config file
	configPath := filepath.Join(tempDir, "config.yaml")
	testConfig := `
server:
  port: 9090
  host: "127.0.0.1"

scanner:
  frequency: "30m"
  rateLimit: 500
  scanAllPorts: true
  disablePing: false
  targetNetwork: "10.0.0.0/24"

database:
  path: "./test.db"
`
	err = ioutil.WriteFile(configPath, []byte(testConfig), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Test loading the configuration
	cfg := GetConfig()
	err = cfg.LoadConfig(configPath)
	if err != nil {
		t.Errorf("LoadConfig returned error: %v", err)
	}

	// Check that values were loaded correctly
	if cfg.Server.Port != 9090 {
		t.Errorf("Expected port 9090, got %d", cfg.Server.Port)
	}

	if cfg.Server.Host != "127.0.0.1" {
		t.Errorf("Expected host 127.0.0.1, got %s", cfg.Server.Host)
	}

	if cfg.Scanner.RateLimit != 500 {
		t.Errorf("Expected rate limit 500, got %d", cfg.Scanner.RateLimit)
	}

	if !cfg.Scanner.ScanAllPorts {
		t.Errorf("Expected ScanAllPorts true, got false")
	}

	if cfg.Scanner.DisablePing {
		t.Errorf("Expected DisablePing false, got true")
	}

	if cfg.Scanner.TargetNetwork != "10.0.0.0/24" {
		t.Errorf("Expected TargetNetwork 10.0.0.0/24, got %s", cfg.Scanner.TargetNetwork)
	}

	if cfg.Database.Path != "./test.db" {
		t.Errorf("Expected Database.Path ./test.db, got %s", cfg.Database.Path)
	}
}

func TestReload(t *testing.T) {
	// Create a temporary directory for the test
	tempDir, err := ioutil.TempDir("", "config-reload-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create an initial config file
	configPath := filepath.Join(tempDir, "config.yaml")
	initialConfig := `
server:
  port: 9090
`
	err = ioutil.WriteFile(configPath, []byte(initialConfig), 0644)
	if err != nil {
		t.Fatalf("Failed to write initial config: %v", err)
	}

	// Load the initial configuration
	cfg := GetConfig()
	err = cfg.LoadConfig(configPath)
	if err != nil {
		t.Errorf("LoadConfig returned error: %v", err)
	}

	// Verify initial value
	if cfg.Server.Port != 9090 {
		t.Errorf("Expected initial port 9090, got %d", cfg.Server.Port)
	}

	// Update the config file
	updatedConfig := `
server:
  port: 8080
`
	err = ioutil.WriteFile(configPath, []byte(updatedConfig), 0644)
	if err != nil {
		t.Fatalf("Failed to write updated config: %v", err)
	}

	// Reload the configuration
	err = cfg.Reload()
	if err != nil {
		t.Errorf("Reload returned error: %v", err)
	}

	// Verify the updated value
	if cfg.Server.Port != 8080 {
		t.Errorf("Expected updated port 8080, got %d", cfg.Server.Port)
	}
}

func TestGetScanFrequency(t *testing.T) {
	// Create a test Config with a specific frequency
	cfg := GetConfig()
	cfg.Scanner.Frequency = "2h30m"

	// Get the scan frequency
	duration, err := cfg.GetScanFrequency()
	if err != nil {
		t.Errorf("GetScanFrequency returned error: %v", err)
	}

	// Check that the duration was parsed correctly
	expectedDuration, _ := time.ParseDuration("2h30m")
	if duration != expectedDuration {
		t.Errorf("Expected duration %v, got %v", expectedDuration, duration)
	}

	// Test invalid duration
	cfg.Scanner.Frequency = "invalid"
	_, err = cfg.GetScanFrequency()
	if err == nil {
		t.Errorf("Expected error for invalid frequency, got nil")
	}
}

func TestValidate(t *testing.T) {
	// Test valid configuration
	cfg := GetConfig()
	cfg.Server.Port = 8080
	cfg.Scanner.Frequency = "1h"
	cfg.Scanner.RateLimit = 1000
	cfg.Database.Path = "./data.db"

	err := cfg.Validate()
	if err != nil {
		t.Errorf("Validate returned error for valid config: %v", err)
	}

	// Test invalid port
	cfg.Server.Port = 0
	err = cfg.Validate()
	if err == nil {
		t.Errorf("Expected error for invalid port, got nil")
	}
	cfg.Server.Port = 8080 // Reset

	// Test invalid frequency
	cfg.Scanner.Frequency = "invalid"
	err = cfg.Validate()
	if err == nil {
		t.Errorf("Expected error for invalid frequency, got nil")
	}
	cfg.Scanner.Frequency = "1h" // Reset

	// Test invalid rate limit
	cfg.Scanner.RateLimit = 0
	err = cfg.Validate()
	if err == nil {
		t.Errorf("Expected error for invalid rate limit, got nil")
	}
	cfg.Scanner.RateLimit = 1000 // Reset

	// Test missing database path
	cfg.Database.Path = ""
	err = cfg.Validate()
	if err == nil {
		t.Errorf("Expected error for missing database path, got nil")
	}
}

func TestSaveConfig(t *testing.T) {
	// Create a temporary directory for the test
	tempDir, err := ioutil.TempDir("", "config-save-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a config instance with specific values
	cfg := GetConfig()
	cfg.Database.Path = "./test.db"
	cfg.Server.Port = 9999
	cfg.Scanner.Frequency = "5m"
	cfg.Scanner.TargetNetwork = "192.168.0.0/16"

	// Save the configuration
	savePath := filepath.Join(tempDir, "saved-config.yaml")
	err = cfg.SaveConfig(savePath)
	if err != nil {
		t.Errorf("SaveConfig returned error: %v", err)
	}

	// Verify the file was created
	if _, err := os.Stat(savePath); os.IsNotExist(err) {
		t.Errorf("Config file was not created at %s", savePath)
	}

	// Load the saved file to verify content
	newCfg := GetConfig()
	err = newCfg.LoadConfig(savePath)
	if err != nil {
		t.Errorf("Failed to load saved config: %v", err)
	}

	// Check that values were saved correctly
	if newCfg.Server.Port != 9999 {
		t.Errorf("Expected port 9999, got %d", newCfg.Server.Port)
	}

	if newCfg.Scanner.Frequency != "5m" {
		t.Errorf("Expected frequency 5m, got %s", newCfg.Scanner.Frequency)
	}

	if newCfg.Scanner.TargetNetwork != "192.168.0.0/16" {
		t.Errorf("Expected TargetNetwork 192.168.0.0/16, got %s", newCfg.Scanner.TargetNetwork)
	}
}
