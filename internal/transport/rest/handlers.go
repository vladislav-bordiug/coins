package rest

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"

	"avitotest/internal/contextkeys"
	"avitotest/internal/models"
)

type ServiceInterface interface {
	GenerateToken(context context.Context, req models.AuthRequest) (string, int, error)
	GetInfo(context context.Context, user *models.Claims) (models.InfoResponse, int, error)
	SendCoins(context context.Context, user *models.Claims, req models.SendCoinRequest) (int, error)
	BuyItem(context context.Context, user *models.Claims, item string) (int, error)
}

type Handler struct {
	services ServiceInterface
}

func NewHandler(services ServiceInterface) *Handler {
	return &Handler{services: services}
}

func getUserFromContext(ctx context.Context) (*models.Claims, error) {
	user, ok := ctx.Value(contextkeys.UserContextKey).(*models.Claims)
	if !ok {
		return nil, fmt.Errorf("user not found in context")
	}

	return user, nil
}

func (h *Handler) AuthHandler(w http.ResponseWriter, r *http.Request) {
	var req models.AuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Неверный запрос", http.StatusBadRequest)
		return
	}

	tokenStr, status, err := h.services.GenerateToken(r.Context(), req)

	if err != nil {
		http.Error(w, err.Error(), status)
		return
	}

	resp := models.AuthResponse{Token: tokenStr}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, "Ошибка записи ответа", status)
		return
	}
}

func (h *Handler) InfoHandler(w http.ResponseWriter, r *http.Request) {
	user, err := getUserFromContext(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	info, status, err := h.services.GetInfo(r.Context(), user)

	if err != nil {
		http.Error(w, err.Error(), status)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(info); err != nil {
		http.Error(w, "Ошибка записи ответа", status)
		return
	}
}

func (h *Handler) SendCoinHandler(w http.ResponseWriter, r *http.Request) {
	user, err := getUserFromContext(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	var req models.SendCoinRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Неверный запрос", http.StatusBadRequest)
		return
	}

	status, err := h.services.SendCoins(r.Context(), user, req)

	if err != nil {
		http.Error(w, err.Error(), status)
		return
	}

	w.WriteHeader(http.StatusOK)

	if _, err := w.Write([]byte("Монеты успешно отправлены")); err != nil {
		http.Error(w, "Монеты успешно отправлены", status)
		return
	}
}

func (h *Handler) BuyHandler(w http.ResponseWriter, r *http.Request) {
	user, err := getUserFromContext(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	item := vars["item"]
	status, err := h.services.BuyItem(r.Context(), user, item)

	if err != nil {
		http.Error(w, err.Error(), status)
		return
	}

	w.WriteHeader(http.StatusOK)

	if _, err := w.Write([]byte("Покупка прошла успешно")); err != nil {
		http.Error(w, "Покупка прошла успешно", status)
		return
	}
}
