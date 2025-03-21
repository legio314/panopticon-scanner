// internal/models/models_test.go
package models

import (
	"encoding/json"
	"testing"
	"time"
)

// TestDeviceJSON tests Device JSON serialization and deserialization
func TestDeviceJSON(t *testing.T) {
	// Create a Device
	now := time.Now().Truncate(time.Second) // Truncate to avoid fractional seconds comparison issues
	device := Device{
		ID:           1,
		IPAddress:    "192.168.1.1",
		MACAddress:   "00:11:22:33:44:55",
		Hostname:     "test-device",
		OSFingerprint: "Linux 5.4",
		FirstSeen:    now,
		LastSeen:     now,
		PortCount:    3,
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(device)
	if err != nil {
		t.Fatalf("Failed to marshal Device to JSON: %v", err)
	}

	// Unmarshal back to Device
	var unmarshaledDevice Device
	err = json.Unmarshal(jsonData, &unmarshaledDevice)
	if err != nil {
		t.Fatalf("Failed to unmarshal Device from JSON: %v", err)
	}

	// Verify the fields match
	if unmarshaledDevice.ID != device.ID {
		t.Errorf("ID mismatch: got %d, expected %d", unmarshaledDevice.ID, device.ID)
	}
	if unmarshaledDevice.IPAddress != device.IPAddress {
		t.Errorf("IPAddress mismatch: got %s, expected %s", unmarshaledDevice.IPAddress, device.IPAddress)
	}
	if unmarshaledDevice.MACAddress != device.MACAddress {
		t.Errorf("MACAddress mismatch: got %s, expected %s", unmarshaledDevice.MACAddress, device.MACAddress)
	}
	if unmarshaledDevice.Hostname != device.Hostname {
		t.Errorf("Hostname mismatch: got %s, expected %s", unmarshaledDevice.Hostname, device.Hostname)
	}
	if unmarshaledDevice.OSFingerprint != device.OSFingerprint {
		t.Errorf("OSFingerprint mismatch: got %s, expected %s", unmarshaledDevice.OSFingerprint, device.OSFingerprint)
	}
	if !unmarshaledDevice.FirstSeen.Equal(device.FirstSeen) {
		t.Errorf("FirstSeen mismatch: got %v, expected %v", unmarshaledDevice.FirstSeen, device.FirstSeen)
	}
	if !unmarshaledDevice.LastSeen.Equal(device.LastSeen) {
		t.Errorf("LastSeen mismatch: got %v, expected %v", unmarshaledDevice.LastSeen, device.LastSeen)
	}
	if unmarshaledDevice.PortCount != device.PortCount {
		t.Errorf("PortCount mismatch: got %d, expected %d", unmarshaledDevice.PortCount, device.PortCount)
	}
}

// TestPortJSON tests Port JSON serialization and deserialization
func TestPortJSON(t *testing.T) {
	// Create a Port
	now := time.Now().Truncate(time.Second) // Truncate to avoid fractional seconds comparison issues
	port := Port{
		ID:             1,
		DeviceID:       2,
		PortNumber:     80,
		Protocol:       "tcp",
		ServiceName:    "http",
		ServiceVersion: "Apache/2.4.41",
		FirstSeen:      now,
		LastSeen:       now,
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(port)
	if err != nil {
		t.Fatalf("Failed to marshal Port to JSON: %v", err)
	}

	// Unmarshal back to Port
	var unmarshaledPort Port
	err = json.Unmarshal(jsonData, &unmarshaledPort)
	if err != nil {
		t.Fatalf("Failed to unmarshal Port from JSON: %v", err)
	}

	// Verify the fields match
	if unmarshaledPort.ID != port.ID {
		t.Errorf("ID mismatch: got %d, expected %d", unmarshaledPort.ID, port.ID)
	}
	if unmarshaledPort.DeviceID != port.DeviceID {
		t.Errorf("DeviceID mismatch: got %d, expected %d", unmarshaledPort.DeviceID, port.DeviceID)
	}
	if unmarshaledPort.PortNumber != port.PortNumber {
		t.Errorf("PortNumber mismatch: got %d, expected %d", unmarshaledPort.PortNumber, port.PortNumber)
	}
	if unmarshaledPort.Protocol != port.Protocol {
		t.Errorf("Protocol mismatch: got %s, expected %s", unmarshaledPort.Protocol, port.Protocol)
	}
	if unmarshaledPort.ServiceName != port.ServiceName {
		t.Errorf("ServiceName mismatch: got %s, expected %s", unmarshaledPort.ServiceName, port.ServiceName)
	}
	if unmarshaledPort.ServiceVersion != port.ServiceVersion {
		t.Errorf("ServiceVersion mismatch: got %s, expected %s", unmarshaledPort.ServiceVersion, port.ServiceVersion)
	}
	if !unmarshaledPort.FirstSeen.Equal(port.FirstSeen) {
		t.Errorf("FirstSeen mismatch: got %v, expected %v", unmarshaledPort.FirstSeen, port.FirstSeen)
	}
	if !unmarshaledPort.LastSeen.Equal(port.LastSeen) {
		t.Errorf("LastSeen mismatch: got %v, expected %v", unmarshaledPort.LastSeen, port.LastSeen)
	}
}

// TestScanJSON tests Scan JSON serialization and deserialization
func TestScanJSON(t *testing.T) {
	// Create a Scan
	now := time.Now().Truncate(time.Second) // Truncate to avoid fractional seconds comparison issues
	scan := Scan{
		ID:           1,
		Timestamp:    now,
		Template:     "default",
		Duration:     60,
		DevicesFound: 5,
		PortsFound:   25,
		Status:       "completed",
		ErrorMessage: "",
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(scan)
	if err != nil {
		t.Fatalf("Failed to marshal Scan to JSON: %v", err)
	}

	// Unmarshal back to Scan
	var unmarshaledScan Scan
	err = json.Unmarshal(jsonData, &unmarshaledScan)
	if err != nil {
		t.Fatalf("Failed to unmarshal Scan from JSON: %v", err)
	}

	// Verify the fields match
	if unmarshaledScan.ID != scan.ID {
		t.Errorf("ID mismatch: got %d, expected %d", unmarshaledScan.ID, scan.ID)
	}
	if !unmarshaledScan.Timestamp.Equal(scan.Timestamp) {
		t.Errorf("Timestamp mismatch: got %v, expected %v", unmarshaledScan.Timestamp, scan.Timestamp)
	}
	if unmarshaledScan.Template != scan.Template {
		t.Errorf("Template mismatch: got %s, expected %s", unmarshaledScan.Template, scan.Template)
	}
	if unmarshaledScan.Duration != scan.Duration {
		t.Errorf("Duration mismatch: got %d, expected %d", unmarshaledScan.Duration, scan.Duration)
	}
	if unmarshaledScan.DevicesFound != scan.DevicesFound {
		t.Errorf("DevicesFound mismatch: got %d, expected %d", unmarshaledScan.DevicesFound, scan.DevicesFound)
	}
	if unmarshaledScan.PortsFound != scan.PortsFound {
		t.Errorf("PortsFound mismatch: got %d, expected %d", unmarshaledScan.PortsFound, scan.PortsFound)
	}
	if unmarshaledScan.Status != scan.Status {
		t.Errorf("Status mismatch: got %s, expected %s", unmarshaledScan.Status, scan.Status)
	}
	if unmarshaledScan.ErrorMessage != scan.ErrorMessage {
		t.Errorf("ErrorMessage mismatch: got %s, expected %s", unmarshaledScan.ErrorMessage, scan.ErrorMessage)
	}
}

// TestChangeJSON tests Change JSON serialization and deserialization
func TestChangeJSON(t *testing.T) {
	// Create a Change
	now := time.Now().Truncate(time.Second) // Truncate to avoid fractional seconds comparison issues
	change := Change{
		ID:         1,
		ScanID:     2,
		DeviceID:   3,
		ChangeType: "new_device",
		Details:    "New device discovered: 192.168.1.1",
		Timestamp:  now,
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(change)
	if err != nil {
		t.Fatalf("Failed to marshal Change to JSON: %v", err)
	}

	// Unmarshal back to Change
	var unmarshaledChange Change
	err = json.Unmarshal(jsonData, &unmarshaledChange)
	if err != nil {
		t.Fatalf("Failed to unmarshal Change from JSON: %v", err)
	}

	// Verify the fields match
	if unmarshaledChange.ID != change.ID {
		t.Errorf("ID mismatch: got %d, expected %d", unmarshaledChange.ID, change.ID)
	}
	if unmarshaledChange.ScanID != change.ScanID {
		t.Errorf("ScanID mismatch: got %d, expected %d", unmarshaledChange.ScanID, change.ScanID)
	}
	if unmarshaledChange.DeviceID != change.DeviceID {
		t.Errorf("DeviceID mismatch: got %d, expected %d", unmarshaledChange.DeviceID, change.DeviceID)
	}
	if unmarshaledChange.ChangeType != change.ChangeType {
		t.Errorf("ChangeType mismatch: got %s, expected %s", unmarshaledChange.ChangeType, change.ChangeType)
	}
	if unmarshaledChange.Details != change.Details {
		t.Errorf("Details mismatch: got %s, expected %s", unmarshaledChange.Details, change.Details)
	}
	if !unmarshaledChange.Timestamp.Equal(change.Timestamp) {
		t.Errorf("Timestamp mismatch: got %v, expected %v", unmarshaledChange.Timestamp, change.Timestamp)
	}
}

// TestScanParametersJSON tests ScanParameters JSON serialization and deserialization
func TestScanParametersJSON(t *testing.T) {
	// Create ScanParameters
	params := ScanParameters{
		Template:      "thorough",
		TargetNetwork: "192.168.0.0/16",
		RateLimit:     500,
		ScanAllPorts:  true,
		DisablePing:   true,
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("Failed to marshal ScanParameters to JSON: %v", err)
	}

	// Unmarshal back to ScanParameters
	var unmarshaledParams ScanParameters
	err = json.Unmarshal(jsonData, &unmarshaledParams)
	if err != nil {
		t.Fatalf("Failed to unmarshal ScanParameters from JSON: %v", err)
	}

	// Verify the fields match
	if unmarshaledParams.Template != params.Template {
		t.Errorf("Template mismatch: got %s, expected %s", unmarshaledParams.Template, params.Template)
	}
	if unmarshaledParams.TargetNetwork != params.TargetNetwork {
		t.Errorf("TargetNetwork mismatch: got %s, expected %s", unmarshaledParams.TargetNetwork, params.TargetNetwork)
	}
	if unmarshaledParams.RateLimit != params.RateLimit {
		t.Errorf("RateLimit mismatch: got %d, expected %d", unmarshaledParams.RateLimit, params.RateLimit)
	}
	if unmarshaledParams.ScanAllPorts != params.ScanAllPorts {
		t.Errorf("ScanAllPorts mismatch: got %v, expected %v", unmarshaledParams.ScanAllPorts, params.ScanAllPorts)
	}
	if unmarshaledParams.DisablePing != params.DisablePing {
		t.Errorf("DisablePing mismatch: got %v, expected %v", unmarshaledParams.DisablePing, params.DisablePing)
	}
}

// TestSystemStatusJSON tests SystemStatus JSON serialization and deserialization
func TestSystemStatusJSON(t *testing.T) {
	// Create a SystemStatus
	now := time.Now().Truncate(time.Second) // Truncate to avoid fractional seconds comparison issues
	status := SystemStatus{
		Status:           "ok",
		LastScan:         now,
		DeviceCount:      10,
		DatabaseSize:     1024 * 1024,
		CPUUsage:         5.2,
		MemoryUsage:      256 * 1024 * 1024,
		DiskUsage:        2 * 1024 * 1024 * 1024,
		ScannerStatus:    "idle",
		MaintenanceStats: "Backup completed successfully",
		Version:          "0.1.0",
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(status)
	if err != nil {
		t.Fatalf("Failed to marshal SystemStatus to JSON: %v", err)
	}

	// Unmarshal back to SystemStatus
	var unmarshaledStatus SystemStatus
	err = json.Unmarshal(jsonData, &unmarshaledStatus)
	if err != nil {
		t.Fatalf("Failed to unmarshal SystemStatus from JSON: %v", err)
	}

	// Verify the fields match
	if unmarshaledStatus.Status != status.Status {
		t.Errorf("Status mismatch: got %s, expected %s", unmarshaledStatus.Status, status.Status)
	}
	if !unmarshaledStatus.LastScan.Equal(status.LastScan) {
		t.Errorf("LastScan mismatch: got %v, expected %v", unmarshaledStatus.LastScan, status.LastScan)
	}
	if unmarshaledStatus.DeviceCount != status.DeviceCount {
		t.Errorf("DeviceCount mismatch: got %d, expected %d", unmarshaledStatus.DeviceCount, status.DeviceCount)
	}
	if unmarshaledStatus.DatabaseSize != status.DatabaseSize {
		t.Errorf("DatabaseSize mismatch: got %d, expected %d", unmarshaledStatus.DatabaseSize, status.DatabaseSize)
	}
	if unmarshaledStatus.CPUUsage != status.CPUUsage {
		t.Errorf("CPUUsage mismatch: got %f, expected %f", unmarshaledStatus.CPUUsage, status.CPUUsage)
	}
	if unmarshaledStatus.MemoryUsage != status.MemoryUsage {
		t.Errorf("MemoryUsage mismatch: got %d, expected %d", unmarshaledStatus.MemoryUsage, status.MemoryUsage)
	}
	if unmarshaledStatus.DiskUsage != status.DiskUsage {
		t.Errorf("DiskUsage mismatch: got %d, expected %d", unmarshaledStatus.DiskUsage, status.DiskUsage)
	}
	if unmarshaledStatus.ScannerStatus != status.ScannerStatus {
		t.Errorf("ScannerStatus mismatch: got %s, expected %s", unmarshaledStatus.ScannerStatus, status.ScannerStatus)
	}
	if unmarshaledStatus.MaintenanceStats != status.MaintenanceStats {
		t.Errorf("MaintenanceStats mismatch: got %s, expected %s", unmarshaledStatus.MaintenanceStats, status.MaintenanceStats)
	}
	if unmarshaledStatus.Version != status.Version {
		t.Errorf("Version mismatch: got %s, expected %s", unmarshaledStatus.Version, status.Version)
	}
}

// TestDeviceDetails tests DeviceDetails structure and methods
func TestDeviceDetails(t *testing.T) {
	// Create a Device
	device := Device{
		ID:           1,
		IPAddress:    "192.168.1.100",
		MACAddress:   "00:11:22:33:44:55",
		Hostname:     "test-server",
		OSFingerprint: "Linux 5.4",
		FirstSeen:    time.Now().Add(-24 * time.Hour),
		LastSeen:     time.Now(),
	}

	// Create Ports
	ports := []*Port{
		{
			ID:             1,
			DeviceID:       1,
			PortNumber:     22,
			Protocol:       "tcp",
			ServiceName:    "ssh",
			ServiceVersion: "OpenSSH 8.2p1",
			FirstSeen:      time.Now().Add(-24 * time.Hour),
			LastSeen:       time.Now(),
		},
		{
			ID:             2,
			DeviceID:       1,
			PortNumber:     80,
			Protocol:       "tcp",
			ServiceName:    "http",
			ServiceVersion: "nginx 1.18.0",
			FirstSeen:      time.Now().Add(-24 * time.Hour),
			LastSeen:       time.Now(),
		},
	}

	// Create DeviceDetails
	deviceDetails := DeviceDetails{
		Device: device,
		Ports:  ports,
	}

	// Verify DeviceDetails fields
	if deviceDetails.ID != device.ID {
		t.Errorf("ID mismatch: got %d, expected %d", deviceDetails.ID, device.ID)
	}
	if deviceDetails.IPAddress != device.IPAddress {
		t.Errorf("IPAddress mismatch: got %s, expected %s", deviceDetails.IPAddress, device.IPAddress)
	}
	if len(deviceDetails.Ports) != len(ports) {
		t.Errorf("Ports count mismatch: got %d, expected %d", len(deviceDetails.Ports), len(ports))
	}

	// Test JSON marshaling and unmarshaling
	jsonData, err := json.Marshal(deviceDetails)
	if err != nil {
		t.Fatalf("Failed to marshal DeviceDetails to JSON: %v", err)
	}

	var unmarshaledDeviceDetails DeviceDetails
	err = json.Unmarshal(jsonData, &unmarshaledDeviceDetails)
	if err != nil {
		t.Fatalf("Failed to unmarshal DeviceDetails from JSON: %v", err)
	}

	// Verify unmarshaled data
	if unmarshaledDeviceDetails.ID != deviceDetails.ID {
		t.Errorf("Unmarshaled ID mismatch: got %d, expected %d", unmarshaledDeviceDetails.ID, deviceDetails.ID)
	}
	if unmarshaledDeviceDetails.IPAddress != deviceDetails.IPAddress {
		t.Errorf("Unmarshaled IPAddress mismatch: got %s, expected %s", unmarshaledDeviceDetails.IPAddress, deviceDetails.IPAddress)
	}
	if len(unmarshaledDeviceDetails.Ports) != len(deviceDetails.Ports) {
		t.Errorf("Unmarshaled Ports count mismatch: got %d, expected %d", len(unmarshaledDeviceDetails.Ports), len(deviceDetails.Ports))
	}
}

// TestNetworkStats tests NetworkStats JSON serialization and deserialization
func TestNetworkStats(t *testing.T) {
	// Create NetworkStats
	stats := NetworkStats{
		TotalDevices: 25,
		TotalPorts:   120,
		OSDistribution: map[string]int{
			"Linux":   12,
			"Windows": 8,
			"macOS":   5,
		},
		PortDistribution: map[int]int{
			22:   18,
			80:   22,
			443:  15,
			3389: 8,
		},
		ServiceDistribution: map[string]int{
			"ssh":  18,
			"http": 22,
			"https": 15,
			"rdp":  8,
		},
		NewDevices:            3,
		ChangedDevices:        5,
		ActiveVulnerabilities: 2,
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(stats)
	if err != nil {
		t.Fatalf("Failed to marshal NetworkStats to JSON: %v", err)
	}

	// Unmarshal back to NetworkStats
	var unmarshaledStats NetworkStats
	err = json.Unmarshal(jsonData, &unmarshaledStats)
	if err != nil {
		t.Fatalf("Failed to unmarshal NetworkStats from JSON: %v", err)
	}

	// Verify the fields match
	if unmarshaledStats.TotalDevices != stats.TotalDevices {
		t.Errorf("TotalDevices mismatch: got %d, expected %d", unmarshaledStats.TotalDevices, stats.TotalDevices)
	}
	if unmarshaledStats.TotalPorts != stats.TotalPorts {
		t.Errorf("TotalPorts mismatch: got %d, expected %d", unmarshaledStats.TotalPorts, stats.TotalPorts)
	}

	// Check OS distribution
	if len(unmarshaledStats.OSDistribution) != len(stats.OSDistribution) {
		t.Errorf("OSDistribution length mismatch: got %d, expected %d", len(unmarshaledStats.OSDistribution), len(stats.OSDistribution))
	}
	for os, count := range stats.OSDistribution {
		if unmarshaledStats.OSDistribution[os] != count {
			t.Errorf("OSDistribution count mismatch for %s: got %d, expected %d", os, unmarshaledStats.OSDistribution[os], count)
		}
	}

	// Check port distribution
	if len(unmarshaledStats.PortDistribution) != len(stats.PortDistribution) {
		t.Errorf("PortDistribution length mismatch: got %d, expected %d", len(unmarshaledStats.PortDistribution), len(stats.PortDistribution))
	}
	for port, count := range stats.PortDistribution {
		if unmarshaledStats.PortDistribution[port] != count {
			t.Errorf("PortDistribution count mismatch for port %d: got %d, expected %d", port, unmarshaledStats.PortDistribution[port], count)
		}
	}

	// Check other fields
	if unmarshaledStats.NewDevices != stats.NewDevices {
		t.Errorf("NewDevices mismatch: got %d, expected %d", unmarshaledStats.NewDevices, stats.NewDevices)
	}
	if unmarshaledStats.ChangedDevices != stats.ChangedDevices {
		t.Errorf("ChangedDevices mismatch: got %d, expected %d", unmarshaledStats.ChangedDevices, stats.ChangedDevices)
	}
	if unmarshaledStats.ActiveVulnerabilities != stats.ActiveVulnerabilities {
		t.Errorf("ActiveVulnerabilities mismatch: got %d, expected %d", unmarshaledStats.ActiveVulnerabilities, stats.ActiveVulnerabilities)
	}
}
