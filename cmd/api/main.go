package main

import (
	"log"
	"net/http"

	"l45/internal/generator"
	"l45/internal/httpapi"
	"l45/internal/repository"
)

func main() {
	orders := generator.GenerateOrders(10000)

	repo := repository.NewRepository(orders)
	handler := httpapi.NewHandler(repo)

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	addr := ":8080"

	log.Printf("api server started on %s", addr)
	log.Printf("try: http://localhost%s/order?id=order-1", addr)

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}
