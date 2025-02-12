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

func TestE2E_Buy_Success(t *testing.T) {

	username := fmt.Sprintf("e2e_test_user_%d", time.Now().UnixNano())
	password := "testpassword"

	authURL := "http://localhost:8080/api/auth"
	authReq := models.AuthRequest{
		Username: username,
		Password: password,
	}
	reqBody, err := json.Marshal(authReq)
	if err != nil {
		t.Fatalf("Не удалось сериализовать запрос аутентификации: %v", err)
	}

	resp, err := http.Post(authURL, "application/json", bytes.NewReader(reqBody))
	if err != nil {
		t.Fatalf("Ошибка запроса к /api/auth: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Ожидался статус 200 на /api/auth, получен %d, тело: %s", resp.StatusCode, string(body))
	}

	var authResp models.AuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		t.Fatalf("Не удалось разобрать ответ /api/auth: %v", err)
	}
	if authResp.Token == "" {
		t.Fatalf("Получен пустой JWT-токен")
	}

	item := "t-shirt"
	buyURL := fmt.Sprintf("http://localhost:8080/api/buy/%s", item)

	req, err := http.NewRequest("GET", buyURL, nil)
	if err != nil {
		t.Fatalf("Ошибка создания запроса к /api/buy: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+authResp.Token)

	client := &http.Client{}
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("Ошибка вызова /api/buy: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Ожидался статус 200 на /api/buy, получен %d, тело: %s", resp.StatusCode, string(body))
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Не удалось прочитать тело ответа /api/buy: %v", err)
	}
	bodyStr := string(bodyBytes)
	expected := "Покупка прошла успешно"
	if bodyStr != expected {
		t.Fatalf("Ожидался ответ %q, получен %q", expected, bodyStr)
	}
}
