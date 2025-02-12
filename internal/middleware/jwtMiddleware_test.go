package middleware

import (
	"avitotest/internal/contextkeys"
	"avitotest/internal/models"
	"github.com/golang-jwt/jwt/v4"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestJwtMiddleware_NoAuthHeader(t *testing.T) {
	secret := []byte("mysecret")
	mw := NewMiddleware(secret)

	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	handler := mw.JwtMiddleware(next)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401 Unauthorized, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "Нет заголовка Authorization") {
		t.Errorf("Expected error message about missing Authorization header")
	}
	if nextCalled {
		t.Error("Next handler should not be called when no Authorization header is present")
	}
}

func TestJwtMiddleware_InvalidFormat(t *testing.T) {
	secret := []byte("mysecret")
	mw := NewMiddleware(secret)

	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)

	req.Header.Set("Authorization", "InvalidFormatToken")
	rr := httptest.NewRecorder()

	handler := mw.JwtMiddleware(next)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401 Unauthorized, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "Неправильный формат заголовка Authorization") {
		t.Errorf("Expected error message about invalid Authorization header format")
	}
	if nextCalled {
		t.Error("Next handler should not be called for invalid header format")
	}
}

func TestJwtMiddleware_InvalidToken(t *testing.T) {
	secret := []byte("mysecret")
	mw := NewMiddleware(secret)

	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)

	req.Header.Set("Authorization", "Bearer invalid.token.value")
	rr := httptest.NewRecorder()

	handler := mw.JwtMiddleware(next)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401 Unauthorized, got %d", rr.Code)
	}

	if !strings.Contains(rr.Body.String(), "Неправибльный токен:") {
		t.Errorf("Expected error message about invalid token")
	}
	if nextCalled {
		t.Error("Next handler should not be called for invalid token")
	}
}

func TestJwtMiddleware_WrongSigningMethod(t *testing.T) {
	secret := []byte("mysecret")
	mw := NewMiddleware(secret)

	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	})

	claims := &models.Claims{
		UserID:   1,
		Username: "user1",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString(secret)
	if err != nil {
		t.Fatalf("Error signing token: %v", err)
	}

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	rr := httptest.NewRecorder()

	handler := mw.JwtMiddleware(next)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401 Unauthorized, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "неправильный метод подписи") {
		t.Errorf("Expected error message about wrong signing method")
	}
	if nextCalled {
		t.Error("Next handler should not be called for token with wrong signing method")
	}
}

func TestJwtMiddleware_ValidToken(t *testing.T) {
	secret := []byte("mysecret")
	mw := NewMiddleware(secret)

	nextCalled := false
	var capturedClaims *models.Claims
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true

		val := r.Context().Value(contextkeys.UserContextKey)
		if claims, ok := val.(*models.Claims); ok {
			capturedClaims = claims
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	claims := &models.Claims{
		UserID:   1,
		Username: "user1",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS512, claims)
	tokenStr, err := token.SignedString(secret)
	if err != nil {
		t.Fatalf("Error signing token: %v", err)
	}

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	rr := httptest.NewRecorder()

	handler := mw.JwtMiddleware(next)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200 OK, got %d", rr.Code)
	}
	if !nextCalled {
		t.Error("Next handler was not called for valid token")
	}
	if capturedClaims == nil {
		t.Error("Claims not set in request context")
	} else {
		if capturedClaims.UserID != 1 || capturedClaims.Username != "user1" {
			t.Errorf("Claims values do not match; got %+v", capturedClaims)
		}
	}
}
