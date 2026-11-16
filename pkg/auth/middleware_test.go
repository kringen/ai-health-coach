package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"health-coach/pkg/models"
)

func TestGetUserFromContext(t *testing.T) {
	user := &models.User{
		ID:    1,
		Email: "test@example.com",
		Name:  "Test User",
	}

	ctx := context.WithValue(context.Background(), UserContextKey, user)

	retrievedUser, ok := GetUserFromContext(ctx)
	if !ok {
		t.Error("Expected to retrieve user from context")
	}

	if retrievedUser.Email != "test@example.com" {
		t.Errorf("Expected email 'test@example.com', got %s", retrievedUser.Email)
	}
}

func TestGetUserFromContextNotFound(t *testing.T) {
	ctx := context.Background()

	_, ok := GetUserFromContext(ctx)
	if ok {
		t.Error("Expected not to find user in empty context")
	}
}

func TestRequireAuth(t *testing.T) {
	user := &models.User{
		ID:    1,
		Email: "test@example.com",
		Name:  "Test User",
	}

	// Test with authenticated user
	ctx := context.WithValue(context.Background(), UserContextKey, user)
	req := httptest.NewRequest("GET", "/", nil)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	retrievedUser, ok := RequireAuth(w, req)
	if !ok {
		t.Error("Expected authentication to succeed")
	}

	if retrievedUser.Email != "test@example.com" {
		t.Errorf("Expected email 'test@example.com', got %s", retrievedUser.Email)
	}

	// Test without authenticated user
	req2 := httptest.NewRequest("GET", "/", nil)
	w2 := httptest.NewRecorder()

	_, ok2 := RequireAuth(w2, req2)
	if ok2 {
		t.Error("Expected authentication to fail")
	}

	if w2.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w2.Code)
	}
}
