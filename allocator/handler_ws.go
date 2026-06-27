package main

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/coder/websocket"
)

// ─── /ws ハンドラ（WSリバースプロキシ）──────────────────────────
// 1. クエリパラメータからJWTを取得して検証
// 2. JWTに埋め込まれたPod URLにWSで接続
// 3. ブラウザ ↔ Pod 間のメッセージを双方向に中継する

func wsProxyHandler(logger *slog.Logger, allowedOrigin string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// ── JWT検証 ───────────────────────────────────────────────
		tokenString := r.URL.Query().Get("token")
		if tokenString == "" {
			http.Error(w, "token required", http.StatusUnauthorized)
			return
		}

		podURL, err := verifyToken(tokenString)
		if err != nil {
			logger.Warn("invalid token", "err", err)
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}

		// ── ブラウザ側のWSアップグレード ──────────────────────────
		clientConn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			OriginPatterns: []string{allowedOrigin},
		})
		if err != nil {
			logger.Error("failed to accept websocket", "err", err)
			return
		}
		defer clientConn.CloseNow()

		logger.Info("ws proxy started", "pod_url", podURL)

		// ── Pod側のWSに接続 ──────────────────────────────────────
		// http:// → ws:// に変換
		podWsURL := strings.Replace(podURL, "http://", "ws://", 1) + "/ws"

		podConn, _, err := websocket.Dial(r.Context(), podWsURL, nil)
		if err != nil {
			logger.Error("failed to connect to pod", "err", err, "pod_url", podWsURL)
			clientConn.Close(websocket.StatusInternalError, "failed to connect to terminal")
			return
		}
		defer podConn.CloseNow()

		ctx := r.Context()
		errCh := make(chan error, 2)

		// ── ブラウザ → Pod（入力・リサイズ） ─────────────────────
		go func() {
			for {
				msgType, data, err := clientConn.Read(ctx)
				if err != nil {
					errCh <- err
					return
				}
				if err := podConn.Write(ctx, msgType, data); err != nil {
					errCh <- err
					return
				}
			}
		}()

		// ── Pod → ブラウザ（bash出力） ────────────────────────────
		go func() {
			for {
				msgType, data, err := podConn.Read(ctx)
				if err != nil {
					errCh <- err
					return
				}
				if err := clientConn.Write(ctx, msgType, data); err != nil {
					errCh <- err
					return
				}
			}
		}()

		// どちらかが切断したらプロキシ終了
		<-errCh
		logger.Info("ws proxy ended", "pod_url", podURL)
	}
}
