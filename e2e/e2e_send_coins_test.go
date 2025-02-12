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

func registerUser(t *testing.T, username, password string) string {
	authURL := "http://localhost:8080/api/auth"
	authReq := models.AuthRequest{
		Username: username,
		Password: password,
	}
	reqBody, err := json.Marshal(authReq)
	if err != nil {
		t.Fatalf("Ошибка маршалинга запроса аутентификации: %v", err)
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
		t.Fatalf("Ошибка декодирования ответа /api/auth: %v", err)
	}
	if authResp.Token == "" {
		t.Fatalf("Получен пустой JWT-токен")
	}
	return authResp.Token
}

func TestE2E_SendCoin_Success(t *testing.T) {

	senderUsername := fmt.Sprintf("e2e_sender_%d", time.Now().UnixNano())
	receiverUsername := fmt.Sprintf("e2e_receiver_%d", time.Now().UnixNano())
	password := "testpassword"

	senderToken := registerUser(t, senderUsername, password)

	registerUser(t, receiverUsername, password)

	sendCoinURL := "http://localhost:8080/api/sendCoin"
	sendReq := models.SendCoinRequest{
		ToUser: receiverUsername,
		Amount: 50,
	}
	reqBody, err := json.Marshal(sendReq)
	if err != nil {
		t.Fatalf("Ошибка маршалинга запроса send coin: %v", err)
	}

	req, err := http.NewRequest("POST", sendCoinURL, bytes.NewReader(reqBody))
	if err != nil {
		t.Fatalf("Ошибка создания запроса к /api/sendCoin: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+senderToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Ошибка отправки запроса /api/sendCoin: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Ожидался статус 200 на /api/sendCoin, получен %d, тело: %s", resp.StatusCode, string(body))
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Ошибка чтения тела ответа /api/sendCoin: %v", err)
	}
	bodyStr := string(bodyBytes)
	expected := "Монеты успешно отправлены"
	if bodyStr != expected {
		t.Fatalf("Ожидался ответ %q, получен %q", expected, bodyStr)
	}
}
