// Package scanner implements network scanning functionality for the Panopticon Scanner.
// It provides an interface for running both scheduled and manual network scans using nmap,
// processes scan results, and stores discovered devices and ports in the database.
package scanner

import (
	"context"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"panopticon-scanner/internal/config"
	"panopticon-scanner/internal/database"
	"panopticon-scanner/internal/models"
)

// ScanService represents the network scanning service
type ScanService struct {
	config            *config.Config
	db                *database.DB
	logger            zerolog.Logger
	scanLock          sync.Mutex
	isScanning        bool
	scanStats         *ScanStats
	scanSchedule      *time.Ticker
	stopChan          chan struct{}
	mockModeForTesting bool
}

// ScanStats tracks statistics for the current/last scan
type ScanStats struct {
	ScanID       int64
	StartTime    time.Time
	EndTime      time.Time
	Status       string
	DevicesFound int
	PortsFound   int
	Error        error
}

// ScanTemplate defines a scan configuration template
type ScanTemplate struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	NmapArgs    []string `yaml:"nmapArgs"`
	RateLimit   int      `yaml:"rateLimit"`
}

// New creates a new scan service
func New(cfg *config.Config, db *database.DB) *ScanService {
	return &ScanService{
		config:   cfg,
		db:       db,
		logger:   log.With().Str("component", "scanner").Logger(),
		scanLock: sync.Mutex{},
		scanStats: &ScanStats{
			Status: "idle",
		},
		stopChan: make(chan struct{}),
	}
}

// Start initializes and starts the scan service
func (s *ScanService) Start() error {
	s.logger.Info().Msg("Starting scan service")

	// Create scan output directory if it doesn't exist
	if err := os.MkdirAll(s.config.Scanner.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create scan output directory: %w", err)
	}

	// Start the scan scheduler
	if s.config.Scanner.EnableScheduler {
		s.StartScheduler()
	}

	return nil
}

// Stop gracefully stops the scan service
func (s *ScanService) Stop() error {
	s.logger.Info().Msg("Stopping scan service")

	// Stop the scheduler if running
	if s.scanSchedule != nil {
		s.scanSchedule.Stop()
		close(s.stopChan)
	}

	// If a scan is in progress, let it finish
	s.scanLock.Lock()
	defer s.scanLock.Unlock()

	return nil
}

// StartScheduler initiates the scan scheduler based on configuration
func (s *ScanService) StartScheduler() {
	s.scanLock.Lock()
	defer s.scanLock.Unlock()

	// Parse scan frequency from config
	frequency, err := time.ParseDuration(s.config.Scanner.Frequency)
	if err != nil {
		s.logger.Error().Err(err).Msg("Invalid scan frequency in config, using default 1h")
		frequency = 1 * time.Hour
	}

	s.logger.Info().Str("frequency", frequency.String()).Msg("Starting scan scheduler")

	// Stop existing scheduler if running
	if s.scanSchedule != nil {
		s.scanSchedule.Stop()
	}

	// Create new ticker for the scan schedule
	s.scanSchedule = time.NewTicker(frequency)

	// Run the scanner on a schedule
	go func() {
		// Run initial scan immediately
		s.RunScan(context.Background(), s.config.Scanner.DefaultTemplate)

		for {
			select {
			case <-s.scanSchedule.C:
				s.logger.Info().Msg("Running scheduled scan")
				s.RunScan(context.Background(), s.config.Scanner.DefaultTemplate)
			case <-s.stopChan:
				s.logger.Info().Msg("Scan scheduler stopped")
				return
			}
		}
	}()
}

// GetStatus returns the current scanner status
func (s *ScanService) GetStatus() ScanStats {
	s.scanLock.Lock()
	defer s.scanLock.Unlock()

	return *s.scanStats
}

// RunScan performs a network scan using the specified template
func (s *ScanService) RunScan(ctx context.Context, templateName string) (int64, error) {
	// Ensure only one scan runs at a time
	s.scanLock.Lock()
	if s.isScanning {
		s.scanLock.Unlock()
		return 0, fmt.Errorf("a scan is already in progress")
	}

	// Update scan status
	s.isScanning = true
	s.scanStats = &ScanStats{
		StartTime: time.Now(),
		Status:    "running",
	}
	s.scanLock.Unlock()

	// Ensure we update status when done
	defer func() {
		s.scanLock.Lock()
		s.isScanning = false
		s.scanStats.EndTime = time.Now()
		s.scanLock.Unlock()
	}()

	// Check if we're in mock mode for testing
	if s.mockModeForTesting {
		s.logger.Info().Str("template", templateName).Msg("Running mock scan for testing")
		
		// Create a mock scan record
		dbScanID, err := s.db.CreateScan(templateName)
		
		if err != nil {
			s.updateScanError(fmt.Errorf("failed to record mock scan in database: %w", err))
			return 0, err
		}
		
		s.scanLock.Lock()
		s.scanStats.ScanID = dbScanID
		s.scanLock.Unlock()
		
		// Simulate scan completion
		time.Sleep(100 * time.Millisecond)
		
		// Mock successful scan
		mockDeviceCount := 5
		mockPortCount := 15
		
		// Update scan status in database
		duration := time.Since(s.scanStats.StartTime)
		err = s.updateScanInDB(dbScanID, "completed", mockDeviceCount, mockPortCount, duration)
		if err != nil {
			s.logger.Error().Err(err).Msg("Failed to update mock scan record in database")
		}
		
		// Update scan stats
		s.scanLock.Lock()
		s.scanStats.Status = "completed"
		s.scanStats.DevicesFound = mockDeviceCount
		s.scanStats.PortsFound = mockPortCount
		s.scanLock.Unlock()
		
		s.logger.Info().
			Int64("scanID", dbScanID).
			Int("devices", mockDeviceCount).
			Int("ports", mockPortCount).
			Dur("duration", duration).
			Msg("Mock scan completed successfully")
		
		return dbScanID, nil
	}

	// Log scan start
	s.logger.Info().Str("template", templateName).Msg("Starting network scan")

	// Load scan template
	template, err := s.getScanTemplate(templateName)
	if err != nil {
		s.updateScanError(err)
		return 0, err
	}

	// Create unique output file for this scan
	scanID := uuid.New().String()
	outputPath := filepath.Join(s.config.Scanner.OutputDir, fmt.Sprintf("scan_%s.xml", scanID))

	// Prepare scan command
	nmapCmd, err := s.prepareScanCommand(template, outputPath)
	if err != nil {
		s.updateScanError(err)
		return 0, err
	}

	// Record scan in database before starting
	dbScanID, err := s.db.CreateScan(templateName)

	if err != nil {
		s.updateScanError(fmt.Errorf("failed to record scan in database: %w", err))
		return 0, err
	}

	s.scanLock.Lock()
	s.scanStats.ScanID = dbScanID
	s.scanLock.Unlock()

	// Execute the nmap scan
	s.logger.Debug().Str("command", strings.Join(nmapCmd.Args, " ")).Msg("Executing nmap command")

	// Capture output for logging
	stdout, err := nmapCmd.StdoutPipe()
	if err != nil {
		s.updateScanError(err)
		return dbScanID, err
	}

	stderr, err := nmapCmd.StderrPipe()
	if err != nil {
		s.updateScanError(err)
		return dbScanID, err
	}

	// Start the command
	if err := nmapCmd.Start(); err != nil {
		s.updateScanError(fmt.Errorf("failed to start nmap: %w", err))
		s.updateScanInDB(dbScanID, "error", 0, 0, time.Since(s.scanStats.StartTime))
		return dbScanID, err
	}

	// Read command output in background
	go func() {
		output, _ := ioutil.ReadAll(stdout)
		if len(output) > 0 {
			s.logger.Debug().Str("stdout", string(output)).Msg("nmap output")
		}
	}()

	go func() {
		errOutput, _ := ioutil.ReadAll(stderr)
		if len(errOutput) > 0 {
			s.logger.Warn().Str("stderr", string(errOutput)).Msg("nmap error output")
		}
	}()

	// Wait for command to complete
	if err := nmapCmd.Wait(); err != nil {
		s.updateScanError(fmt.Errorf("nmap command failed: %w", err))
		s.updateScanInDB(dbScanID, "error", 0, 0, time.Since(s.scanStats.StartTime))
		return dbScanID, err
	}

	// Process scan results
	deviceCount, portCount, err := s.processScanResults(outputPath)
	if err != nil {
		s.updateScanError(fmt.Errorf("failed to process scan results: %w", err))
		s.updateScanInDB(dbScanID, "error", deviceCount, portCount, time.Since(s.scanStats.StartTime))
		return dbScanID, err
	}

	// Update scan status in database
	duration := time.Since(s.scanStats.StartTime)
	err = s.updateScanInDB(dbScanID, "completed", deviceCount, portCount, duration)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to update scan record in database")
	}

	// Update scan stats
	s.scanLock.Lock()
	s.scanStats.Status = "completed"
	s.scanStats.DevicesFound = deviceCount
	s.scanStats.PortsFound = portCount
	s.scanLock.Unlock()

	s.logger.Info().
		Int64("scanID", dbScanID).
		Int("devices", deviceCount).
		Int("ports", portCount).
		Dur("duration", duration).
		Msg("Scan completed successfully")

	return dbScanID, nil
}

// RunManualScan executes a scan with custom parameters
func (s *ScanService) RunManualScan(ctx context.Context, params models.ScanParameters) (int64, error) {
	// Ensure only one scan runs at a time
	s.scanLock.Lock()
	if s.isScanning {
		s.scanLock.Unlock()
		return 0, fmt.Errorf("a scan is already in progress")
	}

	// Update scan status
	s.isScanning = true
	s.scanStats = &ScanStats{
		StartTime: time.Now(),
		Status:    "running",
	}
	s.scanLock.Unlock()

	// Ensure we update status when done
	defer func() {
		s.scanLock.Lock()
		s.isScanning = false
		s.scanStats.EndTime = time.Now()
		s.scanLock.Unlock()
	}()

	// Check if we're in mock mode for testing
	if s.mockModeForTesting {
		s.logger.Info().
			Str("template", params.Template).
			Str("targetNetwork", params.TargetNetwork).
			Int("rateLimit", params.RateLimit).
			Bool("scanAllPorts", params.ScanAllPorts).
			Bool("disablePing", params.DisablePing).
			Msg("Running mock manual scan for testing")
		
		// Create a mock scan record
		dbScanID, err := s.db.CreateScan(params.Template)
		
		if err != nil {
			s.updateScanError(fmt.Errorf("failed to record mock manual scan in database: %w", err))
			return 0, err
		}
		
		s.scanLock.Lock()
		s.scanStats.ScanID = dbScanID
		s.scanLock.Unlock()
		
		// Simulate scan completion
		time.Sleep(100 * time.Millisecond)
		
		// Mock successful scan
		mockDeviceCount := 3
		mockPortCount := 10
		
		// Update scan status in database
		duration := time.Since(s.scanStats.StartTime)
		err = s.updateScanInDB(dbScanID, "completed", mockDeviceCount, mockPortCount, duration)
		if err != nil {
			s.logger.Error().Err(err).Msg("Failed to update mock manual scan record in database")
		}
		
		// Update scan stats
		s.scanLock.Lock()
		s.scanStats.Status = "completed"
		s.scanStats.DevicesFound = mockDeviceCount
		s.scanStats.PortsFound = mockPortCount
		s.scanLock.Unlock()
		
		s.logger.Info().
			Int64("scanID", dbScanID).
			Int("devices", mockDeviceCount).
			Int("ports", mockPortCount).
			Dur("duration", duration).
			Msg("Mock manual scan completed successfully")
		
		return dbScanID, nil
	}

	// Log scan start
	s.logger.Info().
		Str("template", params.Template).
		Str("targetNetwork", params.TargetNetwork).
		Int("rateLimit", params.RateLimit).
		Bool("scanAllPorts", params.ScanAllPorts).
		Bool("disablePing", params.DisablePing).
		Msg("Starting manual network scan")

	// Load base scan template
	template, err := s.getScanTemplate(params.Template)
	if err != nil {
		s.updateScanError(err)
		return 0, err
	}

	// Override template settings with custom parameters if provided
	if params.RateLimit > 0 {
		template.RateLimit = params.RateLimit
	}

	// Create unique output file for this scan
	scanID := uuid.New().String()
	outputPath := filepath.Join(s.config.Scanner.OutputDir, fmt.Sprintf("scan_%s.xml", scanID))

	// Record scan in database before starting
	dbScanID, err := s.db.CreateScan(params.Template)

	if err != nil {
		s.updateScanError(fmt.Errorf("failed to record scan in database: %w", err))
		return 0, err
	}

	s.scanLock.Lock()
	s.scanStats.ScanID = dbScanID
	s.scanLock.Unlock()

	// Prepare scan command with custom parameters
	nmapCmd, err := s.prepareManualScanCommand(template, outputPath, params)
	if err != nil {
		s.updateScanError(err)
		s.updateScanInDB(dbScanID, "error", 0, 0, time.Since(s.scanStats.StartTime))
		return dbScanID, err
	}

	// Execute the nmap scan
	s.logger.Debug().Str("command", strings.Join(nmapCmd.Args, " ")).Msg("Executing nmap command")

	// Capture output for logging
	stdout, err := nmapCmd.StdoutPipe()
	if err != nil {
		s.updateScanError(err)
		return dbScanID, err
	}

	stderr, err := nmapCmd.StderrPipe()
	if err != nil {
		s.updateScanError(err)
		return dbScanID, err
	}

	// Start the command
	if err := nmapCmd.Start(); err != nil {
		s.updateScanError(fmt.Errorf("failed to start nmap: %w", err))
		s.updateScanInDB(dbScanID, "error", 0, 0, time.Since(s.scanStats.StartTime))
		return dbScanID, err
	}

	// Read command output in background
	go func() {
		output, _ := ioutil.ReadAll(stdout)
		if len(output) > 0 {
			s.logger.Debug().Str("stdout", string(output)).Msg("nmap output")
		}
	}()

	go func() {
		errOutput, _ := ioutil.ReadAll(stderr)
		if len(errOutput) > 0 {
			s.logger.Warn().Str("stderr", string(errOutput)).Msg("nmap error output")
		}
	}()

	// Wait for command to complete
	if err := nmapCmd.Wait(); err != nil {
		s.updateScanError(fmt.Errorf("nmap command failed: %w", err))
		s.updateScanInDB(dbScanID, "error", 0, 0, time.Since(s.scanStats.StartTime))
		return dbScanID, err
	}

	// Process scan results
	deviceCount, portCount, err := s.processScanResults(outputPath)
	if err != nil {
		s.updateScanError(fmt.Errorf("failed to process scan results: %w", err))
		s.updateScanInDB(dbScanID, "error", deviceCount, portCount, time.Since(s.scanStats.StartTime))
		return dbScanID, err
	}

	// Update scan status in database
	duration := time.Since(s.scanStats.StartTime)
	err = s.updateScanInDB(dbScanID, "completed", deviceCount, portCount, duration)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to update scan record in database")
	}

	// Update scan stats
	s.scanLock.Lock()
	s.scanStats.Status = "completed"
	s.scanStats.DevicesFound = deviceCount
	s.scanStats.PortsFound = portCount
	s.scanLock.Unlock()

	s.logger.Info().
		Int64("scanID", dbScanID).
		Int("devices", deviceCount).
		Int("ports", portCount).
		Dur("duration", duration).
		Msg("Manual scan completed successfully")

	return dbScanID, nil
}

// getScanTemplate retrieves the specified scan template from configuration
func (s *ScanService) getScanTemplate(name string) (*ScanTemplate, error) {
	// For MVP, use hardcoded templates - later these would come from config or database
	templates := s.getTemplates()

	// Find the requested template
	template, exists := templates[name]
	if !exists {
		s.logger.Warn().Str("template", name).Msg("Template not found, using default")
		template = templates["default"]
	}

	return template, nil
}

// getTemplates returns all available scan templates
func (s *ScanService) getTemplates() map[string]*ScanTemplate {
	return map[string]*ScanTemplate{
		"default": {
			Name:        "default",
			Description: "Standard network scan",
			NmapArgs:    []string{"-sS", "-sV", "-O", "--osscan-limit"},
			RateLimit:   1000,
		},
		"quick": {
			Name:        "quick",
			Description: "Fast scan of common ports",
			NmapArgs:    []string{"-sS", "-F"},
			RateLimit:   2000,
		},
		"thorough": {
			Name:        "thorough",
			Description: "Detailed scan of all ports",
			NmapArgs:    []string{"-sS", "-sV", "-p-", "-O", "--osscan-guess"},
			RateLimit:   500,
		},
		"stealth": {
			Name:        "stealth",
			Description: "Low-impact scan for sensitive networks",
			NmapArgs:    []string{"-sS", "-T2", "--max-retries", "1"},
			RateLimit:   100,
		},
	}
}

// GetScanTemplates returns all available scan templates
func (s *ScanService) GetScanTemplates() ([]models.ScanTemplate, error) {
	templates := s.getTemplates()
	result := make([]models.ScanTemplate, 0, len(templates))

	for id, template := range templates {
		result = append(result, models.ScanTemplate{
			ID:          id,
			Name:        template.Name,
			Description: template.Description,
			NmapArgs:    template.NmapArgs,
			RateLimit:   template.RateLimit,
		})
	}

	return result, nil
}

// GetScan retrieves a scan by ID
func (s *ScanService) GetScan(scanID int64) (*models.Scan, error) {
	return s.db.GetScan(scanID)
}

// GetRecentScans retrieves recent scans
func (s *ScanService) GetRecentScans(limit int) ([]*models.Scan, error) {
	return s.db.GetRecentScans(limit)
}

// prepareScanCommand builds the nmap command with appropriate arguments
func (s *ScanService) prepareScanCommand(template *ScanTemplate, outputPath string) (*exec.Cmd, error) {
	// Start with basic arguments
	args := []string{
		"-oX", outputPath, // XML output for parsing
	}

	// Add template-specific arguments
	args = append(args, template.NmapArgs...)

	// Add rate limiting
	args = append(args, "--max-rate", strconv.Itoa(template.RateLimit))

	// Add target network from configuration
	if s.config.Scanner.TargetNetwork == "" {
		return nil, fmt.Errorf("no target network specified in configuration")
	}
	args = append(args, s.config.Scanner.TargetNetwork)

	// Create the command
	cmd := exec.Command("nmap", args...)

	return cmd, nil
}

// prepareManualScanCommand builds a customized nmap command based on scan parameters
func (s *ScanService) prepareManualScanCommand(template *ScanTemplate, outputPath string, params models.ScanParameters) (*exec.Cmd, error) {
	// Start with basic arguments
	args := []string{
		"-oX", outputPath, // XML output for parsing
	}

	// Handle custom scan parameters
	if params.ScanAllPorts {
		// Replace port specification with all ports (-p-)
		customArgs := make([]string, 0, len(template.NmapArgs))
		for _, arg := range template.NmapArgs {
			// Skip any existing port specifications
			if strings.HasPrefix(arg, "-p") {
				continue
			}
			customArgs = append(customArgs, arg)
		}
		args = append(args, customArgs...)
		args = append(args, "-p-") // Scan all ports
	} else {
		// Use template arguments
		args = append(args, template.NmapArgs...)
	}

	// Handle disable ping option
	if params.DisablePing {
		// Check if -Pn is already in the arguments
		hasPingDisabled := false
		for _, arg := range args {
			if arg == "-Pn" {
				hasPingDisabled = true
				break
			}
		}
		if !hasPingDisabled {
			args = append(args, "-Pn") // Skip host discovery
		}
	}

	// Add rate limiting
	rateLimit := template.RateLimit
	if params.RateLimit > 0 {
		rateLimit = params.RateLimit
	}
	args = append(args, "--max-rate", strconv.Itoa(rateLimit))

	// Use custom target network if provided, otherwise use from config
	targetNetwork := s.config.Scanner.TargetNetwork
	if params.TargetNetwork != "" {
		targetNetwork = params.TargetNetwork
	}

	if targetNetwork == "" {
		return nil, fmt.Errorf("no target network specified")
	}
	args = append(args, targetNetwork)

	// Create the command
	cmd := exec.Command("nmap", args...)

	return cmd, nil
}

// processScanResults parses the nmap XML output and stores results in database
func (s *ScanService) processScanResults(outputPath string) (deviceCount int, portCount int, err error) {
	s.logger.Debug().Str("file", outputPath).Msg("Processing scan results")

	// Read the XML output file
	xmlData, err := ioutil.ReadFile(outputPath)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to read scan output file: %w", err)
	}

	// Parse the XML
	var result NmapRun
	if err := xml.Unmarshal(xmlData, &result); err != nil {
		return 0, 0, fmt.Errorf("failed to parse nmap XML: %w", err)
	}

	// Process each host found
	for _, host := range result.Hosts {
		// Skip hosts that are not "up"
		if host.Status.State != "up" {
			continue
		}

		// Extract IP address
		var ipAddress, macAddress string
		for _, addr := range host.Addresses {
			if addr.AddrType == "ipv4" {
				ipAddress = addr.Addr
			} else if addr.AddrType == "mac" {
				macAddress = addr.Addr
			}
		}

		if ipAddress == "" {
			s.logger.Debug().Interface("host", host).Msg("Skipping host with no IP address")
			continue
		}

		// Extract hostname
		hostname := ""
		if len(host.Hostnames.Hostname) > 0 {
			hostname = host.Hostnames.Hostname[0].Name
		}

		// Extract OS detection
		osFingerprint := ""
		if len(host.Os.OsMatches) > 0 {
			osFingerprint = host.Os.OsMatches[0].Name
		}

		// Log device found
		s.logger.Debug().
			Str("ip", ipAddress).
			Str("hostname", hostname).
			Str("os", osFingerprint).
			Int("ports", len(host.Ports.Port)).
			Msg("Found device")

		// Save to database using transaction
		deviceID, err := s.db.SaveDevice(&models.Device{
			IPAddress:     ipAddress,
			MACAddress:    macAddress,
			Hostname:      hostname,
			OSFingerprint: osFingerprint,
			FirstSeen:     time.Now(),
			LastSeen:      time.Now(),
		})

		if err != nil {
			s.logger.Error().Err(err).Str("ip", ipAddress).Msg("Failed to save device")
			continue
		}

		deviceCount++

		// Process ports for this host
		for _, port := range host.Ports.Port {
			if port.State.State != "open" {
				continue
			}

			portNum, err := strconv.Atoi(port.PortID)
			if err != nil {
				s.logger.Warn().Err(err).Str("port", port.PortID).Msg("Invalid port number")
				continue
			}

			// Build service info
			serviceName := port.Service.Name
			serviceVersion := ""
			if port.Service.Product != "" {
				serviceVersion = port.Service.Product
				if port.Service.Version != "" {
					serviceVersion += " " + port.Service.Version
				}
			}

			// Save port to database
			err = s.db.SavePort(&models.Port{
				DeviceID:       deviceID,
				PortNumber:     portNum,
				Protocol:       port.Protocol,
				ServiceName:    serviceName,
				ServiceVersion: serviceVersion,
				FirstSeen:      time.Now(),
				LastSeen:       time.Now(),
			})

			if err != nil {
				s.logger.Error().Err(err).
					Int64("deviceID", deviceID).
					Int("port", portNum).
					Msg("Failed to save port")
				continue
			}

			portCount++
		}
	}

	// Compress the XML file to save space if enabled
	if s.config.Scanner.CompressOutput {
		go s.compressOutputFile(outputPath)
	}

	return deviceCount, portCount, nil
}

// compressOutputFile compresses the scan output file
func (s *ScanService) compressOutputFile(filePath string) {
	s.logger.Debug().Str("file", filePath).Msg("Compressing scan output file")

	// Create gzip command
	cmd := exec.Command("gzip", filePath)

	// Execute the command
	if err := cmd.Run(); err != nil {
		s.logger.Error().Err(err).Str("file", filePath).Msg("Failed to compress scan output file")
	}
}

// updateScanError updates the scan status with error information
func (s *ScanService) updateScanError(err error) {
	s.scanLock.Lock()
	defer s.scanLock.Unlock()

	s.scanStats.Status = "error"
	s.scanStats.Error = err
	s.logger.Error().Err(err).Msg("Scan error occurred")
}

// updateScanInDB updates the scan record in the database
func (s *ScanService) updateScanInDB(scanID int64, status string, deviceCount, portCount int, duration time.Duration) error {
	return s.db.UpdateScan(scanID, status, deviceCount, portCount, duration, "")
}

// Clean removes old scan data
func (s *ScanService) Clean() error {
	s.logger.Info().Msg("Cleaning old scan data")

	// Remove old output files based on retention policy
	return s.cleanOutputFiles()
}

// cleanOutputFiles removes scan output files older than retention period
func (s *ScanService) cleanOutputFiles() error {
	// Skip if no retention period set
	if s.config.Scanner.OutputRetentionDays <= 0 {
		return nil
	}

	// Calculate cutoff time
	cutoff := time.Now().Add(-time.Hour * 24 * time.Duration(s.config.Scanner.OutputRetentionDays))

	// Walk through output directory
	return filepath.Walk(s.config.Scanner.OutputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Check if file is old enough to delete
		if info.ModTime().Before(cutoff) {
			s.logger.Debug().Str("file", path).Msg("Removing old scan output file")
			if err := os.Remove(path); err != nil {
				s.logger.Error().Err(err).Str("file", path).Msg("Failed to remove old scan file")
			}
		}

		return nil
	})
}

// NmapRun represents the root XML element from nmap output
type NmapRun struct {
	XMLName xml.Name `xml:"nmaprun"`
	Hosts   []Host   `xml:"host"`
}

// Host represents a host found during scanning
type Host struct {
	Status    Status    `xml:"status"`
	Addresses []Address `xml:"address"`
	Hostnames Hostnames `xml:"hostnames"`
	Ports     Ports     `xml:"ports"`
	Os        Os        `xml:"os"`
}

// Status represents the status of a host
type Status struct {
	State string `xml:"state,attr"`
}

// Address represents a network address
type Address struct {
	Addr     string `xml:"addr,attr"`
	AddrType string `xml:"addrtype,attr"`
}

// Hostnames contains hostname information
type Hostnames struct {
	Hostname []Hostname `xml:"hostname"`
}

// Hostname represents a hostname
type Hostname struct {
	Name string `xml:"name,attr"`
}

// Ports contains port information
type Ports struct {
	Port []Port `xml:"port"`
}

// Port represents a port
type Port struct {
	Protocol string  `xml:"protocol,attr"`
	PortID   string  `xml:"portid,attr"`
	State    State   `xml:"state"`
	Service  Service `xml:"service"`
}

// State represents the state of a port
type State struct {
	State string `xml:"state,attr"`
}

// Service represents a service detected on a port
type Service struct {
	Name    string `xml:"name,attr"`
	Product string `xml:"product,attr"`
	Version string `xml:"version,attr"`
}

// Os contains operating system detection information
type Os struct {
	OsMatches []OsMatch `xml:"osmatch"`
}

// OsMatch represents an OS detection match
type OsMatch struct {
	Name      string  `xml:"name,attr"`
	Accuracy  string  `xml:"accuracy,attr"`
	OsClasses []OsClass `xml:"osclass"`
}

// OsClass represents an OS class
type OsClass struct {
	Type     string `xml:"type,attr"`
	Vendor   string `xml:"vendor,attr"`
	Family   string `xml:"osfamily,attr"`
	Gen      string `xml:"osgen,attr"`
	Accuracy string `xml:"accuracy,attr"`
}

// The following methods are used for testing only

// SetStatusForTesting sets the scan status for testing purposes
func (s *ScanService) SetStatusForTesting(status string) {
	s.scanLock.Lock()
	defer s.scanLock.Unlock()
	s.scanStats.Status = status
}

// SetDevicesFoundForTesting sets the devices found count for testing purposes
func (s *ScanService) SetDevicesFoundForTesting(count int) {
	s.scanLock.Lock()
	defer s.scanLock.Unlock()
	s.scanStats.DevicesFound = count
}

// SetPortsFoundForTesting sets the ports found count for testing purposes
func (s *ScanService) SetPortsFoundForTesting(count int) {
	s.scanLock.Lock()
	defer s.scanLock.Unlock()
	s.scanStats.PortsFound = count
}

// SetScanIDForTesting sets the scan ID for testing purposes
func (s *ScanService) SetScanIDForTesting(id int64) {
	s.scanLock.Lock()
	defer s.scanLock.Unlock()
	s.scanStats.ScanID = id
}

// SetStartTimeForTesting sets the scan start time for testing purposes
func (s *ScanService) SetStartTimeForTesting(t time.Time) {
	s.scanLock.Lock()
	defer s.scanLock.Unlock()
	s.scanStats.StartTime = t
}

// SetEndTimeForTesting sets the scan end time for testing purposes
func (s *ScanService) SetEndTimeForTesting(t time.Time) {
	s.scanLock.Lock()
	defer s.scanLock.Unlock()
	s.scanStats.EndTime = t
}

// SetMockModeForTesting enables or disables mock mode for testing
func (s *ScanService) SetMockModeForTesting(enabled bool) {
	s.scanLock.Lock()
	defer s.scanLock.Unlock()
	if enabled {
		s.mockModeForTesting = true
	} else {
		s.mockModeForTesting = false
	}
}
