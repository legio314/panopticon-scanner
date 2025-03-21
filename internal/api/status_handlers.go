// internal/api/status_handlers.go
package api

import (
	"encoding/json"
	"net/http"
	"runtime"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"

	"panopticon-scanner/internal/config"
	"panopticon-scanner/internal/database"
	"panopticon-scanner/internal/scanner"
)

// StatusHandler handles system status-related API endpoints
type StatusHandler struct {
	db          *database.DB
	scanService *scanner.ScanService
	cfg         *config.Config
	startTime   time.Time
}

// NewStatusHandler creates a new status handler
func NewStatusHandler(db *database.DB, scanService *scanner.ScanService, cfg *config.Config) *StatusHandler {
	return &StatusHandler{
		db:          db,
		scanService: scanService,
		cfg:         cfg,
		startTime:   time.Now(),
	}
}

// RegisterRoutes registers the status routes
func (h *StatusHandler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/api/status", h.getSystemStatus).Methods("GET")
	r.HandleFunc("/api/status/health", h.getHealthCheck).Methods("GET")
	r.HandleFunc("/api/status/database", h.getDatabaseStatus).Methods("GET")
}

// getSystemStatus returns the overall system status
func (h *StatusHandler) getSystemStatus(w http.ResponseWriter, r *http.Request) {
	logger := log.With().Str("handler", "getSystemStatus").Logger()

	// Get database stats
	dbStats, err := h.db.GetDatabaseStats()
	if err != nil {
		logger.Error().Err(err).Msg("Failed to retrieve database stats")
	}

	// Get current scan status
	scanStatus := h.scanService.GetStatus()

	// Build memory stats
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	// Calculate uptime
	uptime := time.Since(h.startTime)

	// Build response
	response := map[string]interface{}{
		"status":    "healthy", // Default status
		"version":   "1.0.0",   // TODO: Get this from a version package
		"uptime":    uptime.String(),
		"startTime": h.startTime,
		"system": map[string]interface{}{
			"goVersion":   runtime.Version(),
			"goArch":      runtime.GOARCH,
			"goOS":        runtime.GOOS,
			"numCPU":      runtime.NumCPU(),
			"numGoroutine": runtime.NumGoroutine(),
		},
		"memory": map[string]interface{}{
			"alloc":        memStats.Alloc / 1024 / 1024,         // MB
			"totalAlloc":   memStats.TotalAlloc / 1024 / 1024,     // MB
			"sys":          memStats.Sys / 1024 / 1024,            // MB
			"numGC":        memStats.NumGC,
			"heapObjects":  memStats.HeapObjects,
		},
		"config": map[string]interface{}{
			"serverPort":      h.cfg.Server.Port,
			"scanFrequency":   h.cfg.Scanner.Frequency,
			"targetNetwork":   h.cfg.Scanner.TargetNetwork,
			"authEnabled":     h.cfg.Auth.Enabled,
			"loggingLevel":    h.cfg.Logging.Level,
		},
		"scanner": map[string]interface{}{
			"status":         scanStatus.Status,
			"currentScanID":  scanStatus.ScanID,
			"lastScanTime":   dbStats["lastScanTime"],
			"devicesFound":   dbStats["deviceCount"],
			"portsFound":     dbStats["portCount"],
			"schedulerEnabled": h.cfg.Scanner.EnableScheduler,
		},
		"database": map[string]interface{}{
			"size":          dbStats["sizeBytes"],
			"scanCount":     dbStats["scanCount"],
			"deviceCount":   dbStats["deviceCount"],
			"portCount":     dbStats["portCount"],
			"path":          h.cfg.Database.Path,
			"lastBackup":    "N/A", // Would need to track this separately
		},
		"timestamp": time.Now(),
	}

	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.Error().Err(err).Msg("Failed to encode system status")
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// getHealthCheck returns a simple health check response
func (h *StatusHandler) getHealthCheck(w http.ResponseWriter, r *http.Request) {
	logger := log.With().Str("handler", "getHealthCheck").Logger()

	// Simple health check - check DB connection
	err := h.db.Ping()
	var status string
	if err != nil {
		status = "unhealthy"
		logger.Error().Err(err).Msg("Database ping failed")
	} else {
		status = "healthy"
	}

	// Build response
	response := map[string]interface{}{
		"status":    status,
		"timestamp": time.Now(),
		"uptime":    time.Since(h.startTime).String(),
	}

	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.Error().Err(err).Msg("Failed to encode health check response")
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// getDatabaseStatus returns detailed database status information
func (h *StatusHandler) getDatabaseStatus(w http.ResponseWriter, r *http.Request) {
	logger := log.With().Str("handler", "getDatabaseStatus").Logger()

	// Get database stats
	dbStats, err := h.db.GetDatabaseStats()
	if err != nil {
		logger.Error().Err(err).Msg("Failed to retrieve database stats")
		http.Error(w, "Failed to retrieve database status", http.StatusInternalServerError)
		return
	}

	// Calculate size in MB for better readability
	sizeBytes, _ := dbStats["sizeBytes"].(int64)
	sizeMB := float64(sizeBytes) / 1024 / 1024

	// Extract OS distribution for a separate section
	osDistribution := dbStats["osDistribution"]

	// Build response
	response := map[string]interface{}{
		"status":           "online",
		"path":             h.cfg.Database.Path,
		"sizeBytes":        sizeBytes,
		"sizeMB":           sizeMB,
		"scanCount":        dbStats["scanCount"],
		"deviceCount":      dbStats["deviceCount"],
		"portCount":        dbStats["portCount"],
		"lastScanTime":     dbStats["lastScanTime"],
		"osDistribution":   osDistribution,
		"retentionDays":    h.cfg.Database.DataRetentionDays,
		"backupFrequency":  h.cfg.Database.BackupFrequency,
		"journalMode":      "WAL", // From the PRAGMA settings
		"synchronousMode":  "NORMAL", // From the PRAGMA settings
		"timestamp":        time.Now(),
	}

	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.Error().Err(err).Msg("Failed to encode database status")
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}