package main

import (
	"log/slog"
	"net/http"
	"os"
)

func main() {
	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":8080"
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout,&slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	mux := http.NewServeMux()
	mux.HandleFunc("/health",healthHandler)
	mux.HandleFunc("/ws",wsHandler(logger))

	logger.Info("starting terminal server", "addr", addr)
	if err := http.ListenAndServe(addr,mux); err != nil {
		logger.Error("server error", "err", err)
		os.Exit(1)
	}
}



