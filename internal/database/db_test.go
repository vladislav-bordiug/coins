package database

import (
	"context"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"net/http"
	"regexp"
	"testing"
)

func TestSelectIDPassHashQuery(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	db := NewPGXDatabase(mock)
	username := "testuser"
	expectedID := 42
	expectedHash := "hashedpassword"

	mock.ExpectQuery("SELECT id, password FROM users WHERE username=\\$1").
		WithArgs(username).
		WillReturnRows(pgxmock.NewRows([]string{"id", "password"}).AddRow(expectedID, expectedHash))

	id, hash, err := db.SelectIDPassHashQuery(context.Background(), username)
	assert.NoError(t, err)
	assert.Equal(t, expectedID, id)
	assert.Equal(t, expectedHash, hash)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestInsertUserQuery(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	db := NewPGXDatabase(mock)
	username := "newuser"
	passhash := "passhash"
	expectedID := 100

	mock.ExpectQuery("INSERT INTO users \\(username, password, coins\\) VALUES \\(\\$1, \\$2, 300\\) RETURNING id").
		WithArgs(username, passhash).
		WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(expectedID))

	id, err := db.InsertUserQuery(context.Background(), username, passhash)
	assert.NoError(t, err)
	assert.Equal(t, expectedID, id)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestSelectCoinsQuery(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	db := NewPGXDatabase(mock)
	userID := 1
	expectedCoins := 500

	mock.ExpectQuery("SELECT coins FROM users WHERE id=\\$1").
		WithArgs(userID).
		WillReturnRows(pgxmock.NewRows([]string{"coins"}).AddRow(expectedCoins))

	coins, err := db.SelectCoinsQuery(context.Background(), userID)
	assert.NoError(t, err)
	assert.Equal(t, expectedCoins, coins)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestSelectUserItemsQuery(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	db := NewPGXDatabase(mock)
	userID := 1

	rows := pgxmock.NewRows([]string{"item", "quantity"}).
		AddRow("sword", 1).
		AddRow("shield", 2)

	mock.ExpectQuery("SELECT item, quantity FROM user_inventory WHERE user_id=\\$1").
		WithArgs(userID).
		WillReturnRows(rows)

	inventory, err := db.SelectUserItemsQuery(context.Background(), userID)
	assert.NoError(t, err)
	assert.Len(t, inventory, 2)
	assert.Equal(t, "sword", inventory[0].Type)
	assert.Equal(t, 1, inventory[0].Quantity)
	assert.Equal(t, "shield", inventory[1].Type)
	assert.Equal(t, 2, inventory[1].Quantity)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestSelectReceivedMoneyQuery(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	db := NewPGXDatabase(mock)
	userID := 1

	rows := pgxmock.NewRows([]string{"username", "amount"}).
		AddRow("alice", 100).
		AddRow("bob", 50)

	mock.ExpectQuery("SELECT u\\.username, ct\\.amount FROM coin_transactions").
		WithArgs(userID).
		WillReturnRows(rows)

	received, err := db.SelectReceivedMoneyQuery(context.Background(), userID)
	assert.NoError(t, err)
	assert.Len(t, received, 2)

	assert.Equal(t, "alice", received[0].FromUser)
	assert.Equal(t, 100, received[0].Amount)
	assert.Equal(t, "bob", received[1].FromUser)
	assert.Equal(t, 50, received[1].Amount)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestSelectSentMoneyQuery(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	db := NewPGXDatabase(mock)
	userID := 1

	rows := pgxmock.NewRows([]string{"username", "amount"}).
		AddRow("charlie", 75)

	mock.ExpectQuery("SELECT u\\.username, ct\\.amount FROM coin_transactions").
		WithArgs(userID).
		WillReturnRows(rows)

	sent, err := db.SelectSentMoneyQuery(context.Background(), userID)
	assert.NoError(t, err)
	assert.Len(t, sent, 1)

	assert.Equal(t, "charlie", sent[0].ToUser)
	assert.Equal(t, 75, sent[0].Amount)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestSendCoins_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	db := NewPGXDatabase(mock)
	ctx := context.Background()
	userid := 1
	touser := "receiver"
	amount := 50
	receiverID := 2
	senderCoins := 100

	mock.ExpectBegin()

	mock.ExpectQuery("SELECT id FROM users WHERE username=\\$1").
		WithArgs(touser).
		WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(receiverID))

	mock.ExpectQuery("SELECT coins FROM users WHERE id=\\$1").
		WithArgs(userid).
		WillReturnRows(pgxmock.NewRows([]string{"coins"}).AddRow(senderCoins))

	mock.ExpectExec("UPDATE users SET coins = coins - \\$1 WHERE id=\\$2").
		WithArgs(amount, userid).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	mock.ExpectExec(regexp.QuoteMeta("UPDATE users SET coins = coins + $1 WHERE id=$2")).
		WithArgs(amount, receiverID).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	mock.ExpectExec("INSERT INTO coin_transactions \\(from_user_id, to_user_id, amount\\) VALUES \\(\\$1, \\$2, \\$3\\)").
		WithArgs(userid, receiverID, amount).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	mock.ExpectCommit()

	code, err := db.SendCoins(ctx, userid, touser, amount)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestSendCoins_InsufficientCoins(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	db := NewPGXDatabase(mock)
	ctx := context.Background()
	userid := 1
	touser := "receiver"
	amount := 200
	receiverID := 2
	senderCoins := 100

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT id FROM users WHERE username=\\$1").
		WithArgs(touser).
		WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(receiverID))
	mock.ExpectQuery("SELECT coins FROM users WHERE id=\\$1").
		WithArgs(userid).
		WillReturnRows(pgxmock.NewRows([]string{"coins"}).AddRow(senderCoins))
	mock.ExpectRollback()

	code, err := db.SendCoins(ctx, userid, touser, amount)
	assert.Error(t, err)
	assert.Equal(t, "недостаточно монет", err.Error())
	assert.Equal(t, http.StatusInternalServerError, code)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestBuyItem_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	db := NewPGXDatabase(mock)
	ctx := context.Background()
	userid := 1
	price := 75
	item := "potion"
	availableCoins := 100

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT coins FROM users WHERE id=\\$1").
		WithArgs(userid).
		WillReturnRows(pgxmock.NewRows([]string{"coins"}).AddRow(availableCoins))

	mock.ExpectExec("UPDATE users SET coins = coins - \\$1 WHERE id=\\$2").
		WithArgs(price, userid).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	mock.ExpectExec("INSERT INTO user_inventory \\(user_id, item, quantity\\)").
		WithArgs(userid, item).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	mock.ExpectCommit()

	code, err := db.BuyItem(ctx, userid, price, item)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestBuyItem_InsufficientCoins(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	db := NewPGXDatabase(mock)
	ctx := context.Background()
	userid := 1
	price := 150
	item := "potion"
	availableCoins := 100

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT coins FROM users WHERE id=\\$1").
		WithArgs(userid).
		WillReturnRows(pgxmock.NewRows([]string{"coins"}).AddRow(availableCoins))
	mock.ExpectRollback()

	code, err := db.BuyItem(ctx, userid, price, item)
	assert.Error(t, err)
	assert.Equal(t, "недостаточно монет для покупки", err.Error())
	assert.Equal(t, http.StatusBadRequest, code)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}
