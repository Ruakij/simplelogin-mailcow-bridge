package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"git.ruekov.eu/ruakij/simplelogin-mailcow-bridge/internal/api"
	"git.ruekov.eu/ruakij/simplelogin-mailcow-bridge/internal/auth"
	"git.ruekov.eu/ruakij/simplelogin-mailcow-bridge/internal/config"
	"git.ruekov.eu/ruakij/simplelogin-mailcow-bridge/internal/logger"
	"git.ruekov.eu/ruakij/simplelogin-mailcow-bridge/internal/mailcow"
)

// Logger middleware to log all requests
func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		requestID := fmt.Sprintf("%d", time.Now().UnixNano())

		// Create a logger with request ID for this request
		log := logger.WithRequestID(requestID)

		// Create a custom response writer to capture the status code
		rw := &responseWriter{w, http.StatusOK}

		// Process request
		next.ServeHTTP(rw, r)

		// Calculate duration
		duration := time.Since(start)
		durationFormatted := logger.FormatDuration(duration)

		// Log the request with appropriate level based on status code
		logMsg := fmt.Sprintf("[%s] %s %s %s - %d %s",
			r.RemoteAddr, r.Method, r.RequestURI, r.Proto, rw.statusCode, durationFormatted)

		if rw.statusCode >= 500 {
			log.Error(logMsg)
		} else if rw.statusCode >= 400 {
			log.Warn(logMsg)
		} else {
			log.Info(logMsg)
		}
	})
}

// Custom response writer to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func setupLogging(cfg *config.Config) {
	// Configure the logger
	logger.SetupGlobal(cfg.LogLevel, cfg.LogColorize)

	// Log startup with configured level
	logger.Info("Logging initialized with level: %s, colors: %v", cfg.LogLevel, cfg.LogColorize)
}

// setupCacheCleanup sets up a background goroutine to periodically clean the auth cache
func setupCacheCleanup(authModule *auth.AuthModule, interval time.Duration) {
	ticker := time.NewTicker(interval)
	log := logger.WithComponent("CacheCleanup")

	go func() {
		for range ticker.C {
			total, valid := authModule.CacheStats()
			if total > 0 {
				cleaned := authModule.CleanupCache()
				if cleaned > 0 {
					log.Info("Auth cache stats - Total: %d, Valid: %d, Cleaned: %d", total, valid, cleaned)
				}
			}
		}
	}()

	log.Info("Auth cache cleanup initialized with interval: %s", interval)
}

func main() {
	// Load configuration first (without logging)
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize logging with configured level
	setupLogging(cfg)
	logger.Info("Starting SimpleLogin-Mailcow Bridge service")

	// Log configuration details
	logger.Info("Configuration loaded successfully. Using port: %d, Auth method: %s", cfg.Port, cfg.MailcowAuthMethod)

	// Log cache configuration
	if cfg.AuthCacheTTL > 0 {
		logger.Info("Auth caching enabled with TTL: %d seconds", cfg.AuthCacheTTL)
	} else {
		logger.Info("Auth caching disabled")
	}

	// Initialize Mailcow API client
	mailcowLog := logger.WithComponent("Mailcow")
	mailcowLog.Info("Initializing Mailcow API client with URL: %s", cfg.MailcowAdminAPIURL)

	mailcowClient, err := mailcow.NewMailcowClient(cfg.MailcowAdminAPIURL, cfg.MailcowAdminAPIKey)
	if err != nil {
		logger.Fatal("Failed to initialize Mailcow API client: %v", err)
	}
	mailcowLog.Info("Mailcow API client initialized successfully")

	// Initialize authentication module
	authLog := logger.WithComponent("Auth")
	authLog.Info("Initializing authentication module with method: %s, server: %s", cfg.MailcowAuthMethod, cfg.MailcowServerAddress)

	authModule, err := auth.NewAuthModule(cfg.MailcowAuthMethod, cfg.MailcowServerAddress, cfg.AuthCacheTTL)
	if err != nil {
		logger.Fatal("Failed to initialize authentication module: %v", err)
	}
	authLog.Info("Authentication module initialized successfully")

	// Setup cache cleanup if caching is enabled
	if cfg.AuthCacheTTL > 0 {
		setupCacheCleanup(authModule, 10*time.Second)
	}

	// Initialize API
	apiLog := logger.WithComponent("API")
	apiLog.Info("Initializing API endpoints")

	apiHandler := api.NewAPI(cfg, mailcowClient, authModule)
	apiLog.Info("API initialized successfully")

	// Add request logging middleware
	handler := requestLogger(apiHandler.Router())

	// Start server
	serverAddr := fmt.Sprintf(":%d", cfg.Port)
	logger.Info("Server starting and listening on port %d...", cfg.Port)
	if err := http.ListenAndServe(serverAddr, handler); err != nil {
		logger.Fatal("Server failed to start: %v", err)
	}
}
