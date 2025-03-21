// Package database provides database operations for the Panopticon Scanner application.
// It handles all interactions with the SQLite database including initialization,
// optimization, and CRUD operations for devices, scans, and other data.
package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"panopticon-scanner/internal/models"
)

// DB represents the database connection
type DB struct {
	*sql.DB
	Path   string // Exported for integration tests
	logger *zerolog.Logger
	sync.Mutex
}

// New creates a new database connection
func New(path string) (*DB, error) {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Open database connection
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Set connection parameters
	db.SetMaxOpenConns(1) // SQLite supports only one writer at a time
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(time.Hour)

	// Create logger
	logger := log.With().Str("component", "database").Logger()

	// Create DB instance
	dbInstance := &DB{
		DB:     db,
		Path:   path,
		logger: &logger,
	}

	// Initialize the database schema
	if err := dbInstance.initializeDB(); err != nil {
		db.Close()
		return nil, err
	}

	// Run PRAGMA statements for optimization
	if err := dbInstance.optimizeDB(); err != nil {
		logger.Warn().Err(err).Msg("Failed to set some database optimization parameters")
	}

	return dbInstance, nil
}

// Initialize database schema
func (db *DB) initializeDB() error {
	db.logger.Info().Msg("Initializing database schema")

	schema := `
	-- Devices table
	CREATE TABLE IF NOT EXISTS devices (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		ip_address TEXT NOT NULL,
		mac_address TEXT,
		hostname TEXT,
		os_fingerprint TEXT,
		first_seen TIMESTAMP NOT NULL,
		last_seen TIMESTAMP NOT NULL,
		UNIQUE(ip_address, mac_address)
	);

	-- Ports table
	CREATE TABLE IF NOT EXISTS ports (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		device_id INTEGER NOT NULL,
		port_number INTEGER NOT NULL,
		protocol TEXT NOT NULL,
		service_name TEXT,
		service_version TEXT,
		first_seen TIMESTAMP NOT NULL,
		last_seen TIMESTAMP NOT NULL,
		FOREIGN KEY (device_id) REFERENCES devices(id) ON DELETE CASCADE,
		UNIQUE(device_id, port_number, protocol)
	);

	-- Scans table
	CREATE TABLE IF NOT EXISTS scans (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp TIMESTAMP NOT NULL,
		template TEXT NOT NULL,
		duration INTEGER DEFAULT 0,
		devices_found INTEGER DEFAULT 0,
		ports_found INTEGER DEFAULT 0,
		status TEXT NOT NULL,
		error_message TEXT
	);

	-- Changes table
	CREATE TABLE IF NOT EXISTS changes (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		scan_id INTEGER NOT NULL,
		device_id INTEGER NOT NULL,
		change_type TEXT NOT NULL,
		details TEXT NOT NULL,
		timestamp TIMESTAMP NOT NULL,
		FOREIGN KEY (scan_id) REFERENCES scans(id) ON DELETE CASCADE,
		FOREIGN KEY (device_id) REFERENCES devices(id) ON DELETE CASCADE
	);

	-- Configuration table
	CREATE TABLE IF NOT EXISTS configuration (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL,
		encrypted BOOLEAN DEFAULT FALSE,
		description TEXT
	);

	-- Logs table
	CREATE TABLE IF NOT EXISTS logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		level TEXT NOT NULL,
		message TEXT NOT NULL,
		component TEXT NOT NULL,
		timestamp TIMESTAMP NOT NULL
	);

	-- Create indexes
	CREATE INDEX IF NOT EXISTS idx_devices_ip ON devices(ip_address);
	CREATE INDEX IF NOT EXISTS idx_devices_mac ON devices(mac_address);
	CREATE INDEX IF NOT EXISTS idx_ports_device_id ON ports(device_id);
	CREATE INDEX IF NOT EXISTS idx_ports_port_protocol ON ports(port_number, protocol);
	CREATE INDEX IF NOT EXISTS idx_scans_timestamp ON scans(timestamp);
	CREATE INDEX IF NOT EXISTS idx_changes_scan_id ON changes(scan_id);
	CREATE INDEX IF NOT EXISTS idx_changes_device_id ON changes(device_id);
	CREATE INDEX IF NOT EXISTS idx_logs_level_component ON logs(level, component);
	CREATE INDEX IF NOT EXISTS idx_logs_timestamp ON logs(timestamp);
	`

	_, err := db.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to initialize database schema: %w", err)
	}

	return nil
}

// optimizeDB sets SQLite optimization parameters
func (db *DB) optimizeDB() error {
	// Enable WAL mode for better concurrency
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return err
	}

	// Set synchronous mode to NORMAL for better performance with adequate safety
	if _, err := db.Exec("PRAGMA synchronous=NORMAL"); err != nil {
		return err
	}

	// Enable foreign keys
	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		return err
	}

	// Make sure dates are stored in ISO8601 string format
	if _, err := db.Exec("PRAGMA datetime_bytes=iso8601"); err != nil {
		db.logger.Warn().Err(err).Msg("Failed to set datetime_bytes PRAGMA - SQLite may be using different timestamp format")
	}

	// Set cache size for better performance
	if _, err := db.Exec("PRAGMA cache_size=-20000"); err != nil { // Approx 20MB cache
		db.logger.Warn().Err(err).Msg("Failed to set cache_size PRAGMA")
	}

	// Set mmap_size for improved performance
	if _, err := db.Exec("PRAGMA mmap_size=134217728"); err != nil { // 128MB
		db.logger.Warn().Err(err).Msg("Failed to set mmap_size PRAGMA - older SQLite version might not support this")
	}

	// Set busy timeout to avoid "database is locked" errors
	if _, err := db.Exec("PRAGMA busy_timeout=10000"); err != nil { // 10 seconds
		db.logger.Warn().Err(err).Msg("Failed to set busy_timeout PRAGMA")
	}

	return nil
}

// ExecuteWithRetry attempts to execute a function with retries for transient errors
func (db *DB) ExecuteWithRetry(maxRetries int, retryDelay time.Duration, operation func() error) error {
	var err error
	for attempt := 0; attempt < maxRetries; attempt++ {
		err = operation()
		if err == nil {
			return nil
		}
		
		// Check if the error is one we should retry
		if strings.Contains(err.Error(), "database is locked") || 
			strings.Contains(err.Error(), "busy") {
			db.logger.Warn().
				Err(err).
				Int("attempt", attempt+1).
				Int("maxRetries", maxRetries).
				Msg("Retrying database operation")
			
			// Wait before retrying
			time.Sleep(retryDelay)
			
			// Increase delay for next attempt
			retryDelay = retryDelay * 2
			continue
		}
		
		// Not a retryable error
		break
	}
	
	return fmt.Errorf("database operation failed after %d attempts: %w", maxRetries, err)
}

// SaveDevice saves or updates a device in the database
func (db *DB) SaveDevice(device *models.Device) (int64, error) {
	db.Lock()
	defer db.Unlock()

	// Start a transaction
	tx, err := db.Begin()
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Ensure transaction is rolled back in case of error
	defer func() {
		if tx != nil {
			tx.Rollback()
		}
	}()

	// Round timestamps to the nearest hour for deduplication
	roundedTime := time.Now().Truncate(time.Hour)

	// Check if device exists
	var id int64
	err = tx.QueryRow(
		`SELECT id FROM devices WHERE ip_address = ? AND
		 (mac_address = ? OR (mac_address IS NULL AND ? IS NULL))`,
		device.IPAddress, device.MACAddress, device.MACAddress,
	).Scan(&id)

	if err == sql.ErrNoRows {
		// Insert new device
		res, err := tx.Exec(
			`INSERT INTO devices (ip_address, mac_address, hostname, os_fingerprint, first_seen, last_seen)
			 VALUES (?, ?, ?, ?, ?, ?)`,
			device.IPAddress, device.MACAddress, device.Hostname, device.OSFingerprint,
			roundedTime, roundedTime,
		)
		if err != nil {
			return 0, fmt.Errorf("failed to insert device: %w", err)
		}

		id, err = res.LastInsertId()
		if err != nil {
			return 0, fmt.Errorf("failed to get inserted device ID: %w", err)
		}

		// Log new device discovery
		db.logger.Info().
			Str("ip", device.IPAddress).
			Str("hostname", device.Hostname).
			Int64("id", id).
			Msg("New device discovered")

		// Get latest scan ID for change record or create a dummy record if needed
		var scanID int64
		scanErr := tx.QueryRow("SELECT COALESCE(MAX(id), 1) FROM scans").Scan(&scanID)
		if scanErr != nil {
			scanID = 1 // Fallback to ID 1 if query fails
			db.logger.Warn().Err(scanErr).Msg("Failed to get latest scan ID, using default")
		}
		
		// Insert change record
		if _, err := tx.Exec(
			`INSERT INTO changes (scan_id, device_id, change_type, details, timestamp)
			 VALUES (?, ?, ?, ?, ?)`,
			scanID, id, "new_device", fmt.Sprintf("New device discovered: %s", device.IPAddress),
			time.Now(),
		); err != nil {
			db.logger.Warn().Err(err).Int64("deviceID", id).Msg("Failed to record device change")
		}
	} else if err != nil {
		return 0, fmt.Errorf("failed to check if device exists: %w", err)
	} else {
		// Device exists, check if we need to update
		var oldHostname, oldOsFingerprint string
		var oldMacAddress sql.NullString

		err = tx.QueryRow(
			`SELECT hostname, os_fingerprint, mac_address FROM devices WHERE id = ?`,
			id,
		).Scan(&oldHostname, &oldOsFingerprint, &oldMacAddress)

		if err != nil {
			return 0, fmt.Errorf("failed to retrieve existing device data: %w", err)
		}

		// Check if anything changed
		hasChanges := false
		changeDetails := ""

		// Compare MAC address (handle NULL case)
		oldMac := ""
		if oldMacAddress.Valid {
			oldMac = oldMacAddress.String
		}

		if device.MACAddress != oldMac && device.MACAddress != "" {
			hasChanges = true
			changeDetails += fmt.Sprintf("MAC address changed: %s -> %s; ", oldMac, device.MACAddress)
		}

		// Compare hostname
		if device.Hostname != oldHostname && device.Hostname != "" {
			hasChanges = true
			changeDetails += fmt.Sprintf("Hostname changed: %s -> %s; ", oldHostname, device.Hostname)
		}

		// Compare OS fingerprint
		if device.OSFingerprint != oldOsFingerprint && device.OSFingerprint != "" {
			hasChanges = true
			changeDetails += fmt.Sprintf("OS changed: %s -> %s; ", oldOsFingerprint, device.OSFingerprint)
		}

		// Update device if anything changed
		if hasChanges || device.LastSeen.After(time.Now().Add(-time.Hour)) {
			// Only update non-empty fields
			macValue := device.MACAddress
			hostnameValue := device.Hostname
			osValue := device.OSFingerprint

			// If new values are empty, keep old values
			if macValue == "" && oldMacAddress.Valid {
				macValue = oldMacAddress.String
			}

			if hostnameValue == "" {
				hostnameValue = oldHostname
			}

			if osValue == "" {
				osValue = oldOsFingerprint
			}

			_, err = tx.Exec(
				`UPDATE devices
				 SET mac_address = ?, hostname = ?, os_fingerprint = ?, last_seen = ?
				 WHERE id = ?`,
				macValue, hostnameValue, osValue, roundedTime, id,
			)

			if err != nil {
				return 0, fmt.Errorf("failed to update device: %w", err)
			}

			// Record change if anything significant changed
			if hasChanges {
				// Get latest scan ID for change record
				var scanID int64
				scanErr := tx.QueryRow("SELECT COALESCE(MAX(id), 1) FROM scans").Scan(&scanID)
				if scanErr != nil {
					scanID = 1 // Fallback to ID 1 if query fails
					db.logger.Warn().Err(scanErr).Msg("Failed to get latest scan ID, using default")
				}
				
				if _, err := tx.Exec(
					`INSERT INTO changes (scan_id, device_id, change_type, details, timestamp)
					 VALUES (?, ?, ?, ?, ?)`,
					scanID, id, "device_change", changeDetails, time.Now(),
				); err != nil {
					db.logger.Warn().Err(err).Int64("deviceID", id).Msg("Failed to record device change")
				}
			}

			db.logger.Debug().
				Int64("id", id).
				Str("ip", device.IPAddress).
				Bool("hasChanges", hasChanges).
				Msg("Updated existing device")
		}
	}

	// Commit the transaction
	if err = tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}
	
	// Set tx to nil to prevent rollback in deferred function
	tx = nil

	return id, nil
}

// SavePort saves or updates a port in the database
func (db *DB) SavePort(port *models.Port) error {
	db.Lock()
	defer db.Unlock()

	// Start a transaction
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Ensure transaction is rolled back in case of error
	defer func() {
		if tx != nil {
			tx.Rollback()
		}
	}()

	// Round timestamps to the nearest hour for deduplication
	roundedTime := time.Now().Truncate(time.Hour)

	// Check if port exists
	var id int64
	var oldServiceName, oldServiceVersion string

	err = tx.QueryRow(
		`SELECT id, service_name, service_version
		 FROM ports
		 WHERE device_id = ? AND port_number = ? AND protocol = ?`,
		port.DeviceID, port.PortNumber, port.Protocol,
	).Scan(&id, &oldServiceName, &oldServiceVersion)

	if err == sql.ErrNoRows {
		// Insert new port
		res, err := tx.Exec(
			`INSERT INTO ports (device_id, port_number, protocol, service_name, service_version, first_seen, last_seen)
			 VALUES (?, ?, ?, ?, ?, ?, ?)`,
			port.DeviceID, port.PortNumber, port.Protocol, port.ServiceName, port.ServiceVersion,
			roundedTime, roundedTime,
		)

		if err != nil {
			return fmt.Errorf("failed to insert port: %w", err)
		}

		id, err = res.LastInsertId()
		if err != nil {
			return fmt.Errorf("failed to get inserted port ID: %w", err)
		}

		// Log new port discovery
		db.logger.Debug().
			Int64("deviceID", port.DeviceID).
			Int("port", port.PortNumber).
			Str("protocol", port.Protocol).
			Msg("New port discovered")

		// Get latest scan ID for change record
		var scanID int64
		scanErr := tx.QueryRow("SELECT COALESCE(MAX(id), 1) FROM scans").Scan(&scanID)
		if scanErr != nil {
			scanID = 1 // Fallback to ID 1 if query fails
			db.logger.Warn().Err(scanErr).Msg("Failed to get latest scan ID, using default")
		}
		
		// Insert change record for new port
		if _, err := tx.Exec(
			`INSERT INTO changes (scan_id, device_id, change_type, details, timestamp)
			 VALUES (?, ?, ?, ?, ?)`,
			scanID, port.DeviceID, "new_port",
			fmt.Sprintf("New port discovered: %d/%s - %s",
				port.PortNumber, port.Protocol, port.ServiceName),
			time.Now(),
		); err != nil {
			db.logger.Warn().Err(err).Int64("portID", id).Msg("Failed to record port change")
		}

	} else if err != nil {
		return fmt.Errorf("failed to check if port exists: %w", err)
	} else {
		// Port exists, check if service information changed
		serviceChanged := (port.ServiceName != oldServiceName && port.ServiceName != "") ||
						  (port.ServiceVersion != oldServiceVersion && port.ServiceVersion != "")

		// Update port if service changed or last seen needs to be updated
		if serviceChanged || port.LastSeen.After(time.Now().Add(-time.Hour)) {
			// Only update non-empty fields
			serviceNameValue := port.ServiceName
			serviceVersionValue := port.ServiceVersion

			// If new values are empty, keep old values
			if serviceNameValue == "" {
				serviceNameValue = oldServiceName
			}

			if serviceVersionValue == "" {
				serviceVersionValue = oldServiceVersion
			}

			_, err = tx.Exec(
				`UPDATE ports
				 SET service_name = ?, service_version = ?, last_seen = ?
				 WHERE id = ?`,
				serviceNameValue, serviceVersionValue, roundedTime, id,
			)

			if err != nil {
				return fmt.Errorf("failed to update port: %w", err)
			}

			// Record change if service information changed
			if serviceChanged {
				// Get latest scan ID for change record
				var scanID int64
				scanErr := tx.QueryRow("SELECT COALESCE(MAX(id), 1) FROM scans").Scan(&scanID)
				if scanErr != nil {
					scanID = 1 // Fallback to ID 1 if query fails
					db.logger.Warn().Err(scanErr).Msg("Failed to get latest scan ID, using default")
				}
				
				if _, err := tx.Exec(
					`INSERT INTO changes (scan_id, device_id, change_type, details, timestamp)
					 VALUES (?, ?, ?, ?, ?)`,
					scanID, port.DeviceID, "port_change",
					fmt.Sprintf("Service on port %d/%s changed: %s %s -> %s %s",
						port.PortNumber, port.Protocol,
						oldServiceName, oldServiceVersion,
						serviceNameValue, serviceVersionValue),
					time.Now(),
				); err != nil {
					db.logger.Warn().Err(err).Int64("portID", id).Msg("Failed to record port change")
				}
			}

			db.logger.Debug().
				Int64("id", id).
				Int("port", port.PortNumber).
				Bool("serviceChanged", serviceChanged).
				Msg("Updated existing port")
		}
	}

	// Commit the transaction
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	
	// Set tx to nil to prevent rollback in deferred function
	tx = nil

	return nil
}

// GetDevice retrieves a device by ID
func (db *DB) GetDevice(id int64) (*models.Device, error) {
	var device models.Device

	err := db.QueryRow(
		`SELECT id, ip_address, mac_address, hostname, os_fingerprint, first_seen, last_seen
		 FROM devices WHERE id = ?`, id,
	).Scan(
		&device.ID,
		&device.IPAddress,
		&device.MACAddress,
		&device.Hostname,
		&device.OSFingerprint,
		&device.FirstSeen,
		&device.LastSeen,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get device: %w", err)
	}

	// Get port count
	err = db.QueryRow(
		`SELECT COUNT(*) FROM ports WHERE device_id = ?`, id,
	).Scan(&device.PortCount)

	if err != nil {
		return nil, fmt.Errorf("failed to get port count: %w", err)
	}

	return &device, nil
}

// GetDeviceByIP retrieves a device by IP address
func (db *DB) GetDeviceByIP(ipAddress string) (*models.Device, error) {
	var device models.Device

	err := db.QueryRow(
		`SELECT id, ip_address, mac_address, hostname, os_fingerprint, first_seen, last_seen
		 FROM devices WHERE ip_address = ?`, ipAddress,
	).Scan(
		&device.ID,
		&device.IPAddress,
		&device.MACAddress,
		&device.Hostname,
		&device.OSFingerprint,
		&device.FirstSeen,
		&device.LastSeen,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get device by IP: %w", err)
	}

	// Get port count
	err = db.QueryRow(
		`SELECT COUNT(*) FROM ports WHERE device_id = ?`, device.ID,
	).Scan(&device.PortCount)

	if err != nil {
		return nil, fmt.Errorf("failed to get port count: %w", err)
	}

	return &device, nil
}

// GetDeviceDetails retrieves a device with its ports
func (db *DB) GetDeviceDetails(id int64) (*models.DeviceDetails, error) {
	// Get the device first
	device, err := db.GetDevice(id)
	if err != nil {
		return nil, err
	}

	// Get the ports for this device
	rows, err := db.Query(
		`SELECT id, device_id, port_number, protocol, service_name, service_version, first_seen, last_seen
		 FROM ports WHERE device_id = ? ORDER BY port_number`, id,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get ports: %w", err)
	}
	defer rows.Close()

	var ports []*models.Port
	for rows.Next() {
		var port models.Port
		err := rows.Scan(
			&port.ID,
			&port.DeviceID,
			&port.PortNumber,
			&port.Protocol,
			&port.ServiceName,
			&port.ServiceVersion,
			&port.FirstSeen,
			&port.LastSeen,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan port row: %w", err)
		}
		ports = append(ports, &port)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating port rows: %w", err)
	}

	return &models.DeviceDetails{
		Device: *device,
		Ports:  ports,
	}, nil
}

// GetAllDevices retrieves all devices
func (db *DB) GetAllDevices() ([]*models.Device, error) {
	rows, err := db.Query(
		`SELECT d.id, d.ip_address, d.mac_address, d.hostname, d.os_fingerprint, d.first_seen, d.last_seen,
		 (SELECT COUNT(*) FROM ports WHERE device_id = d.id) as port_count
		 FROM devices d
		 ORDER BY d.last_seen DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query devices: %w", err)
	}
	defer rows.Close()

	var devices []*models.Device
	for rows.Next() {
		var device models.Device
		err := rows.Scan(
			&device.ID,
			&device.IPAddress,
			&device.MACAddress,
			&device.Hostname,
			&device.OSFingerprint,
			&device.FirstSeen,
			&device.LastSeen,
			&device.PortCount,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan device row: %w", err)
		}
		devices = append(devices, &device)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating device rows: %w", err)
	}

	return devices, nil
}

// CreateScan creates a new scan record
func (db *DB) CreateScan(template string) (int64, error) {
	result, err := db.Exec(
		`INSERT INTO scans (timestamp, template, status, duration, devices_found, ports_found)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		time.Now(), template, "running", 0, 0, 0,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to create scan: %w", err)
	}

	scanID, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get inserted scan ID: %w", err)
	}

	return scanID, nil
}

// CreateScanFromModel creates a new scan with all fields provided
func (db *DB) CreateScanFromModel(scan *models.Scan) (int64, error) {
	result, err := db.Exec(
		`INSERT INTO scans (timestamp, template, status, duration, devices_found, ports_found)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		scan.Timestamp, scan.Template, scan.Status, scan.Duration, scan.DevicesFound, scan.PortsFound,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to create scan: %w", err)
	}

	scanID, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get inserted scan ID: %w", err)
	}

	return scanID, nil
}

// UpdateScan updates an existing scan record
func (db *DB) UpdateScan(id int64, status string, devicesFound, portsFound int, duration time.Duration, errorMsg string) error {
	if status == "" {
		return fmt.Errorf("status cannot be empty")
	}

	// Convert duration to seconds (as INTEGER in SQLite)
	durationSecs := int(duration.Seconds())

	_, err := db.Exec(
		`UPDATE scans
		 SET status = ?, devices_found = ?, ports_found = ?, duration = ?, error_message = ?
		 WHERE id = ?`,
		status, devicesFound, portsFound, durationSecs, errorMsg, id,
	)

	if err != nil {
		return fmt.Errorf("failed to update scan #%d: %w", id, err)
	}

	return nil
}

// UpdateScanFromModel updates a scan using a Scan model
func (db *DB) UpdateScanFromModel(scan *models.Scan) error {
	_, err := db.Exec(
		`UPDATE scans SET status = ?, duration = ?, devices_found = ?, ports_found = ?, error_message = ?
		 WHERE id = ?`,
		scan.Status, scan.Duration, scan.DevicesFound, scan.PortsFound, scan.ErrorMessage, scan.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update scan: %w", err)
	}

	return nil
}

// GetScan retrieves a scan by ID
func (db *DB) GetScan(id int64) (*models.Scan, error) {
	var scan models.Scan
	var errorMsg sql.NullString

	err := db.QueryRow(
		`SELECT id, timestamp, template, duration, devices_found, ports_found, status, error_message
		 FROM scans WHERE id = ?`, id,
	).Scan(
		&scan.ID,
		&scan.Timestamp,
		&scan.Template,
		&scan.Duration,
		&scan.DevicesFound,
		&scan.PortsFound,
		&scan.Status,
		&errorMsg,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get scan: %w", err)
	}

	if errorMsg.Valid {
		scan.ErrorMessage = errorMsg.String
	}

	return &scan, nil
}

// GetRecentScans retrieves recent scans with a limit
func (db *DB) GetRecentScans(limit int) ([]*models.Scan, error) {
	rows, err := db.Query(
		`SELECT id, timestamp, template, duration, devices_found, ports_found, status, error_message
		 FROM scans
		 ORDER BY timestamp DESC
		 LIMIT ?`, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query recent scans: %w", err)
	}
	defer rows.Close()

	var scans []*models.Scan
	for rows.Next() {
		var scan models.Scan
		var errorMsg sql.NullString

		err := rows.Scan(
			&scan.ID,
			&scan.Timestamp,
			&scan.Template,
			&scan.Duration,
			&scan.DevicesFound,
			&scan.PortsFound,
			&scan.Status,
			&errorMsg,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		if errorMsg.Valid {
			scan.ErrorMessage = errorMsg.String
		}

		scans = append(scans, &scan)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating scan rows: %w", err)
	}

	return scans, nil
}

// RecordScan is a convenience method to create and update a scan in one step
func (db *DB) RecordScan(duration time.Duration, devicesFound, portsFound int, status string) (int64, error) {
	// Create scan with "default" template
	scanID, err := db.CreateScan("default")
	if err != nil {
		return 0, err
	}
	
	// Update the scan record
	err = db.UpdateScan(scanID, status, devicesFound, portsFound, duration, "")
	if err != nil {
		return 0, err
	}
	
	return scanID, nil
}

// OptimizeDatabase performs database maintenance operations
func (db *DB) OptimizeDatabase() error {
	db.Lock()
	defer db.Unlock()

	db.logger.Info().Msg("Optimizing database")

	// Run VACUUM to rebuild the database and reclaim space
	_, err := db.Exec("VACUUM")
	if err != nil {
		return fmt.Errorf("failed to vacuum database: %w", err)
	}

	// Rebuild indexes
	_, err = db.Exec("REINDEX")
	if err != nil {
		return fmt.Errorf("failed to reindex database: %w", err)
	}

	// Run ANALYZE to update statistics for query planning
	_, err = db.Exec("ANALYZE")
	if err != nil {
		return fmt.Errorf("failed to analyze database: %w", err)
	}

	// Refresh PRAGMA settings as they may reset after VACUUM
	if err := db.optimizeDB(); err != nil {
		db.logger.Warn().Err(err).Msg("Failed to reset optimization parameters after vacuum")
	}

	return nil
}

// BackupDatabase creates a backup of the database
func (db *DB) BackupDatabase() (string, error) {
	db.Lock()
	defer db.Unlock()

	// Create backup directory
	backupDir := filepath.Join(filepath.Dir(db.Path), "backups")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Create backup filename with timestamp
	timestamp := time.Now().Format("20060102_150405")
	baseFilename := filepath.Base(db.Path)
	extIdx := strings.LastIndex(baseFilename, ".")
	var backupFilename string
	if extIdx > 0 {
		backupFilename = fmt.Sprintf("%s_%s%s", baseFilename[:extIdx], timestamp, baseFilename[extIdx:])
	} else {
		backupFilename = fmt.Sprintf("%s_%s", baseFilename, timestamp)
	}
	backupPath := filepath.Join(backupDir, backupFilename)

	// Checkpoint the WAL first to ensure all changes are in the main DB file
	_, err := db.Exec("PRAGMA wal_checkpoint(FULL)")
	if err != nil {
		db.logger.Warn().Err(err).Msg("Failed to checkpoint WAL before backup")
	}

	// Instead of copying the file (which could be locked), use SQLite's VACUUM INTO
	// This creates a consistent backup of the database
	_, err = db.Exec(fmt.Sprintf("VACUUM INTO '%s'", backupPath))
	if err != nil {
		// Fall back to file copy if VACUUM INTO fails (it requires SQLite 3.27.0+)
		if fileErr := copyFile(db.Path, backupPath); fileErr != nil {
			return "", fmt.Errorf("failed to backup database (both VACUUM INTO and file copy failed): %w", fileErr)
		}
		db.logger.Warn().Err(err).Msg("VACUUM INTO failed, used file copy backup instead")
	}

	db.logger.Info().Str("path", backupPath).Msg("Database backup created")

	return backupPath, nil
}

// Helper function to copy a file
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dstFile.Close()

	if _, err := dstFile.ReadFrom(srcFile); err != nil {
		return fmt.Errorf("failed to copy file contents: %w", err)
	}

	return nil
}

// SearchDevices searches devices by IP, hostname, or OS
func (db *DB) SearchDevices(query string) ([]*models.Device, error) {
	// Add wildcards for LIKE query
	likeQuery := "%" + query + "%"

	rows, err := db.Query(
		`SELECT d.id, d.ip_address, d.mac_address, d.hostname, d.os_fingerprint, d.first_seen, d.last_seen,
		(SELECT COUNT(*) FROM ports WHERE device_id = d.id) as port_count
		FROM devices d
		WHERE d.ip_address LIKE ? OR d.hostname LIKE ? OR d.os_fingerprint LIKE ? OR d.mac_address LIKE ?
		ORDER BY d.last_seen DESC`,
		likeQuery, likeQuery, likeQuery, likeQuery,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to search devices: %w", err)
	}
	defer rows.Close()

	var devices []*models.Device
	for rows.Next() {
		var device models.Device
		err := rows.Scan(
			&device.ID,
			&device.IPAddress,
			&device.MACAddress,
			&device.Hostname,
			&device.OSFingerprint,
			&device.FirstSeen,
			&device.LastSeen,
			&device.PortCount,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan device row: %w", err)
		}
		devices = append(devices, &device)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating device rows: %w", err)
	}

	return devices, nil
}

// CleanOldData removes data older than the retention period
func (db *DB) CleanOldData(retentionDays int) (int, error) {
	db.Lock()
	defer db.Unlock()

	// Calculate cutoff date
	cutoff := time.Now().AddDate(0, 0, -retentionDays)

	// Start transaction
	tx, err := db.Begin()
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Ensure transaction is rolled back in case of error
	defer func() {
		if tx != nil {
			tx.Rollback()
		}
	}()

	// Order is important for foreign key constraints
	// First delete changes (which reference devices and scans)
	res, err := tx.Exec("DELETE FROM changes WHERE timestamp < ?", cutoff)
	if err != nil {
		return 0, fmt.Errorf("failed to delete old changes: %w", err)
	}
	changeCount, _ := res.RowsAffected()

	// Delete old scans
	res, err = tx.Exec("DELETE FROM scans WHERE timestamp < ?", cutoff)
	if err != nil {
		return 0, fmt.Errorf("failed to delete old scans: %w", err)
	}
	scanCount, _ := res.RowsAffected()

	// Delete old devices (their ports will cascade due to foreign key)
	res, err = tx.Exec("DELETE FROM devices WHERE last_seen < ?", cutoff)
	if err != nil {
		return 0, fmt.Errorf("failed to delete old devices: %w", err)
	}
	deviceCount, _ := res.RowsAffected()

	// Delete old logs
	res, err = tx.Exec("DELETE FROM logs WHERE timestamp < ?", cutoff)
	if err != nil {
		return 0, fmt.Errorf("failed to delete old logs: %w", err)
	}
	logCount, _ := res.RowsAffected()

	// Commit the transaction
	if err = tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}
	
	// Set tx to nil to prevent rollback in deferred function
	tx = nil

	totalDeleted := int(scanCount + deviceCount + changeCount + logCount)

	db.logger.Info().
		Int("scans", int(scanCount)).
		Int("devices", int(deviceCount)).
		Int("changes", int(changeCount)).
		Int("logs", int(logCount)).
		Int("total", totalDeleted).
		Msg("Cleaned old data")

	return totalDeleted, nil
}

// AddLogEntry adds a log entry to the database
func (db *DB) AddLogEntry(level, message, component string) error {
	_, err := db.Exec(
		"INSERT INTO logs (level, message, component, timestamp) VALUES (?, ?, ?, ?)",
		level, message, component, time.Now(),
	)
	return err
}

// GetLogEntries retrieves log entries with filtering options
func (db *DB) GetLogEntries(limit int, level, component string) ([]*models.Log, error) {
	// Build the query based on filters
	query := "SELECT id, level, message, component, timestamp FROM logs"
	var args []interface{}
	var conditions []string

	if level != "" {
		conditions = append(conditions, "level = ?")
		args = append(args, level)
	}

	if component != "" {
		conditions = append(conditions, "component = ?")
		args = append(args, component)
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += " ORDER BY timestamp DESC LIMIT ?"
	args = append(args, limit)

	// Execute the query
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query logs: %w", err)
	}
	defer rows.Close()

	var logs []*models.Log
	for rows.Next() {
		var log models.Log
		err := rows.Scan(
			&log.ID,
			&log.Level,
			&log.Message,
			&log.Component,
			&log.Timestamp,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan log row: %w", err)
		}
		logs = append(logs, &log)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating log rows: %w", err)
	}

	return logs, nil
}

// GetDatabaseStats returns statistics about the database
func (db *DB) GetDatabaseStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})
	
	// We'll use direct queries instead of a transaction to avoid timestamp format issues
	// Get device count
	var deviceCount int
	err := db.QueryRow("SELECT COUNT(*) FROM devices").Scan(&deviceCount)
	if err != nil {
		return nil, fmt.Errorf("failed to get device count: %w", err)
	}
	stats["deviceCount"] = deviceCount

	// Get port count
	var portCount int
	err = db.QueryRow("SELECT COUNT(*) FROM ports").Scan(&portCount)
	if err != nil {
		return nil, fmt.Errorf("failed to get port count: %w", err)
	}
	stats["portCount"] = portCount

	// Get scan count
	var scanCount int
	err = db.QueryRow("SELECT COUNT(*) FROM scans").Scan(&scanCount)
	if err != nil {
		return nil, fmt.Errorf("failed to get scan count: %w", err)
	}
	stats["scanCount"] = scanCount

	// Get last scan time - as a string first, then convert
	var lastScanTimeStr sql.NullString
	err = db.QueryRow("SELECT MAX(timestamp) FROM scans").Scan(&lastScanTimeStr)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to get last scan time: %w", err)
	}
	
	if lastScanTimeStr.Valid && lastScanTimeStr.String != "" {
		// Try several timestamp formats since SQLite doesn't enforce a specific one
		var lastScan time.Time
		var parseErr error
		
		formats := []string{
			time.RFC3339,                        // Standard format
			"2006-01-02 15:04:05.999999Z07:00",  // Common SQLite format
			"2006-01-02 15:04:05.999999-07:00",  // Another variation
			"2006-01-02 15:04:05-07:00",         // Without microseconds
			"2006-01-02 15:04:05",               // Without timezone
			"2006-01-02T15:04:05.999999Z07:00",  // With T separator
		}
		
		for _, format := range formats {
			lastScan, parseErr = time.Parse(format, lastScanTimeStr.String)
			if parseErr == nil {
				break
			}
		}
		
		if parseErr != nil {
			db.logger.Warn().Err(parseErr).Str("timestamp", lastScanTimeStr.String).Msg("Failed to parse scan timestamp")
			stats["lastScanTime"] = time.Time{} // Zero time
		} else {
			stats["lastScanTime"] = lastScan
		}
	} else {
		stats["lastScanTime"] = time.Time{} // Zero time
	}

	// Get database file size
	fileInfo, err := os.Stat(db.Path)
	if err != nil {
		db.logger.Warn().Err(err).Msg("Failed to get database file size")
		stats["sizeBytes"] = int64(0)
	} else {
		stats["sizeBytes"] = fileInfo.Size()
	}

	// Get OS distribution
	osDistribution := make(map[string]int)
	rows, err := db.Query("SELECT COALESCE(os_fingerprint, 'Unknown') as os, COUNT(*) FROM devices GROUP BY os_fingerprint")
	if err != nil {
		db.logger.Warn().Err(err).Msg("Failed to get OS distribution")
	} else {
		defer rows.Close()
		for rows.Next() {
			var os string
			var count int
			if err := rows.Scan(&os, &count); err != nil {
				db.logger.Warn().Err(err).Msg("Failed to scan OS distribution row")
				continue
			}
			osDistribution[os] = count
		}
		
		if err = rows.Err(); err != nil {
			db.logger.Warn().Err(err).Msg("Error iterating OS distribution rows")
		}
	}
	stats["osDistribution"] = osDistribution

	// Get service distribution
	serviceDistribution := make(map[string]int)
	rows, err = db.Query("SELECT COALESCE(service_name, 'Unknown') as service, COUNT(*) FROM ports GROUP BY service_name")
	if err != nil {
		db.logger.Warn().Err(err).Msg("Failed to get service distribution")
	} else {
		defer rows.Close()
		for rows.Next() {
			var service string
			var count int
			if err := rows.Scan(&service, &count); err != nil {
				db.logger.Warn().Err(err).Msg("Failed to scan service distribution row")
				continue
			}
			serviceDistribution[service] = count
		}
		
		if err = rows.Err(); err != nil {
			db.logger.Warn().Err(err).Msg("Error iterating service distribution rows")
		}
	}
	stats["serviceDistribution"] = serviceDistribution
	
	// Get changes count by type
	changeTypeDistribution := make(map[string]int)
	rows, err = db.Query("SELECT change_type, COUNT(*) FROM changes GROUP BY change_type")
	if err != nil {
		db.logger.Warn().Err(err).Msg("Failed to get change type distribution")
	} else {
		defer rows.Close()
		for rows.Next() {
			var changeType string
			var count int
			if err := rows.Scan(&changeType, &count); err != nil {
				db.logger.Warn().Err(err).Msg("Failed to scan change type row")
				continue
			}
			changeTypeDistribution[changeType] = count
		}
		
		if err = rows.Err(); err != nil {
			db.logger.Warn().Err(err).Msg("Error iterating change types")
		}
	}
	stats["changeTypeDistribution"] = changeTypeDistribution

	return stats, nil
}