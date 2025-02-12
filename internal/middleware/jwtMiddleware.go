package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v4"

	"avitotest/internal/contextkeys"
	"avitotest/internal/models"
)

type Middleware struct {
	jwtSecret []byte
}

func NewMiddleware(jwtSecret []byte) *Middleware {
	return &Middleware{jwtSecret: jwtSecret}
}

func (m *Middleware) JwtMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Нет заголовка Authorization", http.StatusUnauthorized)
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, "Неправильный формат заголовка Authorization", http.StatusUnauthorized)
			return
		}

		tokenStr := parts[1]
		token, err := jwt.ParseWithClaims(tokenStr, &models.Claims{}, func(token *jwt.Token) (interface{}, error) {
			if token.Method.Alg() != jwt.SigningMethodHS512.Alg() {
				return nil, fmt.Errorf("неправильный метод подписи: %v", token.Header["alg"])
			}

			return m.jwtSecret, nil
		})

		if err != nil || !token.Valid {
			http.Error(w, "Неправибльный токен: "+err.Error(), http.StatusUnauthorized)
			return
		}

		claims, ok := token.Claims.(*models.Claims)
		if !ok {
			http.Error(w, "Неправильные утверждения полезной нагрузки токена", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), contextkeys.UserContextKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
