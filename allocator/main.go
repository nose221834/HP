package main

import (
	"log/slog"
	"net/http"
	"os"
)

func main() {
	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":9090"
	}

	allowedOrigin := os.Getenv("ALLOWED_ORIGIN")
	if allowedOrigin == "" {
		allowedOrigin = "http://localhost:3000"
	}

	wsPublicURL := os.Getenv("WS_PUBLIC_URL")
	if wsPublicURL == "" {
		wsPublicURL = "ws://localhost:9090"
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// ─── Discovery の初期化 ─────────────────────────────────────
	// DISCOVERY_MODE=static → StaticDiscovery（開発用）
	// DISCOVERY_MODE=k8s    → K8sDiscovery（本番用）
	mode := os.Getenv("DISCOVERY_MODE")
	if mode == "" {
		mode = "static" // デフォルトは開発用
	}

	var discovery PodDiscovery
	switch mode {
	case "static":
		d, err := NewStaticDiscovery()
		if err != nil {
			logger.Error("failed to init static discovery", "err", err)
			os.Exit(1)
		}
		discovery = d
		logger.Info("using static discovery")
	case "k8s":
		d, err := NewK8sDiscovery()
		if err != nil {
			logger.Error("failed to init k8s discovery", "err", err)
			os.Exit(1)
		}
		discovery = d
		logger.Info("using k8s discovery")
	default:
		logger.Error("unknown DISCOVERY_MODE", "mode", mode)
		os.Exit(1)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/allocate", allocateHandler(logger, discovery, wsPublicURL))
	mux.HandleFunc("/ws", wsProxyHandler(logger, allowedOrigin))

	logger.Info("starting allocator", "addr", addr, "mode", mode)
	if err := http.ListenAndServe(addr, mux); err != nil {
		logger.Error("server error", "err", err)
		os.Exit(1)
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}
