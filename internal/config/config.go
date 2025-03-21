// Package config manages the Panopticon Scanner application configuration.
// It handles loading, validating, and providing access to configuration settings
// from YAML files. It includes defaults for all settings and implements thread-safe
// access to configuration values.
package config

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	Server struct {
		Port              int    `yaml:"port"`
		Host              string `yaml:"host"`
		AllowedOrigins    []string `yaml:"allowedOrigins"`
		ReadTimeout       int    `yaml:"readTimeout"`
		WriteTimeout      int    `yaml:"writeTimeout"`
		ShutdownTimeout   int    `yaml:"shutdownTimeout"`
	} `yaml:"server"`

	Scanner struct {
		Frequency            string   `yaml:"frequency"`
		RateLimit            int      `yaml:"rateLimit"`
		ScanAllPorts         bool     `yaml:"scanAllPorts"`
		DisablePing          bool     `yaml:"disablePing"`
		TargetNetwork        string   `yaml:"targetNetwork"`
		OutputDir            string   `yaml:"outputDir"`
		OutputRetentionDays  int      `yaml:"outputRetentionDays"`
		CompressOutput       bool     `yaml:"compressOutput"`
		EnableScheduler      bool     `yaml:"enableScheduler"`
		DefaultTemplate      string   `yaml:"defaultTemplate"`
		Templates            []string `yaml:"templates"`
		ExcludeHosts         []string `yaml:"excludeHosts"`
		EnableOSDetection    bool     `yaml:"enableOSDetection"`
		EnableVersionDetection bool   `yaml:"enableVersionDetection"`
	} `yaml:"scanner"`

	Database struct {
		Path                 string `yaml:"path"`
		BackupDir            string `yaml:"backupDir"`
		BackupFrequency      string `yaml:"backupFrequency"`
		OptimizeFrequency    string `yaml:"optimizeFrequency"`
		DataRetentionDays    int    `yaml:"dataRetentionDays"`
		MaxConnections       int    `yaml:"maxConnections"`
		EnableForeignKeys    bool   `yaml:"enableForeignKeys"`
		JournalMode          string `yaml:"journalMode"`
		SynchronousMode      string `yaml:"synchronousMode"`
	} `yaml:"database"`

	Auth struct {
		Enabled              bool   `yaml:"enabled"`
		Username             string `yaml:"username"`
		PasswordHash         string `yaml:"passwordHash"`
		SessionTimeout       int    `yaml:"sessionTimeout"`
		UseHTTPS             bool   `yaml:"useHttps"`
		JWTSecret            string `yaml:"jwtSecret"`
	} `yaml:"auth"`

	Logging struct {
		Level                string `yaml:"level"`
		Format               string `yaml:"format"`
		OutputPath           string `yaml:"outputPath"`
		MaxSize              int    `yaml:"maxSize"`
		MaxBackups           int    `yaml:"maxBackups"`
		MaxAge               int    `yaml:"maxAge"`
		Compress             bool   `yaml:"compress"`
	} `yaml:"logging"`

	Maintenance struct {
		Schedule             string `yaml:"schedule"`
		DatabaseBackup       bool   `yaml:"databaseBackup"`
		DatabaseOptimize     bool   `yaml:"databaseOptimize"`
		CleanupOldData       bool   `yaml:"cleanupOldData"`
		CleanupOutputFiles   bool   `yaml:"cleanupOutputFiles"`
	} `yaml:"maintenance"`

	Reporting struct {
		OutputDir            string `yaml:"outputDir"`
		EnableScheduledReports bool  `yaml:"enableScheduledReports"`
		DefaultFormat        string `yaml:"defaultFormat"`
	} `yaml:"reporting"`

	Advanced struct {
		PerformanceProfiling bool   `yaml:"performanceProfiling"`
		ProfilingEndpoint    string `yaml:"profilingEndpoint"`
		MetricsEnabled       bool   `yaml:"metricsEnabled"`
		MetricsEndpoint      string `yaml:"metricsEndpoint"`
		DiagnosticsEnabled   bool   `yaml:"diagnosticsEnabled"`
		DiagnosticsInterval  string `yaml:"diagnosticsInterval"`
	} `yaml:"advanced"`

	path string
	mu   sync.RWMutex
}

var (
	instance *Config
	once     sync.Once
)

// GetConfig returns the singleton configuration instance
func GetConfig() *Config {
	once.Do(func() {
		instance = &Config{}
		setDefaults(instance)
	})
	return instance
}

// LoadConfig loads configuration from a YAML file
func (c *Config) LoadConfig(path string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Save path for potential reloading
	c.path = path

	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("configuration file does not exist: %s", path)
	}

	// Read file
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read configuration file: %w", err)
	}

	// Unmarshal YAML
	if err := yaml.Unmarshal(data, c); err != nil {
		return fmt.Errorf("failed to parse configuration file: %w", err)
	}

	// Create directories if they don't exist
	dirs := []string{
		c.Scanner.OutputDir,
		c.Database.BackupDir,
		c.Reporting.OutputDir,
		filepath.Dir(c.Database.Path),
		filepath.Dir(c.Logging.OutputPath),
	}

	for _, dir := range dirs {
		if dir != "" {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", dir, err)
			}
		}
	}

	// Validate configuration
	if err := c.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	log.Info().Str("path", path).Msg("Configuration loaded successfully")
	return nil
}

// Reload reloads the configuration from the file
func (c *Config) Reload() error {
	if c.path == "" {
		return errors.New("configuration was not loaded from a file")
	}
	return c.LoadConfig(c.path)
}

// SaveConfig saves the current configuration to a file
func (c *Config) SaveConfig(path string) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal configuration: %w", err)
	}

	if err := ioutil.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write configuration file: %w", err)
	}

	return nil
}

// Validate checks that the configuration is valid
func (c *Config) Validate() error {
	// Server validation
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}

	// Scanner validation
	if c.Scanner.Frequency != "" {
		if _, err := time.ParseDuration(c.Scanner.Frequency); err != nil {
			return fmt.Errorf("invalid scan frequency: %s", c.Scanner.Frequency)
		}
	}

	if c.Scanner.RateLimit <= 0 {
		return fmt.Errorf("invalid rate limit: %d", c.Scanner.RateLimit)
	}

	// Database validation
	if c.Database.Path == "" {
		return errors.New("database path is required")
	}

	if c.Database.BackupFrequency != "" {
		if _, err := time.ParseDuration(c.Database.BackupFrequency); err != nil {
			return fmt.Errorf("invalid backup frequency: %s", c.Database.BackupFrequency)
		}
	}

	if c.Database.OptimizeFrequency != "" {
		if _, err := time.ParseDuration(c.Database.OptimizeFrequency); err != nil {
			return fmt.Errorf("invalid optimize frequency: %s", c.Database.OptimizeFrequency)
		}
	}

	return nil
}

// GetScanFrequency returns the scan frequency as a parsed duration
func (c *Config) GetScanFrequency() (time.Duration, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return time.ParseDuration(c.Scanner.Frequency)
}

// GetBackupFrequency returns the backup frequency as a parsed duration
func (c *Config) GetBackupFrequency() (time.Duration, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return time.ParseDuration(c.Database.BackupFrequency)
}

// GetOptimizeFrequency returns the optimize frequency as a parsed duration
func (c *Config) GetOptimizeFrequency() (time.Duration, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return time.ParseDuration(c.Database.OptimizeFrequency)
}

// setDefaults initializes the configuration with default values
func setDefaults(c *Config) {
	// Server defaults
	c.Server.Port = 8080
	c.Server.Host = "127.0.0.1"
	c.Server.AllowedOrigins = []string{"*"}
	c.Server.ReadTimeout = 30
	c.Server.WriteTimeout = 30
	c.Server.ShutdownTimeout = 10

	// Scanner defaults
	c.Scanner.Frequency = "1h"
	c.Scanner.RateLimit = 1000
	c.Scanner.ScanAllPorts = false
	c.Scanner.DisablePing = true
	c.Scanner.TargetNetwork = "192.168.1.0/24"
	c.Scanner.OutputDir = "./data/scans"
	c.Scanner.OutputRetentionDays = 30
	c.Scanner.CompressOutput = true
	c.Scanner.EnableScheduler = true
	c.Scanner.DefaultTemplate = "default"
	c.Scanner.EnableOSDetection = true
	c.Scanner.EnableVersionDetection = true

	// Database defaults
	c.Database.Path = "./data/panopticon.db"
	c.Database.BackupDir = "./data/backups"
	c.Database.BackupFrequency = "168h" // 1 week
	c.Database.OptimizeFrequency = "24h" // 1 day
	c.Database.DataRetentionDays = 730 // 2 years
	c.Database.MaxConnections = 10
	c.Database.EnableForeignKeys = true
	c.Database.JournalMode = "WAL"
	c.Database.SynchronousMode = "NORMAL"

	// Auth defaults
	c.Auth.Enabled = false
	c.Auth.SessionTimeout = 3600 // 1 hour
	c.Auth.UseHTTPS = false

	// Logging defaults
	c.Logging.Level = "info"
	c.Logging.Format = "json"
	c.Logging.OutputPath = "./data/logs/panopticon.log"
	c.Logging.MaxSize = 10 // 10 MB
	c.Logging.MaxBackups = 5
	c.Logging.MaxAge = 30 // 30 days
	c.Logging.Compress = true

	// Maintenance defaults
	c.Maintenance.Schedule = "0 2 * * *" // 2 AM daily
	c.Maintenance.DatabaseBackup = true
	c.Maintenance.DatabaseOptimize = true
	c.Maintenance.CleanupOldData = true
	c.Maintenance.CleanupOutputFiles = true

	// Reporting defaults
	c.Reporting.OutputDir = "./data/reports"
	c.Reporting.EnableScheduledReports = false
	c.Reporting.DefaultFormat = "html"

	// Advanced defaults
	c.Advanced.PerformanceProfiling = false
	c.Advanced.MetricsEnabled = false
	c.Advanced.DiagnosticsEnabled = true
	c.Advanced.DiagnosticsInterval = "1h"
}
