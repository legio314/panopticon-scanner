// internal/api/scan_handlers_test.go
package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gorilla/mux"

	"panopticon-scanner/internal/config"
	"panopticon-scanner/internal/database"
	"panopticon-scanner/internal/models"
	"panopticon-scanner/internal/scanner"
)

// setupTestEnvironment creates a test environment for the API tests
func setupTestEnvironment(t *testing.T) (string, *config.Config, *database.DB, *scanner.ScanService, *ScanHandler) {
	// Create a temporary directory for the test
	tempDir, err := os.MkdirTemp("", "api-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Create subdirectories
	os.MkdirAll(filepath.Join(tempDir, "scans"), 0755)
	os.MkdirAll(filepath.Join(tempDir, "data"), 0755)

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
	scanService := scanner.New(cfg, db)
	err = scanService.Start()
	if err != nil {
		t.Fatalf("Failed to start scanner service: %v", err)
	}

	// Create scan handler
	scanHandler := NewScanHandler(scanService)

	return tempDir, cfg, db, scanService, scanHandler
}

// createTestScans creates test scans in the database
func createTestScans(t *testing.T, db *database.DB, count int) []int64 {
	var scanIDs []int64

	for i := 0; i < count; i++ {
		// Create a scan with test data
		scan := &models.Scan{
			Timestamp:    time.Now().Add(-time.Duration(i) * time.Hour), // Spaced 1 hour apart
			Template:     "default",
			Duration:     60 + i*10, // Different durations
			DevicesFound: 10 + i,    // Different device counts
			PortsFound:   50 + i*5,  // Different port counts
			Status:       "completed",
		}

		// Create scan with template
		scanID, err := db.CreateScan(scan.Template)
		if err != nil {
			t.Fatalf("Failed to create test scan with template: %v", err)
		}
		
		// Update with details
		err = db.UpdateScan(
			scanID,
			scan.Status,
			scan.DevicesFound,
			scan.PortsFound,
			time.Duration(scan.Duration) * time.Second,
			"",
		)
		if err != nil {
			t.Fatalf("Failed to update test scan: %v", err)
		}

		scanIDs = append(scanIDs, scanID)
	}

	return scanIDs
}

// TestGetScans tests the getScans handler
func TestGetScans(t *testing.T) {
	tempDir, _, db, _, scanHandler := setupTestEnvironment(t)
	defer os.RemoveAll(tempDir)
	defer db.Close()

	// Create test scans
	createTestScans(t, db, 5)

	// Create a request to the getScans handler
	req, err := http.NewRequest("GET", "/api/scans", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Record the response
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(scanHandler.getScans)

	// Serve the request
	handler.ServeHTTP(rr, req)

	// Check the status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Check the content type
	contentType := rr.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Handler returned wrong content type: got %v want %v", contentType, "application/json")
	}

	// Parse the response
	var scans []*models.Scan
	if err := json.Unmarshal(rr.Body.Bytes(), &scans); err != nil {
		t.Errorf("Failed to parse response: %v", err)
	}

	// Check that we got the expected number of scans (default limit is 10)
	if len(scans) != 5 {
		t.Errorf("Expected 5 scans, got %d", len(scans))
	}

	// Test with limit parameter
	req, err = http.NewRequest("GET", "/api/scans?limit=2", nil)
	if err != nil {
		t.Fatalf("Failed to create request with limit: %v", err)
	}

	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler with limit returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	if err := json.Unmarshal(rr.Body.Bytes(), &scans); err != nil {
		t.Errorf("Failed to parse response with limit: %v", err)
	}

	if len(scans) != 2 {
		t.Errorf("Expected 2 scans with limit, got %d", len(scans))
	}
}

// TestGetScan tests the getScan handler
func TestGetScan(t *testing.T) {
	tempDir, _, db, _, scanHandler := setupTestEnvironment(t)
	defer os.RemoveAll(tempDir)
	defer db.Close()

	// Create test scans
	scanIDs := createTestScans(t, db, 1)
	scanID := scanIDs[0]

	// Create a router to use with the route parameters
	router := mux.NewRouter()
	router.HandleFunc("/api/scans/{id}", scanHandler.getScan).Methods("GET")

	// Create a request to the getScan handler
	req, err := http.NewRequest("GET", fmt.Sprintf("/api/scans/%d", scanID), nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Record the response
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	// Check the status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Parse the response
	var scan models.Scan
	if err := json.Unmarshal(rr.Body.Bytes(), &scan); err != nil {
		t.Errorf("Failed to parse response: %v", err)
	}

	// Check the scan ID
	if scan.ID != scanID {
		t.Errorf("Expected scan ID %d, got %d", scanID, scan.ID)
	}

	// Test with non-existent scan ID
	req, err = http.NewRequest("GET", "/api/scans/9999", nil)
	if err != nil {
		t.Fatalf("Failed to create request with non-existent ID: %v", err)
	}

	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	// Should return 404 for non-existent scan
	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("Handler with non-existent ID returned wrong status code: got %v want %v", status, http.StatusNotFound)
	}

	// Test with invalid scan ID format
	req, err = http.NewRequest("GET", "/api/scans/invalid", nil)
	if err != nil {
		t.Fatalf("Failed to create request with invalid ID: %v", err)
	}

	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	// Should return 400 for invalid scan ID
	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("Handler with invalid ID format returned wrong status code: got %v want %v", status, http.StatusBadRequest)
	}
}

// TestStartScan tests the startScan handler
func TestStartScan(t *testing.T) {
	tempDir, _, db, scanService, scanHandler := setupTestEnvironment(t)
	defer os.RemoveAll(tempDir)
	defer db.Close()

	// Create a request to start a scan with default template
	req, err := http.NewRequest("POST", "/api/scans", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Record the response
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(scanHandler.startScan)

	// Mock the scan service to not actually run the scan
	originalStatus := scanService.GetStatus()

	// Serve the request
	handler.ServeHTTP(rr, req)

	// Check the status code - should be 202 Accepted
	if status := rr.Code; status != http.StatusAccepted {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusAccepted)
	}

	// Parse the response
	var response map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to parse response: %v", err)
	}

	// Check the response fields
	if msg, ok := response["message"]; !ok || msg != "Scan started" {
		t.Errorf("Expected message 'Scan started', got %v", msg)
	}

	if template, ok := response["template"]; !ok || template != "default" {
		t.Errorf("Expected template 'default', got %v", template)
	}

	// Reset any leftover scan status
	scanService.SetStatusForTesting("idle")

	// Test conflict case - mock a scan in progress
	scanService.SetStatusForTesting("running")

	req, err = http.NewRequest("POST", "/api/scans", nil)
	if err != nil {
		t.Fatalf("Failed to create request for conflict test: %v", err)
	}

	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// Should return 409 Conflict when scan is already in progress
	if status := rr.Code; status != http.StatusConflict {
		t.Errorf("Handler with scan in progress returned wrong status code: got %v want %v", status, http.StatusConflict)
	}

	// Reset the scan status for other tests
	scanService.SetStatusForTesting(originalStatus.Status)
}

// TestGetScanStatus tests the getScanStatus handler
func TestGetScanStatus(t *testing.T) {
	tempDir, _, db, scanService, scanHandler := setupTestEnvironment(t)
	defer os.RemoveAll(tempDir)
	defer db.Close()

	// Test with idle status
	scanService.SetStatusForTesting("idle")

	req, err := http.NewRequest("GET", "/api/scans/status", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(scanHandler.GetScanStatus)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to parse response: %v", err)
	}

	if status, ok := response["status"]; !ok || status != "idle" {
		t.Errorf("Expected status 'idle', got %v", status)
	}

	// Test with running status
	scanService.SetStatusForTesting("running")
	startTime := time.Now().Add(-30 * time.Second)
	scanService.SetStartTimeForTesting(startTime)
	scanService.SetScanIDForTesting(123)

	req, err = http.NewRequest("GET", "/api/scans/status", nil)
	if err != nil {
		t.Fatalf("Failed to create request for running status: %v", err)
	}

	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler with running status returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to parse running status response: %v", err)
	}

	if status, ok := response["status"]; !ok || status != "running" {
		t.Errorf("Expected status 'running', got %v", status)
	}

	if scanID, ok := response["scanID"]; !ok || int64(scanID.(float64)) != 123 {
		t.Errorf("Expected scanID 123, got %v", scanID)
	}

	// Test with completed status
	scanService.SetStatusForTesting("completed")
	scanService.SetDevicesFoundForTesting(10)
	scanService.SetPortsFoundForTesting(50)
	scanService.SetEndTimeForTesting(time.Now())

	req, err = http.NewRequest("GET", "/api/scans/status", nil)
	if err != nil {
		t.Fatalf("Failed to create request for completed status: %v", err)
	}

	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler with completed status returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to parse completed status response: %v", err)
	}

	if status, ok := response["status"]; !ok || status != "completed" {
		t.Errorf("Expected status 'completed', got %v", status)
	}

	if devicesFound, ok := response["devicesFound"]; !ok || int(devicesFound.(float64)) != 10 {
		t.Errorf("Expected devicesFound 10, got %v", devicesFound)
	}

	if portsFound, ok := response["portsFound"]; !ok || int(portsFound.(float64)) != 50 {
		t.Errorf("Expected portsFound 50, got %v", portsFound)
	}

	// Reset the scan status for other tests
	scanService.SetStatusForTesting("idle")
}

// TestGetScanTemplates tests the getScanTemplates handler
func TestGetScanTemplates(t *testing.T) {
	tempDir, _, db, _, scanHandler := setupTestEnvironment(t)
	defer os.RemoveAll(tempDir)
	defer db.Close()

	req, err := http.NewRequest("GET", "/api/scans/templates", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(scanHandler.GetScanTemplates)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var templates []models.ScanTemplate
	if err := json.Unmarshal(rr.Body.Bytes(), &templates); err != nil {
		t.Errorf("Failed to parse response: %v", err)
	}

	// Check that we got templates
	if len(templates) == 0 {
		t.Errorf("Expected at least one template, got none")
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

// Note: All testing helper methods have been moved to the scanner package
