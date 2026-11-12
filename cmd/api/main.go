// Command api is the HTTP API entry point of go-shop.
//
// All dependencies are wired by hand in main: no DI container, the whole wiring fits on one screen.
package main

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/newhand31/go-shop/internal/handler"
	"github.com/newhand31/go-shop/internal/repository"
	"github.com/newhand31/go-shop/internal/service"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Build from the inside out: repository → service → handler.
	repo := repository.NewMemoryProducts()
	svc := service.NewProductService(repo)
	h := handler.New(svc, logger)

	logger.Info("api listening", "port", port)
	if err := http.ListenAndServe(":"+port, h); err != nil {
		logger.Error("server stopped", "err", err)
		os.Exit(1)
	}
}
