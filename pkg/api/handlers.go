package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"health-coach/pkg/auth"
	"health-coach/pkg/database"
	"health-coach/pkg/models"
	"health-coach/pkg/ollama"

	"github.com/gorilla/mux"
)

// Handler contains all the dependencies for API handlers
type Handler struct {
	db     *database.DB
	ollama *ollama.Client
}

// NewHandler creates a new API handler instance
func NewHandler(db *database.DB, ollamaClient *ollama.Client) *Handler {
	return &Handler{
		db:     db,
		ollama: ollamaClient,
	}
}

// SetupRoutes configures all API routes
func (h *Handler) SetupRoutes(router *mux.Router) {
	api := router.PathPrefix("/api").Subrouter()

	// Health check
	api.HandleFunc("/health", h.HealthCheck).Methods("GET")

	// Journal entries
	api.HandleFunc("/journal/{date}", h.GetJournalEntry).Methods("GET")
	api.HandleFunc("/journal", h.CreateJournalEntry).Methods("POST")
	api.HandleFunc("/journal/{id}", h.UpdateJournalEntry).Methods("PUT")
	api.HandleFunc("/journal/{date}/chat", h.GetChatHistory).Methods("GET")

	// AI evaluation endpoints
	api.HandleFunc("/ollama/evaluate", h.EvaluateEntry).Methods("POST")

	// Data visualization endpoints
	api.HandleFunc("/data/summary", h.GetDataSummary).Methods("GET")
	api.HandleFunc("/data/trends", h.GetTrends).Methods("GET")
}

// HealthCheck provides a simple health check endpoint
func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	log.Printf("HealthCheck: Starting health check")

	status := map[string]string{
		"status": "healthy",
		"time":   time.Now().Format(time.RFC3339),
	}

	// Check database connection
	log.Printf("HealthCheck: Checking database connection")
	if err := h.db.Ping(); err != nil {
		log.Printf("HealthCheck: Database unhealthy: %v", err)
		status["database"] = "unhealthy: " + err.Error()
		w.WriteHeader(http.StatusServiceUnavailable)
	} else {
		log.Printf("HealthCheck: Database healthy")
		status["database"] = "healthy"
	}

	// Check Ollama connection
	log.Printf("HealthCheck: Checking Ollama connection")
	if err := h.ollama.HealthCheck(); err != nil {
		log.Printf("HealthCheck: Ollama unhealthy: %v", err)
		status[h.ollama.Provider()] = "unhealthy: " + err.Error()
		// Don't fail the health check if the AI provider is down, as it is not critical
	} else {
		log.Printf("HealthCheck: %s healthy", h.ollama.Provider())
		status[h.ollama.Provider()] = "healthy"
	}

	w.Header().Set("Content-Type", "application/json")
	log.Printf("HealthCheck: Returning response: %+v", status)
	json.NewEncoder(w).Encode(status)
}

// ReadinessCheck verifies the database dependency without requiring a user session.
func (h *Handler) ReadinessCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := h.db.Ping(); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{"status": "unavailable"})
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"status": "ready"})
}

// GetJournalEntry retrieves a journal entry for a specific date
func (h *Handler) GetJournalEntry(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.RequireAuth(w, r)
	if !ok {
		return
	}

	vars := mux.Vars(r)
	dateStr := vars["date"]

	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		http.Error(w, "Invalid date format. Use YYYY-MM-DD", http.StatusBadRequest)
		return
	}

	entry, err := h.db.GetDailyEntry(user.ID, date)
	if err != nil {
		http.Error(w, "Failed to retrieve journal entry", http.StatusInternalServerError)
		return
	}

	if entry == nil {
		// Return an empty entry template for the requested date
		entry = &models.DailyEntry{
			UserID: user.ID,
			Date:   models.CustomDate{Time: date},
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entry)
}

// CreateJournalEntry creates a new journal entry
func (h *Handler) CreateJournalEntry(w http.ResponseWriter, r *http.Request) {
	log.Printf("CreateJournalEntry: Starting request")

	user, ok := auth.RequireAuth(w, r)
	if !ok {
		log.Printf("CreateJournalEntry: Authentication failed")
		return
	}
	log.Printf("CreateJournalEntry: Authenticated user ID %d", user.ID)

	var entry models.DailyEntry
	if err := json.NewDecoder(r.Body).Decode(&entry); err != nil {
		log.Printf("CreateJournalEntry: Failed to decode JSON: %v", err)
		http.Error(w, "Invalid JSON data", http.StatusBadRequest)
		return
	}
	log.Printf("CreateJournalEntry: Decoded entry: %+v", entry)

	// Set the user ID from the authenticated user
	entry.UserID = user.ID

	// If no date is provided, use today's date
	if entry.Date.IsZero() {
		entry.Date = models.CustomDate{Time: time.Now().Truncate(24 * time.Hour)}
		log.Printf("CreateJournalEntry: No date provided, using today: %v", entry.Date.Time)
	}

	log.Printf("CreateJournalEntry: Entry date: %v, current time: %v", entry.Date.Time, time.Now())
	// Validate the entry date is not in the future
	if entry.Date.After(time.Now()) {
		log.Printf("CreateJournalEntry: Date is in future, rejecting")
		http.Error(w, "Cannot create entries for future dates", http.StatusBadRequest)
		return
	}

	log.Printf("CreateJournalEntry: Checking for existing entry for user %d on date %v", user.ID, entry.Date.Time)
	// Check if entry already exists for this date
	existing, err := h.db.GetDailyEntry(user.ID, entry.Date.Time)
	log.Printf("CreateJournalEntry: GetDailyEntry returned, existing=%v, err=%v", existing != nil, err)
	if err != nil {
		log.Printf("CreateJournalEntry: Error checking existing entry: %v", err)
		http.Error(w, "Failed to check existing entry", http.StatusInternalServerError)
		return
	}

	if existing != nil {
		log.Printf("CreateJournalEntry: Found existing entry, updating it")
		// Update existing entry instead
		existing.FoodIntake = entry.FoodIntake
		existing.WaterIntake = entry.WaterIntake
		existing.BedTime = entry.BedTime
		existing.WakeTime = entry.WakeTime
		existing.SleepQuality = entry.SleepQuality
		existing.AlcoholUnits = entry.AlcoholUnits
		existing.ExerciseType = entry.ExerciseType
		existing.ExerciseDuration = entry.ExerciseDuration
		existing.ExerciseIntensity = entry.ExerciseIntensity
		existing.Notes = entry.Notes

		if err := h.db.UpdateDailyEntry(existing); err != nil {
			log.Printf("CreateJournalEntry: Failed to update existing entry: %v", err)
			http.Error(w, "Failed to update journal entry", http.StatusInternalServerError)
			return
		}

		log.Printf("CreateJournalEntry: Successfully updated existing entry")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(existing)
		return
	}

	log.Printf("CreateJournalEntry: No existing entry found, creating new one")
	// Create new entry
	if err := h.db.CreateDailyEntry(&entry); err != nil {
		log.Printf("CreateJournalEntry: Failed to create new entry: %v", err)
		http.Error(w, "Failed to create journal entry", http.StatusInternalServerError)
		return
	}

	log.Printf("CreateJournalEntry: Successfully created new entry with ID %d", entry.ID)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(entry)
}

// UpdateJournalEntry updates an existing journal entry
func (h *Handler) UpdateJournalEntry(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.RequireAuth(w, r)
	if !ok {
		return
	}

	vars := mux.Vars(r)
	entryID, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid entry ID", http.StatusBadRequest)
		return
	}

	var entry models.DailyEntry
	if err := json.NewDecoder(r.Body).Decode(&entry); err != nil {
		http.Error(w, "Invalid JSON data", http.StatusBadRequest)
		return
	}

	// Set the ID and user ID
	entry.ID = entryID
	entry.UserID = user.ID

	if err := h.db.UpdateDailyEntry(&entry); err != nil {
		http.Error(w, "Failed to update journal entry", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entry)
}

// EvaluationRequest represents a request for AI evaluation
type EvaluationRequest struct {
	Date           string `json:"date"`
	EvaluationType string `json:"evaluation_type"`    // "daily", "historical", or "followup"
	Question       string `json:"question,omitempty"` // For follow-up questions
}

// EvaluateEntry performs AI evaluation of a journal entry
func (h *Handler) EvaluateEntry(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.RequireAuth(w, r)
	if !ok {
		return
	}

	var req EvaluationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON data", http.StatusBadRequest)
		return
	}

	date, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		http.Error(w, "Invalid date format. Use YYYY-MM-DD", http.StatusBadRequest)
		return
	}

	// Get the current entry
	entry, err := h.db.GetDailyEntry(user.ID, date)
	if err != nil {
		http.Error(w, "Failed to retrieve journal entry", http.StatusInternalServerError)
		return
	}

	if entry == nil {
		http.Error(w, "No journal entry found for the specified date", http.StatusNotFound)
		return
	}

	var evaluation *models.OllamaResponse

	if req.EvaluationType == "historical" {
		// Get historical data for RAG evaluation
		recentEntries, err := h.db.GetRecentEntries(user.ID, 30) // Last 30 entries
		if err != nil {
			log.Printf("EvaluateEntry: Failed to retrieve historical data: %v", err)
			http.Error(w, "Failed to retrieve historical data", http.StatusInternalServerError)
			return
		}

		summary, err := h.db.GetHistoricalSummary(user.ID, 90) // Last 90 days
		if err != nil {
			log.Printf("EvaluateEntry: Failed to retrieve historical summary: %v", err)
			http.Error(w, "Failed to retrieve historical summary", http.StatusInternalServerError)
			return
		}

		evaluation, err = h.ollama.EvaluateHistorical(*entry, recentEntries, summary)
		if err != nil {
			log.Printf("EvaluateEntry: Failed to generate historical evaluation: %v", err)
			http.Error(w, "Failed to generate historical evaluation", http.StatusInternalServerError)
			return
		}
	} else if req.EvaluationType == "followup" {
		// Handle follow-up questions
		if req.Question == "" {
			http.Error(w, "Question is required for follow-up type", http.StatusBadRequest)
			return
		}

		evaluation, err = h.ollama.AnswerFollowupQuestion(*entry, req.Question)
		if err != nil {
			log.Printf("EvaluateEntry: Failed to generate follow-up response: %v", err)
			http.Error(w, "Failed to generate follow-up response", http.StatusInternalServerError)
			return
		}
	} else {
		// Perform daily evaluation, enriched with a compact last-7-days vs last-30-days trend
		lastWeek, weekErr := h.db.GetRecentEntries(user.ID, 7)
		if weekErr != nil {
			log.Printf("EvaluateEntry: Failed to retrieve last 7 days for trend context: %v", weekErr)
		}

		summary30, summaryErr := h.db.GetHistoricalSummary(user.ID, 30)
		if summaryErr != nil {
			log.Printf("EvaluateEntry: Failed to retrieve 30-day summary for trend context: %v", summaryErr)
		}

		evaluation, err = h.ollama.EvaluateDaily(*entry, lastWeek, summary30)
		if err != nil {
			log.Printf("EvaluateEntry: Failed to generate daily evaluation: %v", err)
			http.Error(w, "Failed to generate daily evaluation", http.StatusInternalServerError)
			return
		}
	}

	if saveErr := h.saveChatExchange(user.ID, date, req.EvaluationType, req.Question, evaluation); saveErr != nil {
		log.Printf("EvaluateEntry: Failed to persist chat history: %v", saveErr)
	}

	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")

	// Return the markdown content directly
	if evaluation.Markdown != "" {
		w.Write([]byte(evaluation.Markdown))
	} else {
		// Fallback to JSON format if no markdown is available
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(evaluation)
	}
}

// saveChatExchange persists the user prompt and AI response for a day's evaluation chat.
func (h *Handler) saveChatExchange(userID int64, date time.Time, evaluationType, question string, evaluation *models.OllamaResponse) error {
	if evaluationType == "" {
		evaluationType = "daily"
	}

	userContent := question
	if evaluationType != "followup" {
		userContent = fmt.Sprintf("Please provide a %s health evaluation for %s.", evaluationType, date.Format("Monday, January 2, 2006"))
	}

	now := time.Now()
	aiFormat := "json"
	aiContent := ""
	if evaluation.Markdown != "" {
		aiFormat = "markdown"
		aiContent = evaluation.Markdown
	} else {
		encoded, err := json.Marshal(evaluation)
		if err != nil {
			return fmt.Errorf("failed to encode evaluation for chat history: %w", err)
		}
		aiContent = string(encoded)
	}

	messages := []models.ChatMessage{
		{Role: "user", Type: evaluationType, Format: "text", Content: userContent, CreatedAt: now},
		{Role: "ai", Type: evaluationType, Format: aiFormat, Content: aiContent, CreatedAt: now},
	}

	return h.db.AppendChatMessages(userID, date, messages)
}

// GetChatHistory retrieves the stored AI evaluation chat for a specific day
func (h *Handler) GetChatHistory(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.RequireAuth(w, r)
	if !ok {
		return
	}

	vars := mux.Vars(r)
	date, err := time.Parse("2006-01-02", vars["date"])
	if err != nil {
		http.Error(w, "Invalid date format. Use YYYY-MM-DD", http.StatusBadRequest)
		return
	}

	history, err := h.db.GetChatHistory(user.ID, date)
	if err != nil {
		http.Error(w, "Failed to retrieve chat history", http.StatusInternalServerError)
		return
	}

	messages := []models.ChatMessage{}
	if history != nil {
		messages = history.Messages
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"messages": messages})
}

// GetDataSummary provides summary statistics for data visualization
func (h *Handler) GetDataSummary(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.RequireAuth(w, r)
	if !ok {
		return
	}

	// Get query parameters
	daysStr := r.URL.Query().Get("days")
	days := 30 // Default to 30 days
	if daysStr != "" {
		if d, err := strconv.Atoi(daysStr); err == nil && d > 0 {
			days = d
		}
	}

	summary, err := h.db.GetHistoricalSummary(user.ID, days)
	if err != nil {
		http.Error(w, "Failed to retrieve data summary", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}

// TrendData represents trend data for visualization
type TrendData struct {
	Date         string  `json:"date"`
	WaterIntake  float64 `json:"water_intake"`
	SleepQuality *int    `json:"sleep_quality"`
	AlcoholUnits float64 `json:"alcohol_units"`
	HasExercise  bool    `json:"has_exercise"`
}

// GetTrends provides trend data for visualization
func (h *Handler) GetTrends(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.RequireAuth(w, r)
	if !ok {
		return
	}

	// Get query parameters
	daysStr := r.URL.Query().Get("days")
	days := 30 // Default to 30 days
	if daysStr != "" {
		if d, err := strconv.Atoi(daysStr); err == nil && d > 0 {
			days = d
		}
	}

	entries, err := h.db.GetRecentEntries(user.ID, days)
	if err != nil {
		http.Error(w, "Failed to retrieve trend data", http.StatusInternalServerError)
		return
	}

	var trends []TrendData
	for _, entry := range entries {
		trend := TrendData{
			Date:         entry.Date.Time.Format("2006-01-02"),
			WaterIntake:  entry.WaterIntake,
			SleepQuality: entry.SleepQuality,
			AlcoholUnits: entry.AlcoholUnits,
			HasExercise:  entry.ExerciseType != "",
		}
		trends = append(trends, trend)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(trends)
}

// ErrorResponse represents an API error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Code    int    `json:"code"`
}

// SendError sends a structured error response
func SendError(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)

	response := ErrorResponse{
		Error:   http.StatusText(code),
		Message: message,
		Code:    code,
	}

	json.NewEncoder(w).Encode(response)
}
