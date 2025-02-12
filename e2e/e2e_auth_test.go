package e2e

import (
	"avitotest/internal/models"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"
)

func TestE2E_GetToken_Success(t *testing.T) {

	username := fmt.Sprintf("e2e_token_user_%d", time.Now().UnixNano())
	password := "testpassword"

	authURL := "http://localhost:8080/api/auth"
	authReq := models.AuthRequest{
		Username: username,
		Password: password,
	}
	reqBody, err := json.Marshal(authReq)
	if err != nil {
		t.Fatalf("Ошибка маршалинга запроса: %v", err)
	}

	resp, err := http.Post(authURL, "application/json", bytes.NewReader(reqBody))
	if err != nil {
		t.Fatalf("Ошибка выполнения запроса к /api/auth: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Ожидался статус 200, получен %d. Тело ответа: %s", resp.StatusCode, string(body))
	}

	var authResp models.AuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		t.Fatalf("Ошибка декодирования ответа: %v", err)
	}
	if authResp.Token == "" {
		t.Fatalf("Получен пустой JWT-токен")
	}
}
