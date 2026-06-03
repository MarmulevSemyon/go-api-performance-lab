package httpapi

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"sync"

	"l45/internal/repository"
)

type JSONCache struct {
	mu    sync.RWMutex
	items map[string][]byte
}

func NewJSONCache() *JSONCache {
	return &JSONCache{
		items: make(map[string][]byte),
	}
}

func (c *JSONCache) Get(id string) ([]byte, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	data, ok := c.items[id]
	return data, ok
}

func (c *JSONCache) Set(id string, data []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[id] = data
}

type Handler struct {
	repo      repository.OrderReader
	jsonCache *JSONCache
	debug     bool
}

func NewHandler(repo repository.OrderReader) *Handler {
	return &Handler{
		repo:      repo,
		jsonCache: NewJSONCache(),
		debug:     false,
	}
}

func (h *Handler) SetDebug(debug bool) {
	h.debug = debug
}

func (h *Handler) logf(format string, args ...any) {
	if h.debug {
		log.Printf(format, args...)
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

	h.logf("request order id=%s", id)

	if data, ok := h.jsonCache.Get(id); ok {
		writeJSON(w, data)
		return
	}

	order, ok, err := h.repo.GetOrderByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			http.Error(w, "order not found", http.StatusNotFound)
			return
		}

		h.logf("get order error: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if !ok {
		http.Error(w, "order not found", http.StatusNotFound)
		return
	}

	data, err := json.Marshal(order)
	if err != nil {
		h.logf("json marshal error: %v", err)
		http.Error(w, "json marshal error", http.StatusInternalServerError)
		return
	}

	h.jsonCache.Set(id, data)

	writeJSON(w, data)
}

func writeJSON(w http.ResponseWriter, data []byte) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}
