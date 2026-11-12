// Command api 是 go-shop 的 HTTP API 進入點。
//
// 所有依賴都在 main 手動組裝：沒有 DI 容器，接線一眼看完。
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

	// 由內往外 new：repository → service → handler。
	repo := repository.NewMemoryProducts()
	svc := service.NewProductService(repo)
	h := handler.New(svc, logger)

	logger.Info("api listening", "port", port)
	if err := http.ListenAndServe(":"+port, h); err != nil {
		logger.Error("server stopped", "err", err)
		os.Exit(1)
	}
}
