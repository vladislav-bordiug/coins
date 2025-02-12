package models

import "github.com/golang-jwt/jwt/v4"

type Claims struct {
	jwt.RegisteredClaims
	Username string `json:"username"`
	UserID   int    `json:"user_id"`
}

type AuthRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type AuthResponse struct {
	Token string `json:"token"`
}

type SendCoinRequest struct {
	ToUser string `json:"toUser"`
	Amount int    `json:"amount"`
}

type InventoryItem struct {
	Type     string `json:"type"`
	Quantity int    `json:"quantity"`
}

type TransactionRecord struct {
	FromUser string `json:"fromUser,omitempty"`
	ToUser   string `json:"toUser,omitempty"`
	Amount   int    `json:"amount"`
}

type CoinHistoryResponse struct {
	Received []TransactionRecord `json:"received"`
	Sent     []TransactionRecord `json:"sent"`
}

type InfoResponse struct {
	CoinHistory CoinHistoryResponse `json:"coinHistory"`
	Inventory   []InventoryItem     `json:"inventory"`
	Coins       int                 `json:"coins"`
}
