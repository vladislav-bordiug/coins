package services

import (
	"avitotest/internal/models"
	"context"
	"errors"
	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"testing"
)

type MockDatabase struct {
	mock.Mock
}

func (m *MockDatabase) SelectIDPassHashQuery(ctx context.Context, username string) (int, string, error) {
	args := m.Called(ctx, username)
	return args.Int(0), args.String(1), args.Error(2)
}

func (m *MockDatabase) InsertUserQuery(ctx context.Context, username string, passhash string) (int, error) {
	args := m.Called(ctx, username, passhash)
	return args.Int(0), args.Error(1)
}

func (m *MockDatabase) SelectCoinsQuery(ctx context.Context, userid int) (int, error) {
	args := m.Called(ctx, userid)
	return args.Int(0), args.Error(1)
}

func (m *MockDatabase) SelectUserItemsQuery(ctx context.Context, userid int) ([]models.InventoryItem, error) {
	args := m.Called(ctx, userid)
	return args.Get(0).([]models.InventoryItem), args.Error(1)
}

func (m *MockDatabase) SelectReceivedMoneyQuery(ctx context.Context, userid int) ([]models.TransactionRecord, error) {
	args := m.Called(ctx, userid)
	return args.Get(0).([]models.TransactionRecord), args.Error(1)
}

func (m *MockDatabase) SelectSentMoneyQuery(ctx context.Context, userid int) ([]models.TransactionRecord, error) {
	args := m.Called(ctx, userid)
	return args.Get(0).([]models.TransactionRecord), args.Error(1)
}

func (m *MockDatabase) SendCoins(ctx context.Context, userid int, touser string, amount int) (int, error) {
	args := m.Called(ctx, userid, touser, amount)
	return args.Int(0), args.Error(1)
}

func (m *MockDatabase) BuyItem(ctx context.Context, userid int, price int, item string) (int, error) {
	args := m.Called(ctx, userid, price, item)
	return args.Int(0), args.Error(1)
}

func TestGenerateToken_UserExists_Success(t *testing.T) {
	mockDB := new(MockDatabase)
	jwtSecret := []byte("secret")

	service := NewService(mockDB, jwtSecret, map[string]int{})
	ctx := context.Background()

	authReq := models.AuthRequest{
		Username: "user1",
		Password: "password",
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	assert.NoError(t, err)

	mockDB.
		On("SelectIDPassHashQuery", ctx, "user1").
		Return(1, string(hashed), nil).
		Once()

	tokenStr, status, err := service.GenerateToken(ctx, authReq)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, status)
	assert.NotEmpty(t, tokenStr)

	parsedToken, err := jwt.ParseWithClaims(tokenStr, &models.Claims{}, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})
	assert.NoError(t, err)
	claims, ok := parsedToken.Claims.(*models.Claims)
	assert.True(t, ok)
	assert.Equal(t, 1, claims.UserID)
	assert.Equal(t, "user1", claims.Username)

	mockDB.AssertExpectations(t)
}

func TestGenerateToken_NewUser_Success(t *testing.T) {
	mockDB := new(MockDatabase)
	jwtSecret := []byte("secret")
	service := NewService(mockDB, jwtSecret, map[string]int{})
	ctx := context.Background()

	authReq := models.AuthRequest{
		Username: "newuser",
		Password: "newpassword",
	}

	mockDB.
		On("SelectIDPassHashQuery", ctx, "newuser").
		Return(0, "", errors.New("user not found")).
		Once()

	mockDB.
		On("InsertUserQuery", ctx, "newuser", mock.AnythingOfType("string")).
		Return(2, nil).
		Once()

	tokenStr, status, err := service.GenerateToken(ctx, authReq)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, status)
	assert.NotEmpty(t, tokenStr)

	parsedToken, err := jwt.ParseWithClaims(tokenStr, &models.Claims{}, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})
	assert.NoError(t, err)
	claims, ok := parsedToken.Claims.(*models.Claims)
	assert.True(t, ok)
	assert.Equal(t, 2, claims.UserID)
	assert.Equal(t, "newuser", claims.Username)

	mockDB.AssertExpectations(t)
}

func TestGenerateToken_IncorrectPassword(t *testing.T) {
	mockDB := new(MockDatabase)
	jwtSecret := []byte("secret")
	service := NewService(mockDB, jwtSecret, map[string]int{})
	ctx := context.Background()

	authReq := models.AuthRequest{
		Username: "user1",
		Password: "wrongpassword",
	}

	correctHash, err := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	assert.NoError(t, err)

	mockDB.
		On("SelectIDPassHashQuery", ctx, "user1").
		Return(1, string(correctHash), nil).
		Once()

	tokenStr, status, err := service.GenerateToken(ctx, authReq)
	assert.Error(t, err)
	assert.Equal(t, "неверный пароль", err.Error())
	assert.Equal(t, http.StatusUnauthorized, status)
	assert.Empty(t, tokenStr)

	mockDB.AssertExpectations(t)
}

func TestGetInfo_Success(t *testing.T) {
	mockDB := new(MockDatabase)
	jwtSecret := []byte("secret")
	service := NewService(mockDB, jwtSecret, map[string]int{})
	ctx := context.Background()

	claims := &models.Claims{
		UserID:   1,
		Username: "user1",
	}

	mockDB.
		On("SelectCoinsQuery", ctx, 1).
		Return(100, nil).
		Once()

	inv := []models.InventoryItem{
		{Type: "sword", Quantity: 1},
	}
	mockDB.
		On("SelectUserItemsQuery", ctx, 1).
		Return(inv, nil).
		Once()

	received := []models.TransactionRecord{
		{FromUser: "alice", Amount: 50},
	}
	mockDB.
		On("SelectReceivedMoneyQuery", ctx, 1).
		Return(received, nil).
		Once()

	sent := []models.TransactionRecord{
		{ToUser: "bob", Amount: 30},
	}
	mockDB.
		On("SelectSentMoneyQuery", ctx, 1).
		Return(sent, nil).
		Once()

	info, status, err := service.GetInfo(ctx, claims)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, 100, info.Coins)
	assert.Equal(t, inv, info.Inventory)
	assert.Equal(t, received, info.CoinHistory.Received)
	assert.Equal(t, sent, info.CoinHistory.Sent)

	mockDB.AssertExpectations(t)
}

func TestGetInfo_ErrorOnCoins(t *testing.T) {
	mockDB := new(MockDatabase)
	jwtSecret := []byte("secret")
	service := NewService(mockDB, jwtSecret, map[string]int{})
	ctx := context.Background()

	claims := &models.Claims{
		UserID:   1,
		Username: "user1",
	}

	mockDB.
		On("SelectCoinsQuery", ctx, 1).
		Return(0, errors.New("db error")).
		Once()

	info, status, err := service.GetInfo(ctx, claims)
	assert.Error(t, err)
	assert.Equal(t, http.StatusInternalServerError, status)

	assert.Equal(t, models.InfoResponse{}, info)

	mockDB.AssertExpectations(t)
}

func TestSendCoins_Success(t *testing.T) {
	mockDB := new(MockDatabase)
	jwtSecret := []byte("secret")
	service := NewService(mockDB, jwtSecret, map[string]int{})
	ctx := context.Background()

	claims := &models.Claims{
		UserID:   1,
		Username: "user1",
	}
	req := models.SendCoinRequest{
		ToUser: "user2",
		Amount: 50,
	}

	mockDB.
		On("SendCoins", ctx, 1, "user2", 50).
		Return(http.StatusOK, nil).
		Once()

	status, err := service.SendCoins(ctx, claims, req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, status)

	mockDB.AssertExpectations(t)
}

func TestSendCoins_InvalidAmount(t *testing.T) {
	mockDB := new(MockDatabase)
	jwtSecret := []byte("secret")
	service := NewService(mockDB, jwtSecret, map[string]int{})
	ctx := context.Background()

	claims := &models.Claims{
		UserID:   1,
		Username: "user1",
	}

	req := models.SendCoinRequest{
		ToUser: "user2",
		Amount: 0,
	}

	status, err := service.SendCoins(ctx, claims, req)
	assert.Error(t, err)
	assert.Equal(t, "количество монет должно быть положительным", err.Error())
	assert.Equal(t, http.StatusBadRequest, status)
}

func TestSendCoins_InvalidRecipient(t *testing.T) {
	mockDB := new(MockDatabase)
	jwtSecret := []byte("secret")
	service := NewService(mockDB, jwtSecret, map[string]int{})
	ctx := context.Background()

	claims := &models.Claims{
		UserID:   1,
		Username: "user1",
	}

	reqEmpty := models.SendCoinRequest{
		ToUser: "",
		Amount: 50,
	}
	status, err := service.SendCoins(ctx, claims, reqEmpty)
	assert.Error(t, err)
	assert.Equal(t, "неверный получатель", err.Error())
	assert.Equal(t, http.StatusBadRequest, status)

	reqSame := models.SendCoinRequest{
		ToUser: "user1",
		Amount: 50,
	}
	status, err = service.SendCoins(ctx, claims, reqSame)
	assert.Error(t, err)
	assert.Equal(t, "неверный получатель", err.Error())
	assert.Equal(t, http.StatusBadRequest, status)
}

func TestBuyItem_Success(t *testing.T) {
	mockDB := new(MockDatabase)

	merch := map[string]int{
		"potion": 50,
	}
	jwtSecret := []byte("secret")
	service := NewService(mockDB, jwtSecret, merch)
	ctx := context.Background()

	claims := &models.Claims{
		UserID:   1,
		Username: "user1",
	}

	mockDB.
		On("BuyItem", ctx, 1, 50, "potion").
		Return(http.StatusOK, nil).
		Once()

	status, err := service.BuyItem(ctx, claims, "potion")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, status)

	mockDB.AssertExpectations(t)
}

func TestBuyItem_ItemNotFound(t *testing.T) {
	mockDB := new(MockDatabase)
	merch := map[string]int{
		"potion": 50,
	}
	jwtSecret := []byte("secret")
	service := NewService(mockDB, jwtSecret, merch)
	ctx := context.Background()

	claims := &models.Claims{
		UserID:   1,
		Username: "user1",
	}

	status, err := service.BuyItem(ctx, claims, "sword")
	assert.Error(t, err)
	assert.Equal(t, "товар не найден", err.Error())
	assert.Equal(t, http.StatusInternalServerError, status)
}

func TestBuyItem_DatabaseError(t *testing.T) {
	mockDB := new(MockDatabase)
	merch := map[string]int{
		"potion": 50,
	}
	jwtSecret := []byte("secret")
	service := NewService(mockDB, jwtSecret, merch)
	ctx := context.Background()

	claims := &models.Claims{
		UserID:   1,
		Username: "user1",
	}

	mockDB.
		On("BuyItem", ctx, 1, 50, "potion").
		Return(http.StatusInternalServerError, errors.New("db error")).
		Once()

	status, err := service.BuyItem(ctx, claims, "potion")
	assert.Error(t, err)
	assert.Equal(t, "db error", err.Error())
	assert.Equal(t, http.StatusInternalServerError, status)

	mockDB.AssertExpectations(t)
}
