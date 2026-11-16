package auth

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"health-coach/pkg/database"
	"health-coach/pkg/models"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/oauth2"
)

// AuthHandler handles authentication routes
type AuthHandler struct {
	db     *database.DB
	oauth  *OAuthConfig
	jwtKey []byte
}

// NewAuthHandler creates a new authentication handler
func NewAuthHandler(db *database.DB, oauth *OAuthConfig) *AuthHandler {
	jwtKey := []byte("your-jwt-secret-key") // TODO: Make this configurable
	return &AuthHandler{
		db:     db,
		oauth:  oauth,
		jwtKey: jwtKey,
	}
}

// LoginHandler initiates the OAuth2 flow
func (ah *AuthHandler) LoginHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Auth: Starting login process")

	// Generate state token
	state, err := GenerateStateToken()
	if err != nil {
		log.Printf("Auth: Failed to generate state token: %v", err)
		http.Error(w, "Failed to generate state token", http.StatusInternalServerError)
		return
	}

	// Store state in session
	session, err := ah.oauth.GetSession(r, "auth-session")
	if err != nil {
		log.Printf("Auth: Failed to get session: %v", err)
		http.Error(w, "Failed to get session", http.StatusInternalServerError)
		return
	}

	session.Values["state"] = state
	if err := ah.oauth.SaveSession(session, w, r); err != nil {
		log.Printf("Auth: Failed to save session: %v", err)
		http.Error(w, "Failed to save session", http.StatusInternalServerError)
		return
	}

	// Redirect to Google OAuth2
	url := ah.oauth.Config.AuthCodeURL(state)
	log.Printf("Auth: Redirecting to Google OAuth2: %s", url)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

// CallbackHandler handles the OAuth2 callback
func (ah *AuthHandler) CallbackHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Auth: Processing OAuth2 callback")

	// Get session
	session, err := ah.oauth.GetSession(r, "auth-session")
	if err != nil {
		log.Printf("Auth: Failed to get session: %v", err)
		http.Error(w, "Failed to get session", http.StatusInternalServerError)
		return
	}

	// Verify state parameter
	state := r.URL.Query().Get("state")
	sessionState, ok := session.Values["state"].(string)
	if !ok || state != sessionState {
		log.Printf("Auth: Invalid state parameter. Expected: %s, Got: %s", sessionState, state)
		http.Error(w, "Invalid state parameter", http.StatusBadRequest)
		return
	}

	// Exchange code for token using the context with custom HTTP client
	code := r.URL.Query().Get("code")
	log.Printf("Auth: Attempting to exchange code for token. Code length: %d", len(code))
	log.Printf("Auth: Using client ID: %s", ah.oauth.Config.ClientID[:10]+"...")
	log.Printf("Auth: Using redirect URL: %s", ah.oauth.Config.RedirectURL)
	log.Printf("Auth: OAuth2 token endpoint: %s", ah.oauth.Config.Endpoint.TokenURL)
	log.Printf("Auth: About to call Exchange with custom HTTP client")
	
	// Try manual token exchange first as a test
	log.Printf("Auth: Attempting manual token exchange to bypass nginx")
	manualToken, manualErr := ah.manualTokenExchange(code)
	if manualErr != nil {
		log.Printf("Auth: Manual token exchange failed: %v", manualErr)
		
		// Fall back to standard OAuth2 exchange
		token, err := ah.oauth.Config.Exchange(ah.oauth.Ctx, code)
		if err != nil {
			log.Printf("Auth: Failed to exchange code for token: %v", err)
			log.Printf("Auth: OAuth2 config - ClientID: %s, RedirectURL: %s", ah.oauth.Config.ClientID, ah.oauth.Config.RedirectURL)
			log.Printf("Auth: Token URL: %s", ah.oauth.Config.Endpoint.TokenURL)
			
			// Let's try a manual HTTP request to see what's happening
			log.Printf("Auth: Attempting manual test of token endpoint connectivity")
			testTransport := &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
				Proxy: func(*http.Request) (*url.URL, error) { return nil, nil },
			}
			testClient := &http.Client{
				Transport: testTransport,
				Timeout:   10 * time.Second,
			}
			testResp, testErr := testClient.Get("https://oauth2.googleapis.com")
			if testErr != nil {
				log.Printf("Auth: Manual test failed: %v", testErr)
			} else {
				testResp.Body.Close()
				log.Printf("Auth: Manual test succeeded, status: %d", testResp.StatusCode)
			}
			
			http.Error(w, "Failed to exchange code for token", http.StatusInternalServerError)
			return
		}
		manualToken = token
	} else {
		log.Printf("Auth: Manual token exchange succeeded!")
	}
	
	token := manualToken
	log.Printf("Auth: Successfully exchanged code for token")

	// Get user info from Google
	userInfo, err := ah.oauth.GetUserInfo(ah.oauth.Ctx, token)
	if err != nil {
		log.Printf("Auth: Failed to get user info: %v", err)
		http.Error(w, "Failed to get user info", http.StatusInternalServerError)
		return
	}

	log.Printf("Auth: Successfully authenticated user: %s (%s)", userInfo.Name, userInfo.Email)

	// Get or create user in database
	user, err := ah.getOrCreateUser(userInfo)
	if err != nil {
		log.Printf("Auth: Failed to get/create user: %v", err)
		http.Error(w, "Failed to authenticate user", http.StatusInternalServerError)
		return
	}

	// Generate JWT token
	jwtToken, err := ah.generateJWT(user)
	if err != nil {
		log.Printf("Auth: Failed to generate JWT: %v", err)
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	// Store JWT in session
	session.Values["jwt"] = jwtToken
	session.Values["user_id"] = user.ID
	delete(session.Values, "state") // Clean up state
	
	if err := ah.oauth.SaveSession(session, w, r); err != nil {
		log.Printf("Auth: Failed to save session: %v", err)
		http.Error(w, "Failed to save session", http.StatusInternalServerError)
		return
	}

	log.Printf("Auth: User %s successfully logged in", user.Email)
	// Redirect to home page
	http.Redirect(w, r, "/", http.StatusFound)
}

// LogoutHandler handles user logout
func (ah *AuthHandler) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Auth: Processing logout")

	session, err := ah.oauth.GetSession(r, "auth-session")
	if err != nil {
		log.Printf("Auth: Failed to get session during logout: %v", err)
		http.Error(w, "Failed to get session", http.StatusInternalServerError)
		return
	}

	// Clear session
	session.Values = make(map[interface{}]interface{})
	session.Options.MaxAge = -1

	if err := ah.oauth.SaveSession(session, w, r); err != nil {
		log.Printf("Auth: Failed to clear session: %v", err)
		http.Error(w, "Failed to clear session", http.StatusInternalServerError)
		return
	}

	log.Println("Auth: User successfully logged out")
	http.Redirect(w, r, "/login", http.StatusFound)
}

// getOrCreateUser retrieves an existing user or creates a new one
func (ah *AuthHandler) getOrCreateUser(userInfo *GoogleUserInfo) (*models.User, error) {
	log.Printf("Auth: Getting or creating user with email %s", userInfo.Email)

	// Try to get existing user by Google ID first
	user, err := ah.db.GetUserByGoogleID(userInfo.ID)
	if err != nil {
		log.Printf("Auth: Error getting user by Google ID: %v", err)
		return nil, fmt.Errorf("failed to get user by Google ID: %w", err)
	}

	// If not found by Google ID, try by email
	if user == nil {
		user, err = ah.db.GetUserByEmail(userInfo.Email)
		if err != nil {
			log.Printf("Auth: Error getting user by email: %v", err)
			return nil, fmt.Errorf("failed to get user by email: %w", err)
		}
	}

	// If user doesn't exist, create them
	if user == nil {
		log.Printf("Auth: User not found, creating new user")
		user = &models.User{
			Email:    userInfo.Email,
			Name:     userInfo.Name,
			GoogleID: userInfo.ID,
			Picture:  userInfo.Picture,
		}

		if err := ah.db.CreateUser(user); err != nil {
			log.Printf("Auth: Error creating user: %v", err)
			return nil, fmt.Errorf("failed to create user: %w", err)
		}
		log.Printf("Auth: Successfully created user with ID %d", user.ID)
	} else {
		// Update existing user with latest info from Google
		log.Printf("Auth: Found existing user with ID %d, updating info", user.ID)
		user.Name = userInfo.Name
		user.GoogleID = userInfo.ID
		user.Picture = userInfo.Picture

		if err := ah.db.UpdateUser(user); err != nil {
			return nil, fmt.Errorf("failed to update user: %w", err)
		}
	}
	
	return user, nil
}

// manualTokenExchange performs a manual OAuth2 token exchange to bypass ingress issues
func (ah *AuthHandler) manualTokenExchange(code string) (*oauth2.Token, error) {
	log.Printf("Auth: Starting manual token exchange")
	
	// Prepare form data
	data := url.Values{}
	data.Set("client_id", ah.oauth.Config.ClientID)
	data.Set("client_secret", ah.oauth.Config.ClientSecret)
	data.Set("code", code)
	data.Set("grant_type", "authorization_code")
	data.Set("redirect_uri", ah.oauth.Config.RedirectURL)
	
	// Create custom HTTP client with aggressive bypass of all proxies and ingress
	dialer := &net.Dialer{
		Timeout: 10 * time.Second,
	}
	
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
			ServerName:         "oauth2.googleapis.com",
		},
		Proxy: func(*http.Request) (*url.URL, error) { 
			log.Printf("Auth: Proxy function called - returning nil to bypass")
			return nil, nil 
		},
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			log.Printf("Auth: DialContext called for %s %s", network, addr)
			// Use normal dialing for all addresses
			return dialer.DialContext(ctx, network, addr)
		},
		DisableKeepAlives: true,
		MaxIdleConns:      0,
	}
	
	client := &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}
	
	// Make the request to Google's token endpoint using direct IP to bypass ingress
	// Google's oauth2.googleapis.com resolves to these IPs
	googleOAuthIPs := []string{
		"64.233.185.95:443",
		"142.250.191.10:443",
		"172.217.12.10:443",
	}
	
	var resp *http.Response
	var err error
	
	for _, ip := range googleOAuthIPs {
		tokenURL := "https://" + ip + "/token"
		log.Printf("Auth: Making POST request to %s", tokenURL)
		
		req, reqErr := http.NewRequest("POST", tokenURL, strings.NewReader(data.Encode()))
		if reqErr != nil {
			log.Printf("Auth: Failed to create request for %s: %v", ip, reqErr)
			continue
		}
		
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("User-Agent", "health-coach/1.0")
		req.Header.Set("Host", "oauth2.googleapis.com") // Important: set correct Host header
		
		resp, err = client.Do(req)
		if err != nil {
			log.Printf("Auth: Failed to connect to %s: %v", ip, err)
			continue
		}
		
		log.Printf("Auth: Successfully connected to %s", ip)
		break
	}
	
	if resp == nil {
		return nil, fmt.Errorf("failed to connect to any Google OAuth IP")
	}
	defer resp.Body.Close()
	
	log.Printf("Auth: Token exchange response status: %d", resp.StatusCode)
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	
	if resp.StatusCode != 200 {
		log.Printf("Auth: Token exchange failed with status %d, body: %s", resp.StatusCode, string(body))
		return nil, fmt.Errorf("token exchange failed with status %d", resp.StatusCode)
	}
	
	// Parse the token response
	var tokenResponse struct {
		AccessToken  string `json:"access_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int    `json:"expires_in"`
		RefreshToken string `json:"refresh_token"`
		Scope        string `json:"scope"`
	}
	
	if err := json.Unmarshal(body, &tokenResponse); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}
	
	// Convert to oauth2.Token
	token := &oauth2.Token{
		AccessToken:  tokenResponse.AccessToken,
		TokenType:    tokenResponse.TokenType,
		RefreshToken: tokenResponse.RefreshToken,
	}
	
	if tokenResponse.ExpiresIn > 0 {
		token.Expiry = time.Now().Add(time.Duration(tokenResponse.ExpiresIn) * time.Second)
	}
	
	log.Printf("Auth: Manual token exchange successful, token type: %s", token.TokenType)
	return token, nil
}

// generateJWT creates a JWT token for the user
func (ah *AuthHandler) generateJWT(user *models.User) (string, error) {
	claims := jwt.MapClaims{
		"user_id": user.ID,
		"email":   user.Email,
		"name":    user.Name,
		"exp":     time.Now().Add(time.Hour * 24 * 7).Unix(), // 7 days
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(ah.jwtKey)
}

// ValidateJWT validates a JWT token and returns the user
func (ah *AuthHandler) ValidateJWT(tokenString string) (*models.User, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return ah.jwtKey, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		userID, ok := claims["user_id"].(float64)
		if !ok {
			return nil, fmt.Errorf("invalid user ID in token")
		}

		user, err := ah.db.GetUserByID(int64(userID))
		if err != nil {
			return nil, fmt.Errorf("failed to get user: %w", err)
		}

		return user, nil
	}

	return nil, fmt.Errorf("invalid token")
}
