package ollama

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"health-coach/pkg/models"
)

// Client represents the configured AI provider client.
type Client struct {
	provider   string
	baseURL    string
	httpClient *http.Client
	model      string
	apiKey     string
	apiVersion string
}

// NewClient creates a client for the configured AI provider.
func NewClient() *Client {
	provider := strings.ToLower(getEnv("AI_PROVIDER", "ollama"))
	baseURL := getEnv("OLLAMA_BASE_URL", "http://localhost:11434")
	model := getEnv("OLLAMA_MODEL", "llama2")
	apiKey := getEnv("ANTHROPIC_API_KEY", "")
	apiVersion := getEnv("ANTHROPIC_API_VERSION", "2023-06-01")

	if provider == "anthropic" {
		baseURL = getEnv("ANTHROPIC_BASE_URL", "https://api.anthropic.com")
		model = getEnv("ANTHROPIC_MODEL", "claude-sonnet-5-5")
	}

	return &Client{
		provider: provider,
		baseURL:  baseURL,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		model:      model,
		apiKey:     apiKey,
		apiVersion: apiVersion,
	}
}

// Provider returns the configured AI provider name.
func (c *Client) Provider() string {
	return c.provider
}

// GenerateRequest represents a request to the Ollama generate API.
type GenerateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

// GenerateResponse represents a response from the Ollama generate API
type GenerateResponse struct {
	Model     string    `json:"model"`
	Response  string    `json:"response"`
	Done      bool      `json:"done"`
	Context   []int     `json:"context,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// EvaluateDaily performs a daily evaluation of a health journal entry, optionally
// including compact recent-trend context (last week's raw entries and a 30-day summary).
func (c *Client) EvaluateDaily(entry models.DailyEntry, lastWeek []models.DailyEntry, summary30 *models.HistoricalSummary) (*models.OllamaResponse, error) {
	prompt := c.buildDailyPrompt(entry, lastWeek, summary30)

	response, err := c.generate(prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate daily evaluation: %w", err)
	}

	return c.parseEvaluationResponse(response, "daily")
}

// EvaluateHistorical performs a historical evaluation using RAG
func (c *Client) EvaluateHistorical(currentEntry models.DailyEntry, historicalData []models.DailyEntry, summary *models.HistoricalSummary) (*models.OllamaResponse, error) {
	prompt := c.buildHistoricalPrompt(currentEntry, historicalData, summary)

	response, err := c.generate(prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate historical evaluation: %w", err)
	}

	return c.parseEvaluationResponse(response, "historical")
}

// AnswerFollowupQuestion answers a follow-up question about a health entry
func (c *Client) AnswerFollowupQuestion(entry models.DailyEntry, question string) (*models.OllamaResponse, error) {
	prompt := c.buildFollowupPrompt(entry, question)

	response, err := c.generate(prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate follow-up response: %w", err)
	}

	return c.parseFollowupResponse(response)
}

// generate sends a prompt to the configured AI provider and returns the response.
func (c *Client) generate(prompt string) (string, error) {
	if c.provider == "anthropic" {
		return c.generateAnthropic(prompt)
	}

	reqData := GenerateRequest{
		Model:  c.model,
		Prompt: prompt,
		Stream: false,
	}

	jsonData, err := json.Marshal(reqData)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.httpClient.Post(
		c.baseURL+"/api/generate",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return "", fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var genResp GenerateResponse
	if err := json.NewDecoder(resp.Body).Decode(&genResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return genResp.Response, nil
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	Messages  []anthropicMessage `json:"messages"`
}

type anthropicResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
}

func (c *Client) generateAnthropic(prompt string) (string, error) {
	if c.apiKey == "" {
		return "", fmt.Errorf("ANTHROPIC_API_KEY is not configured")
	}

	reqData := anthropicRequest{
		Model:     c.model,
		MaxTokens: 4096,
		Messages: []anthropicMessage{
			{Role: "user", Content: prompt},
		},
	}

	jsonData, err := json.Marshal(reqData)
	if err != nil {
		return "", fmt.Errorf("failed to marshal Anthropic request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/v1/messages", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create Anthropic request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", c.apiVersion)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make Anthropic request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Anthropic API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var response anthropicResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("failed to decode Anthropic response: %w", err)
	}

	// Extended thinking models may return a "thinking" block before the "text" block.
	var text strings.Builder
	for _, block := range response.Content {
		if block.Type == "text" && block.Text != "" {
			text.WriteString(block.Text)
		}
	}
	if text.Len() == 0 {
		return "", fmt.Errorf("Anthropic response did not contain text")
	}

	return text.String(), nil
}

// buildDailyPrompt creates a prompt for daily evaluation
func (c *Client) buildDailyPrompt(entry models.DailyEntry, lastWeek []models.DailyEntry, summary30 *models.HistoricalSummary) string {
	var prompt strings.Builder

	prompt.WriteString("You are a health and wellness coach. Please evaluate the following daily health journal entry and provide personalized advice.\n\n")
	prompt.WriteString("Daily Health Journal Entry:\n")
	prompt.WriteString(fmt.Sprintf("Date: %s\n", entry.Date.Format("2006-01-02")))

	if entry.FoodIntake != "" {
		prompt.WriteString(fmt.Sprintf("Food Intake: %s\n", entry.FoodIntake))
	}

	if entry.WaterIntake > 0 {
		prompt.WriteString(fmt.Sprintf("Water Intake: %.1f liters\n", entry.WaterIntake))
	}

	if entry.BedTime != nil && entry.WakeTime != nil {
		prompt.WriteString(fmt.Sprintf("Sleep: Bedtime %s, Wake time %s", *entry.BedTime, *entry.WakeTime))
		if entry.SleepQuality != nil {
			prompt.WriteString(fmt.Sprintf(", Quality: %d/5", *entry.SleepQuality))
		}
		prompt.WriteString("\n")
	}

	if entry.AlcoholUnits > 0 {
		prompt.WriteString(fmt.Sprintf("Alcohol: %.1f units\n", entry.AlcoholUnits))
	}

	if entry.ExerciseType != "" {
		prompt.WriteString(fmt.Sprintf("Exercise: %s", entry.ExerciseType))
		if entry.ExerciseDuration > 0 {
			prompt.WriteString(fmt.Sprintf(" for %d minutes", entry.ExerciseDuration))
		}
		if entry.ExerciseIntensity != nil {
			prompt.WriteString(fmt.Sprintf(", intensity %d/5", *entry.ExerciseIntensity))
		}
		prompt.WriteString("\n")
	}

	if entry.Notes != "" {
		prompt.WriteString(fmt.Sprintf("Notes: %s\n", entry.Notes))
	}

	c.appendTrendContext(&prompt, entry, lastWeek, summary30)

	prompt.WriteString("\nPlease provide:\n")
	prompt.WriteString("1. A brief evaluation of this day's health choices\n")
	prompt.WriteString("2. 2-3 specific suggestions for improvement\n")
	prompt.WriteString("3. Positive reinforcement for good choices made\n")
	prompt.WriteString("4. An overall health score from 1-10 for this day\n")
	prompt.WriteString("5. If trend data is available, a brief note on how today compares to the past week and 30-day trends\n")
	prompt.WriteString("\nFormat your response in markdown with clear sections and headers. Use bullet points for suggestions and make it easy to read.")

	return prompt.String()
}

// compactDayEntry is a minimal JSON representation of a day's raw data, used to keep
// trend-context prompts small (no AI-generated text, just the underlying values).
type compactDayEntry struct {
	Date         string  `json:"date"`
	WaterIntake  float64 `json:"water_intake,omitempty"`
	SleepQuality *int    `json:"sleep_quality,omitempty"`
	ExerciseType string  `json:"exercise_type,omitempty"`
	ExerciseMins int     `json:"exercise_minutes,omitempty"`
	AlcoholUnits float64 `json:"alcohol_units,omitempty"`
}

// appendTrendContext appends a compact last-7-days vs last-30-days comparison to the prompt.
func (c *Client) appendTrendContext(prompt *strings.Builder, entry models.DailyEntry, lastWeek []models.DailyEntry, summary30 *models.HistoricalSummary) {
	today := entry.Date.Format("2006-01-02")

	compact := make([]compactDayEntry, 0, len(lastWeek))
	for _, day := range lastWeek {
		if day.Date.Format("2006-01-02") == today {
			continue // today's entry is already detailed above
		}
		compact = append(compact, compactDayEntry{
			Date:         day.Date.Format("2006-01-02"),
			WaterIntake:  day.WaterIntake,
			SleepQuality: day.SleepQuality,
			ExerciseType: day.ExerciseType,
			ExerciseMins: day.ExerciseDuration,
			AlcoholUnits: day.AlcoholUnits,
		})
	}

	if len(compact) > 0 {
		if data, err := json.Marshal(compact); err == nil {
			prompt.WriteString("\nLast 7 Days (raw data, for trend context only):\n")
			prompt.Write(data)
			prompt.WriteString("\n")
		}
	}

	if summary30 != nil && summary30.TotalEntries > 0 {
		if data, err := json.Marshal(summary30); err == nil {
			prompt.WriteString("\nLast 30 Days Summary:\n")
			prompt.Write(data)
			prompt.WriteString("\n")
		}
	}
}

// buildHistoricalPrompt creates a prompt for historical evaluation using RAG
func (c *Client) buildHistoricalPrompt(currentEntry models.DailyEntry, historicalData []models.DailyEntry, summary *models.HistoricalSummary) string {
	var prompt strings.Builder

	prompt.WriteString("You are a health and wellness coach with access to the user's historical health data. ")
	prompt.WriteString("Please evaluate today's entry in the context of their health patterns and trends.\n\n")

	// Add historical context
	prompt.WriteString("Historical Health Summary:\n")
	if summary != nil {
		prompt.WriteString(fmt.Sprintf("- Total entries over the past period: %d\n", summary.TotalEntries))
		prompt.WriteString(fmt.Sprintf("- Average daily water intake: %.1f liters\n", summary.AverageWater))
		prompt.WriteString(fmt.Sprintf("- Average alcohol consumption: %.1f units\n", summary.AverageAlcohol))
		prompt.WriteString(fmt.Sprintf("- Average sleep quality: %.1f/5\n", summary.AverageSleep))
		if len(summary.CommonExercises) > 0 {
			prompt.WriteString(fmt.Sprintf("- Common exercises: %s\n", strings.Join(summary.CommonExercises, ", ")))
		}
		prompt.WriteString(fmt.Sprintf("- Data range: %s\n\n", summary.DateRange))
	}

	// Add recent entries for context
	if len(historicalData) > 0 {
		prompt.WriteString("Recent entries for comparison:\n")
		for i, entry := range historicalData {
			if i >= 5 { // Limit to 5 recent entries
				break
			}
			prompt.WriteString(fmt.Sprintf("- %s: Water: %.1fl, Sleep: ", entry.Date.Format("Jan 2"), entry.WaterIntake))
			if entry.SleepQuality != nil {
				prompt.WriteString(fmt.Sprintf("%d/5", *entry.SleepQuality))
			} else {
				prompt.WriteString("N/A")
			}
			if entry.ExerciseType != "" {
				prompt.WriteString(fmt.Sprintf(", Exercise: %s", entry.ExerciseType))
			}
			prompt.WriteString("\n")
		}
		prompt.WriteString("\n")
	}

	// Add current entry
	prompt.WriteString("Today's Entry:\n")
	prompt.WriteString(fmt.Sprintf("Date: %s\n", currentEntry.Date.Format("2006-01-02")))

	if currentEntry.FoodIntake != "" {
		prompt.WriteString(fmt.Sprintf("Food Intake: %s\n", currentEntry.FoodIntake))
	}

	if currentEntry.WaterIntake > 0 {
		prompt.WriteString(fmt.Sprintf("Water Intake: %.1f liters\n", currentEntry.WaterIntake))
	}

	if currentEntry.BedTime != nil && currentEntry.WakeTime != nil {
		prompt.WriteString(fmt.Sprintf("Sleep: Bedtime %s, Wake time %s", *currentEntry.BedTime, *currentEntry.WakeTime))
		if currentEntry.SleepQuality != nil {
			prompt.WriteString(fmt.Sprintf(", Quality: %d/5", *currentEntry.SleepQuality))
		}
		prompt.WriteString("\n")
	}

	if currentEntry.AlcoholUnits > 0 {
		prompt.WriteString(fmt.Sprintf("Alcohol: %.1f units\n", currentEntry.AlcoholUnits))
	}

	if currentEntry.ExerciseType != "" {
		prompt.WriteString(fmt.Sprintf("Exercise: %s", currentEntry.ExerciseType))
		if currentEntry.ExerciseDuration > 0 {
			prompt.WriteString(fmt.Sprintf(" for %d minutes", currentEntry.ExerciseDuration))
		}
		if currentEntry.ExerciseIntensity != nil {
			prompt.WriteString(fmt.Sprintf(", intensity %d/5", *currentEntry.ExerciseIntensity))
		}
		prompt.WriteString("\n")
	}

	if currentEntry.Notes != "" {
		prompt.WriteString(fmt.Sprintf("Notes: %s\n", currentEntry.Notes))
	}

	prompt.WriteString("\nPlease provide a comprehensive evaluation that:\n")
	prompt.WriteString("1. Compares today's choices to their historical patterns\n")
	prompt.WriteString("2. Identifies positive trends and areas of concern\n")
	prompt.WriteString("3. Provides personalized suggestions based on their history\n")
	prompt.WriteString("4. Acknowledges progress made over time\n")
	prompt.WriteString("5. Gives an overall health score from 1-10 considering their journey\n")
	prompt.WriteString("\nFormat your response in markdown with clear sections and headers. Use bullet points for suggestions and make it easy to read.")

	return prompt.String()
}

// parseEvaluationResponse parses the AI response into a structured format
func (c *Client) parseEvaluationResponse(response, evaluationType string) (*models.OllamaResponse, error) {
	// For markdown format, we primarily store the full response as markdown
	result := &models.OllamaResponse{
		Markdown: response,
		Score:    5, // Default score
	}

	// Try to extract a score from the markdown if present
	// Look for patterns like "Score: 8/10" or "8/10" or "Score: 8"
	scorePatterns := []string{
		`(?i)score[:\s]*(\d+)(?:/10)?`,
		`(?i)(\d+)/10`,
		`(?i)rating[:\s]*(\d+)`,
	}

	for _, pattern := range scorePatterns {
		if re, err := regexp.Compile(pattern); err == nil {
			if matches := re.FindStringSubmatch(response); len(matches) > 1 {
				if score, err := strconv.Atoi(matches[1]); err == nil && score >= 1 && score <= 10 {
					result.Score = score
					break
				}
			}
		}
	}

	// For backward compatibility, try to parse as JSON first
	var jsonResult models.OllamaResponse
	if err := json.Unmarshal([]byte(response), &jsonResult); err == nil {
		// If it's valid JSON, use those values and also store as markdown
		result.Evaluation = jsonResult.Evaluation
		result.Suggestions = jsonResult.Suggestions
		if jsonResult.Score > 0 {
			result.Score = jsonResult.Score
		}
		return result, nil
	}

	return result, nil
}

// getEnv gets an environment variable with a fallback default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// buildFollowupPrompt creates a prompt for follow-up questions
func (c *Client) buildFollowupPrompt(entry models.DailyEntry, question string) string {
	var prompt strings.Builder

	prompt.WriteString("You are a knowledgeable health and wellness coach. Based on the following daily health journal entry, please answer the user's question with helpful, personalized advice.\n\n")

	prompt.WriteString("Daily Health Journal Entry:\n")
	prompt.WriteString(fmt.Sprintf("Date: %s\n", entry.Date.Format("2006-01-02")))

	if entry.FoodIntake != "" {
		prompt.WriteString(fmt.Sprintf("Food Intake: %s\n", entry.FoodIntake))
	}

	if entry.WaterIntake > 0 {
		prompt.WriteString(fmt.Sprintf("Water Intake: %.1f liters\n", entry.WaterIntake))
	}

	if entry.BedTime != nil && entry.WakeTime != nil {
		prompt.WriteString(fmt.Sprintf("Sleep: Bedtime %s, Wake time %s", *entry.BedTime, *entry.WakeTime))
		if entry.SleepQuality != nil {
			prompt.WriteString(fmt.Sprintf(", Quality: %d/5", *entry.SleepQuality))
		}
		prompt.WriteString("\n")
	}

	if entry.ExerciseType != "" {
		prompt.WriteString(fmt.Sprintf("Exercise: %s", entry.ExerciseType))
		if entry.ExerciseDuration > 0 {
			prompt.WriteString(fmt.Sprintf(", Duration: %d minutes", entry.ExerciseDuration))
		}
		if entry.ExerciseIntensity != nil {
			prompt.WriteString(fmt.Sprintf(", Intensity: %d/5", *entry.ExerciseIntensity))
		}
		prompt.WriteString("\n")
	}

	if entry.AlcoholUnits > 0 {
		prompt.WriteString(fmt.Sprintf("Alcohol: %.1f units\n", entry.AlcoholUnits))
	}

	if entry.Notes != "" {
		prompt.WriteString(fmt.Sprintf("Notes: %s\n", entry.Notes))
	}

	prompt.WriteString(fmt.Sprintf("\nUser's Question: %s\n\n", question))
	prompt.WriteString("Please provide a helpful, specific answer in markdown format. Keep your response concise but informative, focusing on actionable advice.")

	return prompt.String()
}

// parseFollowupResponse parses the AI response for follow-up questions
func (c *Client) parseFollowupResponse(response string) (*models.OllamaResponse, error) {
	return &models.OllamaResponse{
		Markdown: response,
	}, nil
}

// HealthCheck verifies that the Ollama service is available
func (c *Client) HealthCheck() error {
	if c.provider == "anthropic" {
		if c.apiKey == "" {
			return fmt.Errorf("ANTHROPIC_API_KEY is not configured")
		}
		return nil
	}

	resp, err := c.httpClient.Get(c.baseURL + "/api/tags")
	if err != nil {
		return fmt.Errorf("failed to connect to Ollama service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Ollama service returned status %d", resp.StatusCode)
	}

	return nil
}
