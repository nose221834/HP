package main

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"

	"github.com/coder/websocket"
	"github.com/creack/pty"
)

// メッセージ型
//   1. 入力データ (バイナリ): キーストロークをそのまま PTY に流すため型不要
//   2. リサイズ  (テキスト): ターミナルのサイズ変更

type resizeMsg struct {
	Type string `json:"type"`
	Rows uint16 `json:"rows"`
	Cols uint16 `json:"cols"`
}

func wsHandler(logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn,err := websocket.Accept(w,r,&websocket.AcceptOptions{
			InsecureSkipVerify: true,
		})
		if err != nil {
			logger.Error("web socket accept failed", "err", err)
			return
		}
		defer conn.CloseNow()

		logger.Info("websocket connected","remote",r.RemoteAddr)

		if err := runTerminal(r.Context(),conn,logger); err != nil {
			logger.Info("terminal session ended","err",err)
		}
	}
}

func runTerminal(ctx context.Context,conn *websocket.Conn,logger *slog.Logger) error {

	cmd := exec.CommandContext(ctx,"/bin/bash")
	cmd.Env = append(os.Environ(),
		"TERM=xterm-256color",
		"HOME=/home/nonroot",
	)

	ptmx,err := pty.Start(cmd)
	if err != nil {
		return err
	}
	defer func() {
		ptmx.Close()
		cmd.Wait()
	}()

	// セッションコンテキスト: bash 終了または WS 切断で全 goroutine を止める
	ctx,cancel := context.WithCancel(ctx)
	defer cancel()

	errCh := make(chan error, 2)

	// PTY → WebSocket
	go func() {
		buf := make([]byte,4096)
		for {
			n,err := ptmx.Read(buf)
			if n > 0 {
				if werr := conn.Write(ctx,websocket.MessageBinary,buf[:n]); werr != nil {
					errCh <- werr
					return
				}
			}
			if err != nil {
				errCh <- err
			return
			}
		}
	}()

	// WebSocket → PTY
	go func() {
		for {
			msgType,data,err := conn.Read(ctx)
			if err != nil {
				errCh <- err
				return
			}

			switch msgType{
				case websocket.MessageBinary:
					if _,err := ptmx.Write(data); err != nil {
						errCh <- err
						return
					}
				case websocket.MessageText:
					var msg resizeMsg
					if err := json.Unmarshal(data,&msg); err != nil {
						logger.Warn("unknown text message", "data", string(data))
					}
					if msg.Type == "resize" {
						if err := pty.Setsize(ptmx,&pty.Winsize{
							Rows: msg.Rows,
							Cols: msg.Cols,
						}); err != nil {
							logger.Warn("resize failed", "err", err)
						}
					}
				}
		}
	}()

	// どちらかの goroutine が終了したらセッション終了
	err = <-errCh
	if isNormalClose(err) {
		return nil
	}
	return err
}
// isNormalClose は正常なセッション終了（EOF・WS切断）を判定する
func isNormalClose(err error) bool {
	if err == nil || err == io.EOF {
		return true
	}
	return websocket.CloseStatus(err) != -1
}
