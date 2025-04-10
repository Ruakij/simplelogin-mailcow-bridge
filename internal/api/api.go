package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"

	"git.ruekov.eu/ruakij/simplelogin-mailcow-bridge/internal/alias"
	"git.ruekov.eu/ruakij/simplelogin-mailcow-bridge/internal/auth"
	"git.ruekov.eu/ruakij/simplelogin-mailcow-bridge/internal/config"
	"git.ruekov.eu/ruakij/simplelogin-mailcow-bridge/internal/mailcow"
)

// API is the API handler
type API struct {
	config        *config.Config
	mailcowClient *mailcow.MailcowClient
	authModule    *auth.AuthModule
	router        *mux.Router
}

// NewAPI creates a new API handler
func NewAPI(cfg *config.Config, mailcowClient *mailcow.MailcowClient, authModule *auth.AuthModule) *API {
	api := &API{
		config:        cfg,
		mailcowClient: mailcowClient,
		authModule:    authModule,
		router:        mux.NewRouter(),
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
}

func (a *API) handleNewAlias(w http.ResponseWriter, r *http.Request) {
	// Get username and password from request
	username, password, ok := r.BasicAuth()
	if !ok {
		http.Error(w, "Unauthorized: Basic authentication required", http.StatusUnauthorized)
		return
	}

	// Authenticate user against Mailcow
	if err := a.authModule.Authenticate(username, password); err != nil {
		errorMsg := fmt.Sprintf("Authentication failed: %v", err)
		log.Println(errorMsg)
		http.Error(w, errorMsg, http.StatusUnauthorized)
		return
	}

	// Generate alias
	generatedAlias, err := alias.GenerateAlias(username, a.config.AliasGenerationPattern)
	if err != nil {
		errorMsg := fmt.Sprintf("Failed to generate alias: %v", err)
		log.Println(errorMsg)
		http.Error(w, errorMsg, http.StatusInternalServerError)
		return
	}

	// Create alias in Mailcow
	if err := a.mailcowClient.CreateAlias(generatedAlias, username); err != nil {
		errorMsg := fmt.Sprintf("Failed to create alias in Mailcow: %v", err)
		log.Println(errorMsg)
		http.Error(w, errorMsg, http.StatusInternalServerError)
		return
	}

	// Set expiration date
	expirationDate := time.Now().AddDate(a.config.AliasValidityPeriod, 0, 0).Format(time.RFC3339)

	// Prepare response
	response := map[string]string{
		"alias":           generatedAlias,
		"expiration_date": expirationDate,
	}

	// Return response as JSON
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}
