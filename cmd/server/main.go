package main

import (
	"html/template"
	"log"
	"net/http"
	"os"

	"health-coach/pkg/api"
	"health-coach/pkg/auth"
	"health-coach/pkg/database"
	"health-coach/pkg/ollama"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables from .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	// Initialize database connection
	db, err := database.NewConnection()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Run database migrations
	if err := db.RunMigrations(); err != nil {
		log.Fatalf("Failed to run database migrations: %v", err)
	}

	// Initialize the configured AI client
	aiClient := ollama.NewClient()

	// Test the configured AI provider (non-fatal if it fails)
	if err := aiClient.HealthCheck(); err != nil {
		log.Printf("Warning: %s service not available: %v", aiClient.Provider(), err)
		log.Println("AI evaluation features will be disabled")
	} else {
		log.Printf("Successfully configured %s service", aiClient.Provider())
	}

	// Initialize OAuth2 configuration
	oauthConfig := auth.NewOAuthConfig()

	// Initialize authentication handler
	authHandler := auth.NewAuthHandler(db, oauthConfig)

	// Initialize authentication middleware
	authMiddleware := auth.NewAuthMiddleware(db, authHandler)

	// Initialize API handlers
	apiHandler := api.NewHandler(db, aiClient)

	// Setup router
	router := mux.NewRouter()
	router.HandleFunc("/healthz", apiHandler.ReadinessCheck).Methods("GET")

	// Setup authentication routes (these bypass the auth middleware)
	router.HandleFunc("/auth/login", authHandler.LoginHandler).Methods("GET")
	router.HandleFunc("/auth/callback", authHandler.CallbackHandler).Methods("GET")
	router.HandleFunc("/auth/logout", authHandler.LogoutHandler).Methods("GET", "POST")

	// Apply authentication middleware to all other routes
	protected := router.PathPrefix("/").Subrouter()
	protected.Use(authMiddleware.Middleware)

	// Static file serving
	protected.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./web/assets/"))))

	// Setup API routes
	apiHandler.SetupRoutes(protected)

	// Setup web routes
	setupWebRoutes(protected)

	// Login page route (unprotected)
	router.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		templates := template.Must(template.ParseGlob("web/templates/*.html"))
		if err := templates.ExecuteTemplate(w, "login.html", nil); err != nil {
			http.Error(w, "Failed to render template", http.StatusInternalServerError)
			return
		}
	}).Methods("GET")

	// Get port from environment
	port := getEnv("PORT", "8080")

	log.Printf("Starting health coach server on port %s", port)
	log.Printf("Environment: %s", getEnv("ENVIRONMENT", "development"))
	log.Printf("Google OAuth2 configured with client ID: %s", getEnv("GOOGLE_CLIENT_ID", "NOT_SET"))

	if err := http.ListenAndServe(":"+port, router); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

// setupWebRoutes configures web page routes
func setupWebRoutes(router *mux.Router) {
	// Parse templates
	templates := template.Must(template.ParseGlob("web/templates/*.html"))

	// Home page (calendar view)
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if err := templates.ExecuteTemplate(w, "calendar.html", nil); err != nil {
			http.Error(w, "Failed to render template", http.StatusInternalServerError)
			return
		}
	}).Methods("GET")

	// Journal entry page
	router.HandleFunc("/journal", func(w http.ResponseWriter, r *http.Request) {
		date := r.URL.Query().Get("date")
		if date == "" {
			date = "today"
		}

		data := map[string]interface{}{
			"Date": date,
		}

		if err := templates.ExecuteTemplate(w, "journal.html", data); err != nil {
			http.Error(w, "Failed to render template", http.StatusInternalServerError)
			return
		}
	}).Methods("GET")

	// Dashboard page
	router.HandleFunc("/dashboard", func(w http.ResponseWriter, r *http.Request) {
		if err := templates.ExecuteTemplate(w, "dashboard.html", nil); err != nil {
			http.Error(w, "Failed to render template", http.StatusInternalServerError)
			return
		}
	}).Methods("GET")

	// AI Chat page
	router.HandleFunc("/chat", func(w http.ResponseWriter, r *http.Request) {
		date := r.URL.Query().Get("date")
		evalType := r.URL.Query().Get("type")

		data := map[string]interface{}{
			"Date":           date,
			"EvaluationType": evalType,
		}

		if err := templates.ExecuteTemplate(w, "chat.html", data); err != nil {
			http.Error(w, "Failed to render template", http.StatusInternalServerError)
			return
		}
	}).Methods("GET")
}

// getEnv retrieves an environment variable with a fallback default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
