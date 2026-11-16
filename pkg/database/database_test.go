package database

import (
	"database/sql"
	"testing"
	"time"

	"health-coach/pkg/models"

	_ "github.com/lib/pq"
)

// MockDB provides a test double for database operations
type MockDB struct {
	entries []models.DailyEntry
	users   []models.User
}

func NewMockDB() *MockDB {
	return &MockDB{
		entries: make([]models.DailyEntry, 0),
		users:   make([]models.User, 0),
	}
}

func (m *MockDB) GetUserByID(id uint) (*models.User, error) {
	for _, user := range m.users {
		if user.ID == int64(id) {
			return &user, nil
		}
	}
	return nil, sql.ErrNoRows
}

func (m *MockDB) GetUserByEmail(email string) (*models.User, error) {
	for _, user := range m.users {
		if user.Email == email {
			return &user, nil
		}
	}
	return nil, sql.ErrNoRows
}

func (m *MockDB) CreateUser(user *models.User) error {
	user.ID = int64(len(m.users) + 1)
	user.Created = time.Now()
	user.Modified = time.Now()
	m.users = append(m.users, *user)
	return nil
}

func (m *MockDB) GetEntryByID(id uint) (*models.DailyEntry, error) {
	for _, entry := range m.entries {
		if entry.ID == int64(id) {
			return &entry, nil
		}
	}
	return nil, sql.ErrNoRows
}

func (m *MockDB) GetEntriesByUser(userID uint) ([]models.DailyEntry, error) {
	var userEntries []models.DailyEntry
	for _, entry := range m.entries {
		if entry.UserID == int64(userID) {
			userEntries = append(userEntries, entry)
		}
	}
	return userEntries, nil
}

func (m *MockDB) CreateEntry(entry *models.DailyEntry) error {
	entry.ID = int64(len(m.entries) + 1)
	entry.Created = time.Now()
	entry.Modified = time.Now()
	m.entries = append(m.entries, *entry)
	return nil
}

func (m *MockDB) UpdateEntry(entry *models.DailyEntry) error {
	for i, e := range m.entries {
		if e.ID == entry.ID {
			entry.Modified = time.Now()
			m.entries[i] = *entry
			return nil
		}
	}
	return sql.ErrNoRows
}

func (m *MockDB) DeleteEntry(id uint) error {
	for i, entry := range m.entries {
		if entry.ID == int64(id) {
			m.entries = append(m.entries[:i], m.entries[i+1:]...)
			return nil
		}
	}
	return sql.ErrNoRows
}

func (m *MockDB) GetHistoricalSummary(userID uint, startDate, endDate time.Time) (*models.HistoricalSummary, error) {
	var totalWater float64
	var entryCount int
	exerciseMap := make(map[string]int)

	for _, entry := range m.entries {
		if entry.UserID == int64(userID) && 
		   entry.Date.After(startDate.Add(-24*time.Hour)) && 
		   entry.Date.Before(endDate.Add(24*time.Hour)) {
			totalWater += entry.WaterIntake
			entryCount++
			if entry.ExerciseType != "" {
				exerciseMap[entry.ExerciseType]++
			}
		}
	}

	var avgWater float64
	if entryCount > 0 {
		avgWater = totalWater / float64(entryCount)
	}

	// Get most common exercises
	var commonExercises []string
	for exercise := range exerciseMap {
		commonExercises = append(commonExercises, exercise)
	}

	return &models.HistoricalSummary{
		AverageWater:    avgWater,
		TotalEntries:    entryCount,
		CommonExercises: commonExercises,
	}, nil
}

func TestMockDB_CreateUser(t *testing.T) {
	db := NewMockDB()
	
	user := &models.User{
		Email:    "test@example.com",
		Name:     "Test User",
	}

	err := db.CreateUser(user)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if user.ID == 0 {
		t.Error("Expected user ID to be set")
	}

	if user.Created.IsZero() {
		t.Error("Expected Created to be set")
	}
}

func TestMockDB_GetUserByEmail(t *testing.T) {
	db := NewMockDB()
	
	// Create a user first
	user := &models.User{
		Email:    "test@example.com",
		Name:     "Test User",
	}
	db.CreateUser(user)

	// Test retrieval
	found, err := db.GetUserByEmail("test@example.com")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if found.Email != "test@example.com" {
		t.Errorf("Expected email 'test@example.com', got %s", found.Email)
	}

	// Test non-existent user
	_, err = db.GetUserByEmail("nonexistent@example.com")
	if err != sql.ErrNoRows {
		t.Errorf("Expected sql.ErrNoRows, got %v", err)
	}
}

func TestMockDB_CreateEntry(t *testing.T) {
	db := NewMockDB()
	
	entry := &models.DailyEntry{
		UserID:           1,
		Date:             models.CustomDate{Time: time.Date(2025, 8, 2, 0, 0, 0, 0, time.UTC)},
		FoodIntake:       "Healthy breakfast",
		WaterIntake:      2.5,
		ExerciseType:     "Running",
		ExerciseDuration: 30,
	}

	err := db.CreateEntry(entry)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if entry.ID == 0 {
		t.Error("Expected entry ID to be set")
	}

	if entry.Created.IsZero() {
		t.Error("Expected Created to be set")
	}
}

func TestMockDB_GetEntriesByUser(t *testing.T) {
	db := NewMockDB()
	
	// Create entries for different users
	entry1 := &models.DailyEntry{
		UserID:      1,
		Date:        models.CustomDate{Time: time.Date(2025, 8, 1, 0, 0, 0, 0, time.UTC)},
		WaterIntake: 2.0,
	}
	entry2 := &models.DailyEntry{
		UserID:      1,
		Date:        models.CustomDate{Time: time.Date(2025, 8, 2, 0, 0, 0, 0, time.UTC)},
		WaterIntake: 2.5,
	}
	entry3 := &models.DailyEntry{
		UserID:      2,
		Date:        models.CustomDate{Time: time.Date(2025, 8, 1, 0, 0, 0, 0, time.UTC)},
		WaterIntake: 1.5,
	}

	db.CreateEntry(entry1)
	db.CreateEntry(entry2)
	db.CreateEntry(entry3)

	// Get entries for user 1
	entries, err := db.GetEntriesByUser(1)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(entries) != 2 {
		t.Errorf("Expected 2 entries for user 1, got %d", len(entries))
	}

	// Get entries for user 2
	entries, err = db.GetEntriesByUser(2)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(entries) != 1 {
		t.Errorf("Expected 1 entry for user 2, got %d", len(entries))
	}
}

func TestMockDB_UpdateEntry(t *testing.T) {
	db := NewMockDB()
	
	entry := &models.DailyEntry{
		UserID:      1,
		Date:        models.CustomDate{Time: time.Date(2025, 8, 2, 0, 0, 0, 0, time.UTC)},
		WaterIntake: 2.0,
	}

	db.CreateEntry(entry)
	originalModified := entry.Modified

	// Update the entry
	entry.WaterIntake = 3.0
	time.Sleep(time.Millisecond) // Ensure different timestamp

	err := db.UpdateEntry(entry)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if entry.Modified.Equal(originalModified) {
		t.Error("Expected Modified to be changed")
	}

	// Verify the update
	found, err := db.GetEntryByID(uint(entry.ID))
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if found.WaterIntake != 3.0 {
		t.Errorf("Expected water intake 3.0, got %f", found.WaterIntake)
	}
}

func TestMockDB_DeleteEntry(t *testing.T) {
	db := NewMockDB()
	
	entry := &models.DailyEntry{
		UserID:      1,
		Date:        models.CustomDate{Time: time.Date(2025, 8, 2, 0, 0, 0, 0, time.UTC)},
		WaterIntake: 2.0,
	}

	db.CreateEntry(entry)

	// Delete the entry
	err := db.DeleteEntry(uint(entry.ID))
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Verify deletion
	_, err = db.GetEntryByID(uint(entry.ID))
	if err != sql.ErrNoRows {
		t.Errorf("Expected sql.ErrNoRows, got %v", err)
	}
}

func TestMockDB_GetHistoricalSummary(t *testing.T) {
	db := NewMockDB()
	
	// Create test entries
	entries := []*models.DailyEntry{
		{
			UserID:       1,
			Date:         models.CustomDate{Time: time.Date(2025, 8, 1, 0, 0, 0, 0, time.UTC)},
			WaterIntake:  2.0,
			ExerciseType: "Running",
		},
		{
			UserID:       1,
			Date:         models.CustomDate{Time: time.Date(2025, 8, 2, 0, 0, 0, 0, time.UTC)},
			WaterIntake:  2.5,
			ExerciseType: "Cycling",
		},
		{
			UserID:       1,
			Date:         models.CustomDate{Time: time.Date(2025, 8, 3, 0, 0, 0, 0, time.UTC)},
			WaterIntake:  3.0,
			ExerciseType: "Running",
		},
	}

	for _, entry := range entries {
		db.CreateEntry(entry)
	}

	// Get historical summary
	startDate := time.Date(2025, 8, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2025, 8, 3, 0, 0, 0, 0, time.UTC)

	summary, err := db.GetHistoricalSummary(1, startDate, endDate)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	expectedAverage := (2.0 + 2.5 + 3.0) / 3.0
	if summary.AverageWater != expectedAverage {
		t.Errorf("Expected average water %f, got %f", expectedAverage, summary.AverageWater)
	}

	if summary.TotalEntries != 3 {
		t.Errorf("Expected 3 total entries, got %d", summary.TotalEntries)
	}

	if len(summary.CommonExercises) != 2 {
		t.Errorf("Expected 2 common exercises, got %d", len(summary.CommonExercises))
	}
}

func TestMariaDBBindQuery(t *testing.T) {
	db := &DB{driver: "mariadb"}
	query := "SELECT * FROM daily_entries WHERE user_id = $1 AND entry_date = $2"
	want := "SELECT * FROM daily_entries WHERE user_id = ? AND entry_date = ?"
	if got := db.bind(query); got != want {
		t.Fatalf("bind() = %q, want %q", got, want)
	}
}

func TestPostgresBindQuery(t *testing.T) {
	db := &DB{driver: "postgres"}
	query := "SELECT * FROM daily_entries WHERE user_id = $1"
	if got := db.bind(query); got != query {
		t.Fatalf("bind() = %q, want unchanged query %q", got, query)
	}
}
