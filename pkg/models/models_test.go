package models

import (
	"testing"
	"time"
)

func TestDailyEntry(t *testing.T) {
	entry := &DailyEntry{
		ID:               1,
		UserID:           1,
		Date:             CustomDate{Time: time.Now()},
		FoodIntake:       "Breakfast: oatmeal, Lunch: salad",
		WaterIntake:      2.5,
		AlcoholUnits:     0.0,
		ExerciseType:     "Running",
		ExerciseDuration: 30,
		Notes:            "Felt great today!",
	}

	if entry.ID != 1 {
		t.Errorf("Expected ID to be 1, got %d", entry.ID)
	}

	if entry.WaterIntake != 2.5 {
		t.Errorf("Expected WaterIntake to be 2.5, got %f", entry.WaterIntake)
	}

	if entry.ExerciseType != "Running" {
		t.Errorf("Expected ExerciseType to be 'Running', got %s", entry.ExerciseType)
	}
}

func TestCustomTimeScanBytes(t *testing.T) {
	var value CustomTime
	if err := value.Scan([]byte("23:45:00")); err != nil {
		t.Fatalf("Scan() error = %v", err)
	}
	if got := value.Format("15:04"); got != "23:45" {
		t.Fatalf("Scan() time = %q, want %q", got, "23:45")
	}
}

func TestUser(t *testing.T) {
	user := &User{
		ID:    1,
		Email: "test@example.com",
		Name:  "Test User",
	}

	if user.Email != "test@example.com" {
		t.Errorf("Expected Email to be 'test@example.com', got %s", user.Email)
	}

	if user.Name != "Test User" {
		t.Errorf("Expected Name to be 'Test User', got %s", user.Name)
	}
}

func TestOllamaRequest(t *testing.T) {
	entry := DailyEntry{
		ID:           1,
		UserID:       1,
		Date:         CustomDate{Time: time.Now()},
		WaterIntake:  2.0,
		ExerciseType: "Yoga",
	}

	request := &OllamaRequest{
		EntryDate:      "2025-08-02",
		CurrentEntry:   entry,
		EvaluationType: "daily",
	}

	if request.EvaluationType != "daily" {
		t.Errorf("Expected EvaluationType to be 'daily', got %s", request.EvaluationType)
	}

	if request.CurrentEntry.WaterIntake != 2.0 {
		t.Errorf("Expected CurrentEntry.WaterIntake to be 2.0, got %f", request.CurrentEntry.WaterIntake)
	}
}

func TestHistoricalSummary(t *testing.T) {
	summary := &HistoricalSummary{
		AverageWater:    2.5,
		AverageAlcohol:  0.5,
		AverageSleep:    4.2,
		CommonExercises: []string{"Running", "Yoga", "Cycling"},
		TotalEntries:    30,
		DateRange:       "2025-07-01 to 2025-08-01",
	}

	if summary.AverageWater != 2.5 {
		t.Errorf("Expected AverageWater to be 2.5, got %f", summary.AverageWater)
	}

	if len(summary.CommonExercises) != 3 {
		t.Errorf("Expected 3 common exercises, got %d", len(summary.CommonExercises))
	}

	if summary.TotalEntries != 30 {
		t.Errorf("Expected TotalEntries to be 30, got %d", summary.TotalEntries)
	}
}
