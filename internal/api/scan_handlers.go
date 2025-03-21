// Package api provides HTTP handlers for the Panopticon Scanner REST API.
// It includes handlers for device management, scan operations, system status,
// and other functions exposed through the API.
package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"

	"panopticon-scanner/internal/models"
	"panopticon-scanner/internal/scanner"
)

// ScanHandler handles scan-related API endpoints
type ScanHandler struct {
	scanService *scanner.ScanService
}

// NewScanHandler creates a new scan handler
func NewScanHandler(scanService *scanner.ScanService) *ScanHandler {
	return &ScanHandler{
		scanService: scanService,
	}
}

// RegisterRoutes registers the scan routes
func (h *ScanHandler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/api/scans", h.getScans).Methods("GET")
	r.HandleFunc("/api/scans/{id}", h.getScan).Methods("GET")
	r.HandleFunc("/api/scans", h.startScan).Methods("POST")
	r.HandleFunc("/api/scans/status", h.GetScanStatus).Methods("GET")
	r.HandleFunc("/api/scans/templates", h.GetScanTemplates).Methods("GET")
}

// getScans returns a list of recent scans
func (h *ScanHandler) getScans(w http.ResponseWriter, r *http.Request) {
	logger := log.With().Str("handler", "getScans").Logger()

	// Parse query parameters
	limit := 10 // Default limit
	if limitParam := r.URL.Query().Get("limit"); limitParam != "" {
		parsedLimit, err := strconv.Atoi(limitParam)
		if err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	// Get scans from database
	scans, err := h.scanService.GetRecentScans(limit)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to retrieve scans")
		http.Error(w, "Failed to retrieve scans", http.StatusInternalServerError)
		return
	}

	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(scans); err != nil {
		logger.Error().Err(err).Msg("Failed to encode scans")
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// getScan returns a specific scan by ID
func (h *ScanHandler) getScan(w http.ResponseWriter, r *http.Request) {
	logger := log.With().Str("handler", "getScan").Logger()

	// Parse scan ID from URL
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		logger.Error().Err(err).Str("id", idStr).Msg("Invalid scan ID")
		http.Error(w, "Invalid scan ID", http.StatusBadRequest)
		return
	}

	// Get scan from database
	scan, err := h.scanService.GetScan(id)
	if err != nil {
		logger.Error().Err(err).Int64("id", id).Msg("Failed to retrieve scan")
		http.Error(w, "Scan not found", http.StatusNotFound)
		return
	}

	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(scan); err != nil {
		logger.Error().Err(err).Msg("Failed to encode scan")
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// startScan initiates a new network scan
func (h *ScanHandler) startScan(w http.ResponseWriter, r *http.Request) {
	logger := log.With().Str("handler", "startScan").Logger()

	// Check if a scan is already running
	status := h.scanService.GetStatus()
	if status.Status == "running" {
		logger.Warn().Msg("Scan already in progress")
		http.Error(w, "A scan is already in progress", http.StatusConflict)
		return
	}

	// Parse scan parameters from request body
	var params models.ScanParameters
	if r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
			logger.Error().Err(err).Msg("Failed to parse scan parameters")
			http.Error(w, "Invalid scan parameters", http.StatusBadRequest)
			return
		}
	}

	// Use default template if none specified
	if params.Template == "" {
		params.Template = "default"
	}

	// Validate parameters
	if params.RateLimit < 0 {
		logger.Warn().Int("rateLimit", params.RateLimit).Msg("Invalid rate limit provided")
		http.Error(w, "Invalid rate limit: must be a positive number", http.StatusBadRequest)
		return
	}

	// Log the scan request details
	logger.Info().
		Str("template", params.Template).
		Str("targetNetwork", params.TargetNetwork).
		Int("rateLimit", params.RateLimit).
		Bool("scanAllPorts", params.ScanAllPorts).
		Bool("disablePing", params.DisablePing).
		Msg("Scan requested")

	// Start the scan in a goroutine
	go func() {
		_, err := h.scanService.RunManualScan(r.Context(), params)
		if err != nil {
			logger.Error().Err(err).Msg("Scan failed")
		}
	}()

	// Return success response
	response := map[string]interface{}{
		"message": "Scan started",
		"template": params.Template,
		"timestamp": time.Now(),
	}

	// Include additional parameters in response if they were provided
	if params.TargetNetwork != "" {
		response["targetNetwork"] = params.TargetNetwork
	}
	if params.RateLimit > 0 {
		response["rateLimit"] = params.RateLimit
	}
	if params.ScanAllPorts {
		response["scanAllPorts"] = true
	}
	if params.DisablePing {
		response["disablePing"] = true
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted) // 202 Accepted
	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.Error().Err(err).Msg("Failed to encode response")
	}
}

// GetScanStatus returns the current status of the scanner
func (h *ScanHandler) GetScanStatus(w http.ResponseWriter, r *http.Request) {
	logger := log.With().Str("handler", "getScanStatus").Logger()

	// Get current scan status
	status := h.scanService.GetStatus()

	// Build response
	response := map[string]interface{}{
		"status": status.Status,
		"startTime": status.StartTime,
		"scanID": status.ScanID,
	}

	if status.EndTime.After(status.StartTime) {
		response["endTime"] = status.EndTime
		response["duration"] = status.EndTime.Sub(status.StartTime).String()
	}

	if status.Status == "completed" || status.Status == "error" {
		response["devicesFound"] = status.DevicesFound
		response["portsFound"] = status.PortsFound
	}

	if status.Status == "error" && status.Error != nil {
		response["error"] = status.Error.Error()
	}

	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.Error().Err(err).Msg("Failed to encode scan status")
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// GetScanTemplates returns the available scan templates
func (h *ScanHandler) GetScanTemplates(w http.ResponseWriter, r *http.Request) {
	logger := log.With().Str("handler", "getScanTemplates").Logger()

	// Get templates from scan service
	templates, err := h.scanService.GetScanTemplates()
	if err != nil {
		logger.Error().Err(err).Msg("Failed to retrieve scan templates")
		http.Error(w, "Failed to retrieve scan templates", http.StatusInternalServerError)
		return
	}

	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(templates); err != nil {
		logger.Error().Err(err).Msg("Failed to encode scan templates")
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}