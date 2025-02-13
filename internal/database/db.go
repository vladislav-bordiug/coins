package database

import (
	"context"
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"avitotest/internal/models"
)

type Database interface {
	SelectIDPassHashQuery(ctx context.Context, username string) (userID int, storedHash string, err error)
	InsertUserQuery(ctx context.Context, username string, passhash string) (int, error)
	SelectCoinsQuery(ctx context.Context, userid int) (int, error)
	SelectUserItemsQuery(ctx context.Context, userid int) ([]models.InventoryItem, error)
	SelectReceivedMoneyQuery(ctx context.Context, userid int) ([]models.TransactionRecord, error)
	SelectSentMoneyQuery(ctx context.Context, userid int) ([]models.TransactionRecord, error)
	SendCoins(ctx context.Context, userid int, touser string, amount int) (int, error)
	BuyItem(ctx context.Context, userid int, price int, item string) (int, error)
}

type DBPool interface {
	Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, arguments ...interface{}) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, arguments ...interface{}) pgx.Row
	Begin(ctx context.Context) (pgx.Tx, error)
}

type PGXDatabase struct {
	pool DBPool
}

func NewPGXDatabase(pool DBPool) *PGXDatabase {
	return &PGXDatabase{pool: pool}
}

func (db *PGXDatabase) SelectIDPassHashQuery(ctx context.Context, username string) (userID int, storedHash string, err error) {
	err = db.pool.QueryRow(ctx,
		"SELECT id, password FROM users WHERE username=$1", username).Scan(&userID, &storedHash)

	return userID, storedHash, err
}

func (db *PGXDatabase) InsertUserQuery(ctx context.Context, username, passhash string) (int, error) {
	var userID int
	err := db.pool.QueryRow(ctx,
		"INSERT INTO users (username, password, coins) VALUES ($1, $2, 1000) RETURNING id",
		username, passhash).Scan(&userID)

	return userID, err
}

func (db *PGXDatabase) SelectCoinsQuery(ctx context.Context, userid int) (int, error) {
	var coins int

	err := db.pool.QueryRow(ctx, "SELECT coins FROM users WHERE id=$1", userid).Scan(&coins)

	return coins, err
}

func (db *PGXDatabase) SelectUserItemsQuery(ctx context.Context, userid int) ([]models.InventoryItem, error) {
	var inventory []models.InventoryItem

	rows, err := db.pool.Query(ctx, "SELECT item, quantity FROM user_inventory WHERE user_id=$1", userid)

	if err != nil {
		return inventory, err
	}

	defer rows.Close()

	for rows.Next() {
		var item models.InventoryItem
		if err := rows.Scan(&item.Type, &item.Quantity); err != nil {
			return inventory, err
		}

		inventory = append(inventory, item)
	}

	return inventory, nil
}

func (db *PGXDatabase) SelectReceivedMoneyQuery(ctx context.Context, userid int) ([]models.TransactionRecord, error) {
	var received []models.TransactionRecord

	rows, err := db.pool.Query(ctx,
		`SELECT u.username, ct.amount FROM coin_transactions ct
         JOIN users u ON ct.from_user_id = u.id
         WHERE ct.to_user_id=$1 ORDER BY ct.created_at`, userid)

	if err != nil {
		return received, err
	}

	defer rows.Close()

	for rows.Next() {
		var tr models.TransactionRecord
		if err := rows.Scan(&tr.FromUser, &tr.Amount); err != nil {
			return received, err
		}

		received = append(received, tr)
	}

	return received, nil
}

func (db *PGXDatabase) SelectSentMoneyQuery(ctx context.Context, userid int) ([]models.TransactionRecord, error) {
	var sent []models.TransactionRecord

	rows, err := db.pool.Query(ctx,
		`SELECT u.username, ct.amount FROM coin_transactions ct
         JOIN users u ON ct.to_user_id = u.id
         WHERE ct.from_user_id=$1 ORDER BY ct.created_at`, userid)

	if err != nil {
		return sent, err
	}

	defer rows.Close()

	for rows.Next() {
		var tr models.TransactionRecord
		if err := rows.Scan(&tr.ToUser, &tr.Amount); err != nil {
			return sent, err
		}

		sent = append(sent, tr)
	}

	return sent, nil
}

func (db *PGXDatabase) SendCoins(ctx context.Context, userid int, touser string, amount int) (int, error) {
	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var receiverID int
	err = tx.QueryRow(ctx, "SELECT id FROM users WHERE username=$1", touser).Scan(&receiverID)

	if err != nil {
		return http.StatusBadRequest, err
	}

	var senderCoins int
	err = tx.QueryRow(ctx, "SELECT coins FROM users WHERE id=$1", userid).Scan(&senderCoins)

	if err != nil {
		return http.StatusInternalServerError, err
	}

	if senderCoins < amount {
		return http.StatusInternalServerError, errors.New("недостаточно монет")
	}

	_, err = tx.Exec(ctx, "UPDATE users SET coins = coins - $1 WHERE id=$2", amount, userid)

	if err != nil {
		return http.StatusInternalServerError, err
	}

	_, err = tx.Exec(ctx, "UPDATE users SET coins = coins + $1 WHERE id=$2", amount, receiverID)

	if err != nil {
		return http.StatusInternalServerError, err
	}

	_, err = tx.Exec(ctx,
		"INSERT INTO coin_transactions (from_user_id, to_user_id, amount) VALUES ($1, $2, $3)",
		userid, receiverID, amount)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	if err := tx.Commit(ctx); err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, nil
}

func (db *PGXDatabase) BuyItem(ctx context.Context, userid, price int, item string) (int, error) {
	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var coins int
	err = tx.QueryRow(ctx, "SELECT coins FROM users WHERE id=$1", userid).Scan(&coins)

	if err != nil {
		return http.StatusInternalServerError, err
	}

	if coins < price {
		return http.StatusBadRequest, errors.New("недостаточно монет для покупки")
	}

	_, err = tx.Exec(ctx, "UPDATE users SET coins = coins - $1 WHERE id=$2", price, userid)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	_, err = tx.Exec(ctx, `
        INSERT INTO user_inventory (user_id, item, quantity)
        VALUES ($1, $2, 1)
        ON CONFLICT (user_id, item)
        DO UPDATE SET quantity = user_inventory.quantity + 1
    `, userid, item)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	if err := tx.Commit(ctx); err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, nil
}
