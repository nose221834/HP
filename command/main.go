package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
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
	Pwd     string `json:"pwd,omitempty"`
}

// グローバル変数として現在のディレクトリを保持
var currentDir = "/home/nonroot"

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
				Pwd:     currentDir,
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
	// コマンドを解析して、cdコマンドの場合は特別に処理
	parts := strings.Fields(cmd)
	if len(parts) > 0 && parts[0] == "cd" {
		if len(parts) == 1 {
			// cd のみの場合はホームディレクトリに移動
			currentDir = "/home/nonroot"
			return CommandResult{
				Status:  "success",
				Command: cmd,
				Result:  "",
				Pwd:     currentDir,
			}, nil
		}

		// 新しいディレクトリパスを計算
		newDir := parts[1]
		if !strings.HasPrefix(newDir, "/") {
			// 相対パスの場合
			newDir = filepath.Join(currentDir, newDir)
		}
		newDir = filepath.Clean(newDir)

		// ディレクトリの存在確認
		if _, err := os.Stat(newDir); os.IsNotExist(err) {
			return CommandResult{
				Status:  "error",
				Command: cmd,
				Error:   fmt.Sprintf("ディレクトリが存在しません: %s", newDir),
				Pwd:     currentDir,
			}, nil
		}

		currentDir = newDir
		return CommandResult{
			Status:  "success",
			Command: cmd,
			Result:  "",
			Pwd:     currentDir,
		}, nil
	}

	// 通常のコマンド実行
	cmdObj := exec.Command("bash", "-c", cmd)
	cmdObj.Dir = currentDir // 現在のディレクトリを設定
	output, err := cmdObj.CombinedOutput()
	outputStr := strings.TrimSpace(string(output))

	if err != nil {
		log.Printf("コマンド実行エラー: %v, 出力: %s", err, outputStr)
		return CommandResult{
			Status:  "error",
			Command: cmd,
			Error:   fmt.Sprintf("コマンド実行エラー: %v", err),
			Result:  outputStr,
			Pwd:     currentDir,
		}, nil
	}

	result := CommandResult{
		Status:  "success",
		Command: cmd,
		Result:  outputStr,
		Pwd:     currentDir,
	}
	log.Printf("コマンド実行成功: %+v", result)
	return result, nil
}
