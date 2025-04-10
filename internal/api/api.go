package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"

	"git.ruekov.eu/ruakij/simplelogin-mailcow-bridge/internal/alias"
	"git.ruekov.eu/ruakij/simplelogin-mailcow-bridge/internal/auth"
	"git.ruekov.eu/ruakij/simplelogin-mailcow-bridge/internal/config"
	"git.ruekov.eu/ruakij/simplelogin-mailcow-bridge/internal/logger"
	"git.ruekov.eu/ruakij/simplelogin-mailcow-bridge/internal/mailcow"
)

// API is the API handler
type API struct {
	config        *config.Config
	mailcowClient *mailcow.MailcowClient
	authModule    *auth.AuthModule
	router        *mux.Router
	logger        *logger.Logger
}

// NewAPI creates a new API handler
func NewAPI(cfg *config.Config, mailcowClient *mailcow.MailcowClient, authModule *auth.AuthModule) *API {
	api := &API{
		config:        cfg,
		mailcowClient: mailcowClient,
		authModule:    authModule,
		router:        mux.NewRouter(),
		logger:        logger.WithComponent("API"),
	}
	api.routes()
	return api
}

// Router returns the router
func (a *API) Router() http.Handler {
	return a.router
}

func (a *API) routes() {
	a.router.HandleFunc("/api/alias/random/new", a.handleNewAlias).Methods("POST")
	a.logger.Debug("Registered route: POST /api/alias/random/new")
}

func (a *API) handleNewAlias(w http.ResponseWriter, r *http.Request) {
	requestID := fmt.Sprintf("%d", time.Now().UnixNano())
	log := a.logger.WithRequestID(requestID)

	log.Info("Processing new alias request")

	// Get username and password from the Authentication header
	// Format: "Authentication: username:password"
	authHeader := r.Header.Get("Authentication")
	if authHeader == "" {
		log.Warn("Authentication failed: No Authentication header provided")
		http.Error(w, "Unauthorized: Authentication header required", http.StatusUnauthorized)
		return
	}

	// Split the header value to get username and password
	credentials := strings.SplitN(authHeader, ":", 2)
	if len(credentials) != 2 {
		log.Warn("Authentication failed: Invalid Authentication header format")
		http.Error(w, "Unauthorized: Authentication header must be in the format 'username:password'", http.StatusUnauthorized)
		return
	}

	username := credentials[0]
	password := credentials[1]

	// Mask username for logging
	maskedUser := username
	if len(maskedUser) > 3 {
		maskedUser = maskedUser[:3] + "***"
	}
	log.Info("Authenticating user: %s", maskedUser)

	// Authenticate user against Mailcow
	if err := a.authModule.Authenticate(username, password); err != nil {
		errorMsg := fmt.Sprintf("Authentication failed: %v", err)
		log.Warn("%s", errorMsg)
		http.Error(w, errorMsg, http.StatusUnauthorized)
		return
	}
	log.Info("User %s authenticated successfully", maskedUser)

	// Generate alias
	log.Info("Generating alias using pattern: %s", a.config.AliasGenerationPattern)
	generatedAlias, err := alias.GenerateAlias(username, a.config.AliasGenerationPattern)
	if err != nil {
		errorMsg := fmt.Sprintf("Failed to generate alias: %v", err)
		log.Error("%s", errorMsg)
		http.Error(w, errorMsg, http.StatusInternalServerError)
		return
	}
	log.Info("Generated alias: %s", generatedAlias)

	// Create alias in Mailcow
	log.Info("Creating alias in Mailcow: %s -> %s", generatedAlias, maskedUser)
	if err := a.mailcowClient.CreateAlias(generatedAlias, username); err != nil {
		errorMsg := fmt.Sprintf("Failed to create alias in Mailcow: %v", err)
		log.Error("%s", errorMsg)
		http.Error(w, errorMsg, http.StatusInternalServerError)
		return
	}
	log.Info("Alias created successfully in Mailcow")

	// Set expiration date
	expirationDate := time.Now().AddDate(a.config.AliasValidityPeriod, 0, 0).Format(time.RFC3339)
	log.Debug("Setting expiration date: %s", expirationDate)

	// Prepare response
	response := map[string]string{
		"alias":           generatedAlias,
		"expiration_date": expirationDate,
	}

	// Return response as JSON
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Error("Failed to encode response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.Info("Successfully completed new alias request")
}
