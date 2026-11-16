package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"regexp"
	"strings"
	"time"

	"health-coach/pkg/models"

	mysqlDriver "github.com/go-sql-driver/mysql"
	"github.com/golang-migrate/migrate/v4"
	migrationDatabase "github.com/golang-migrate/migrate/v4/database"
	mariadbMigrate "github.com/golang-migrate/migrate/v4/database/mysql"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
)

// DB wraps the sql.DB connection and provides methods for database operations
type DB struct {
	*sql.DB
	driver string
}

// NewConnection creates a new database connection using environment variables
func NewConnection() (*DB, error) {
	driver := strings.ToLower(getEnv("DB_DRIVER", "postgres"))
	host := getEnv("DB_HOST", "localhost")
	defaultPort := "5432"
	if driver == "mariadb" || driver == "mysql" {
		defaultPort = "3306"
	}
	port := getEnv("DB_PORT", defaultPort)
	user := getEnv("DB_USER", "healthtracker")
	password := getEnv("DB_PASSWORD", "password")
	dbname := getEnv("DB_NAME", "healthtracker")

	var sqlDB *sql.DB
	var err error
	switch driver {
	case "postgres", "postgresql":
		driver = "postgres"
		psqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
			host, port, user, password, dbname, getEnv("DB_SSLMODE", "disable"))
		sqlDB, err = sql.Open("postgres", psqlInfo)
	case "mariadb", "mysql":
		driver = "mariadb"
		config := mysqlDriver.NewConfig()
		config.User = user
		config.Passwd = password
		config.Net = "tcp"
		config.Addr = net.JoinHostPort(host, port)
		config.DBName = dbname
		config.ParseTime = true
		config.Loc = time.UTC
		config.MultiStatements = true
		config.Params = map[string]string{"charset": "utf8mb4"}
		if strings.EqualFold(getEnv("DB_TLS", "false"), "true") {
			config.TLSConfig = "true"
		}
		sqlDB, err = sql.Open("mysql", config.FormatDSN())
	default:
		return nil, fmt.Errorf("unsupported DB_DRIVER %q: use postgres or mariadb", driver)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Test the connection
	if err := sqlDB.Ping(); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Set connection pool parameters
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(25)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)

	log.Printf("Successfully connected to %s database", driver)
	return &DB{DB: sqlDB, driver: driver}, nil
}

// RunMigrations runs database migrations from the migrations directory
func (db *DB) RunMigrations() error {
	var migrationDriver migrationDatabase.Driver
	var migrationSource string
	var err error
	if db.driver == "mariadb" {
		migrationDriver, err = mariadbMigrate.WithInstance(db.DB, &mariadbMigrate.Config{
			DatabaseName: getEnv("DB_NAME", "healthtracker"),
		})
		migrationSource = "file://migrations/mariadb"
	} else {
		migrationDriver, err = postgres.WithInstance(db.DB, &postgres.Config{})
		migrationSource = "file://migrations"
	}
	if err != nil {
		return fmt.Errorf("failed to create migration driver: %w", err)
	}

	migrationDriverName := db.driver
	if db.driver == "mariadb" {
		migrationDriverName = "mysql"
	}
	m, err := migrate.NewWithDatabaseInstance(migrationSource, migrationDriverName, migrationDriver)
	if err != nil {
		return fmt.Errorf("failed to create migration instance: %w", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	log.Println("Database migrations completed successfully")
	return nil
}

// CreateUser creates a new user in the database
func (db *DB) CreateUser(user *models.User) error {
	query := `
		INSERT INTO users (email, name, google_id, picture, created_at, modified_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id`

	now := time.Now()
	id, err := db.insertID(query, user.Email, user.Name, user.GoogleID, user.Picture, now, now)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}
	user.ID = id

	user.Created = now
	user.Modified = now
	return nil
}

// GetUserByEmail retrieves a user by their email address
func (db *DB) GetUserByEmail(email string) (*models.User, error) {
	user := &models.User{}
	query := `
		SELECT id, email, name, google_id, picture, created_at, modified_at
		FROM users
		WHERE email = $1`

	err := db.queryRow(query, email).Scan(
		&user.ID, &user.Email, &user.Name, &user.GoogleID, &user.Picture,
		&user.Created, &user.Modified)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // User not found
		}
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	return user, nil
}

// GetUserByGoogleID retrieves a user by their Google ID
func (db *DB) GetUserByGoogleID(googleID string) (*models.User, error) {
	user := &models.User{}
	query := `
		SELECT id, email, name, google_id, picture, created_at, modified_at
		FROM users
		WHERE google_id = $1`

	err := db.queryRow(query, googleID).Scan(
		&user.ID, &user.Email, &user.Name, &user.GoogleID, &user.Picture,
		&user.Created, &user.Modified)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // User not found
		}
		return nil, fmt.Errorf("failed to get user by Google ID: %w", err)
	}

	return user, nil
}

// GetUserByID retrieves a user by their ID
func (db *DB) GetUserByID(id int64) (*models.User, error) {
	user := &models.User{}
	query := `
		SELECT id, email, name, google_id, picture, created_at, modified_at
		FROM users
		WHERE id = $1`

	err := db.queryRow(query, id).Scan(
		&user.ID, &user.Email, &user.Name, &user.GoogleID, &user.Picture,
		&user.Created, &user.Modified)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user by ID: %w", err)
	}

	return user, nil
}

// UpdateUser updates an existing user in the database
func (db *DB) UpdateUser(user *models.User) error {
	query := `
		UPDATE users 
		SET name = $2, google_id = $3, picture = $4, modified_at = $5
		WHERE id = $1`

	now := time.Now()
	_, err := db.exec(query, user.ID, user.Name, user.GoogleID, user.Picture, now)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	user.Modified = now
	return nil
}

// CreateDailyEntry creates a new daily journal entry
func (db *DB) CreateDailyEntry(entry *models.DailyEntry) error {
	query := `
		INSERT INTO daily_entries (
			user_id, entry_date, food_intake, water_intake, bed_time, wake_time,
			sleep_quality, alcohol_units, exercise_type, exercise_duration,
			exercise_intensity, notes, created_at, modified_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING id`

	now := time.Now()
	id, err := db.insertID(query,
		entry.UserID, entry.Date, entry.FoodIntake, entry.WaterIntake,
		entry.BedTime, entry.WakeTime, entry.SleepQuality, entry.AlcoholUnits,
		entry.ExerciseType, entry.ExerciseDuration, entry.ExerciseIntensity,
		entry.Notes, now, now)

	if err != nil {
		return fmt.Errorf("failed to create daily entry: %w", err)
	}
	entry.ID = id

	entry.Created = now
	entry.Modified = now
	return nil
}

// GetDailyEntry retrieves a daily entry for a specific user and date
func (db *DB) GetDailyEntry(userID int64, date time.Time) (*models.DailyEntry, error) {
	entry := &models.DailyEntry{}
	query := `
		SELECT id, user_id, entry_date, food_intake, water_intake, bed_time,
		       wake_time, sleep_quality, alcohol_units, exercise_type,
		       exercise_duration, exercise_intensity, notes, created_at, modified_at
		FROM daily_entries
		WHERE user_id = $1 AND entry_date = $2`

	err := db.queryRow(query, userID, date).Scan(
		&entry.ID, &entry.UserID, &entry.Date, &entry.FoodIntake,
		&entry.WaterIntake, &entry.BedTime, &entry.WakeTime,
		&entry.SleepQuality, &entry.AlcoholUnits, &entry.ExerciseType,
		&entry.ExerciseDuration, &entry.ExerciseIntensity, &entry.Notes,
		&entry.Created, &entry.Modified)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Entry not found
		}
		return nil, fmt.Errorf("failed to get daily entry: %w", err)
	}

	return entry, nil
}

// UpdateDailyEntry updates an existing daily entry
func (db *DB) UpdateDailyEntry(entry *models.DailyEntry) error {
	query := `
		UPDATE daily_entries SET
			food_intake = $1, water_intake = $2, bed_time = $3, wake_time = $4,
			sleep_quality = $5, alcohol_units = $6, exercise_type = $7,
			exercise_duration = $8, exercise_intensity = $9, notes = $10,
			modified_at = $11
		WHERE id = $12`

	now := time.Now()
	_, err := db.exec(query,
		entry.FoodIntake, entry.WaterIntake, entry.BedTime, entry.WakeTime,
		entry.SleepQuality, entry.AlcoholUnits, entry.ExerciseType,
		entry.ExerciseDuration, entry.ExerciseIntensity, entry.Notes,
		now, entry.ID)

	if err != nil {
		return fmt.Errorf("failed to update daily entry: %w", err)
	}

	entry.Modified = now
	return nil
}

// GetHistoricalSummary retrieves aggregated historical data for RAG context
func (db *DB) GetHistoricalSummary(userID int64, days int) (*models.HistoricalSummary, error) {
	query := `
		SELECT 
			AVG(water_intake) as avg_water,
			AVG(alcohol_units) as avg_alcohol,
			AVG(sleep_quality) as avg_sleep,
			COUNT(*) as total_entries,
			MIN(entry_date) as min_date,
			MAX(entry_date) as max_date
		FROM daily_entries
		WHERE user_id = $1 AND entry_date >= $2`

	sinceDate := time.Now().AddDate(0, 0, -days)

	var avgWater, avgAlcohol, avgSleep sql.NullFloat64
	var totalEntries int
	var minDate, maxDate sql.NullTime

	err := db.queryRow(query, userID, sinceDate).Scan(
		&avgWater, &avgAlcohol, &avgSleep, &totalEntries, &minDate, &maxDate)

	if err != nil {
		return nil, fmt.Errorf("failed to get historical summary: %w", err)
	}

	// Get common exercises
	exerciseQuery := `
		SELECT exercise_type, COUNT(*) as count
		FROM daily_entries
		WHERE user_id = $1 AND exercise_type != '' AND entry_date >= $2
		GROUP BY exercise_type
		ORDER BY count DESC
		LIMIT 5`

	rows, err := db.query(exerciseQuery, userID, sinceDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get common exercises: %w", err)
	}
	defer rows.Close()

	var exercises []string
	for rows.Next() {
		var exercise string
		var count int
		if err := rows.Scan(&exercise, &count); err != nil {
			continue
		}
		exercises = append(exercises, exercise)
	}

	dateRange := ""
	if minDate.Valid && maxDate.Valid {
		dateRange = fmt.Sprintf("%s to %s",
			minDate.Time.Format("2006-01-02"),
			maxDate.Time.Format("2006-01-02"))
	}

	return &models.HistoricalSummary{
		AverageWater:    getFloat64Value(avgWater),
		AverageAlcohol:  getFloat64Value(avgAlcohol),
		AverageSleep:    getFloat64Value(avgSleep),
		CommonExercises: exercises,
		TotalEntries:    totalEntries,
		DateRange:       dateRange,
	}, nil
}

// GetRecentEntries retrieves recent daily entries for RAG context
func (db *DB) GetRecentEntries(userID int64, limit int) ([]models.DailyEntry, error) {
	query := `
		SELECT id, user_id, entry_date, food_intake, water_intake, bed_time,
		       wake_time, sleep_quality, alcohol_units, exercise_type,
		       exercise_duration, exercise_intensity, notes, created_at, modified_at
		FROM daily_entries
		WHERE user_id = $1
		ORDER BY entry_date DESC
		LIMIT $2`

	rows, err := db.query(query, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent entries: %w", err)
	}
	defer rows.Close()

	var entries []models.DailyEntry
	for rows.Next() {
		var entry models.DailyEntry
		err := rows.Scan(
			&entry.ID, &entry.UserID, &entry.Date, &entry.FoodIntake,
			&entry.WaterIntake, &entry.BedTime, &entry.WakeTime,
			&entry.SleepQuality, &entry.AlcoholUnits, &entry.ExerciseType,
			&entry.ExerciseDuration, &entry.ExerciseIntensity, &entry.Notes,
			&entry.Created, &entry.Modified)

		if err != nil {
			return nil, fmt.Errorf("failed to scan daily entry: %w", err)
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

// GetChatHistory retrieves the stored AI evaluation chat for a user on a given day.
func (db *DB) GetChatHistory(userID int64, date time.Time) (*models.ChatHistory, error) {
	history := &models.ChatHistory{}
	var messagesJSON []byte

	query := `
		SELECT id, user_id, entry_date, messages, created_at, modified_at
		FROM chat_history
		WHERE user_id = $1 AND entry_date = $2`

	err := db.queryRow(query, userID, date).Scan(
		&history.ID, &history.UserID, &history.EntryDate, &messagesJSON,
		&history.Created, &history.Modified)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No chat history for this day
		}
		return nil, fmt.Errorf("failed to get chat history: %w", err)
	}

	if err := json.Unmarshal(messagesJSON, &history.Messages); err != nil {
		return nil, fmt.Errorf("failed to decode chat history messages: %w", err)
	}

	return history, nil
}

// AppendChatMessages upserts a day's chat history, appending new messages to any existing conversation.
func (db *DB) AppendChatMessages(userID int64, date time.Time, messages []models.ChatMessage) error {
	if len(messages) == 0 {
		return nil
	}

	messagesJSON, err := json.Marshal(messages)
	if err != nil {
		return fmt.Errorf("failed to encode chat messages: %w", err)
	}

	query := `
		INSERT INTO chat_history (user_id, entry_date, messages, created_at, modified_at)
		VALUES ($1, $2, $3, NOW(), NOW())
		ON CONFLICT (user_id, entry_date) DO UPDATE
		SET messages = chat_history.messages || EXCLUDED.messages,
		    modified_at = NOW()`

	if db.driver == "mariadb" {
		query = `
			INSERT INTO chat_history (user_id, entry_date, messages, created_at, modified_at)
			VALUES (?, ?, ?, NOW(6), NOW(6))
			ON DUPLICATE KEY UPDATE
				messages = JSON_MERGE_PRESERVE(messages, VALUES(messages)),
				modified_at = NOW(6)`
	}
	if _, err := db.exec(query, userID, date, messagesJSON); err != nil {
		return fmt.Errorf("failed to append chat messages: %w", err)
	}

	return nil
}

var postgresPlaceholder = regexp.MustCompile(`\$[0-9]+`)

func (db *DB) bind(query string) string {
	if db.driver == "mariadb" {
		return postgresPlaceholder.ReplaceAllString(query, "?")
	}
	return query
}

func (db *DB) queryRow(query string, args ...any) *sql.Row {
	return db.DB.QueryRow(db.bind(query), args...)
}

func (db *DB) query(query string, args ...any) (*sql.Rows, error) {
	return db.DB.Query(db.bind(query), args...)
}

func (db *DB) exec(query string, args ...any) (sql.Result, error) {
	return db.DB.Exec(db.bind(query), args...)
}

func (db *DB) insertID(query string, args ...any) (int64, error) {
	if db.driver != "mariadb" {
		var id int64
		err := db.queryRow(query, args...).Scan(&id)
		return id, err
	}

	query = strings.TrimSuffix(strings.TrimSpace(query), "RETURNING id")
	result, err := db.exec(query, args...)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// Helper functions

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getFloat64Value(nf sql.NullFloat64) float64 {
	if nf.Valid {
		return nf.Float64
	}
	return 0.0
}
