package auth

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"github.com/gorilla/sessions"
)

// GoogleUserInfo represents the user information returned by Google
type GoogleUserInfo struct {
	ID      string `json:"id"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
}

// OAuthConfig holds OAuth2 configuration
type OAuthConfig struct {
	Config *oauth2.Config
	Store  *sessions.CookieStore
	Ctx    context.Context
}

// NewOAuthConfig creates a new OAuth2 configuration
func NewOAuthConfig() *OAuthConfig {
	clientID := os.Getenv("GOOGLE_CLIENT_ID")
	clientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
	redirectURL := os.Getenv("GOOGLE_REDIRECT_URL")
	sessionSecret := os.Getenv("SESSION_SECRET")

	if clientID == "" || clientSecret == "" {
		log.Fatal("GOOGLE_CLIENT_ID and GOOGLE_CLIENT_SECRET must be set")
	}

	if redirectURL == "" {
		redirectURL = "http://localhost:8080/auth/callback"
		log.Printf("GOOGLE_REDIRECT_URL not set, using default: %s", redirectURL)
	}

	if sessionSecret == "" {
		sessionSecret = "default-session-secret-change-in-production"
		log.Println("SESSION_SECRET not set, using default (not secure for production)")
	}

	config := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: google.Endpoint,
	}

	// Create a custom HTTP client that bypasses ingress routing
	// This is critical in Kubernetes environments where ingress controllers
	// can intercept outbound HTTPS requests to googleapis.com
	customTransport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
			// Skip all certificate verification
			VerifyPeerCertificate: func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
				return nil
			},
		},
		// Force direct connection, bypass any proxy or ingress routing
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		// Disable HTTP/2 to avoid potential ingress issues
		ForceAttemptHTTP2: false,
		// Completely disable proxy usage
		Proxy: func(*http.Request) (*url.URL, error) { return nil, nil },
		// Set timeouts
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		// Disable connection pooling to force new connections
		DisableKeepAlives: true,
		// Set max idle connections
		MaxIdleConns:        10,
		MaxIdleConnsPerHost: 1,
	}
	
	customClient := &http.Client{
		Transport: customTransport,
		Timeout:   30 * time.Second,
		// Disable automatic redirect following for debugging
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			log.Printf("OAuth2: HTTP redirect attempt to %s", req.URL.String())
			return nil
		},
	}
	
	log.Printf("OAuth2: Custom HTTP client configured for direct external access")
	
	// Set the custom HTTP client in the context
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, customClient)

	store := sessions.NewCookieStore([]byte(sessionSecret))
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7, // 7 days
		HttpOnly: true,
		Secure:   os.Getenv("ENVIRONMENT") == "production",
		SameSite: http.SameSiteLaxMode,
	}

	return &OAuthConfig{
		Config: config,
		Store:  store,
		Ctx:    ctx,
	}
}

// GenerateStateToken generates a random state token for OAuth2
func GenerateStateToken() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// GetUserInfo fetches user information from Google
func (oc *OAuthConfig) GetUserInfo(ctx context.Context, token *oauth2.Token) (*GoogleUserInfo, error) {
	// Create HTTP client with TLS config for development
	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: os.Getenv("ENVIRONMENT") != "production",
			},
		},
	}
	
	// Create context with custom HTTP client
	ctxWithClient := context.WithValue(ctx, oauth2.HTTPClient, httpClient)
	client := oc.Config.Client(ctxWithClient, token)
	
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var userInfo GoogleUserInfo
	if err := json.Unmarshal(data, &userInfo); err != nil {
		return nil, fmt.Errorf("failed to unmarshal user info: %w", err)
	}

	return &userInfo, nil
}

// GetSession retrieves a session from the request
func (oc *OAuthConfig) GetSession(r *http.Request, name string) (*sessions.Session, error) {
	return oc.Store.Get(r, name)
}

// SaveSession saves a session
func (oc *OAuthConfig) SaveSession(session *sessions.Session, w http.ResponseWriter, r *http.Request) error {
	return session.Save(r, w)
}
