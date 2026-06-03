package main

import (
	"log"
	"net/http"

	"l45/internal/generator"
	"l45/internal/httpapi"
	"l45/internal/repository"

	_ "net/http/pprof"
)

func main() {
	orders := generator.GenerateOrders(10000)

	repo := repository.NewRepository(orders)
	handler := httpapi.NewHandler(repo)

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	go func() {
		pprofAddr := "localhost:6060"

		log.Printf("pprof server started on http://%s/debug/pprof/", pprofAddr)

		if err := http.ListenAndServe(pprofAddr, nil); err != nil {
			log.Printf("pprof server error: %v", err)
		}
	}()

	addr := ":8080"

	log.Printf("api server started on %s", addr)
	log.Printf("try: curl http://localhost%s/order?id=order-1", addr)

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}
