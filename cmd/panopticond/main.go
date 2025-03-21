// Command panopticond is the main executable for the Panopticon Scanner backend service.
// It initializes the database, scanner service, and HTTP API server, and handles
// graceful shutdown when terminated.
package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"panopticon-scanner/internal/api"
	"panopticon-scanner/internal/config"
	"panopticon-scanner/internal/database"
	"panopticon-scanner/internal/scanner"
)

// Global variables for command line flags
var logLevelFlag string

// parseFlags parses command line flags and returns the config path
func parseFlags() string {
	configPath := flag.String("config", "configs/config.yaml", "Path to configuration file")
	flag.StringVar(&logLevelFlag, "log-level", "info", "Log level (debug, info, warn, error)")
	flag.Parse()
	return *configPath
}

func main() {
	// Parse command line flags
	configPath := parseFlags()

	// Configure logging
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	level, err := zerolog.ParseLevel(logLevelFlag)
	if err != nil {
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)

	// Use colored console output for development
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})

	log.Info().Msg("Starting Panopticon Network Scanner")

	// Load configuration
	cfg := config.GetConfig()
	if err := cfg.LoadConfig(configPath); err != nil {
		log.Fatal().Err(err).Str("path", configPath).Msg("Failed to load configuration")
	}

	// Initialize database
	log.Info().Str("path", cfg.Database.Path).Msg("Initializing database")
	db, err := database.New(cfg.Database.Path)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize database")
	}
	defer db.Close()

	// Initialize scan service
	log.Info().Msg("Initializing scan service")
	scanService := scanner.New(cfg, db)

	if err := scanService.Start(); err != nil {
		log.Fatal().Err(err).Msg("Failed to start scan service")
	}

	// Initialize router and API handlers
	router := mux.NewRouter()

	// Create API handlers
	scanHandler := api.NewScanHandler(scanService)
	deviceHandler := api.NewDeviceHandler(db)
	statusHandler := api.NewStatusHandler(db, scanService, cfg)

	// Register API routes
	scanHandler.RegisterRoutes(router)
	deviceHandler.RegisterRoutes(router)
	statusHandler.RegisterRoutes(router)

	// Register static file server for the Electron UI
	router.PathPrefix("/").Handler(http.FileServer(http.Dir("./ui/build")))

	// Set up CORS
	corsMiddleware := handlers.CORS(
		handlers.AllowedOrigins(cfg.Server.AllowedOrigins),
		handlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}),
		handlers.AllowedHeaders([]string{"Content-Type", "Authorization"}),
	)

	// Set up HTTP server
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	server := &http.Server{
		Addr:         addr,
		Handler:      corsMiddleware(router),
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
	}

	// Start the server in a goroutine
	go func() {
		log.Info().Str("addr", addr).Msg("Starting HTTP server")
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("HTTP server failed")
		}
	}()

	// Set up signal handling for graceful shutdown
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	// Wait for termination signal
	sig := <-signalChan
	log.Info().Str("signal", sig.String()).Msg("Received termination signal")

	// Begin graceful shutdown
	log.Info().Msg("Shutting down...")

	// Create a shutdown context with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(
		context.Background(),
		time.Duration(cfg.Server.ShutdownTimeout) * time.Second,
	)
	defer shutdownCancel()

	// Shutdown HTTP server
	log.Info().Msg("Shutting down HTTP server")
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("HTTP server shutdown failed")
	}

	// Stop scan service
	log.Info().Msg("Stopping scan service")
	if err := scanService.Stop(); err != nil {
		log.Error().Err(err).Msg("Scan service shutdown failed")
	}

	// Optimize database before exit
	log.Info().Msg("Optimizing database before exit")
	if err := db.OptimizeDatabase(); err != nil {
		log.Error().Err(err).Msg("Database optimization failed")
	}

	log.Info().Msg("Panopticon has been shut down gracefully")
}