// internal/scanner/scanner_test.go
package scanner

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"panopticon-scanner/internal/config"
	"panopticon-scanner/internal/database"
	"panopticon-scanner/internal/models"
)

// mockNmapOutput creates a mock nmap XML output file for testing
func mockNmapOutput(t *testing.T, tempDir string) string {
	// Create a mock nmap XML output
	xmlOutput := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE nmaprun>
<nmaprun scanner="nmap">
  <host>
    <status state="up" />
    <address addr="192.168.1.1" addrtype="ipv4" />
    <address addr="00:11:22:33:44:55" addrtype="mac" />
    <hostnames>
      <hostname name="router.local" />
    </hostnames>
    <ports>
      <port protocol="tcp" portid="80">
        <state state="open" />
        <service name="http" product="nginx" version="1.18.0" />
      </port>
      <port protocol="tcp" portid="443">
        <state state="open" />
        <service name="https" product="nginx" version="1.18.0" />
      </port>
      <port protocol="tcp" portid="22">
        <state state="open" />
        <service name="ssh" product="OpenSSH" version="8.2p1" />
      </port>
    </ports>
    <os>
      <osmatch name="Linux 5.4" accuracy="95">
        <osclass type="general purpose" vendor="Linux" osfamily="Linux" osgen="5.X" accuracy="95" />
      </osmatch>
    </os>
  </host>
  <host>
    <status state="up" />
    <address addr="192.168.1.2" addrtype="ipv4" />
    <address addr="AA:BB:CC:DD:EE:FF" addrtype="mac" />
    <hostnames>
      <hostname name="desktop.local" />
    </hostnames>
    <ports>
      <port protocol="tcp" portid="445">
        <state state="open" />
        <service name="microsoft-ds" product="Windows Share" />
      </port>
      <port protocol="tcp" portid="3389">
        <state state="open" />
        <service name="ms-wbt-server" product="Microsoft Terminal Services" />
      </port>
    </ports>
    <os>
      <osmatch name="Windows 10" accuracy="94">
        <osclass type="general purpose" vendor="Microsoft" osfamily="Windows" osgen="10" accuracy="94" />
      </osmatch>
    </os>
  </host>
</nmaprun>`

	// Write the mock XML to a temporary file
	outputPath := filepath.Join(tempDir, "mock_scan.xml")
	err := ioutil.WriteFile(outputPath, []byte(xmlOutput), 0644)
	if err != nil {
		t.Fatalf("Failed to write mock nmap output: %v", err)
	}

	return outputPath
}

// mockCommand creates a mock command that will be used instead of the real nmap
func mockCommand(t *testing.T, tempDir string) {
	// Create a mock nmap script
	mockScript := `#!/bin/sh
# Mock nmap that writes a pre-generated output file
# The second argument should be the output file path
echo "Mock nmap running..."
cp ` + filepath.Join(tempDir, "mock_scan.xml") + ` $2
`
	// Write the mock script to a temporary file
	scriptPath := filepath.Join(tempDir, "nmap")
	err := ioutil.WriteFile(scriptPath, []byte(mockScript), 0755)
	if err != nil {
		t.Fatalf("Failed to write mock nmap script: %v", err)
	}

	// Add the temporary directory to PATH so our mock nmap is found first
	os.Setenv("PATH", tempDir+":"+os.Getenv("PATH"))
}

// setupTestEnvironment creates a test environment for the scanner tests
func setupTestEnvironment(t *testing.T) (string, *config.Config, *database.DB, *ScanService) {
	// Create a temporary directory for the test
	tempDir, err := ioutil.TempDir("", "scanner-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Create subdirectories
	os.MkdirAll(filepath.Join(tempDir, "scans"), 0755)
	os.MkdirAll(filepath.Join(tempDir, "data"), 0755)

	// Setup mock nmap output
	mockNmapOutput(t, tempDir)

	// Setup mock nmap command
	mockCommand(t, tempDir)

	// Setup config
	cfg := config.GetConfig()
	cfg.Scanner.OutputDir = filepath.Join(tempDir, "scans")
	cfg.Scanner.TargetNetwork = "192.168.1.0/24"
	cfg.Scanner.CompressOutput = false // Disable compression for testing
	cfg.Scanner.EnableScheduler = false // Disable scheduler for testing
	cfg.Scanner.DefaultTemplate = "default"
	cfg.Database.Path = filepath.Join(tempDir, "data", "test.db")

	// Setup database
	db, err := database.New(cfg.Database.Path)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}

	// Create scanner service
	scanService := New(cfg, db)

	return tempDir, cfg, db, scanService
}

// TestNew tests the creation of a new scanner service
func TestNew(t *testing.T) {
	cfg := config.GetConfig()
	db, err := database.New(":memory:") // Use in-memory SQLite for this test
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Test creating a new scanner service
	scanService := New(cfg, db)
	if scanService == nil {
		t.Fatal("Failed to create scanner service")
	}

	// Check defaults
	if scanService.scanStats.Status != "idle" {
		t.Errorf("Expected initial status 'idle', got '%s'", scanService.scanStats.Status)
	}

	if scanService.isScanning {
		t.Errorf("Expected isScanning to be false, got true")
	}
}

// TestStart tests starting the scanner service
func TestStart(t *testing.T) {
	tempDir, _, db, scanService := setupTestEnvironment(t)
	defer os.RemoveAll(tempDir)
	defer db.Close()

	// Test starting the scanner service
	err := scanService.Start()
	if err != nil {
		t.Errorf("Failed to start scanner service: %v", err)
	}

	// Make sure output directory exists
	if _, err := os.Stat(scanService.config.Scanner.OutputDir); os.IsNotExist(err) {
		t.Errorf("Output directory was not created: %v", err)
	}

	// Stop the service
	err = scanService.Stop()
	if err != nil {
		t.Errorf("Failed to stop scanner service: %v", err)
	}
}

// TestGetScanTemplates tests retrieving scan templates
func TestGetScanTemplates(t *testing.T) {
	tempDir, _, db, scanService := setupTestEnvironment(t)
	defer os.RemoveAll(tempDir)
	defer db.Close()

	// Test getting scan templates
	templates, err := scanService.GetScanTemplates()
	if err != nil {
		t.Errorf("Failed to get scan templates: %v", err)
	}

	// Check that we have templates
	if len(templates) == 0 {
		t.Errorf("Expected at least one scan template, got none")
	}

	// Check that default template exists
	var hasDefault bool
	for _, template := range templates {
		if template.ID == "default" {
			hasDefault = true
			break
		}
	}
	if !hasDefault {
		t.Errorf("Default template not found in templates")
	}
}

// TestRunScan tests running a network scan
func TestRunScan(t *testing.T) {
	tempDir, _, db, scanService := setupTestEnvironment(t)
	defer os.RemoveAll(tempDir)
	defer db.Close()

	// Start scanner service
	err := scanService.Start()
	if err != nil {
		t.Fatalf("Failed to start scanner service: %v", err)
	}

	// Run a scan
	scanID, err := scanService.RunScan(context.Background(), "default")
	if err != nil {
		t.Fatalf("Failed to run scan: %v", err)
	}

	if scanID <= 0 {
		t.Errorf("Expected positive scan ID, got %d", scanID)
	}

	// Check scan status after completion
	status := scanService.GetStatus()
	if status.Status != "completed" {
		t.Errorf("Expected status 'completed', got '%s'", status.Status)
	}

	if status.DevicesFound != 2 { // Our mock XML has 2 hosts
		t.Errorf("Expected 2 devices found, got %d", status.DevicesFound)
	}

	if status.PortsFound != 5 { // Our mock XML has 5 open ports total
		t.Errorf("Expected 5 ports found, got %d", status.PortsFound)
	}

	// Verify scan was recorded in database
	scan, err := db.GetScan(scanID)
	if err != nil {
		t.Errorf("Failed to get scan from database: %v", err)
	}

	if scan.ID != scanID {
		t.Errorf("Expected scan ID %d, got %d", scanID, scan.ID)
	}

	if scan.Status != "completed" {
		t.Errorf("Expected scan status 'completed', got '%s'", scan.Status)
	}

	if scan.DevicesFound != 2 {
		t.Errorf("Expected scan devices found 2, got %d", scan.DevicesFound)
	}

	if scan.PortsFound != 5 {
		t.Errorf("Expected scan ports found 5, got %d", scan.PortsFound)
	}
}

// TestRunManualScan tests running a manual scan with parameters
func TestRunManualScan(t *testing.T) {
	tempDir, _, db, scanService := setupTestEnvironment(t)
	defer os.RemoveAll(tempDir)
	defer db.Close()

	// Start scanner service
	err := scanService.Start()
	if err != nil {
		t.Fatalf("Failed to start scanner service: %v", err)
	}

	// Run a manual scan with parameters
	params := models.ScanParameters{
		Template: "quick", // Use a different template
	}

	scanID, err := scanService.RunManualScan(context.Background(), params)
	if err != nil {
		t.Fatalf("Failed to run manual scan: %v", err)
	}

	if scanID <= 0 {
		t.Errorf("Expected positive scan ID, got %d", scanID)
	}

	// Check scan status after completion
	status := scanService.GetStatus()
	if status.Status != "completed" {
		t.Errorf("Expected status 'completed', got '%s'", status.Status)
	}

	// Verify scan was recorded in database with the specified template
	scan, err := db.GetScan(scanID)
	if err != nil {
		t.Errorf("Failed to get scan from database: %v", err)
	}

	if scan.Template != "quick" {
		t.Errorf("Expected template 'quick', got '%s'", scan.Template)
	}
}

// TestGetStatus tests retrieving scanner status
func TestGetStatus(t *testing.T) {
	tempDir, _, db, scanService := setupTestEnvironment(t)
	defer os.RemoveAll(tempDir)
	defer db.Close()

	// Initial status should be idle
	status := scanService.GetStatus()
	if status.Status != "idle" {
		t.Errorf("Expected initial status 'idle', got '%s'", status.Status)
	}

	// Run a scan to change the status
	err := scanService.Start()
	if err != nil {
		t.Fatalf("Failed to start scanner service: %v", err)
	}

	// Start a scan in a goroutine
	go func() {
		scanService.RunScan(context.Background(), "default")
	}()

	// Wait a moment for scan to start
	time.Sleep(100 * time.Millisecond)

	// Status during scan should be running or completed
	status = scanService.GetStatus()
	if status.Status != "running" && status.Status != "completed" {
		t.Errorf("Expected status 'running' or 'completed', got '%s'", status.Status)
	}

	// Wait for scan to complete
	time.Sleep(500 * time.Millisecond)

	// Final status should be completed
	status = scanService.GetStatus()
	if status.Status != "completed" {
		t.Errorf("Expected final status 'completed', got '%s'", status.Status)
	}
}

// TestConcurrentScans tests that only one scan can run at a time
func TestConcurrentScans(t *testing.T) {
	tempDir, _, db, scanService := setupTestEnvironment(t)
	defer os.RemoveAll(tempDir)
	defer db.Close()

	// Start scanner service
	err := scanService.Start()
	if err != nil {
		t.Fatalf("Failed to start scanner service: %v", err)
	}

	// Start first scan
	_, err = scanService.RunScan(context.Background(), "default")
	if err != nil {
		t.Fatalf("Failed to run first scan: %v", err)
	}

	// Set isScanning back to true to simulate scan in progress
	scanService.scanLock.Lock()
	scanService.isScanning = true
	scanService.scanStats.Status = "running"
	scanService.scanLock.Unlock()

	// Try to start a second scan
	_, err = scanService.RunScan(context.Background(), "default")
	if err == nil {
		t.Errorf("Expected error when starting concurrent scan, got nil")
	}

	if !strings.Contains(err.Error(), "already in progress") {
		t.Errorf("Expected 'already in progress' error, got: %v", err)
	}
}

// TestGetScan tests retrieving a scan by ID
func TestGetScan(t *testing.T) {
	tempDir, _, db, scanService := setupTestEnvironment(t)
	defer os.RemoveAll(tempDir)
	defer db.Close()

	// Start scanner service
	err := scanService.Start()
	if err != nil {
		t.Fatalf("Failed to start scanner service: %v", err)
	}

	// Run a scan
	scanID, err := scanService.RunScan(context.Background(), "default")
	if err != nil {
		t.Fatalf("Failed to run scan: %v", err)
	}

	// Retrieve the scan
	scan, err := scanService.GetScan(scanID)
	if err != nil {
		t.Errorf("Failed to get scan: %v", err)
	}

	if scan.ID != scanID {
		t.Errorf("Expected scan ID %d, got %d", scanID, scan.ID)
	}

	// Try to get a non-existent scan
	_, err = scanService.GetScan(9999)
	if err == nil {
		t.Errorf("Expected error when getting non-existent scan, got nil")
	}
}

// TestGetRecentScans tests retrieving recent scans
func TestGetRecentScans(t *testing.T) {
	tempDir, _, db, scanService := setupTestEnvironment(t)
	defer os.RemoveAll(tempDir)
	defer db.Close()

	// Start scanner service
	err := scanService.Start()
	if err != nil {
		t.Fatalf("Failed to start scanner service: %v", err)
	}

	// Run multiple scans
	for i := 0; i < 3; i++ {
		_, err := scanService.RunScan(context.Background(), "default")
		if err != nil {
			t.Fatalf("Failed to run scan %d: %v", i, err)
		}
	}

	// Get recent scans
	scans, err := scanService.GetRecentScans(2)
	if err != nil {
		t.Errorf("Failed to get recent scans: %v", err)
	}

	// Should get exactly 2 scans due to limit
	if len(scans) != 2 {
		t.Errorf("Expected 2 recent scans, got %d", len(scans))
	}

	// Scans should be in descending order by timestamp
	if len(scans) >= 2 && scans[0].Timestamp.Before(scans[1].Timestamp) {
		t.Errorf("Expected scans in descending order by timestamp")
	}
}

// TestClean tests cleaning up old scan data
func TestClean(t *testing.T) {
	tempDir, cfg, db, scanService := setupTestEnvironment(t)
	defer os.RemoveAll(tempDir)
	defer db.Close()

	// Set a short retention period for testing
	cfg.Scanner.OutputRetentionDays = 1

	// Create a fake old scan file
	oldFilePath := filepath.Join(cfg.Scanner.OutputDir, "old_scan.xml")
	err := ioutil.WriteFile(oldFilePath, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Set modification time to 2 days ago
	oldTime := time.Now().Add(-48 * time.Hour)
	err = os.Chtimes(oldFilePath, oldTime, oldTime)
	if err != nil {
		t.Fatalf("Failed to set file time: %v", err)
	}

	// Create a newer scan file
	newFilePath := filepath.Join(cfg.Scanner.OutputDir, "new_scan.xml")
	err = ioutil.WriteFile(newFilePath, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Run the cleanup function
	err = scanService.Clean()
	if err != nil {
		t.Errorf("Clean returned error: %v", err)
	}

	// The old file should be deleted
	if _, err := os.Stat(oldFilePath); !os.IsNotExist(err) {
		t.Errorf("Expected old file to be deleted, but it still exists")
	}

	// The new file should still exist
	if _, err := os.Stat(newFilePath); os.IsNotExist(err) {
		t.Errorf("New file was unexpectedly deleted")
	}
}

// TestProcessScanResults tests processing nmap scan results
func TestProcessScanResults(t *testing.T) {
	tempDir, _, db, scanService := setupTestEnvironment(t)
	defer os.RemoveAll(tempDir)
	defer db.Close()

	// Get the mock nmap output file
	outputPath := mockNmapOutput(t, tempDir)

	// Process the scan results directly
	deviceCount, portCount, err := scanService.processScanResults(outputPath)
	if err != nil {
		t.Errorf("Failed to process scan results: %v", err)
	}

	// Verify the counts
	if deviceCount != 2 {
		t.Errorf("Expected 2 devices processed, got %d", deviceCount)
	}

	if portCount != 5 {
		t.Errorf("Expected 5 ports processed, got %d", portCount)
	}

	// Verify that devices were stored in the database
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM devices").Scan(&count)
	if err != nil {
		t.Errorf("Failed to count devices: %v", err)
	}

	if count != 2 {
		t.Errorf("Expected 2 devices in database, got %d", count)
	}

	// Verify that ports were stored in the database
	err = db.QueryRow("SELECT COUNT(*) FROM ports").Scan(&count)
	if err != nil {
		t.Errorf("Failed to count ports: %v", err)
	}

	if count != 5 {
		t.Errorf("Expected 5 ports in database, got %d", count)
	}
}
