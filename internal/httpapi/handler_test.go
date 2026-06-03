package httpapi

import (
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"l45/internal/generator"
	"l45/internal/repository"
)

type benchmarkResponseWriter struct {
	header http.Header
	code   int
	bytes  int
}

func newBenchmarkResponseWriter() *benchmarkResponseWriter {
	return &benchmarkResponseWriter{
		header: make(http.Header),
	}
}

func (w *benchmarkResponseWriter) Header() http.Header {
	return w.header
}

func (w *benchmarkResponseWriter) Write(data []byte) (int, error) {
	if w.code == 0 {
		w.code = http.StatusOK
	}

	w.bytes += len(data)

	return len(data), nil
}

func (w *benchmarkResponseWriter) WriteHeader(statusCode int) {
	w.code = statusCode
}

func (w *benchmarkResponseWriter) Reset() {
	for key := range w.header {
		delete(w.header, key)
	}

	w.code = 0
	w.bytes = 0
}

func TestGetOrderOK(t *testing.T) {
	orders := generator.GenerateOrders(10000)
	repo := repository.NewRepository(orders)
	handler := NewHandler(repo)

	req := httptest.NewRequest(http.MethodGet, "/order?id=order-1", nil)
	rr := httptest.NewRecorder()

	handler.GetOrder(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	contentType := rr.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Fatalf("expected content-type application/json, got %q", contentType)
	}

	if rr.Body.Len() == 0 {
		t.Fatal("expected non-empty response body")
	}
}

func TestGetOrderNotFound(t *testing.T) {
	orders := generator.GenerateOrders(10000)
	repo := repository.NewRepository(orders)
	handler := NewHandler(repo)

	req := httptest.NewRequest(http.MethodGet, "/order?id=unknown", nil)
	rr := httptest.NewRecorder()

	handler.GetOrder(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rr.Code)
	}
}

func BenchmarkGetOrderCacheHit(b *testing.B) {
	log.SetOutput(io.Discard)

	orders := generator.GenerateOrders(10000)
	repo := repository.NewRepository(orders)
	handler := NewHandler(repo)

	req := httptest.NewRequest(http.MethodGet, "/order?id=order-1", nil)
	w := newBenchmarkResponseWriter()

	// Прогрев: первый запрос кладёт заказ в cache.
	handler.GetOrder(w, req)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		w.Reset()

		handler.GetOrder(w, req)

		if w.code != http.StatusOK {
			b.Fatalf("expected status %d, got %d", http.StatusOK, w.code)
		}
	}
}
