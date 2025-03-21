// internal/api/device_handlers.go
package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"

	"panopticon-scanner/internal/database"
	"panopticon-scanner/internal/models"
)

// DeviceHandler handles device-related API endpoints
type DeviceHandler struct {
	db *database.DB
}

// NewDeviceHandler creates a new device handler
func NewDeviceHandler(db *database.DB) *DeviceHandler {
	return &DeviceHandler{
		db: db,
	}
}

// RegisterRoutes registers the device routes
func (h *DeviceHandler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/api/devices", h.getDevices).Methods("GET")
	r.HandleFunc("/api/devices/{id}", h.getDeviceDetail).Methods("GET")
	r.HandleFunc("/api/devices/search", h.SearchDevices).Methods("GET")
	r.HandleFunc("/api/devices/stats", h.GetDeviceStats).Methods("GET")
}

// getDevices returns a list of all devices
func (h *DeviceHandler) getDevices(w http.ResponseWriter, r *http.Request) {
	logger := log.With().Str("handler", "getDevices").Logger()

	// Get all devices from database
	devices, err := h.db.GetAllDevices()
	if err != nil {
		logger.Error().Err(err).Msg("Failed to retrieve devices")
		http.Error(w, "Failed to retrieve devices", http.StatusInternalServerError)
		return
	}

	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(devices); err != nil {
		logger.Error().Err(err).Msg("Failed to encode devices")
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// getDeviceDetail returns detailed information about a specific device
func (h *DeviceHandler) getDeviceDetail(w http.ResponseWriter, r *http.Request) {
	logger := log.With().Str("handler", "getDeviceDetail").Logger()

	// Parse device ID from URL
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		logger.Error().Err(err).Str("id", idStr).Msg("Invalid device ID")
		http.Error(w, "Invalid device ID", http.StatusBadRequest)
		return
	}

	// Get device details from database
	deviceDetail, err := h.db.GetDeviceDetails(id)
	if err != nil {
		logger.Error().Err(err).Int64("id", id).Msg("Failed to retrieve device details")
		http.Error(w, "Device not found", http.StatusNotFound)
		return
	}

	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(deviceDetail); err != nil {
		logger.Error().Err(err).Msg("Failed to encode device details")
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// SearchDevices searches for devices based on query parameters
func (h *DeviceHandler) SearchDevices(w http.ResponseWriter, r *http.Request) {
	logger := log.With().Str("handler", "searchDevices").Logger()

	// Parse query parameter
	query := r.URL.Query().Get("q")
	if query == "" {
		logger.Warn().Msg("Missing query parameter")
		http.Error(w, "Missing query parameter", http.StatusBadRequest)
		return
	}

	// Search devices in database
	devices, err := h.db.SearchDevices(query)
	if err != nil {
		logger.Error().Err(err).Str("query", query).Msg("Failed to search devices")
		http.Error(w, "Failed to search devices", http.StatusInternalServerError)
		return
	}

	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(devices); err != nil {
		logger.Error().Err(err).Msg("Failed to encode search results")
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// GetDeviceStats returns statistics about devices in the network
func (h *DeviceHandler) GetDeviceStats(w http.ResponseWriter, r *http.Request) {
	logger := log.With().Str("handler", "getDeviceStats").Logger()

	// Get database stats
	dbStats, err := h.db.GetDatabaseStats()
	if err != nil {
		logger.Error().Err(err).Msg("Failed to retrieve database stats")
		http.Error(w, "Failed to retrieve device statistics", http.StatusInternalServerError)
		return
	}

	// Build response with network statistics
	stats := models.NetworkStats{
		TotalDevices: dbStats["deviceCount"].(int),
		TotalPorts:   dbStats["portCount"].(int),
	}

	// Include OS distribution if available
	if osDistribution, ok := dbStats["osDistribution"].(map[string]int); ok {
		stats.OSDistribution = osDistribution
	}

	// Calculate port distribution
	portDistribution, err := h.getPortDistribution()
	if err != nil {
		logger.Warn().Err(err).Msg("Failed to get port distribution")
	} else {
		stats.PortDistribution = portDistribution
	}

	// Calculate service distribution
	serviceDistribution, err := h.getServiceDistribution()
	if err != nil {
		logger.Warn().Err(err).Msg("Failed to get service distribution")
	} else {
		stats.ServiceDistribution = serviceDistribution
	}

	// Get new devices in the last 24 hours
	newDevices, err := h.getNewDevicesCount(24 * time.Hour)
	if err != nil {
		logger.Warn().Err(err).Msg("Failed to get new devices count")
	} else {
		stats.NewDevices = newDevices
	}

	// Get devices with changes in the last 24 hours
	changedDevices, err := h.getChangedDevicesCount(24 * time.Hour)
	if err != nil {
		logger.Warn().Err(err).Msg("Failed to get changed devices count")
	} else {
		stats.ChangedDevices = changedDevices
	}

	// Format the response with additional metadata
	response := map[string]interface{}{
		"totalDevices":       stats.TotalDevices,
		"totalPorts":         stats.TotalPorts,
		"osDistribution":     stats.OSDistribution,
		"portDistribution":   stats.PortDistribution,
		"serviceDistribution": stats.ServiceDistribution,
		"newDevices":         stats.NewDevices,
		"changedDevices":     stats.ChangedDevices,
		"lastScanTime":       dbStats["lastScanTime"],
		"generatedAt":        time.Now(),
	}

	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.Error().Err(err).Msg("Failed to encode statistics")
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// Helper methods for statistics

// getPortDistribution returns the distribution of ports across devices
func (h *DeviceHandler) getPortDistribution() (map[int]int, error) {
	portDistribution := make(map[int]int)
	
	rows, err := h.db.Query(`
		SELECT port_number, COUNT(*) as count
		FROM ports
		GROUP BY port_number
		ORDER BY count DESC
		LIMIT 10
	`)
	
	if err != nil {
		return nil, fmt.Errorf("failed to query port distribution: %w", err)
	}
	defer rows.Close()
	
	for rows.Next() {
		var port, count int
		if err := rows.Scan(&port, &count); err != nil {
			return nil, fmt.Errorf("failed to scan port distribution row: %w", err)
		}
		portDistribution[port] = count
	}
	
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating port distribution rows: %w", err)
	}
	
	return portDistribution, nil
}

// getServiceDistribution returns the distribution of services across devices
func (h *DeviceHandler) getServiceDistribution() (map[string]int, error) {
	serviceDistribution := make(map[string]int)
	
	rows, err := h.db.Query(`
		SELECT service_name, COUNT(*) as count
		FROM ports
		WHERE service_name <> ''
		GROUP BY service_name
		ORDER BY count DESC
		LIMIT 10
	`)
	
	if err != nil {
		return nil, fmt.Errorf("failed to query service distribution: %w", err)
	}
	defer rows.Close()
	
	for rows.Next() {
		var service string
		var count int
		if err := rows.Scan(&service, &count); err != nil {
			return nil, fmt.Errorf("failed to scan service distribution row: %w", err)
		}
		if service == "" {
			service = "unknown"
		}
		serviceDistribution[service] = count
	}
	
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating service distribution rows: %w", err)
	}
	
	return serviceDistribution, nil
}

// getNewDevicesCount returns the count of new devices discovered within the given duration
func (h *DeviceHandler) getNewDevicesCount(duration time.Duration) (int, error) {
	var count int
	cutoff := time.Now().Add(-duration)
	
	err := h.db.QueryRow(`
		SELECT COUNT(*) 
		FROM devices 
		WHERE first_seen > ?
	`, cutoff).Scan(&count)
	
	if err != nil {
		return 0, fmt.Errorf("failed to get new devices count: %w", err)
	}
	
	return count, nil
}

// getChangedDevicesCount returns the count of devices with changes within the given duration
func (h *DeviceHandler) getChangedDevicesCount(duration time.Duration) (int, error) {
	var count int
	cutoff := time.Now().Add(-duration)
	
	err := h.db.QueryRow(`
		SELECT COUNT(DISTINCT device_id) 
		FROM changes 
		WHERE timestamp > ? AND change_type IN ('device_change', 'port_change')
	`, cutoff).Scan(&count)
	
	if err != nil {
		return 0, fmt.Errorf("failed to get changed devices count: %w", err)
	}
	
	return count, nil
}