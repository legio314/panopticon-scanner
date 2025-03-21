// internal/database/database_test.go
package database

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"panopticon-scanner/internal/models"
)

// setupTestDB creates a temporary database for testing
func setupTestDB(t *testing.T) (*DB, string, func()) {
	// Create a temporary directory for the test
	tempDir, err := os.MkdirTemp("", "panopticon-db-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	dbPath := filepath.Join(tempDir, "test.db")

	// Create a test database
	db, err := New(dbPath)
	if err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to create database: %v", err)
	}

	// Return a cleanup function
	cleanup := func() {
		db.Close()
		os.RemoveAll(tempDir)
	}

	return db, tempDir, cleanup
}

// TestNew tests database creation and initialization
func TestNew(t *testing.T) {
	db, tempDir, cleanup := setupTestDB(t)
	defer cleanup()

	dbPath := filepath.Join(tempDir, "test.db")

	// Verify the database file was created
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Errorf("Database file was not created at %s", dbPath)
	}

	// Test that the database connection works
	var version string
	err := db.QueryRow("SELECT sqlite_version()").Scan(&version)
	if err != nil {
		t.Errorf("Failed to query database: %v", err)
	}

	if version == "" {
		t.Errorf("Expected SQLite version, got empty string")
	}

	// Check that tables were created
	var tableCount int
	err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table'").Scan(&tableCount)
	if err != nil {
		t.Errorf("Failed to count tables: %v", err)
	}

	// We should have at least our main tables (devices, ports, scans, changes, etc.)
	if tableCount < 5 {
		t.Errorf("Expected at least 5 tables, got %d", tableCount)
	}
}

// TestSaveDevice tests saving and updating device information
func TestSaveDevice(t *testing.T) {
	db, _, cleanup := setupTestDB(t)
	defer cleanup()

	// Test saving a new device
	device := &models.Device{
		IPAddress:     "192.168.1.100",
		MACAddress:    "00:11:22:33:44:55",
		Hostname:      "test-device",
		OSFingerprint: "Linux 5.10",
		FirstSeen:     time.Now(),
		LastSeen:      time.Now(),
	}

	deviceID, err := db.SaveDevice(device)
	if err != nil {
		t.Errorf("Failed to save device: %v", err)
	}

	if deviceID <= 0 {
		t.Errorf("Expected positive device ID, got %d", deviceID)
	}

	// Verify the device was saved correctly
	savedDevice, err := db.GetDevice(deviceID)
	if err != nil {
		t.Errorf("Failed to get saved device: %v", err)
	}

	if savedDevice.IPAddress != device.IPAddress {
		t.Errorf("Expected IP %s, got %s", device.IPAddress, savedDevice.IPAddress)
	}
	if savedDevice.MACAddress != device.MACAddress {
		t.Errorf("Expected MAC %s, got %s", device.MACAddress, savedDevice.MACAddress)
	}
	if savedDevice.Hostname != device.Hostname {
		t.Errorf("Expected hostname %s, got %s", device.Hostname, savedDevice.Hostname)
	}
	if savedDevice.OSFingerprint != device.OSFingerprint {
		t.Errorf("Expected OS %s, got %s", device.OSFingerprint, savedDevice.OSFingerprint)
	}

	// Test updating an existing device
	updatedDevice := &models.Device{
		IPAddress:     "192.168.1.100", // Same IP
		MACAddress:    "00:11:22:33:44:55", // Same MAC
		Hostname:      "updated-device", // Changed hostname
		OSFingerprint: "Windows 10", // Changed OS
		LastSeen:      time.Now(),
	}

	updatedID, err := db.SaveDevice(updatedDevice)
	if err != nil {
		t.Errorf("Failed to update device: %v", err)
	}

	// Should return the same ID for an update
	if updatedID != deviceID {
		t.Errorf("Expected updated ID %d, got %d", deviceID, updatedID)
	}

	// Verify the device was updated correctly
	updatedSavedDevice, err := db.GetDevice(deviceID)
	if err != nil {
		t.Errorf("Failed to get updated device: %v", err)
	}

	if updatedSavedDevice.Hostname != updatedDevice.Hostname {
		t.Errorf("Expected updated hostname %s, got %s", updatedDevice.Hostname, updatedSavedDevice.Hostname)
	}
	if updatedSavedDevice.OSFingerprint != updatedDevice.OSFingerprint {
		t.Errorf("Expected updated OS %s, got %s", updatedDevice.OSFingerprint, updatedSavedDevice.OSFingerprint)
	}

	// Test partial update (only updating some fields)
	partialDevice := &models.Device{
		IPAddress:     "192.168.1.100", // Same IP
		MACAddress:    "00:11:22:33:44:55", // Same MAC
		Hostname:      "", // Empty, should keep previous value
		OSFingerprint: "Linux 5.15", // Changed OS again
		LastSeen:      time.Now(),
	}

	_, err = db.SaveDevice(partialDevice)
	if err != nil {
		t.Errorf("Failed to apply partial update: %v", err)
	}

	// Verify the device was partially updated correctly
	partialSavedDevice, err := db.GetDevice(deviceID)
	if err != nil {
		t.Errorf("Failed to get partially updated device: %v", err)
	}

	// Hostname should still be the previous value
	if partialSavedDevice.Hostname != updatedDevice.Hostname {
		t.Errorf("Expected hostname to remain %s, got %s", updatedDevice.Hostname, partialSavedDevice.Hostname)
	}
	// OS should be updated
	if partialSavedDevice.OSFingerprint != partialDevice.OSFingerprint {
		t.Errorf("Expected updated OS %s, got %s", partialDevice.OSFingerprint, partialSavedDevice.OSFingerprint)
	}
}

// TestSavePort tests saving and updating port information
func TestSavePort(t *testing.T) {
	db, _, cleanup := setupTestDB(t)
	defer cleanup()

	// First, create a device to associate ports with
	device := &models.Device{
		IPAddress:  "192.168.1.200",
		MACAddress: "AA:BB:CC:DD:EE:FF",
		Hostname:   "test-port-device",
	}

	deviceID, err := db.SaveDevice(device)
	if err != nil {
		t.Fatalf("Failed to save device for port test: %v", err)
	}

	// Test saving a new port
	port := &models.Port{
		DeviceID:       deviceID,
		PortNumber:     80,
		Protocol:       "tcp",
		ServiceName:    "http",
		ServiceVersion: "Apache 2.4.41",
		FirstSeen:      time.Now(),
		LastSeen:       time.Now(),
	}

	err = db.SavePort(port)
	if err != nil {
		t.Errorf("Failed to save port: %v", err)
	}

	// Get the device details to verify port was saved
	deviceDetails, err := db.GetDeviceDetails(deviceID)
	if err != nil {
		t.Errorf("Failed to get device details: %v", err)
	}

	if len(deviceDetails.Ports) != 1 {
		t.Errorf("Expected 1 port, got %d", len(deviceDetails.Ports))
	}

	savedPort := deviceDetails.Ports[0]
	if savedPort.PortNumber != port.PortNumber {
		t.Errorf("Expected port number %d, got %d", port.PortNumber, savedPort.PortNumber)
	}
	if savedPort.Protocol != port.Protocol {
		t.Errorf("Expected protocol %s, got %s", port.Protocol, savedPort.Protocol)
	}
	if savedPort.ServiceName != port.ServiceName {
		t.Errorf("Expected service name %s, got %s", port.ServiceName, savedPort.ServiceName)
	}
	if savedPort.ServiceVersion != port.ServiceVersion {
		t.Errorf("Expected service version %s, got %s", port.ServiceVersion, savedPort.ServiceVersion)
	}

	// Test updating an existing port
	updatedPort := &models.Port{
		DeviceID:       deviceID,
		PortNumber:     80, // Same port
		Protocol:       "tcp", // Same protocol
		ServiceName:    "http", // Same service
		ServiceVersion: "Apache 2.4.52", // Updated version
		LastSeen:       time.Now(),
	}

	err = db.SavePort(updatedPort)
	if err != nil {
		t.Errorf("Failed to update port: %v", err)
	}

	// Verify the port was updated correctly
	deviceDetails, err = db.GetDeviceDetails(deviceID)
	if err != nil {
		t.Errorf("Failed to get device details after port update: %v", err)
	}

	if len(deviceDetails.Ports) != 1 {
		t.Errorf("Expected 1 port after update, got %d", len(deviceDetails.Ports))
	}

	updatedSavedPort := deviceDetails.Ports[0]
	if updatedSavedPort.ServiceVersion != updatedPort.ServiceVersion {
		t.Errorf("Expected updated service version %s, got %s", updatedPort.ServiceVersion, updatedSavedPort.ServiceVersion)
	}

	// Add a second port to the device
	port2 := &models.Port{
		DeviceID:       deviceID,
		PortNumber:     443,
		Protocol:       "tcp",
		ServiceName:    "https",
		ServiceVersion: "Apache 2.4.41",
		FirstSeen:      time.Now(),
		LastSeen:       time.Now(),
	}

	err = db.SavePort(port2)
	if err != nil {
		t.Errorf("Failed to save second port: %v", err)
	}

	// Verify both ports exist
	deviceDetails, err = db.GetDeviceDetails(deviceID)
	if err != nil {
		t.Errorf("Failed to get device details after adding second port: %v", err)
	}

	if len(deviceDetails.Ports) != 2 {
		t.Errorf("Expected 2 ports, got %d", len(deviceDetails.Ports))
	}
}

// TestGetAllDevices tests retrieving all devices
func TestGetAllDevices(t *testing.T) {
	db, _, cleanup := setupTestDB(t)
	defer cleanup()

	// Create multiple devices
	devices := []models.Device{
		{
			IPAddress:     "192.168.1.1",
			MACAddress:    "00:11:22:33:44:55",
			Hostname:      "device1",
			OSFingerprint: "Linux",
		},
		{
			IPAddress:     "192.168.1.2",
			MACAddress:    "AA:BB:CC:DD:EE:FF",
			Hostname:      "device2",
			OSFingerprint: "Windows",
		},
		{
			IPAddress:     "192.168.1.3",
			MACAddress:    "11:22:33:44:55:66",
			Hostname:      "device3",
			OSFingerprint: "macOS",
		},
	}

	// Save devices
	for _, device := range devices {
		deviceCopy := device // Create a copy to avoid pointer issues
		_, err := db.SaveDevice(&deviceCopy)
		if err != nil {
			t.Fatalf("Failed to save test device: %v", err)
		}
	}

	// Get all devices
	allDevices, err := db.GetAllDevices()
	if err != nil {
		t.Errorf("Failed to get all devices: %v", err)
	}

	// Verify correct number of devices
	if len(allDevices) != len(devices) {
		t.Errorf("Expected %d devices, got %d", len(devices), len(allDevices))
	}

	// Create a map for easy lookup by IP
	deviceMap := make(map[string]*models.Device)
	for _, device := range allDevices {
		deviceMap[device.IPAddress] = device
	}

	// Verify each device is in the result
	for _, originalDevice := range devices {
		savedDevice, found := deviceMap[originalDevice.IPAddress]
		if !found {
			t.Errorf("Device with IP %s not found in results", originalDevice.IPAddress)
			continue
		}

		if savedDevice.Hostname != originalDevice.Hostname {
			t.Errorf("Expected hostname %s, got %s", originalDevice.Hostname, savedDevice.Hostname)
		}
		if savedDevice.OSFingerprint != originalDevice.OSFingerprint {
			t.Errorf("Expected OS %s, got %s", originalDevice.OSFingerprint, savedDevice.OSFingerprint)
		}
	}
}

// TestSearchDevices tests searching for devices
func TestSearchDevices(t *testing.T) {
	db, _, cleanup := setupTestDB(t)
	defer cleanup()

	// Create devices with different attributes for testing search
	devices := []models.Device{
		{
			IPAddress:     "192.168.1.10",
			MACAddress:    "00:11:22:33:44:55",
			Hostname:      "webserver",
			OSFingerprint: "Ubuntu Linux",
		},
		{
			IPAddress:     "192.168.1.20",
			MACAddress:    "AA:BB:CC:DD:EE:FF",
			Hostname:      "database-server",
			OSFingerprint: "Windows Server",
		},
		{
			IPAddress:     "10.0.0.1",
			MACAddress:    "11:22:33:44:55:66",
			Hostname:      "router",
			OSFingerprint: "RouterOS",
		},
	}

	// Save devices
	for _, device := range devices {
		deviceCopy := device // Create a copy to avoid pointer issues
		_, err := db.SaveDevice(&deviceCopy)
		if err != nil {
			t.Fatalf("Failed to save test device: %v", err)
		}
	}

	// Test search by IP address
	ipResults, err := db.SearchDevices("192.168")
	if err != nil {
		t.Errorf("Failed to search devices by IP: %v", err)
	}

	if len(ipResults) != 2 {
		t.Errorf("Expected 2 devices matching IP pattern, got %d", len(ipResults))
	}

	// Test search by hostname
	hostnameResults, err := db.SearchDevices("server")
	if err != nil {
		t.Errorf("Failed to search devices by hostname: %v", err)
	}

	if len(hostnameResults) != 2 {
		t.Errorf("Expected 2 devices matching hostname pattern, got %d", len(hostnameResults))
	}

	// Test search by OS
	osResults, err := db.SearchDevices("Linux")
	if err != nil {
		t.Errorf("Failed to search devices by OS: %v", err)
	}

	if len(osResults) != 1 {
		t.Errorf("Expected 1 device matching OS pattern, got %d", len(osResults))
	}

	// Test search with no results
	noResults, err := db.SearchDevices("nonexistent")
	if err != nil {
		t.Errorf("Failed to search devices with no match: %v", err)
	}

	if len(noResults) != 0 {
		t.Errorf("Expected 0 devices for non-matching pattern, got %d", len(noResults))
	}
}

// TestCreateAndUpdateScan tests scan record creation and updates
func TestCreateAndUpdateScan(t *testing.T) {
	db, _, cleanup := setupTestDB(t)
	defer cleanup()

	// Create a new scan
	template := "default"

	scanID, err := db.CreateScan(template)
	if err != nil {
		t.Errorf("Failed to create scan: %v", err)
	}

	if scanID <= 0 {
		t.Errorf("Expected positive scan ID, got %d", scanID)
	}

	// Verify scan was created correctly
	savedScan, err := db.GetScan(scanID)
	if err != nil {
		t.Errorf("Failed to get saved scan: %v", err)
	}

	if savedScan.Template != template {
		t.Errorf("Expected template %s, got %s", template, savedScan.Template)
	}
	if savedScan.Status != "running" {
		t.Errorf("Expected status %s, got %s", "running", savedScan.Status)
	}

	// Update the scan
	status := "completed"
	devicesFound := 10
	portsFound := 25
	duration := 60 * time.Second
	errorMsg := ""

	err = db.UpdateScan(scanID, status, devicesFound, portsFound, duration, errorMsg)
	if err != nil {
		t.Errorf("Failed to update scan: %v", err)
	}

	// Verify scan was updated correctly
	updatedSavedScan, err := db.GetScan(scanID)
	if err != nil {
		t.Errorf("Failed to get updated scan: %v", err)
	}

	if updatedSavedScan.Status != status {
		t.Errorf("Expected updated status %s, got %s", status, updatedSavedScan.Status)
	}
	if updatedSavedScan.Duration != int(duration.Seconds()) {
		t.Errorf("Expected updated duration %d, got %d", int(duration.Seconds()), updatedSavedScan.Duration)
	}
	if updatedSavedScan.DevicesFound != devicesFound {
		t.Errorf("Expected updated devices found %d, got %d", devicesFound, updatedSavedScan.DevicesFound)
	}
	if updatedSavedScan.PortsFound != portsFound {
		t.Errorf("Expected updated ports found %d, got %d", portsFound, updatedSavedScan.PortsFound)
	}
}

// TestGetRecentScans tests retrieving recent scans
func TestGetRecentScans(t *testing.T) {
	db, _, cleanup := setupTestDB(t)
	defer cleanup()

	// Create several scans with different timestamps
	scans := []models.Scan{
		{
			Timestamp:    time.Now().Add(-3 * time.Hour),
			Template:     "default",
			Status:       "completed",
			Duration:     60,
			DevicesFound: 5,
			PortsFound:   20,
		},
		{
			Timestamp:    time.Now().Add(-2 * time.Hour),
			Template:     "quick",
			Status:       "completed",
			Duration:     30,
			DevicesFound: 3,
			PortsFound:   10,
		},
		{
			Timestamp:    time.Now().Add(-1 * time.Hour),
			Template:     "thorough",
			Status:       "completed",
			Duration:     120,
			DevicesFound: 8,
			PortsFound:   35,
		},
	}

	// Save scans
	for _, scan := range scans {
		// Create scan with the template
		scanID, err := db.CreateScan(scan.Template)
		if err != nil {
			t.Fatalf("Failed to create test scan: %v", err)
		}
		
		// Update scan with full details
		err = db.UpdateScan(
			scanID, 
			scan.Status, 
			scan.DevicesFound, 
			scan.PortsFound, 
			time.Duration(scan.Duration) * time.Second,
			scan.ErrorMessage,
		)
		if err != nil {
			t.Fatalf("Failed to update test scan: %v", err)
		}
	}

	// Get recent scans with limit 2
	recentScans, err := db.GetRecentScans(2)
	if err != nil {
		t.Errorf("Failed to get recent scans: %v", err)
	}

	// Should return the 2 most recent scans
	if len(recentScans) != 2 {
		t.Errorf("Expected 2 recent scans, got %d", len(recentScans))
	}

	// The first scan should be the most recent one
	if recentScans[0].Template != "thorough" {
		t.Errorf("Expected most recent scan to be 'thorough', got '%s'", recentScans[0].Template)
	}

	// The second scan should be the second most recent
	if recentScans[1].Template != "quick" {
		t.Errorf("Expected second most recent scan to be 'quick', got '%s'", recentScans[1].Template)
	}
}

// TestRecordScan tests the convenience method for recording scans
func TestRecordScan(t *testing.T) {
	db, _, cleanup := setupTestDB(t)
	defer cleanup()

	// Record a scan
	duration := 45 * time.Second
	devicesFound := 7
	portsFound := 30
	status := "completed"

	scanID, err := db.RecordScan(duration, devicesFound, portsFound, status)
	if err != nil {
		t.Errorf("Failed to record scan: %v", err)
	}

	if scanID <= 0 {
		t.Errorf("Expected positive scan ID, got %d", scanID)
	}

	// Verify scan was recorded correctly
	savedScan, err := db.GetScan(scanID)
	if err != nil {
		t.Errorf("Failed to get recorded scan: %v", err)
	}

	if savedScan.Template != "default" {
		t.Errorf("Expected template 'default', got '%s'", savedScan.Template)
	}
	if savedScan.Duration != int(duration.Seconds()) {
		t.Errorf("Expected duration %d, got %d", int(duration.Seconds()), savedScan.Duration)
	}
	if savedScan.DevicesFound != devicesFound {
		t.Errorf("Expected devices found %d, got %d", devicesFound, savedScan.DevicesFound)
	}
	if savedScan.PortsFound != portsFound {
		t.Errorf("Expected ports found %d, got %d", portsFound, savedScan.PortsFound)
	}
	if savedScan.Status != status {
		t.Errorf("Expected status '%s', got '%s'", status, savedScan.Status)
	}
}

// TestOptimizeDatabase tests database optimization
func TestOptimizeDatabase(t *testing.T) {
	db, _, cleanup := setupTestDB(t)
	defer cleanup()

	// Add some data to the database
	device := &models.Device{
		IPAddress:     "192.168.1.100",
		MACAddress:    "00:11:22:33:44:55",
		Hostname:      "test-device",
		OSFingerprint: "Linux",
	}

	_, err := db.SaveDevice(device)
	if err != nil {
		t.Fatalf("Failed to save device: %v", err)
	}

	// Optimize the database
	err = db.OptimizeDatabase()
	if err != nil {
		t.Errorf("Failed to optimize database: %v", err)
	}

	// Verify database is still functional
	devices, err := db.GetAllDevices()
	if err != nil {
		t.Errorf("Database not functional after optimization: %v", err)
	}

	if len(devices) != 1 {
		t.Errorf("Expected 1 device after optimization, got %d", len(devices))
	}
}

// TestBackupDatabase tests database backup functionality
func TestBackupDatabase(t *testing.T) {
	db, tempDir, cleanup := setupTestDB(t)
	defer cleanup()

	// Add some data to the database
	device := &models.Device{
		IPAddress:     "192.168.1.100",
		MACAddress:    "00:11:22:33:44:55",
		Hostname:      "test-device",
		OSFingerprint: "Linux",
	}

	_, err := db.SaveDevice(device)
	if err != nil {
		t.Fatalf("Failed to save device: %v", err)
	}

	// Create backup directory
	backupDir := filepath.Join(tempDir, "backups")
	err = os.MkdirAll(backupDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create backup directory: %v", err)
	}

	// Set the database path to include the test directory for backup
	db.Path = filepath.Join(backupDir, "test.db")
	
	// Backup the database
	backupPath, err := db.BackupDatabase()
	if err != nil {
		t.Errorf("Failed to backup database: %v", err)
	}

	// Verify backup file exists
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		t.Errorf("Backup file does not exist: %s", backupPath)
	}

	// Create a new database from the backup
	backupDB, err := New(backupPath)
	if err != nil {
		t.Errorf("Failed to open backup database: %v", err)
	}
	defer backupDB.Close()

	// Verify data in backup
	devices, err := backupDB.GetAllDevices()
	if err != nil {
		t.Errorf("Failed to query backup database: %v", err)
	}

	if len(devices) != 1 {
		t.Errorf("Expected 1 device in backup, got %d", len(devices))
	}

	if len(devices) > 0 {
		if devices[0].IPAddress != device.IPAddress {
			t.Errorf("Expected IP %s in backup, got %s", device.IPAddress, devices[0].IPAddress)
		}
	}
}

// TestCleanOldData tests data retention policy enforcement
func TestCleanOldData(t *testing.T) {
	db, _, cleanup := setupTestDB(t)
	defer cleanup()

	now := time.Now()

	// Create devices with different last seen times
	devices := []models.Device{
		{
			IPAddress:     "192.168.1.1",
			MACAddress:    "00:11:22:33:44:55",
			Hostname:      "recent-device",
			OSFingerprint: "Linux",
			FirstSeen:     now.Add(-30 * 24 * time.Hour), // 30 days ago
			LastSeen:      now,                            // today
		},
		{
			IPAddress:     "192.168.1.2",
			MACAddress:    "AA:BB:CC:DD:EE:FF",
			Hostname:      "old-device",
			OSFingerprint: "Windows",
			FirstSeen:     now.Add(-100 * 24 * time.Hour), // 100 days ago
			LastSeen:      now.Add(-40 * 24 * time.Hour),  // 40 days ago
		},
		{
			IPAddress:     "192.168.1.3",
			MACAddress:    "11:22:33:44:55:66",
			Hostname:      "very-old-device",
			OSFingerprint: "macOS",
			FirstSeen:     now.Add(-200 * 24 * time.Hour), // 200 days ago
			LastSeen:      now.Add(-100 * 24 * time.Hour), // 100 days ago
		},
	}

	// Save devices and set their last_seen timestamps directly in the database
	for _, device := range devices {
		deviceCopy := device // Create a copy to avoid pointer issues
		deviceID, err := db.SaveDevice(&deviceCopy)
		if err != nil {
			t.Fatalf("Failed to save test device: %v", err)
		}

		// Update the last_seen time directly to set it in the past
		_, err = db.Exec("UPDATE devices SET last_seen = ? WHERE id = ?", device.LastSeen, deviceID)
		if err != nil {
			t.Fatalf("Failed to update device last_seen: %v", err)
		}

		// Add a port to the device to test cascading delete
		port := &models.Port{
			DeviceID:       deviceID,
			PortNumber:     80,
			Protocol:       "tcp",
			ServiceName:    "http",
			ServiceVersion: "Test",
			FirstSeen:      device.FirstSeen,
			LastSeen:       device.LastSeen,
		}

		err = db.SavePort(port)
		if err != nil {
			t.Fatalf("Failed to add port to device: %v", err)
		}
	}

	// Add some scan records
	// Create old scan - manually set old timestamp
	oldScanTemplate := "default"
	oldTimestamp := now.Add(-60 * 24 * time.Hour) // 60 days ago
	
	var err error
	_, err = db.Exec(
		`INSERT INTO scans (timestamp, template, status, duration, devices_found, ports_found)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		oldTimestamp, oldScanTemplate, "completed", 60, 5, 10,
	)
	if err != nil {
		t.Fatalf("Failed to create old scan: %v", err)
	}

	// Create recent scan - manually set recent timestamp
	recentScanTemplate := "default" 
	recentTimestamp := now.Add(-10 * 24 * time.Hour) // 10 days ago
	
	_, err = db.Exec(
		`INSERT INTO scans (timestamp, template, status, duration, devices_found, ports_found)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		recentTimestamp, recentScanTemplate, "completed", 45, 3, 8,
	)
	if err != nil {
		t.Fatalf("Failed to create recent scan: %v", err)
	}

	// Run cleanup with 30-day retention policy
	deleted, err := db.CleanOldData(30)
	if err != nil {
		t.Errorf("Failed to clean old data: %v", err)
	}

	// Should delete 2 devices (old and very old) and 1 scan
	if deleted < 3 {
		t.Errorf("Expected at least 3 items deleted, got %d", deleted)
	}

	// Verify that old devices were deleted
	devicesAfterCleanup, err := db.GetAllDevices()
	if err != nil {
		t.Errorf("Failed to get devices after cleanup: %v", err)
	}

	if len(devicesAfterCleanup) != 1 {
		t.Errorf("Expected 1 device after cleanup, got %d", len(devicesAfterCleanup))
	}

	if len(devicesAfterCleanup) > 0 && devicesAfterCleanup[0].Hostname != "recent-device" {
		t.Errorf("Expected only recent-device to remain, got %s", devicesAfterCleanup[0].Hostname)
	}

	// Verify that old scan was deleted
	scans, err := db.GetRecentScans(10)
	if err != nil {
		t.Errorf("Failed to get scans after cleanup: %v", err)
	}

	if len(scans) != 1 {
		t.Errorf("Expected 1 scan after cleanup, got %d", len(scans))
	}
}

// TestAddLogEntry tests adding and retrieving log entries
func TestAddLogEntry(t *testing.T) {
	db, _, cleanup := setupTestDB(t)
	defer cleanup()

	// Add log entries
	err := db.AddLogEntry("info", "Test info message", "test")
	if err != nil {
		t.Errorf("Failed to add info log entry: %v", err)
	}

	err = db.AddLogEntry("error", "Test error message", "test")
	if err != nil {
		t.Errorf("Failed to add error log entry: %v", err)
	}

	err = db.AddLogEntry("debug", "Test debug message", "scanner")
	if err != nil {
		t.Errorf("Failed to add debug log entry: %v", err)
	}

	// Get all log entries
	logs, err := db.GetLogEntries(10, "", "")
	if err != nil {
		t.Errorf("Failed to get log entries: %v", err)
	}

	if len(logs) != 3 {
		t.Errorf("Expected 3 log entries, got %d", len(logs))
	}

	// Get filtered log entries by level
	infoLogs, err := db.GetLogEntries(10, "info", "")
	if err != nil {
		t.Errorf("Failed to get info logs: %v", err)
	}

	if len(infoLogs) != 1 {
		t.Errorf("Expected 1 info log, got %d", len(infoLogs))
	}

	// Get filtered log entries by component
	scannerLogs, err := db.GetLogEntries(10, "", "scanner")
	if err != nil {
		t.Errorf("Failed to get scanner logs: %v", err)
	}

	if len(scannerLogs) != 1 {
		t.Errorf("Expected 1 scanner log, got %d", len(scannerLogs))
	}

	// Get logs with both filters
	filteredLogs, err := db.GetLogEntries(10, "debug", "scanner")
	if err != nil {
		t.Errorf("Failed to get filtered logs: %v", err)
	}

	if len(filteredLogs) != 1 {
		t.Errorf("Expected 1 filtered log, got %d", len(filteredLogs))
	}

	if len(filteredLogs) > 0 {
		if filteredLogs[0].Level != "debug" || filteredLogs[0].Component != "scanner" {
			t.Errorf("Got wrong filtered log: level=%s, component=%s",
				filteredLogs[0].Level, filteredLogs[0].Component)
		}
	}
}

// TestGetDatabaseStats tests retrieving database statistics
func TestGetDatabaseStats(t *testing.T) {
	db, _, cleanup := setupTestDB(t)
	defer cleanup()

	// Add some data to the database
	device := &models.Device{
		IPAddress:     "192.168.1.100",
		MACAddress:    "00:11:22:33:44:55",
		Hostname:      "test-device",
		OSFingerprint: "Linux",
	}

	deviceID, err := db.SaveDevice(device)
	if err != nil {
		t.Fatalf("Failed to save device: %v", err)
	}

	port := &models.Port{
		DeviceID:       deviceID,
		PortNumber:     80,
		Protocol:       "tcp",
		ServiceName:    "http",
		ServiceVersion: "Test",
	}

	err = db.SavePort(port)
	if err != nil {
		t.Fatalf("Failed to add port: %v", err)
	}

	// Create scan with template
	scanTemplate := "default"
	scanID, err := db.CreateScan(scanTemplate)
	if err != nil {
		t.Fatalf("Failed to create scan: %v", err)
	}
	
	// Update with details
	err = db.UpdateScan(
		scanID,
		"completed",
		1, // devicesFound
		1, // portsFound
		60 * time.Second, // duration
		"", // errorMsg
	)
	if err != nil {
		t.Fatalf("Failed to create scan: %v", err)
	}

	// Get database stats
	stats, err := db.GetDatabaseStats()
	if err != nil {
		t.Errorf("Failed to get database stats: %v", err)
	}

	// Verify stats are present and reasonable
	deviceCount, ok := stats["deviceCount"].(int)
	if !ok || deviceCount != 1 {
		t.Errorf("Expected deviceCount of 1, got %v", stats["deviceCount"])
	}

	portCount, ok := stats["portCount"].(int)
	if !ok || portCount != 1 {
		t.Errorf("Expected portCount of 1, got %v", stats["portCount"])
	}

	scanCount, ok := stats["scanCount"].(int)
	if !ok || scanCount != 1 {
		t.Errorf("Expected scanCount of 1, got %v", stats["scanCount"])
	}

	_, ok = stats["sizeBytes"].(int64)
	if !ok {
		t.Errorf("Expected sizeBytes to be present and of type int64")
	}

	_, hasLastScan := stats["lastScanTime"].(time.Time)
	if !hasLastScan {
		t.Errorf("Expected lastScanTime to be present and of type time.Time")
	}
}

// TestGetDeviceByIP tests retrieving a device by its IP address
func TestGetDeviceByIP(t *testing.T) {
	db, _, cleanup := setupTestDB(t)
	defer cleanup()

	// Create test device
	device := &models.Device{
		IPAddress:     "192.168.1.50",
		MACAddress:    "00:11:22:33:44:55",
		Hostname:      "test-ip-lookup",
		OSFingerprint: "Linux",
	}

	_, err := db.SaveDevice(device)
	if err != nil {
		t.Fatalf("Failed to save device: %v", err)
	}

	// Retrieve device by IP
	retrievedDevice, err := db.GetDeviceByIP("192.168.1.50")
	if err != nil {
		t.Errorf("Failed to get device by IP: %v", err)
	}

	// Verify device details
	if retrievedDevice.IPAddress != device.IPAddress {
		t.Errorf("Expected IP %s, got %s", device.IPAddress, retrievedDevice.IPAddress)
	}
	if retrievedDevice.MACAddress != device.MACAddress {
		t.Errorf("Expected MAC %s, got %s", device.MACAddress, retrievedDevice.MACAddress)
	}
	if retrievedDevice.Hostname != device.Hostname {
		t.Errorf("Expected hostname %s, got %s", device.Hostname, retrievedDevice.Hostname)
	}

	// Test with non-existent IP
	_, err = db.GetDeviceByIP("10.0.0.1")
	if err == nil {
		t.Errorf("Expected error when getting non-existent IP, got nil")
	}
}

// TestTransactionRollback tests transaction rollback on error
func TestTransactionRollback(t *testing.T) {
	db, _, cleanup := setupTestDB(t)
	defer cleanup()

	// Start a transaction
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}

	// Add a device in the transaction
	_, err = tx.Exec(
		`INSERT INTO devices (ip_address, mac_address, hostname, os_fingerprint, first_seen, last_seen)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		"192.168.1.100", "00:11:22:33:44:55", "rollback-test", "Linux",
		time.Now(), time.Now(),
	)
	if err != nil {
		t.Fatalf("Failed to insert device in transaction: %v", err)
	}

	// Rollback the transaction
	err = tx.Rollback()
	if err != nil {
		t.Errorf("Failed to rollback transaction: %v", err)
	}

	// Verify the device was not added
	devices, err := db.GetAllDevices()
	if err != nil {
		t.Errorf("Failed to get devices after rollback: %v", err)
	}

	if len(devices) != 0 {
		t.Errorf("Expected 0 devices after rollback, got %d", len(devices))
	}
}

// TestConcurrentAccess tests concurrent database access
func TestConcurrentAccess(t *testing.T) {
	db, _, cleanup := setupTestDB(t)
	defer cleanup()

	// Number of concurrent operations
	const concurrency = 10

	// Create a wait group to synchronize goroutines
	var wg sync.WaitGroup
	wg.Add(concurrency)

	// Run concurrent device insertions
	for i := 0; i < concurrency; i++ {
		go func(i int) {
			defer wg.Done()

			device := &models.Device{
				IPAddress:     fmt.Sprintf("192.168.1.%d", 100+i),
				MACAddress:    fmt.Sprintf("00:11:22:33:44:%02X", 55+i),
				Hostname:      fmt.Sprintf("concurrent-device-%d", i),
				OSFingerprint: "Linux",
			}

			_, err := db.SaveDevice(device)
			if err != nil {
				t.Errorf("Concurrent device insertion failed: %v", err)
			}
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()

	// Verify all devices were inserted
	devices, err := db.GetAllDevices()
	if err != nil {
		t.Errorf("Failed to get devices after concurrent insertions: %v", err)
	}

	if len(devices) != concurrency {
		t.Errorf("Expected %d devices after concurrent insertions, got %d", concurrency, len(devices))
	}
}
