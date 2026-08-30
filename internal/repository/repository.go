package repository

import (
	"context"
	"errors"
	"sync"

	"github.com/MarmulevSemyon/go-api-performance-lab/internal/model"
)

var ErrNotFound = errors.New("order not found")

type OrderReader interface {
	GetOrderByID(ctx context.Context, id string) (model.Order, bool, error)
}

type Repository struct {
	mu      sync.RWMutex
	cache   map[string]model.Order
	storage map[string]model.Order
}

func NewRepository(storage map[string]model.Order) *Repository {
	return &Repository{
		cache:   make(map[string]model.Order),
		storage: storage,
	}
}

func (r *Repository) GetOrderByID(ctx context.Context, id string) (model.Order, bool, error) {
	select {
	case <-ctx.Done():
		return model.Order{}, false, ctx.Err()
	default:
	}

	r.mu.RLock()
	order, ok := r.cache[id]
	r.mu.RUnlock()

	if ok {
		return order, true, nil
	}

	order, ok = r.storage[id]
	if !ok {
		return model.Order{}, false, ErrNotFound
	}

	r.mu.Lock()
	r.cache[id] = order
	r.mu.Unlock()

	return order, true, nil
}
