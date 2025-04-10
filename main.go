package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"git.ruekov.eu/ruakij/simplelogin-mailcow-bridge/internal/api"
	"git.ruekov.eu/ruakij/simplelogin-mailcow-bridge/internal/auth"
	"git.ruekov.eu/ruakij/simplelogin-mailcow-bridge/internal/config"
	"git.ruekov.eu/ruakij/simplelogin-mailcow-bridge/internal/mailcow"
)

// Logger middleware to log all requests
func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create a custom response writer to capture the status code
		rw := &responseWriter{w, http.StatusOK}

		// Process request
		next.ServeHTTP(rw, r)

		// Calculate duration
		duration := time.Since(start)

		// Log the request
		log.Printf(
			"[%s] %s %s %s - %d %s",
			r.RemoteAddr,
			r.Method,
			r.RequestURI,
			r.Proto,
			rw.statusCode,
			duration,
		)
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

func setupLogging() {
	// Set up log formatting with timestamp
	log.SetOutput(os.Stdout)
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.LUTC | log.Lshortfile)
	log.Println("Logging initialized")
}

// setupCacheCleanup sets up a background goroutine to periodically clean the auth cache
func setupCacheCleanup(authModule *auth.AuthModule, interval time.Duration) {
	ticker := time.NewTicker(interval)

	go func() {
		for range ticker.C {
			total, valid := authModule.CacheStats()
			if total > 0 {
				cleaned := authModule.CleanupCache()
				if cleaned > 0 {
					log.Printf("Auth cache stats - Total: %d, Valid: %d, Cleaned: %d", total, valid, cleaned)
				}
			}
		}
	}()

	log.Printf("Auth cache cleanup initialized with interval: %s", interval)
}

func main() {
	// Initialize logging
	setupLogging()
	log.Println("Starting SimpleLogin-Mailcow Bridge service")

	// Load configuration
	log.Println("Loading configuration...")
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}
	log.Printf("Configuration loaded successfully. Using port: %d, Auth method: %s", cfg.Port, cfg.MailcowAuthMethod)

	// Log cache configuration
	if cfg.AuthCacheTTL > 0 {
		log.Printf("Auth caching enabled with TTL: %d seconds", cfg.AuthCacheTTL)
	} else {
		log.Println("Auth caching disabled")
	}

	// Initialize Mailcow API client
	log.Printf("Initializing Mailcow API client with URL: %s", cfg.MailcowAdminAPIURL)
	mailcowClient, err := mailcow.NewMailcowClient(cfg.MailcowAdminAPIURL, cfg.MailcowAdminAPIKey)
	if err != nil {
		log.Fatalf("Failed to initialize Mailcow API client: %v", err)
	}
	log.Println("Mailcow API client initialized successfully")

	// Initialize authentication module
	log.Printf("Initializing authentication module with method: %s, server: %s", cfg.MailcowAuthMethod, cfg.MailcowServerAddress)
	authModule, err := auth.NewAuthModule(cfg.MailcowAuthMethod, cfg.MailcowServerAddress, cfg.AuthCacheTTL)
	if err != nil {
		log.Fatalf("Failed to initialize authentication module: %v", err)
	}
	log.Println("Authentication module initialized successfully")

	// Setup cache cleanup if caching is enabled
	if cfg.AuthCacheTTL > 0 {
		setupCacheCleanup(authModule, 10*time.Second)
	}

	// Initialize API
	log.Println("Initializing API endpoints")
	apiHandler := api.NewAPI(cfg, mailcowClient, authModule)
	log.Println("API initialized successfully")

	// Add request logging middleware
	handler := requestLogger(apiHandler.Router())

	// Start server
	serverAddr := fmt.Sprintf(":%d", cfg.Port)
	log.Printf("Server starting and listening on port %d...", cfg.Port)
	log.Fatal(http.ListenAndServe(serverAddr, handler))
}
