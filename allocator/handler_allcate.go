package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
)

// ─── レスポンス型 ─────────────────────────────────────────────────

type allocateResponse struct {
	Token string `json:"token"`
	WSUrl string `json:"ws_url"`
}

type errorResponse struct {
	Message string `json:"message"`
}

// ─── /allocate ハンドラ ───────────────────────────────────────────

func allocateHandler(logger *slog.Logger, discovery PodDiscovery, wsPublicURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// GET のみ許可
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		// terminal Pod の URL一覧を取得（Static or k8s）
		podURLs, err := discovery.ListPodURLs(ctx)
		if err != nil {
			logger.Error("failed to list pods", "err", err)
			writeError(w, http.StatusInternalServerError, "failed to list pods")
			return
		}

		// 各Podの /status を確認して空きPodを探す
		podURL := findAvailablePod(ctx, logger, podURLs)
		if podURL == "" {
			// 空きなし → 満員
			logger.Info("no available pods")
			writeError(w, http.StatusServiceUnavailable, "no available terminals")
			return
		}

		// 空きPodのURLをJWTに埋め込んで発行
		token, err := issueToken(podURL)
		if err != nil {
			logger.Error("failed to issue token", "err", err)
			writeError(w, http.StatusInternalServerError, "failed to issue token")
			return
		}

		logger.Info("allocated pod", "pod_url", podURL)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(allocateResponse{
			Token: token,
			// フロントエンドが実際に接続するWS URL（公開URL）
			WSUrl: wsPublicURL + "/ws",
		})
	}
}

// ─── 空き Pod を探す ─────────────────────────────────────────────
// Pod一覧に対して並列で /status を叩き、最初に見つかった空きPodを返す

func findAvailablePod(ctx context.Context, logger *slog.Logger, podURLs []string) string {
	type result struct {
		url string
	}
	ch := make(chan result, len(podURLs))

	for _, url := range podURLs {
		url := url
		go func() {
			if checkPodAvailable(ctx, url) {
				ch <- result{url: url}
			} else {
				ch <- result{url: ""}
			}
		}()
	}

	// 全Podの結果を集める
	for range podURLs {
		r := <-ch
		if r.url != "" {
			return r.url
		}
	}
	return ""
}

// ─── Pod の /status を確認 ───────────────────────────────────────

func checkPodAvailable(ctx context.Context, podURL string) bool {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, podURL+"/status", nil)
	if err != nil {
		return false
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	// 200 OK = 空き、503 = 使用中
	return resp.StatusCode == http.StatusOK
}

// ─── ユーティリティ ───────────────────────────────────────────────

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(errorResponse{Message: message})
}
