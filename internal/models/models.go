// Package models defines the data structures used throughout the Panopticon Scanner.
// It contains all the data models that represent network devices, scans, ports,
// system status, and other entities used by the application.
package models

import "time"

// Device represents a basic network device
type Device struct {
	ID           int64     `json:"id"`
	IPAddress    string    `json:"ipAddress"`
	MACAddress   string    `json:"macAddress"`
	Hostname     string    `json:"hostname"`
	OSFingerprint string    `json:"osFingerprint"`
	FirstSeen    time.Time `json:"firstSeen"`
	LastSeen     time.Time `json:"lastSeen"`
	PortCount    int       `json:"portCount,omitempty"`
}

// DeviceDetails represents a device with its associated ports
type DeviceDetails struct {
	Device
	Ports []*Port `json:"ports"`
}

// Port represents a network port on a device
type Port struct {
	ID             int64     `json:"id"`
	DeviceID       int64     `json:"deviceId"`
	PortNumber     int       `json:"portNumber"`
	Protocol       string    `json:"protocol"`
	ServiceName    string    `json:"serviceName"`
	ServiceVersion string    `json:"serviceVersion"`
	FirstSeen      time.Time `json:"firstSeen"`
	LastSeen       time.Time `json:"lastSeen"`
}

// Scan represents a network scan operation
type Scan struct {
	ID           int64     `json:"id"`
	Timestamp    time.Time `json:"timestamp"`
	Template     string    `json:"template"`
	Duration     int       `json:"duration"`
	DevicesFound int       `json:"devicesFound"`
	PortsFound   int       `json:"portsFound"`
	Status       string    `json:"status"` // running, completed, error
	ErrorMessage string    `json:"errorMessage,omitempty"`
}

// Change represents a detected change in the network
type Change struct {
	ID         int64     `json:"id"`
	ScanID     int64     `json:"scanId"`
	DeviceID   int64     `json:"deviceId"`
	ChangeType string    `json:"changeType"` // new_device, device_change, new_port, port_change, etc.
	Details    string    `json:"details"`
	Timestamp  time.Time `json:"timestamp"`
}

// ScanParameters represents parameters for a manual scan
type ScanParameters struct {
	Template      string `json:"template"`
	TargetNetwork string `json:"targetNetwork,omitempty"`
	RateLimit     int    `json:"rateLimit,omitempty"`
	ScanAllPorts  bool   `json:"scanAllPorts,omitempty"`
	DisablePing   bool   `json:"disablePing,omitempty"`
}

// SystemStatus represents the overall system status
type SystemStatus struct {
	Status           string    `json:"status"` // ok, warning, error
	LastScan         time.Time `json:"lastScan"`
	DeviceCount      int       `json:"deviceCount"`
	DatabaseSize     int64     `json:"databaseSize"` // in bytes
	CPUUsage         float64   `json:"cpuUsage"`     // percentage
	MemoryUsage      int64     `json:"memoryUsage"`  // in bytes
	DiskUsage        int64     `json:"diskUsage"`    // in bytes
	ScannerStatus    string    `json:"scannerStatus"`
	MaintenanceStats string    `json:"maintenanceStats,omitempty"`
	Version          string    `json:"version"`
}

// Config represents application configuration
type Config struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	Encrypted   bool   `json:"encrypted"`
	Description string `json:"description,omitempty"`
}

// Log represents a log entry
type Log struct {
	ID        int64     `json:"id"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
	Component string    `json:"component"`
	Timestamp time.Time `json:"timestamp"`
}

// Report represents a generated report
type Report struct {
	ID         int64     `json:"id"`
	Name       string    `json:"name"`
	Type       string    `json:"type"`
	Format     string    `json:"format"`
	Path       string    `json:"path"`
	CreatedAt  time.Time `json:"createdAt"`
	Parameters string    `json:"parameters"`
}

// Notification represents a system notification
type Notification struct {
	ID        int64     `json:"id"`
	Level     string    `json:"level"` // info, warning, error
	Message   string    `json:"message"`
	Read      bool      `json:"read"`
	Timestamp time.Time `json:"timestamp"`
}

// ReportTemplate represents a report template configuration
type ReportTemplate struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Format      string   `json:"format"`
	Sections    []string `json:"sections"`
}

// ScanTemplate represents a scan template configuration
type ScanTemplate struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	NmapArgs    []string `json:"nmapArgs"`
	RateLimit   int      `json:"rateLimit"`
}

// NetworkStats represents network statistics
type NetworkStats struct {
	TotalDevices       int               `json:"totalDevices"`
	TotalPorts         int               `json:"totalPorts"`
	OSDistribution     map[string]int    `json:"osDistribution"`
	PortDistribution   map[int]int       `json:"portDistribution"`
	ServiceDistribution map[string]int   `json:"serviceDistribution"`
	NewDevices         int               `json:"newDevices"`
	ChangedDevices     int               `json:"changedDevices"`
	ActiveVulnerabilities int            `json:"activeVulnerabilities"`
}

// ScanOptions represents nmap scan options
type ScanOptions struct {
	ScanType          string   `json:"scanType"`          // SYN, connect, UDP, etc.
	PortRange         string   `json:"portRange"`         // "1-1000", "1-65535", etc.
	ServiceDetection  bool     `json:"serviceDetection"`  // -sV
	OSDetection       bool     `json:"osDetection"`       // -O
	ScriptScan        bool     `json:"scriptScan"`        // --script=default
	AggressiveScanning bool    `json:"aggressiveScanning"` // -A
	TimingTemplate    int      `json:"timingTemplate"`    // -T0 to -T5
	DisablePing       bool     `json:"disablePing"`       // -Pn
	CustomArguments   []string `json:"customArguments"`
}
