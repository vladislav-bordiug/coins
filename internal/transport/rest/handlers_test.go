package rest

import (
	"avitotest/internal/contextkeys"
	"avitotest/internal/models"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"net/http"
	"net/http/httptest"
	"testing"
)

type MockService struct {
	mock.Mock
}

func (m *MockService) GenerateToken(ctx context.Context, req models.AuthRequest) (string, int, error) {
	args := m.Called(ctx, req)
	return args.String(0), args.Int(1), args.Error(2)
}

func (m *MockService) GetInfo(ctx context.Context, user *models.Claims) (models.InfoResponse, int, error) {
	args := m.Called(ctx, user)
	return args.Get(0).(models.InfoResponse), args.Int(1), args.Error(2)
}

func (m *MockService) SendCoins(ctx context.Context, user *models.Claims, req models.SendCoinRequest) (int, error) {
	args := m.Called(ctx, user, req)
	return args.Int(0), args.Error(1)
}

func (m *MockService) BuyItem(ctx context.Context, user *models.Claims, item string) (int, error) {
	args := m.Called(ctx, user, item)
	return args.Int(0), args.Error(1)
}

func TestAuthHandler_Success(t *testing.T) {
	mockService := new(MockService)
	handler := NewHandler(mockService)

	authReq := models.AuthRequest{
		Username: "user1",
		Password: "password",
	}

	mockService.
		On("GenerateToken", mock.Anything, authReq).
		Return("token123", http.StatusOK, nil).
		Once()

	body, err := json.Marshal(authReq)
	assert.NoError(t, err)

	req := httptest.NewRequest("POST", "/auth", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.AuthHandler(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var resp models.AuthResponse
	err = json.NewDecoder(rr.Body).Decode(&resp)
	assert.NoError(t, err)
	assert.Equal(t, "token123", resp.Token)

	mockService.AssertExpectations(t)
}

func TestAuthHandler_InvalidJSON(t *testing.T) {
	mockService := new(MockService)
	handler := NewHandler(mockService)

	req := httptest.NewRequest("POST", "/auth", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.AuthHandler(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "Неверный запрос")
}

func TestAuthHandler_ServiceError(t *testing.T) {
	mockService := new(MockService)
	handler := NewHandler(mockService)

	authReq := models.AuthRequest{
		Username: "user1",
		Password: "password",
	}
	mockService.
		On("GenerateToken", mock.Anything, authReq).
		Return("", http.StatusInternalServerError, errors.New("service error")).
		Once()

	body, _ := json.Marshal(authReq)
	req := httptest.NewRequest("POST", "/auth", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.AuthHandler(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.Contains(t, rr.Body.String(), "service error")
	mockService.AssertExpectations(t)
}

func TestInfoHandler_Success(t *testing.T) {
	mockService := new(MockService)
	handler := NewHandler(mockService)

	userClaims := &models.Claims{
		UserID:   1,
		Username: "user1",
	}
	infoResp := models.InfoResponse{
		Coins: 100,
	}
	mockService.
		On("GetInfo", mock.Anything, userClaims).
		Return(infoResp, http.StatusOK, nil).
		Once()

	req := httptest.NewRequest("GET", "/info", nil)

	ctx := context.WithValue(req.Context(), contextkeys.UserContextKey, userClaims)
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	handler.InfoHandler(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var gotResp models.InfoResponse
	err := json.NewDecoder(rr.Body).Decode(&gotResp)
	assert.NoError(t, err)
	assert.Equal(t, infoResp.Coins, gotResp.Coins)
	mockService.AssertExpectations(t)
}

func TestInfoHandler_NoUser(t *testing.T) {
	mockService := new(MockService)
	handler := NewHandler(mockService)

	req := httptest.NewRequest("GET", "/info", nil)

	rr := httptest.NewRecorder()

	handler.InfoHandler(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	assert.Contains(t, rr.Body.String(), "user not found in context")
}

func TestInfoHandler_ServiceError(t *testing.T) {
	mockService := new(MockService)
	handler := NewHandler(mockService)

	userClaims := &models.Claims{
		UserID:   1,
		Username: "user1",
	}
	mockService.
		On("GetInfo", mock.Anything, userClaims).
		Return(models.InfoResponse{}, http.StatusInternalServerError, errors.New("get info error")).
		Once()

	req := httptest.NewRequest("GET", "/info", nil)
	ctx := context.WithValue(req.Context(), contextkeys.UserContextKey, userClaims)
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	handler.InfoHandler(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.Contains(t, rr.Body.String(), "get info error")
	mockService.AssertExpectations(t)
}

func TestSendCoinHandler_Success(t *testing.T) {
	mockService := new(MockService)
	handler := NewHandler(mockService)

	userClaims := &models.Claims{
		UserID:   1,
		Username: "user1",
	}
	sendReq := models.SendCoinRequest{
		ToUser: "user2",
		Amount: 50,
	}
	mockService.
		On("SendCoins", mock.Anything, userClaims, sendReq).
		Return(http.StatusOK, nil).
		Once()

	body, _ := json.Marshal(sendReq)
	req := httptest.NewRequest("POST", "/send", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	ctx := context.WithValue(req.Context(), contextkeys.UserContextKey, userClaims)
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	handler.SendCoinHandler(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "Монеты успешно отправлены")
	mockService.AssertExpectations(t)
}

func TestSendCoinHandler_InvalidJSON(t *testing.T) {
	mockService := new(MockService)
	handler := NewHandler(mockService)

	userClaims := &models.Claims{
		UserID:   1,
		Username: "user1",
	}
	req := httptest.NewRequest("POST", "/send", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), contextkeys.UserContextKey, userClaims)
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	handler.SendCoinHandler(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "Неверный запрос")
}

func TestSendCoinHandler_NoUser(t *testing.T) {
	mockService := new(MockService)
	handler := NewHandler(mockService)

	sendReq := models.SendCoinRequest{
		ToUser: "user2",
		Amount: 50,
	}
	body, _ := json.Marshal(sendReq)
	req := httptest.NewRequest("POST", "/send", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()

	handler.SendCoinHandler(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	assert.Contains(t, rr.Body.String(), "user not found in context")
}

func TestSendCoinHandler_ServiceError(t *testing.T) {
	mockService := new(MockService)
	handler := NewHandler(mockService)

	userClaims := &models.Claims{
		UserID:   1,
		Username: "user1",
	}
	sendReq := models.SendCoinRequest{
		ToUser: "user2",
		Amount: 50,
	}
	mockService.
		On("SendCoins", mock.Anything, userClaims, sendReq).
		Return(http.StatusInternalServerError, errors.New("send coin error")).
		Once()

	body, _ := json.Marshal(sendReq)
	req := httptest.NewRequest("POST", "/send", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), contextkeys.UserContextKey, userClaims)
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	handler.SendCoinHandler(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.Contains(t, rr.Body.String(), "send coin error")
	mockService.AssertExpectations(t)
}

func TestBuyHandler_Success(t *testing.T) {
	mockService := new(MockService)
	handler := NewHandler(mockService)

	userClaims := &models.Claims{
		UserID:   1,
		Username: "user1",
	}
	item := "item1"
	mockService.
		On("BuyItem", mock.Anything, userClaims, item).
		Return(http.StatusOK, nil).
		Once()

	req := httptest.NewRequest("POST", "/buy", nil)

	req = mux.SetURLVars(req, map[string]string{"item": item})
	ctx := context.WithValue(req.Context(), contextkeys.UserContextKey, userClaims)
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	handler.BuyHandler(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "Покупка прошла успешно")
	mockService.AssertExpectations(t)
}

func TestBuyHandler_NoUser(t *testing.T) {
	mockService := new(MockService)
	handler := NewHandler(mockService)

	item := "item1"
	req := httptest.NewRequest("POST", "/buy", nil)
	req = mux.SetURLVars(req, map[string]string{"item": item})
	rr := httptest.NewRecorder()

	handler.BuyHandler(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	assert.Contains(t, rr.Body.String(), "user not found in context")
}

func TestBuyHandler_ServiceError(t *testing.T) {
	mockService := new(MockService)
	handler := NewHandler(mockService)

	userClaims := &models.Claims{
		UserID:   1,
		Username: "user1",
	}
	item := "item1"
	mockService.
		On("BuyItem", mock.Anything, userClaims, item).
		Return(http.StatusInternalServerError, errors.New("buy error")).
		Once()

	req := httptest.NewRequest("POST", "/buy", nil)
	req = mux.SetURLVars(req, map[string]string{"item": item})
	ctx := context.WithValue(req.Context(), contextkeys.UserContextKey, userClaims)
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	handler.BuyHandler(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.Contains(t, rr.Body.String(), "buy error")
	mockService.AssertExpectations(t)
}
