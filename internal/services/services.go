package services

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"golang.org/x/crypto/bcrypt"

	"avitotest/internal/database"
	"avitotest/internal/models"
)

type Service struct {
	database   database.Database
	merchItems map[string]int
	jwtSecret  []byte
}

func NewService(db database.Database, jwtSecret []byte, merchItems map[string]int) *Service {
	return &Service{database: db, jwtSecret: jwtSecret, merchItems: merchItems}
}

func (s *Service) GenerateToken(context context.Context, req models.AuthRequest) (tokenStr string, status int, err error) {
	userID, storedHash, err := s.database.SelectIDPassHashQuery(context, req.Username)

	if err != nil {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			return "", http.StatusInternalServerError, err
		}

		userID, err = s.database.InsertUserQuery(context, req.Username, string(hashedPassword))

		if err != nil {
			return "", http.StatusInternalServerError, err
		}
	} else {
		err = bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(req.Password))
		if err != nil {
			return "", http.StatusUnauthorized, errors.New("неверный пароль")
		}
	}

	const TokenExpireDuration = 24 * time.Hour
	claims := models.Claims{
		UserID:   userID,
		Username: req.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(TokenExpireDuration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS512, claims)
	tokenStr, err = token.SignedString(s.jwtSecret)

	if err != nil {
		return "", http.StatusInternalServerError, err
	}

	return tokenStr, http.StatusOK, nil
}

func (s *Service) GetInfo(context context.Context, user *models.Claims) (info models.InfoResponse, status int, err error) {
	var coins int
	coins, err = s.database.SelectCoinsQuery(context, user.UserID)

	if err != nil {
		return models.InfoResponse{}, http.StatusInternalServerError, err
	}

	inventory, err := s.database.SelectUserItemsQuery(context, user.UserID)
	if err != nil {
		return models.InfoResponse{}, http.StatusInternalServerError, err
	}

	received, err := s.database.SelectReceivedMoneyQuery(context, user.UserID)
	if err != nil {
		return models.InfoResponse{}, http.StatusInternalServerError, err
	}

	sent, err := s.database.SelectSentMoneyQuery(context, user.UserID)
	if err != nil {
		return models.InfoResponse{}, http.StatusInternalServerError, err
	}

	info = models.InfoResponse{
		Coins:     coins,
		Inventory: inventory,
		CoinHistory: models.CoinHistoryResponse{
			Received: received,
			Sent:     sent,
		},
	}

	return info, http.StatusOK, nil
}

func (s *Service) SendCoins(context context.Context, user *models.Claims, req models.SendCoinRequest) (status int, err error) {
	if req.Amount <= 0 {
		return http.StatusBadRequest, errors.New("количество монет должно быть положительным")
	}

	if req.ToUser == "" || req.ToUser == user.Username {
		return http.StatusBadRequest, errors.New("неверный получатель")
	}

	status, err = s.database.SendCoins(context, user.UserID, req.ToUser, req.Amount)

	return status, err
}

func (s *Service) BuyItem(context context.Context, user *models.Claims, item string) (status int, err error) {
	price, ok := s.merchItems[item]
	if !ok {
		return http.StatusInternalServerError, errors.New("товар не найден")
	}

	status, err = s.database.BuyItem(context, user.UserID, price, item)

	return status, err
}
