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

func TestE2E_Info_Success(t *testing.T) {
	username := fmt.Sprintf("e2e_info_user_%d", time.Now().UnixNano())
	password := "testpassword"

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

	infoURL := "http://localhost:8080/api/info"
	req, err := http.NewRequest("GET", infoURL, nil)
	if err != nil {
		t.Fatalf("Ошибка создания запроса к /api/info: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+authResp.Token)

	client := &http.Client{}
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("Ошибка выполнения запроса к /api/info: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Ожидался статус 200 на /api/info, получен %d, тело: %s", resp.StatusCode, string(body))
	}

	var infoResp models.InfoResponse
	if err := json.NewDecoder(resp.Body).Decode(&infoResp); err != nil {
		t.Fatalf("Ошибка декодирования ответа /api/info: %v", err)
	}

	if infoResp.Coins != 1000 {
		t.Errorf("Ожидалось 1000 монет, получено %d", infoResp.Coins)
	}

	if len(infoResp.Inventory) != 0 {
		t.Errorf("Ожидался пустой инвентарь, получено: %+v", infoResp.Inventory)
	}

	if len(infoResp.CoinHistory.Received) != 0 {
		t.Errorf("Ожидалась пустая история входящих транзакций, получено: %+v", infoResp.CoinHistory.Received)
	}
	if len(infoResp.CoinHistory.Sent) != 0 {
		t.Errorf("Ожидалась пустая история исходящих транзакций, получено: %+v", infoResp.CoinHistory.Sent)
	}
}
