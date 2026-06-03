package httpapi

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"l45/internal/repository"
)

type Handler struct {
	repo repository.OrderReader
}

func NewHandler(repo repository.OrderReader) *Handler {
	return &Handler{
		repo: repo,
	}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/order", h.GetOrder)
	mux.HandleFunc("/health", h.Health)
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func (h *Handler) GetOrder(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "missing id", http.StatusBadRequest)
		return
	}

	log.Printf("request order id=%s", id)

	order, ok, err := h.repo.GetOrderByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			http.Error(w, "order not found", http.StatusNotFound)
			return
		}

		log.Printf("get order error: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if !ok {
		http.Error(w, "order not found", http.StatusNotFound)
		return
	}

	data, err := json.Marshal(order)
	if err != nil {
		log.Printf("json marshal error: %v", err)
		http.Error(w, "json marshal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}
