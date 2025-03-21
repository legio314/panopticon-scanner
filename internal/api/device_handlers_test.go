// internal/api/device_handlers_test.go
package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gorilla/mux"

	"panopticon-scanner/internal/database"
	"panopticon-scanner/internal/models"
)

// createTestDevices creates test devices in the database
func createTestDevices(t *testing.T, db *database.DB, count int) []int64 {
	var deviceIDs []int64

	for i := 0; i < count; i++ {
		// Create a device with test data
		device := &models.Device{
			IPAddress:     fmt.Sprintf("192.168.1.%d", 10+i),
			MACAddress:    fmt.Sprintf("00:11:22:33:44:%02x", 10+i),
			Hostname:      fmt.Sprintf("device-%d", i),
			OSFingerprint: []string{"Linux", "Windows", "macOS"}[i%3],
			FirstSeen:     time.Now().Add(-time.Duration(i*24) * time.Hour),
			LastSeen:      time.Now().Add(-time.Duration(i) * time.Hour),
		}

		deviceID, err := db.SaveDevice(device)
		if err != nil {
			t.Fatalf("Failed to create test device: %v", err)
		}

		// Add some ports to the device
		for j := 0; j < 3+i%3; j++ {
			port := &models.Port{
				DeviceID:       deviceID,
				PortNumber:     []int{22, 80, 443, 3389, 8080}[j%5],
				Protocol:       "tcp",
				ServiceName:    []string{"ssh", "http", "https", "rdp", "http-alt"}[j%5],
				ServiceVersion: fmt.Sprintf("Version %d.%d", 1+j%3, j%5),
				FirstSeen:      time.Now().Add(-time.Duration(i*24) * time.Hour),
				LastSeen:       time.Now().Add(-time.Duration(i) * time.Hour),
			}
			err = db.SavePort(port)
			if err != nil {
				t.Fatalf("Failed to create test port: %v", err)
			}
		}

		deviceIDs = append(deviceIDs, deviceID)
	}

	return deviceIDs
}

// TestGetDevices tests the getDevices handler
func TestGetDevices(t *testing.T) {
	tempDir, _, db, _, _ := setupTestEnvironment(t)
	defer os.RemoveAll(tempDir)
	defer db.Close()

	// Create test devices
	deviceIDs := createTestDevices(t, db, 5)

	// Create device handler
	deviceHandler := NewDeviceHandler(db)

	// Create a request to the getDevices handler
	req, err := http.NewRequest("GET", "/api/devices", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Record the response
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(deviceHandler.getDevices)

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
	var devices []models.Device
	if err := json.Unmarshal(rr.Body.Bytes(), &devices); err != nil {
		t.Errorf("Failed to parse response: %v", err)
	}

	// Check that we got the expected number of devices
	if len(devices) != 5 {
		t.Errorf("Expected 5 devices, got %d", len(devices))
	}

	// Verify that the device IDs match
	deviceMap := make(map[int64]bool)
	for _, id := range deviceIDs {
		deviceMap[id] = true
	}

	for _, device := range devices {
		if !deviceMap[device.ID] {
			t.Errorf("Unexpected device ID in response: %d", device.ID)
		}

		// Verify basic device info is present
		if device.IPAddress == "" {
			t.Errorf("Device IP address is empty")
		}
		if device.Hostname == "" {
			t.Errorf("Device hostname is empty")
		}
		if device.PortCount <= 0 {
			t.Errorf("Expected positive port count, got %d", device.PortCount)
		}
	}
}

// TestGetDeviceDetail tests the getDeviceDetail handler
func TestGetDeviceDetail(t *testing.T) {
	tempDir, _, db, _, _ := setupTestEnvironment(t)
	defer os.RemoveAll(tempDir)
	defer db.Close()

	// Create test devices
	deviceIDs := createTestDevices(t, db, 1)
	deviceID := deviceIDs[0]

	// Create device handler
	deviceHandler := NewDeviceHandler(db)

	// Create a router to use with the route parameters
	router := mux.NewRouter()
	router.HandleFunc("/api/devices/{id}", deviceHandler.getDeviceDetail).Methods("GET")

	// Create a request to the getDeviceDetail handler
	req, err := http.NewRequest("GET", fmt.Sprintf("/api/devices/%d", deviceID), nil)
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
	var deviceDetail models.DeviceDetails
	if err := json.Unmarshal(rr.Body.Bytes(), &deviceDetail); err != nil {
		t.Errorf("Failed to parse response: %v", err)
	}

	// Check the device ID
	if deviceDetail.ID != deviceID {
		t.Errorf("Expected device ID %d, got %d", deviceID, deviceDetail.ID)
	}

	// Check that ports are included
	if len(deviceDetail.Ports) == 0 {
		t.Errorf("Expected ports in device detail, got none")
	}

	// Verify port information
	for _, port := range deviceDetail.Ports {
		if port.DeviceID != deviceID {
			t.Errorf("Port has incorrect device ID: expected %d, got %d", deviceID, port.DeviceID)
		}
		if port.PortNumber <= 0 {
			t.Errorf("Invalid port number: %d", port.PortNumber)
		}
		if port.Protocol == "" {
			t.Errorf("Port protocol is empty")
		}
	}

	// Test with non-existent device ID
	req, err = http.NewRequest("GET", "/api/devices/9999", nil)
	if err != nil {
		t.Fatalf("Failed to create request with non-existent ID: %v", err)
	}

	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	// Should return 404 for non-existent device
	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("Handler with non-existent ID returned wrong status code: got %v want %v", status, http.StatusNotFound)
	}

	// Test with invalid device ID format
	req, err = http.NewRequest("GET", "/api/devices/invalid", nil)
	if err != nil {
		t.Fatalf("Failed to create request with invalid ID: %v", err)
	}

	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	// Should return 400 for invalid device ID
	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("Handler with invalid ID format returned wrong status code: got %v want %v", status, http.StatusBadRequest)
	}
}

// TestSearchDevices tests the searchDevices handler
func TestSearchDevices(t *testing.T) {
	tempDir, _, db, _, _ := setupTestEnvironment(t)
	defer os.RemoveAll(tempDir)
	defer db.Close()

	// Create test devices with specific data for search testing
	device1 := &models.Device{
		IPAddress:     "192.168.1.100",
		MACAddress:    "00:11:22:33:44:55",
		Hostname:      "webserver",
		OSFingerprint: "Linux",
		FirstSeen:     time.Now().Add(-24 * time.Hour),
		LastSeen:      time.Now(),
	}
	device1ID, err := db.SaveDevice(device1)
	if err != nil {
		t.Fatalf("Failed to create test device 1: %v", err)
	}

	device2 := &models.Device{
		IPAddress:     "192.168.1.200",
		MACAddress:    "AA:BB:CC:DD:EE:FF",
		Hostname:      "database",
		OSFingerprint: "Windows",
		FirstSeen:     time.Now().Add(-48 * time.Hour),
		LastSeen:      time.Now().Add(-1 * time.Hour),
	}
	device2ID, err := db.SaveDevice(device2)
	if err != nil {
		t.Fatalf("Failed to create test device 2: %v", err)
	}

	// Add ports
	port1 := &models.Port{
		DeviceID:       device1ID,
		PortNumber:     80,
		Protocol:       "tcp",
		ServiceName:    "http",
		ServiceVersion: "nginx 1.18.0",
		FirstSeen:      time.Now().Add(-24 * time.Hour),
		LastSeen:       time.Now(),
	}
	err = db.SavePort(port1)
	if err != nil {
		t.Fatalf("Failed to create test port for device 1: %v", err)
	}

	port2 := &models.Port{
		DeviceID:       device2ID,
		PortNumber:     3306,
		Protocol:       "tcp",
		ServiceName:    "mysql",
		ServiceVersion: "MySQL 8.0.23",
		FirstSeen:      time.Now().Add(-48 * time.Hour),
		LastSeen:       time.Now().Add(-1 * time.Hour),
	}
	err = db.SavePort(port2)
	if err != nil {
		t.Fatalf("Failed to create test port for device 2: %v", err)
	}

	// Create device handler
	deviceHandler := NewDeviceHandler(db)

	// Test search by IP address
	req, err := http.NewRequest("GET", "/api/devices/search?q=192.168.1.100", nil)
	if err != nil {
		t.Fatalf("Failed to create request for IP search: %v", err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(deviceHandler.SearchDevices)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("IP search handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var searchResults []models.Device
	if err := json.Unmarshal(rr.Body.Bytes(), &searchResults); err != nil {
		t.Errorf("Failed to parse IP search response: %v", err)
	}

	if len(searchResults) != 1 {
		t.Errorf("Expected 1 result for IP search, got %d", len(searchResults))
	}

	if len(searchResults) > 0 && searchResults[0].IPAddress != "192.168.1.100" {
		t.Errorf("Expected IP 192.168.1.100, got %s", searchResults[0].IPAddress)
	}

	// Test search by hostname
	req, err = http.NewRequest("GET", "/api/devices/search?q=database", nil)
	if err != nil {
		t.Fatalf("Failed to create request for hostname search: %v", err)
	}

	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Hostname search handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	if err := json.Unmarshal(rr.Body.Bytes(), &searchResults); err != nil {
		t.Errorf("Failed to parse hostname search response: %v", err)
	}

	if len(searchResults) != 1 {
		t.Errorf("Expected 1 result for hostname search, got %d", len(searchResults))
	}

	if len(searchResults) > 0 && searchResults[0].Hostname != "database" {
		t.Errorf("Expected hostname 'database', got %s", searchResults[0].Hostname)
	}

	// Test search by OS
	req, err = http.NewRequest("GET", "/api/devices/search?q=Linux", nil)
	if err != nil {
		t.Fatalf("Failed to create request for OS search: %v", err)
	}

	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("OS search handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	if err := json.Unmarshal(rr.Body.Bytes(), &searchResults); err != nil {
		t.Errorf("Failed to parse OS search response: %v", err)
	}

	if len(searchResults) != 1 {
		t.Errorf("Expected 1 result for OS search, got %d", len(searchResults))
	}

	if len(searchResults) > 0 && searchResults[0].OSFingerprint != "Linux" {
		t.Errorf("Expected OS 'Linux', got %s", searchResults[0].OSFingerprint)
	}

	// Test search with no results
	req, err = http.NewRequest("GET", "/api/devices/search?q=nonexistent", nil)
	if err != nil {
		t.Fatalf("Failed to create request for empty search: %v", err)
	}

	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Empty search handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	if err := json.Unmarshal(rr.Body.Bytes(), &searchResults); err != nil {
		t.Errorf("Failed to parse empty search response: %v", err)
	}

	if len(searchResults) != 0 {
		t.Errorf("Expected 0 results for empty search, got %d", len(searchResults))
	}

	// Test search with empty query
	req, err = http.NewRequest("GET", "/api/devices/search", nil)
	if err != nil {
		t.Fatalf("Failed to create request for missing query: %v", err)
	}

	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// Should return 400 for missing query parameter
	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("Missing query handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
	}
}

// TestGetDeviceStats tests the getDeviceStats handler
func TestGetDeviceStats(t *testing.T) {
	tempDir, _, db, _, _ := setupTestEnvironment(t)
	defer os.RemoveAll(tempDir)
	defer db.Close()

	// Create several test devices
	createTestDevices(t, db, 5)

	// Create device handler
	deviceHandler := NewDeviceHandler(db)

	// Create a request to the getDeviceStats handler
	req, err := http.NewRequest("GET", "/api/devices/stats", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Record the response
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(deviceHandler.GetDeviceStats)

	// Serve the request
	handler.ServeHTTP(rr, req)

	// Check the status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Parse the response
	var stats map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &stats); err != nil {
		t.Errorf("Failed to parse response: %v", err)
	}

	// Check required fields
	if _, ok := stats["totalDevices"]; !ok {
		t.Errorf("Expected totalDevices in stats, not found")
	}

	if _, ok := stats["totalPorts"]; !ok {
		t.Errorf("Expected totalPorts in stats, not found")
	}

	if _, ok := stats["osDistribution"]; !ok {
		t.Errorf("Expected osDistribution in stats, not found")
	}

	if _, ok := stats["lastScanTime"]; !ok {
		t.Errorf("Expected lastScanTime in stats, not found")
	}

	// Check numeric values
	if totalDevices, ok := stats["totalDevices"].(float64); !ok || totalDevices != 5 {
		t.Errorf("Expected totalDevices to be 5, got %v", stats["totalDevices"])
	}

	// Check OS distribution
	osDistribution, ok := stats["osDistribution"].(map[string]interface{})
	if !ok {
		t.Errorf("osDistribution is not a map")
	} else {
		if len(osDistribution) == 0 {
			t.Errorf("Expected non-empty osDistribution")
		}

		// We should have 3 OS types (Linux, Windows, macOS)
		expectedOSCount := 3
		if len(osDistribution) != expectedOSCount {
			t.Errorf("Expected %d OS types, got %d", expectedOSCount, len(osDistribution))
		}
	}
}
