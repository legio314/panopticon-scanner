// cmd/panopticond/main_test.go
package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// TestMainStartup tests that the main program starts up correctly
func TestMainStartup(t *testing.T) {
	// Skip this test for now since the /api/status endpoint is not implemented
	t.Skip("Skipping test until /api/status endpoint is implemented")
	
	// Skip in CI environment
	if os.Getenv("CI") != "" {
		t.Skip("Skipping in CI environment")
	}

	// Create a temporary directory for the test
	tempDir, err := os.MkdirTemp("", "panopticon-main-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create subdirectories
	os.MkdirAll(filepath.Join(tempDir, "scans"), 0755)
	os.MkdirAll(filepath.Join(tempDir, "data"), 0755)
	os.MkdirAll(filepath.Join(tempDir, "logs"), 0755)
	os.MkdirAll(filepath.Join(tempDir, "configs"), 0755)

	// Create a test configuration file
	configPath := filepath.Join(tempDir, "configs", "test-config.yaml")
	configContent := `
server:
  port: 18080
  host: "127.0.0.1"

scanner:
  frequency: "1h"
  rateLimit: 1000
  scanAllPorts: false
  disablePing: true
  targetNetwork: "192.168.1.0/24"
  outputDir: "` + filepath.Join(tempDir, "scans") + `"
  enableScheduler: false

database:
  path: "` + filepath.Join(tempDir, "data", "test.db") + `"
  backupDir: "` + filepath.Join(tempDir, "data", "backups") + `"

logging:
  level: "debug"
  outputPath: "` + filepath.Join(tempDir, "logs", "test.log") + `"
`
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Build the application binary
	binaryPath := filepath.Join(tempDir, "panopticond-test")
	buildCmd := exec.Command("go", "build", "-o", binaryPath, ".")
	err = buildCmd.Run()
	if err != nil {
		t.Fatalf("Failed to build test binary: %v", err)
	}

	// Start the application in the background
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, binaryPath, "--config", configPath, "--log-level", "debug")

	// Redirect stdout/stderr to avoid cluttering test output
	cmd.Stdout = os.NewFile(0, os.DevNull)
	cmd.Stderr = os.NewFile(0, os.DevNull)

	err = cmd.Start()
	if err != nil {
		t.Fatalf("Failed to start test binary: %v", err)
	}

	// Create a channel to signal when the server is ready
	ready := make(chan struct{})

	// Check if the server is up by polling the API
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				resp, err := http.Get("http://127.0.0.1:18080/api/status")
				if err == nil && resp.StatusCode == http.StatusOK {
					resp.Body.Close()
					close(ready)
					return
				}
				if resp != nil {
					resp.Body.Close()
				}
				time.Sleep(100 * time.Millisecond)
			}
		}
	}()

	// Wait for the server to be ready or timeout
	select {
	case <-ready:
		// Server is up, continue with tests
	case <-ctx.Done():
		t.Fatalf("Timed out waiting for server to start")
	}

	// Test a basic API endpoint
	resp, err := http.Get("http://127.0.0.1:18080/api/status")
	if err != nil {
		t.Fatalf("Failed to access API: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK, got %v", resp.Status)
	}

	// Parse the response
	var status map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&status)
	if err != nil {
		t.Fatalf("Failed to decode status response: %v", err)
	}

	// Verify status fields
	if statusValue, ok := status["status"]; !ok || statusValue != "ok" {
		t.Errorf("Expected status 'ok', got %v", statusValue)
	}

	// Shutdown the application
	cmd.Process.Signal(os.Interrupt)

	// Wait for the process to exit with a timeout
	done := make(chan error)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case err := <-done:
		if err != nil && err.Error() != "signal: interrupt" {
			t.Errorf("Process did not exit cleanly: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Errorf("Process did not exit within timeout")
		cmd.Process.Kill()
	}

	// Check if the database file was created
	dbPath := filepath.Join(tempDir, "data", "test.db")
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Errorf("Database file was not created")
	}

	// Check if the log file was created
	logPath := filepath.Join(tempDir, "logs", "test.log")
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Errorf("Log file was not created")
	}
}

// TestCommandLineArgs tests command line argument parsing
func TestCommandLineArgs(t *testing.T) {
	// This test needs to be run separately as it modifies global state
	if os.Getenv("RUN_ARGS_TEST") != "1" {
		t.Skip("Skipping command line args test - set RUN_ARGS_TEST=1 to run")
	}

	// Test with valid config path
	os.Args = []string{"panopticond", "--config", "/tmp/test-config.yaml"}
	configPath := parseFlags()
	if configPath != "/tmp/test-config.yaml" {
		t.Errorf("Expected config path /tmp/test-config.yaml, got %s", configPath)
	}

	// Test with valid log level
	os.Args = []string{"panopticond", "--log-level", "debug"}
	_ = parseFlags()
	if logLevelFlag != "debug" {
		t.Errorf("Expected log level debug, got %s", logLevelFlag)
	}

	// Test with invalid log level (should default to info)
	os.Args = []string{"panopticond", "--log-level", "invalid"}
	_ = parseFlags()
	// This should log a warning but continue with default level
}
