package app

import (
	"net/http"
	"time"

	"github.com/gorilla/mux"

	"avitotest/internal/database"
	"avitotest/internal/middleware"
	"avitotest/internal/services"
	"avitotest/internal/transport/rest"
)

type App struct {
	pool       database.DBPool
	merchItems map[string]int
	port       string
	jwtSecret  []byte
}

func NewApp(pool database.DBPool, port string, jwtSecret []byte, merchItems map[string]int) *App {
	return &App{pool: pool, port: port, jwtSecret: jwtSecret, merchItems: merchItems}
}
func (a *App) Run() error {
	db := database.NewPGXDatabase(a.pool)
	service := services.NewService(db, a.jwtSecret, a.merchItems)
	handler := rest.NewHandler(service)
	middle := middleware.NewMiddleware(a.jwtSecret)

	router := mux.NewRouter()

	router.HandleFunc("/api/auth", handler.AuthHandler).Methods("POST")

	apiRouter := router.PathPrefix("/api").Subrouter()

	apiRouter.Use(middle.JwtMiddleware)
	apiRouter.HandleFunc("/info", handler.InfoHandler).Methods("GET")
	apiRouter.HandleFunc("/sendCoin", handler.SendCoinHandler).Methods("POST")
	apiRouter.HandleFunc("/buy/{item}", handler.BuyHandler).Methods("GET")

	const readmax, writemax, idlemax = 5 * time.Second, 10 * time.Second, 120 * time.Second
	server := &http.Server{
		Addr:         ":" + a.port,
		Handler:      router,
		ReadTimeout:  readmax,
		WriteTimeout: writemax,
		IdleTimeout:  idlemax,
	}
	err := server.ListenAndServe()

	return err
}
