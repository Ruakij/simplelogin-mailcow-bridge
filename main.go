package main

import (
	"fmt"
	"log"
	"net/http"

	"git.ruekov.eu/ruakij/simplelogin-mailcow-bridge/internal/api"
	"git.ruekov.eu/ruakij/simplelogin-mailcow-bridge/internal/auth"
	"git.ruekov.eu/ruakij/simplelogin-mailcow-bridge/internal/config"
	"git.ruekov.eu/ruakij/simplelogin-mailcow-bridge/internal/mailcow"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize Mailcow API client
	mailcowClient, err := mailcow.NewMailcowClient(cfg.MailcowAdminAPIURL, cfg.MailcowAdminAPIKey)
	if err != nil {
		log.Fatalf("Failed to initialize Mailcow API client: %v", err)
	}

	// Initialize authentication module
	authModule, err := auth.NewAuthModule(cfg.MailcowAuthMethod, cfg.MailcowServerAddress)
	if err != nil {
		log.Fatalf("Failed to initialize authentication module: %v", err)
	}

	// Initialize API
	api := api.NewAPI(cfg, mailcowClient, authModule)

	// Start server
	fmt.Printf("Starting server on port %d...\n", cfg.Port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", cfg.Port), api.Router()))
}
