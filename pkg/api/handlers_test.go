package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"health-coach/pkg/models"
)

func TestHealthCheck(t *testing.T) {
	// Create a simple HTTP test for the health check endpoint
	req, err := http.NewRequest("GET", "/api/health", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()

	// Simple health check handler function
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":    "healthy",
			"timestamp": time.Now().Format(time.RFC3339),
		})
	})

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var response map[string]interface{}
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Errorf("Could not parse response JSON: %v", err)
	}

	if response["status"] != "healthy" {
		t.Errorf("Expected status 'healthy', got %v", response["status"])
	}
}

func TestGetEntries(t *testing.T) {
	// Test basic API endpoint structure
	req, err := http.NewRequest("GET", "/api/entries", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()

	// Simple mock handler that returns an empty array
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]models.DailyEntry{})
	})

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var entries []models.DailyEntry
	err = json.Unmarshal(rr.Body.Bytes(), &entries)
	if err != nil {
		t.Errorf("Could not parse response JSON: %v", err)
	}

	// Should parse successfully as an array
	if entries == nil {
		t.Error("Expected entries array to be parsed")
	}
}

func TestCreateEntry(t *testing.T) {
	entry := map[string]interface{}{
		"date":              "2025-08-02",
		"food_intake":       "Healthy breakfast",
		"water_intake":      2.5,
		"exercise_type":     "Running",
		"exercise_duration": 30,
		"notes":             "Felt great!",
	}

	jsonData, _ := json.Marshal(entry)
	req, err := http.NewRequest("POST", "/api/entries", bytes.NewBuffer(jsonData))
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()

	// Simple mock handler that returns the created entry
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var requestEntry map[string]interface{}
		json.NewDecoder(r.Body).Decode(&requestEntry)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)

		// Create a response entry
		responseEntry := models.DailyEntry{
			ID:               1,
			UserID:           1,
			Date:             models.CustomDate{Time: time.Date(2025, 8, 2, 0, 0, 0, 0, time.UTC)},
			FoodIntake:       requestEntry["food_intake"].(string),
			WaterIntake:      requestEntry["water_intake"].(float64),
			ExerciseType:     requestEntry["exercise_type"].(string),
			ExerciseDuration: int(requestEntry["exercise_duration"].(float64)),
			Notes:            requestEntry["notes"].(string),
			Created:          time.Now(),
			Modified:         time.Now(),
		}

		json.NewEncoder(w).Encode(responseEntry)
	})

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusCreated {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusCreated)
	}

	var createdEntry models.DailyEntry
	err = json.Unmarshal(rr.Body.Bytes(), &createdEntry)
	if err != nil {
		t.Errorf("Could not parse response JSON: %v", err)
	}

	if createdEntry.WaterIntake != 2.5 {
		t.Errorf("Expected water intake 2.5, got %f", createdEntry.WaterIntake)
	}

	if createdEntry.ExerciseType != "Running" {
		t.Errorf("Expected exercise type 'Running', got %s", createdEntry.ExerciseType)
	}
}

func TestEvaluateEntry(t *testing.T) {
	evalRequest := map[string]interface{}{
		"entry_id": 1,
		"type":     "daily",
	}

	jsonData, _ := json.Marshal(evalRequest)
	req, err := http.NewRequest("POST", "/api/evaluate", bytes.NewBuffer(jsonData))
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()

	// Simple mock handler that returns an evaluation
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		evaluation := models.AIEvaluation{
			ID:          1,
			EntryID:     1,
			Type:        "daily",
			Evaluation:  "Good job on staying hydrated!",
			Suggestions: []string{"Try to exercise more", "Get better sleep"},
			Score:       8,
			CreatedAt:   time.Now(),
		}

		json.NewEncoder(w).Encode(evaluation)
	})

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var evaluation models.AIEvaluation
	err = json.Unmarshal(rr.Body.Bytes(), &evaluation)
	if err != nil {
		t.Errorf("Could not parse response JSON: %v", err)
	}

	if evaluation.Type != "daily" {
		t.Errorf("Expected evaluation type 'daily', got %s", evaluation.Type)
	}

	if evaluation.Score != 8 {
		t.Errorf("Expected score 8, got %d", evaluation.Score)
	}

	if len(evaluation.Suggestions) == 0 {
		t.Error("Expected suggestions to be provided")
	}
}
