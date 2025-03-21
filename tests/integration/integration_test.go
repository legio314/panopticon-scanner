// tests/integration/integration_test.go
package integration

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gorilla/mux"

	"panopticon-scanner/internal/api"
	"panopticon-scanner/internal/config"
	"panopticon-scanner/internal/database"
	"panopticon-scanner/internal/models"
	"panopticon-scanner/internal/scanner"
)

// setupTestEnvironment creates an integration test environment
func setupTestEnvironment(t *testing.T) (string, *config.Config, *database.DB, *scanner.ScanService, http.Handler) {
	// Create a temporary directory for the test
	tempDir, err := os.MkdirTemp("", "panopticon-integration-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Create subdirectories
	os.MkdirAll(filepath.Join(tempDir, "scans"), 0755)
	os.MkdirAll(filepath.Join(tempDir, "data"), 0755)
	os.MkdirAll(filepath.Join(tempDir, "logs"), 0755)
	os.MkdirAll(filepath.Join(tempDir, "reports"), 0755)

	// Setup configuration
	cfg := config.GetConfig()
	cfg.Server.Port = 8081 // Use different port than main app
	cfg.Scanner.OutputDir = filepath.Join(tempDir, "scans")
	cfg.Scanner.TargetNetwork = "192.168.1.0/24"
	cfg.Scanner.CompressOutput = false // Disable compression for testing
	cfg.Scanner.EnableScheduler = false // Disable scheduler for testing
	cfg.Scanner.DefaultTemplate = "default"
	cfg.Database.Path = filepath.Join(tempDir, "data", "test.db")
	cfg.Logging.OutputPath = filepath.Join(tempDir, "logs", "test.log")
	cfg.Reporting.OutputDir = filepath.Join(tempDir, "reports")

	// Setup database
	db, err := database.New(cfg.Database.Path)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}

	// Create scanner service
	scanService := scanner.New(cfg, db)
	err = scanService.Start()
	if err != nil {
		t.Fatalf("Failed to start scanner service: %v", err)
	}

	// Setup API
	router := mux.NewRouter()

	// Register API handlers
	scanHandler := api.NewScanHandler(scanService)
	deviceHandler := api.NewDeviceHandler(db)

	// Register routes with order that respects route specificity
	// Order matters here - more specific routes should be registered before general ones
	router.HandleFunc("/api/devices/search", deviceHandler.SearchDevices).Methods("GET")
	router.HandleFunc("/api/devices/stats", deviceHandler.GetDeviceStats).Methods("GET")
	router.HandleFunc("/api/scans/status", scanHandler.GetScanStatus).Methods("GET")
	router.HandleFunc("/api/scans/templates", scanHandler.GetScanTemplates).Methods("GET")
	
	// Then register the remaining routes
	scanHandler.RegisterRoutes(router)
	deviceHandler.RegisterRoutes(router)

	return tempDir, cfg, db, scanService, router
}

// teardownTestEnvironment cleans up the test environment
func teardownTestEnvironment(tempDir string, db *database.DB) {
	if db != nil {
		db.Close()
	}
	os.RemoveAll(tempDir)
}

// createTestData creates test device and scan data in the database
func createTestData(t *testing.T, db *database.DB) {
	// Create devices
	devices := []models.Device{
		{
			IPAddress:     "192.168.1.1",
			MACAddress:    "00:11:22:33:44:55",
			Hostname:      "router.test",
			OSFingerprint: "Linux",
			FirstSeen:     time.Now().Add(-48 * time.Hour),
			LastSeen:      time.Now(),
		},
		{
			IPAddress:     "192.168.1.100",
			MACAddress:    "AA:BB:CC:DD:EE:FF",
			Hostname:      "desktop.test",
			OSFingerprint: "Windows",
			FirstSeen:     time.Now().Add(-24 * time.Hour),
			LastSeen:      time.Now(),
		},
	}

	for i, device := range devices {
		deviceID, err := db.SaveDevice(&device)
		if err != nil {
			t.Fatalf("Failed to create test device %d: %v", i, err)
		}

		// Add ports to device
		ports := []models.Port{
			{
				DeviceID:       deviceID,
				PortNumber:     22,
				Protocol:       "tcp",
				ServiceName:    "ssh",
				ServiceVersion: "OpenSSH 8.4p1",
				FirstSeen:      time.Now().Add(-48 * time.Hour),
				LastSeen:       time.Now(),
			},
			{
				DeviceID:       deviceID,
				PortNumber:     80,
				Protocol:       "tcp",
				ServiceName:    "http",
				ServiceVersion: "nginx 1.18.0",
				FirstSeen:      time.Now().Add(-48 * time.Hour),
				LastSeen:       time.Now(),
			},
		}

		for j, port := range ports {
			err := db.SavePort(&port)
			if err != nil {
				t.Fatalf("Failed to create test port %d for device %d: %v", j, i, err)
			}
		}
	}

	// Create scan records
	scans := []models.Scan{
		{
			Timestamp:    time.Now().Add(-48 * time.Hour),
			Template:     "default",
			Duration:     60,
			DevicesFound: 1,
			PortsFound:   2,
			Status:       "completed",
		},
		{
			Timestamp:    time.Now().Add(-24 * time.Hour),
			Template:     "default",
			Duration:     65,
			DevicesFound: 2,
			PortsFound:   4,
			Status:       "completed",
		},
	}

	for i, scan := range scans {
		// Create scan with the template
		scanID, err := db.CreateScan(scan.Template)
		if err != nil {
			t.Fatalf("Failed to create test scan %d: %v", i, err)
		}
		
		// Update with details
		err = db.UpdateScan(
			scanID,
			scan.Status,
			scan.DevicesFound,
			scan.PortsFound,
			time.Duration(scan.Duration) * time.Second,
			scan.ErrorMessage,
		)
		if err != nil {
			t.Fatalf("Failed to update test scan %d: %v", i, err)
		}
	}
}

// TestAPIIntegration tests the entire API flow
func TestAPIIntegration(t *testing.T) {
	tempDir, _, db, _, router := setupTestEnvironment(t)
	defer teardownTestEnvironment(tempDir, db)

	// Create test data
	createTestData(t, db)

	// Create a test server
	server := httptest.NewServer(router)
	defer server.Close()

	// Test API endpoints
	t.Run("GetDevices", func(t *testing.T) {
		// Make request to get devices
		resp, err := http.Get(fmt.Sprintf("%s/api/devices", server.URL))
		if err != nil {
			t.Fatalf("Failed to get devices: %v", err)
		}
		defer resp.Body.Close()

		// Check status code
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status OK, got %v", resp.Status)
		}

		// Parse response
		var devices []models.Device
		if err := json.NewDecoder(resp.Body).Decode(&devices); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		// Verify response
		if len(devices) != 2 {
			t.Errorf("Expected 2 devices, got %d", len(devices))
		}
	})

	t.Run("GetScans", func(t *testing.T) {
		// Make request to get scans
		resp, err := http.Get(fmt.Sprintf("%s/api/scans", server.URL))
		if err != nil {
			t.Fatalf("Failed to get scans: %v", err)
		}
		defer resp.Body.Close()

		// Check status code
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status OK, got %v", resp.Status)
		}

		// Parse response
		var scans []*models.Scan
		if err := json.NewDecoder(resp.Body).Decode(&scans); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		// Verify response
		if len(scans) != 2 {
			t.Errorf("Expected 2 scans, got %d", len(scans))
		}
	})

	t.Run("GetDeviceDetail", func(t *testing.T) {
		// Get a device ID first
		resp, err := http.Get(fmt.Sprintf("%s/api/devices", server.URL))
		if err != nil {
			t.Fatalf("Failed to get devices: %v", err)
		}

		var devices []models.Device
		if err := json.NewDecoder(resp.Body).Decode(&devices); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}
		resp.Body.Close()

		if len(devices) == 0 {
			t.Fatalf("No devices found for detail test")
		}

		deviceID := devices[0].ID

		// Make request to get device detail
		resp, err = http.Get(fmt.Sprintf("%s/api/devices/%d", server.URL, deviceID))
		if err != nil {
			t.Fatalf("Failed to get device detail: %v", err)
		}
		defer resp.Body.Close()

		// Check status code
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status OK, got %v", resp.Status)
		}

		// Parse response
		var deviceDetail models.DeviceDetails
		if err := json.NewDecoder(resp.Body).Decode(&deviceDetail); err != nil {
			t.Fatalf("Failed to decode device detail response: %v", err)
		}

		// Verify response
		if deviceDetail.ID != deviceID {
			t.Errorf("Expected device ID %d, got %d", deviceID, deviceDetail.ID)
		}

		if len(deviceDetail.Ports) == 0 {
			t.Errorf("Expected ports in device detail, got none")
		}
	})

	t.Run("GetScanTemplates", func(t *testing.T) {
		// Make request to get scan templates
		resp, err := http.Get(fmt.Sprintf("%s/api/scans/templates", server.URL))
		if err != nil {
			t.Fatalf("Failed to get scan templates: %v", err)
		}
		defer resp.Body.Close()

		// Check status code
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status OK, got %v", resp.Status)
		}

		// Parse response
		var templates []models.ScanTemplate
		if err := json.NewDecoder(resp.Body).Decode(&templates); err != nil {
			t.Fatalf("Failed to decode templates response: %v", err)
		}

		// Verify response
		if len(templates) == 0 {
			t.Errorf("Expected scan templates, got none")
		}

		// Check for default template
		var hasDefault bool
		for _, template := range templates {
			if template.ID == "default" {
				hasDefault = true
				break
			}
		}
		if !hasDefault {
			t.Errorf("Default template not found")
		}
	})

	t.Run("SearchDevices", func(t *testing.T) {
		// Make request to search for devices
		resp, err := http.Get(fmt.Sprintf("%s/api/devices/search?q=router", server.URL))
		if err != nil {
			t.Fatalf("Failed to search devices: %v", err)
		}
		defer resp.Body.Close()

		// Check status code
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status OK, got %v", resp.Status)
		}

		// Parse response
		var devices []models.Device
		if err := json.NewDecoder(resp.Body).Decode(&devices); err != nil {
			t.Fatalf("Failed to decode search response: %v", err)
		}

		// Should find at least one device
		if len(devices) == 0 {
			t.Errorf("Expected to find device with 'router', got none")
		}

		// Verify the device has 'router' in hostname
		foundRouter := false
		for _, device := range devices {
			if device.Hostname == "router.test" {
				foundRouter = true
				break
			}
		}
		if !foundRouter {
			t.Errorf("Expected to find device with hostname 'router.test'")
		}
	})
}

// TestScanWorkflow tests the complete scanning workflow
func TestScanWorkflow(t *testing.T) {
	tempDir, _, db, scanService, router := setupTestEnvironment(t)
	defer teardownTestEnvironment(tempDir, db)

	// Create a test server
	server := httptest.NewServer(router)
	defer server.Close()

	// Mock the scan function to not actually run nmap
	scanService.SetMockModeForTesting(true)

	// Start a scan
	t.Run("StartScan", func(t *testing.T) {
		// Make request to start scan
		resp, err := http.Post(fmt.Sprintf("%s/api/scans", server.URL), "application/json", nil)
		if err != nil {
			t.Fatalf("Failed to start scan: %v", err)
		}
		defer resp.Body.Close()

		// Should return 202 Accepted
		if resp.StatusCode != http.StatusAccepted {
			t.Errorf("Expected status Accepted, got %v", resp.Status)
		}

		// Parse response
		var response map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode start scan response: %v", err)
		}

		// Verify response fields
		if message, ok := response["message"]; !ok || message != "Scan started" {
			t.Errorf("Expected message 'Scan started', got %v", message)
		}
	})

	// Wait for scan to complete
	time.Sleep(500 * time.Millisecond)

	// Check scan status
	t.Run("GetScanStatus", func(t *testing.T) {
		// Make request to get scan status
		resp, err := http.Get(fmt.Sprintf("%s/api/scans/status", server.URL))
		if err != nil {
			t.Fatalf("Failed to get scan status: %v", err)
		}
		defer resp.Body.Close()

		// Check status code
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status OK, got %v", resp.Status)
		}

		// Parse response
		var status map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
			t.Fatalf("Failed to decode status response: %v", err)
		}

		// In mock mode, status should be 'completed'
		if scanStatus, ok := status["status"]; !ok || (scanStatus != "completed" && scanStatus != "idle") {
			t.Errorf("Expected status 'completed' or 'idle', got %v", scanStatus)
		}
	})

	// Get list of scans to verify the new scan is recorded
	t.Run("VerifyScanRecord", func(t *testing.T) {
		// Make request to get scans
		resp, err := http.Get(fmt.Sprintf("%s/api/scans", server.URL))
		if err != nil {
			t.Fatalf("Failed to get scans: %v", err)
		}
		defer resp.Body.Close()

		// Parse response
		var scans []*models.Scan
		if err := json.NewDecoder(resp.Body).Decode(&scans); err != nil {
			t.Fatalf("Failed to decode scans response: %v", err)
		}

		// Should have at least one scan
		if len(scans) == 0 {
			t.Errorf("Expected at least one scan, got none")
		}

		// Verify the most recent scan has expected status
		if len(scans) > 0 {
			latestScan := scans[0] // Scans are returned in descending order by timestamp
			if latestScan.Status != "completed" && latestScan.Status != "error" {
				t.Errorf("Expected latest scan status 'completed' or 'error', got %s", latestScan.Status)
			}
		}
	})
}

// TestDatabaseMaintenanceIntegration tests the database maintenance operations
func TestDatabaseMaintenanceIntegration(t *testing.T) {
	tempDir, _, db, _, _ := setupTestEnvironment(t)
	defer teardownTestEnvironment(tempDir, db)

	// Create test data
	createTestData(t, db)

	// Test database optimization
	t.Run("OptimizeDatabase", func(t *testing.T) {
		// Perform database optimization
		err := db.OptimizeDatabase()
		if err != nil {
			t.Errorf("Database optimization failed: %v", err)
		}

		// Verify database is still functional by querying a table
		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM devices").Scan(&count)
		if err != nil {
			t.Errorf("Failed to query database after optimization: %v", err)
		}

		// Should still have our test devices
		if count != 2 {
			t.Errorf("Expected 2 devices after optimization, got %d", count)
		}
	})

	// Test database backup
	t.Run("BackupDatabase", func(t *testing.T) {
		// Create backup directory
		backupDir := filepath.Join(tempDir, "backups")
		os.MkdirAll(backupDir, 0755)

		// Perform database backup
		// Set db.Path to include the test directory for backup
		db.Path = filepath.Join(tempDir, "test.db")
		
		// Perform backup
		backupPath, err := db.BackupDatabase()
		if err != nil {
			t.Errorf("Database backup failed: %v", err)
		}

		// Verify backup file exists
		if _, err := os.Stat(backupPath); os.IsNotExist(err) {
			t.Errorf("Backup file does not exist: %s", backupPath)
		}

		// Verify backup is a valid SQLite database
		backupDB, err := sql.Open("sqlite3", backupPath)
		if err != nil {
			t.Errorf("Failed to open backup database: %v", err)
		}
		defer backupDB.Close()

		// Verify backup contains our data
		var count int
		err = backupDB.QueryRow("SELECT COUNT(*) FROM devices").Scan(&count)
		if err != nil {
			t.Errorf("Failed to query backup database: %v", err)
		}

		// Should have our test devices
		if count != 2 {
			t.Errorf("Expected 2 devices in backup, got %d", count)
		}
	})

	// Test data retention policies
	t.Run("CleanOldData", func(t *testing.T) {
		// Add old data that should be cleaned up
		oldTime := time.Now().Add(-365 * 24 * time.Hour) // 1 year old

		oldDevice := models.Device{
			IPAddress:     "192.168.1.200",
			MACAddress:    "11:22:33:44:55:66",
			Hostname:      "old-device.test",
			OSFingerprint: "FreeBSD",
			FirstSeen:     oldTime,
			LastSeen:      oldTime,
		}

		deviceID, err := db.SaveDevice(&oldDevice)
		if err != nil {
			t.Fatalf("Failed to create old test device: %v", err)
		}

		// Mark old device as seen only once a year ago
		_, err = db.Exec("UPDATE devices SET last_seen = ? WHERE id = ?", oldTime, deviceID)
		if err != nil {
			t.Fatalf("Failed to update device last_seen: %v", err)
		}

		// Set retention policy to 6 months
		retentionDays := 180

		// Execute data cleanup
		affected, err := db.CleanOldData(retentionDays)
		if err != nil {
			t.Errorf("Data cleanup failed: %v", err)
		}

		// Should have cleaned up one device
		if affected < 1 {
			t.Errorf("Expected at least 1 device to be cleaned up, got %d", affected)
		}

		// Verify old device is gone
		var exists bool
		err = db.QueryRow("SELECT 1 FROM devices WHERE ip_address = ?", oldDevice.IPAddress).Scan(&exists)
		if err != sql.ErrNoRows {
			t.Errorf("Old device was not cleaned up properly")
		}

		// Verify current devices still exist
		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM devices WHERE hostname IN ('router.test', 'desktop.test')").Scan(&count)
		if err != nil {
			t.Errorf("Failed to query devices after cleanup: %v", err)
		}

		// Should still have our 2 current test devices
		if count != 2 {
			t.Errorf("Expected 2 current devices after cleanup, got %d", count)
		}
	})
}

// Note: All testing methods have been moved to the scanner package
