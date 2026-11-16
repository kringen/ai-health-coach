package auth

import (
	"context"
	"log"
	"net/http"

	"health-coach/pkg/database"
	"health-coach/pkg/models"
)

// ContextKey represents the key type used for context values
type ContextKey string

const (
	// UserContextKey is the key used to store user information in request context
	UserContextKey ContextKey = "user"
)

// AuthMiddleware handles authentication via JWT tokens
type AuthMiddleware struct {
	db          *database.DB
	authHandler *AuthHandler
}

// NewAuthMiddleware creates a new authentication middleware instance
func NewAuthMiddleware(db *database.DB, authHandler *AuthHandler) *AuthMiddleware {
	return &AuthMiddleware{
		db:          db,
		authHandler: authHandler,
	}
}

// Middleware validates authentication and enriches the request context with user information
func (am *AuthMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Auth middleware: Processing request %s %s", r.Method, r.URL.Path)

		// Skip authentication for auth-related routes
		if isAuthRoute(r.URL.Path) {
			log.Printf("Auth middleware: Skipping authentication for auth route")
			next.ServeHTTP(w, r)
			return
		}

		// Check for JWT token in session
		session, err := am.authHandler.oauth.GetSession(r, "auth-session")
		if err != nil {
			log.Printf("Auth middleware: Failed to get session: %v", err)
			redirectToLogin(w, r)
			return
		}

		jwtToken, ok := session.Values["jwt"].(string)
		if !ok || jwtToken == "" {
			log.Printf("Auth middleware: No JWT token found in session")
			redirectToLogin(w, r)
			return
		}

		// Validate JWT token
		user, err := am.authHandler.ValidateJWT(jwtToken)
		if err != nil {
			log.Printf("Auth middleware: Invalid JWT token: %v", err)
			redirectToLogin(w, r)
			return
		}

		log.Printf("Auth middleware: Successfully authenticated user ID %d", user.ID)
		// Add user to request context
		ctx := context.WithValue(r.Context(), UserContextKey, user)
		r = r.WithContext(ctx)

		log.Printf("Auth middleware: Calling next handler for %s %s", r.Method, r.URL.Path)
		// Continue to next handler
		next.ServeHTTP(w, r)
		log.Printf("Auth middleware: Completed request %s %s", r.Method, r.URL.Path)
	})
}

// isAuthRoute checks if the route is authentication-related and should skip auth checks
func isAuthRoute(path string) bool {
	authRoutes := []string{
		"/auth/login",
		"/auth/callback",
		"/auth/logout",
		"/login",
	}

	for _, route := range authRoutes {
		if path == route {
			return true
		}
	}
	return false
}

// redirectToLogin redirects unauthenticated users to the login page
func redirectToLogin(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/login", http.StatusFound)
}

// GetUserFromContext extracts user information from request context
func GetUserFromContext(ctx context.Context) (*models.User, bool) {
	user, ok := ctx.Value(UserContextKey).(*models.User)
	return user, ok
}

// RequireAuth is a helper function to ensure a user is authenticated
func RequireAuth(w http.ResponseWriter, r *http.Request) (*models.User, bool) {
	user, ok := GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return nil, false
	}
	return user, true
}
