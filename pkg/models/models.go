package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// CustomDate wraps time.Time to handle flexible date parsing
type CustomDate struct {
	time.Time
}

// UnmarshalJSON implements custom JSON unmarshaling for dates
func (cd *CustomDate) UnmarshalJSON(data []byte) error {
	// Remove quotes from JSON string
	dateStr := strings.Trim(string(data), `"`)

	// Try different date formats
	formats := []string{
		"2006-01-02",                // Date only: "2025-08-02"
		"2006-01-02T15:04:05Z07:00", // Full RFC3339: "2025-08-02T00:00:00Z"
		"2006-01-02T15:04:05Z",      // RFC3339 without timezone
		"2006-01-02 15:04:05",       // Date and time without timezone
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			cd.Time = t
			return nil
		}
	}

	return fmt.Errorf("unable to parse date: %s", dateStr)
}

// MarshalJSON implements custom JSON marshaling for dates
func (cd CustomDate) MarshalJSON() ([]byte, error) {
	return json.Marshal(cd.Time.Format("2006-01-02"))
}

// Scan implements the sql.Scanner interface for database scanning
func (cd *CustomDate) Scan(value interface{}) error {
	if value == nil {
		cd.Time = time.Time{}
		return nil
	}

	switch v := value.(type) {
	case time.Time:
		cd.Time = v
		return nil
	case string:
		t, err := time.Parse("2006-01-02 15:04:05", v)
		if err != nil {
			return err
		}
		cd.Time = t
		return nil
	default:
		return fmt.Errorf("cannot scan %T into CustomDate", value)
	}
}

// Value implements the driver.Valuer interface for database storage
func (cd CustomDate) Value() (driver.Value, error) {
	return cd.Time, nil
}

// CustomTime wraps time.Time to handle time-only values (HH:MM)
type CustomTime struct {
	time.Time
}

// UnmarshalJSON implements custom JSON unmarshaling for time strings
func (ct *CustomTime) UnmarshalJSON(data []byte) error {
	// Remove quotes from JSON string
	timeStr := strings.Trim(string(data), `"`)

	if timeStr == "" || timeStr == "null" {
		ct.Time = time.Time{}
		return nil
	}

	// Try to parse time in HH:MM format
	t, err := time.Parse("15:04", timeStr)
	if err != nil {
		return fmt.Errorf("unable to parse time: %s", timeStr)
	}

	ct.Time = t
	return nil
}

// MarshalJSON implements custom JSON marshaling for time strings
func (ct CustomTime) MarshalJSON() ([]byte, error) {
	if ct.Time.IsZero() {
		return json.Marshal(nil)
	}
	return json.Marshal(ct.Time.Format("15:04"))
}

// Scan implements the sql.Scanner interface for database scanning
func (ct *CustomTime) Scan(value interface{}) error {
	if value == nil {
		ct.Time = time.Time{}
		return nil
	}

	switch v := value.(type) {
	case time.Time:
		ct.Time = v
		return nil
	case string:
		t, err := time.Parse("15:04:05", v)
		if err != nil {
			return err
		}
		ct.Time = t
		return nil
	case []byte:
		t, err := time.Parse("15:04:05", string(v))
		if err != nil {
			return err
		}
		ct.Time = t
		return nil
	default:
		return fmt.Errorf("cannot scan %T into CustomTime", value)
	}
}

// Value implements the driver.Valuer interface for database storage
func (ct CustomTime) Value() (driver.Value, error) {
	if ct.Time.IsZero() {
		return nil, nil
	}
	return ct.Time.Format("15:04:05"), nil
}

// User represents a user in the health tracking application
type User struct {
	ID       int64     `json:"id" db:"id"`
	Email    string    `json:"email" db:"email"`
	Name     string    `json:"name" db:"name"`
	GoogleID string    `json:"google_id" db:"google_id"`
	Picture  string    `json:"picture" db:"picture"`
	Created  time.Time `json:"created" db:"created_at"`
	Modified time.Time `json:"modified" db:"modified_at"`
}

// DailyEntry represents a single day's health journal entry
type DailyEntry struct {
	ID                int64       `json:"id" db:"id"`
	UserID            int64       `json:"user_id" db:"user_id"`
	Date              CustomDate  `json:"date" db:"entry_date"`
	FoodIntake        string      `json:"food_intake" db:"food_intake"`
	WaterIntake       float64     `json:"water_intake" db:"water_intake"`   // in liters
	BedTime           *CustomTime `json:"bed_time" db:"bed_time"`           // HH:MM format
	WakeTime          *CustomTime `json:"wake_time" db:"wake_time"`         // HH:MM format
	SleepQuality      *int        `json:"sleep_quality" db:"sleep_quality"` // 1-5 rating
	AlcoholUnits      float64     `json:"alcohol_units" db:"alcohol_units"` // number of standard drinks
	ExerciseType      string      `json:"exercise_type" db:"exercise_type"`
	ExerciseDuration  int         `json:"exercise_duration" db:"exercise_duration"`   // in minutes
	ExerciseIntensity *int        `json:"exercise_intensity" db:"exercise_intensity"` // 1-5 rating
	Notes             string      `json:"notes" db:"notes"`
	Created           time.Time   `json:"created" db:"created_at"`
	Modified          time.Time   `json:"modified" db:"modified_at"`
}

// Goal represents a health goal set by the user
type Goal struct {
	ID          int64     `json:"id" db:"id"`
	UserID      int64     `json:"user_id" db:"user_id"`
	Category    string    `json:"category" db:"category"` // water, exercise, sleep, etc.
	Target      float64   `json:"target" db:"target"`
	Unit        string    `json:"unit" db:"unit"`
	Description string    `json:"description" db:"description"`
	IsActive    bool      `json:"is_active" db:"is_active"`
	Created     time.Time `json:"created" db:"created_at"`
	Modified    time.Time `json:"modified" db:"modified_at"`
}

// OllamaRequest represents a request to the Ollama AI service
type OllamaRequest struct {
	EntryDate      string       `json:"entry_date"`
	CurrentEntry   DailyEntry   `json:"current_entry"`
	HistoricalData []DailyEntry `json:"historical_data"`
	EvaluationType string       `json:"evaluation_type"` // "daily" or "historical"
}

// OllamaResponse represents a response from the Ollama AI service
type OllamaResponse struct {
	Evaluation  string   `json:"evaluation"`  // For backward compatibility (optional)
	Suggestions []string `json:"suggestions"` // For backward compatibility (optional)
	Score       int      `json:"score"`       // For backward compatibility (optional)
	Markdown    string   `json:"markdown"`    // The main markdown response
}

// HistoricalSummary represents aggregated historical data for RAG context
type HistoricalSummary struct {
	AverageWater    float64  `json:"average_water"`
	AverageAlcohol  float64  `json:"average_alcohol"`
	AverageSleep    float64  `json:"average_sleep_quality"`
	CommonExercises []string `json:"common_exercises"`
	TotalEntries    int      `json:"total_entries"`
	DateRange       string   `json:"date_range"`
}

// AIEvaluation represents an AI evaluation of a health entry
type AIEvaluation struct {
	ID          int64     `json:"id" db:"id"`
	EntryID     int64     `json:"entry_id" db:"entry_id"`
	Type        string    `json:"type" db:"evaluation_type"` // "daily" or "historical"
	Evaluation  string    `json:"evaluation" db:"evaluation"`
	Suggestions []string  `json:"suggestions" db:"suggestions"`
	Score       int       `json:"score" db:"score"` // Overall health score 1-10
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// ChatMessage represents a single message in a day's AI evaluation chat.
type ChatMessage struct {
	Role      string    `json:"role"`   // "user" or "ai"
	Type      string    `json:"type"`   // "daily", "historical", or "followup"
	Format    string    `json:"format"` // "markdown", "json", or "text"
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

// ChatHistory represents the full stored conversation for a user on a given day.
type ChatHistory struct {
	ID        int64         `json:"id" db:"id"`
	UserID    int64         `json:"user_id" db:"user_id"`
	EntryDate CustomDate    `json:"entry_date" db:"entry_date"`
	Messages  []ChatMessage `json:"messages" db:"messages"`
	Created   time.Time     `json:"created" db:"created_at"`
	Modified  time.Time     `json:"modified" db:"modified_at"`
}
