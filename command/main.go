package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	commandChannel = "terminal:commands"
	resultChannel  = "terminal:results"
)

type CommandResult struct {
	Status  string `json:"status"`
	Command string `json:"command"`
	Result  string `json:"result,omitempty"`
	Error   string `json:"error,omitempty"`
}

func main() {
	// Redisクライアントの設定
	redisOpts := &redis.Options{
		Addr:     "redis:6379",
		Password: "password",
		DB:       0,
	}
	log.Printf("Redis接続設定: %+v", redisOpts)

	rdb := redis.NewClient(redisOpts)
	ctx := context.Background()

	// Redisの接続確認
	for i := 0; i < 5; i++ {
		err := rdb.Ping(ctx).Err()
		if err == nil {
			log.Println("Redis接続成功")
			break
		}
		log.Printf("Redis接続試行 %d/5 失敗: %v", i+1, err)
		if i < 4 {
			time.Sleep(time.Second * 2)
		} else {
			log.Fatalf("Redis接続に失敗しました: %v", err)
		}
	}

	// コマンドチャンネルを購読
	log.Printf("コマンドチャンネル '%s' の購読を開始", commandChannel)
	pubsub := rdb.Subscribe(ctx, commandChannel)
	defer pubsub.Close()

	// 購読状態の確認
	ch := pubsub.Channel()
	log.Println("チャンネル購読準備完了")

	// メッセージ受信ループ
	for msg := range ch {
		log.Printf("メッセージを受信: %s", msg.Payload)

		// コマンドを実行
		command := msg.Payload
		result, err := executeCommand(command)
		if err != nil {
			log.Printf("コマンド実行エラー: %v", err)
			result = CommandResult{
				Status:  "error",
				Command: command,
				Error:   fmt.Sprintf("実行エラー: %v", err),
			}
		}

		// 結果をJSONに変換
		jsonResult, err := json.Marshal(result)
		if err != nil {
			log.Printf("JSON変換エラー: %v", err)
			continue
		}

		// 結果をRedisに送信
		log.Printf("結果を送信: %s", string(jsonResult))
		err = rdb.Publish(ctx, resultChannel, string(jsonResult)).Err()
		if err != nil {
			log.Printf("結果送信エラー: %v", err)
		} else {
			log.Println("結果送信完了")
		}
	}
}

func executeCommand(cmd string) (CommandResult, error) {
	// コマンドの実行
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return CommandResult{
			Status:  "error",
			Command: cmd,
			Error:   "空のコマンド",
		}, nil
	}

	// 許可されたコマンドのチェック
	allowedCommands := map[string]bool{
		"ls":     true,
		"pwd":    true,
		"whoami": true,
		"date":   true,
	}

	if !allowedCommands[parts[0]] {
		return CommandResult{
			Status:  "error",
			Command: cmd,
			Error:   "許可されていないコマンドです",
		}, nil
	}

	// コマンドを実行
	log.Printf("コマンドを実行: %s", cmd)
	cmdObj := exec.Command(parts[0], parts[1:]...)
	output, err := cmdObj.CombinedOutput()
	outputStr := strings.TrimSpace(string(output))

	if err != nil {
		log.Printf("コマンド実行エラー: %v, 出力: %s", err, outputStr)
		return CommandResult{
			Status:  "error",
			Command: cmd,
			Error:   fmt.Sprintf("コマンド実行エラー: %v", err),
			Result:  outputStr, // エラー時も出力を含める
		}, nil
	}

	// 成功時の結果
	result := CommandResult{
		Status:  "success",
		Command: cmd,
		Result:  outputStr,
	}

	log.Printf("コマンド実行成功: %+v", result)
	return result, nil
}
