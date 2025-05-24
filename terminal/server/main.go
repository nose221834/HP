package main

import (
	"context"
	"fmt"
	"log"
)

// 定数を定義
const (
	commandChannel = "terminal:commands"	// コマンド受信用チャンネル
	resultChannel  = "terminal:results"	// 結果送信用チャンネル
)

// main はアプリケーションのエントリーポイント
// Redisとの接続確立とコマンド処理ループを開始
func main() {

	// コンテキストを作成
	ctx := context.Background()

	// RedisのPub/Subをセットアップ
	ch, rdb := setupRedisPubSub(ctx, commandChannel)
	// deferを使用して、プログラム終了時にRedisクライアントをクローズ
	defer rdb.Close()

	// メッセージを受信するためのループを開始
	for msg := range ch {
		// 受信したメッセージをログに出力
		log.Printf("メッセージを受信: %s", msg.Payload)

		// 受信したメッセージをパース
		// 不正であったり、空のメッセージはスキップ
		payload, err := parsePayload(msg.Payload)
		if err != nil {
			log.Printf("パース失敗: %v", err)
			continue
		}

		// コマンドのバリデーション
		if err := valivateCommand(payload.Command); err != nil {
			log.Printf("コマンドバリデーションエラー: %v", err)
			result := CommandResult{
				Status:    "error",
				Command:   payload.Command,
				Error:     fmt.Sprintf("バリデーションエラー: %v", err),
				SessionID: payload.SessionID,
			}
			// 結果をRedisの結果チャンネルに送信
			if err := publishResult(ctx, rdb, &result); err != nil {
				log.Printf("結果のパブリッシュエラー: %v", err)
			}
			continue
		}

		// コマンドを実行し、結果を取得
		result, err := executeCommand(payload.Command, payload.SessionID)
		if err != nil {
			log.Printf("コマンド実行エラー: %v", err)
			result = CommandResult{
				Status:    "error",
				Command:   payload.Command,
				Error:     fmt.Sprintf("実行エラー: %v", err),
				SessionID: payload.SessionID,
			}
			if err := publishResult(ctx, rdb, &result); err != nil {
				log.Printf("結果のパブリッシュエラー: %v", err)
			}
			continue
		}

		// コマンドの実行結果をバリデーション
		if err := validateCommandResult(&result); err != nil {
			log.Printf("コマンド結果バリデーションエラー: %v", err)
			result = CommandResult{
				Status:    "error",
				Command:   payload.Command,
				Error:     fmt.Sprintf("バリデーションエラー: %v", err),
				SessionID: payload.SessionID,
			}
			// 結果をRedisの結果チャンネルに送信
			if err := publishResult(ctx, rdb, &result); err != nil {
				log.Printf("結果のパブリッシュエラー: %v", err)
			}
			continue
		}

		// 結果をRedisの結果チャンネルに送信
		if err := publishResult(ctx, rdb, &result); err != nil {
			log.Printf("結果のパブリッシュエラー: %v", err)
			continue
		}
	}
}
