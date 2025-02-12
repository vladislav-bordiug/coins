package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"avitotest/internal/app"
)

const tshirtprice, cupprice, bookprice, penprice, powerbankpricee = 80, 20, 50, 10, 200
const hoodyprice, umbrellaprice, socksprice, walletprice, pinkhoodypric = 300, 200, 10, 50, 500

var merchItems = map[string]int{
	"t-shirt":    tshirtprice,
	"cup":        cupprice,
	"book":       bookprice,
	"pen":        penprice,
	"powerbank":  powerbankpricee,
	"hoody":      hoodyprice,
	"umbrella":   umbrellaprice,
	"socks":      socksprice,
	"wallet":     walletprice,
	"pink-hoody": pinkhoodypric,
}

func main() {
	dbHost := os.Getenv("DATABASE_HOST")
	dbPort := os.Getenv("DATABASE_PORT")
	dbUser := os.Getenv("DATABASE_USER")
	dbPassword := os.Getenv("DATABASE_PASSWORD")
	dbName := os.Getenv("DATABASE_NAME")
	serverPort := os.Getenv("SERVER_PORT")
	secret := os.Getenv("SECRET")

	if dbHost == "" || dbPort == "" || dbUser == "" || dbPassword == "" || dbName == "" || serverPort == "" || secret == "" {
		log.Fatal("Не все переменные окружения заданы")
	}

	jwtSecret := []byte(secret)

	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s", dbUser, dbPassword, dbHost, dbPort, dbName)

	var err error
	db, err := pgxpool.New(context.Background(), dsn)

	if err != nil {
		log.Fatalf("Ошибка подключения к БД: %v", err)
	}

	application := app.NewApp(db, serverPort, jwtSecret, merchItems)
	if err := application.Run(); err != nil {
		log.Printf("Ошибка выполнения приложения: %v", err)
		db.Close()
		os.Exit(1)
	}
}
