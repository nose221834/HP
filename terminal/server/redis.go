package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// Redisのクライアント設定をbuild時に指定するための変数
// TODO: 以下のように指定してbuildする
// go build -ldflags "-X main.redisHost=localhost -X main.redisPort=6379" -o myapp main.go
var (
	redisHost string 	// Redisサーバのホスト名
	redisPort string 	// Redisサーバのポート番号
	redisPass string	// Redisサーバのパスワード
	redisDB   string 	// RedisサーバのDB番号
)


// build引数の初期化を行う関数
// この関数はmainよりも前に実行される
func init() {
	if redisHost == "" {
		redisHost = "redis"
	}
	if redisPort == "" {
		redisPort = "6379"
	}
	if redisPass == "" {
		redisPass = "password"
	}
	if redisDB == "" {
		// デフォルトのDB番号を指定
		redisDB = "0"
	}
}

func setupRedisPubSub(ctx context.Context, commandChannel string) (<-chan *redis.Message, *redis.Client) {
	// 定数を定義
	const (
		maxRetries = 5	// Redis接続の最大リトライ回数
	)

	// redisDB（string）を int に変換
	dbNum, err := strconv.Atoi(redisDB)
	if err != nil {
		log.Fatalf("REDIS_DB の変換に失敗しました: %v", err)
	}

	// Redisクライアントの設定
	redisOpts := &redis.Options{
		Addr:     redisHost + ":" + redisPort, // Redisサーバのアドレス
		Password: redisPass,                   // パスワード
		DB:       dbNum,                       // DB番号（int型）
	}


	log.Printf("Redis接続設定: %+v", redisOpts)

	// Redisクライアントの作成
	// redis.Optionsを引数に取る
	rdb := redis.NewClient(redisOpts)

	// Redisクライアントと接続
	// 最大5回まで再接続を試行
	// pingを使用して接続確認
	for i := 1; i <= maxRetries; i++ {
		err := rdb.Ping(ctx).Err()
		if err == nil {
			log.Println("Redis接続成功")
			break
		}
		log.Printf("Redis接続試行 %d/%d 失敗: %v", i, maxRetries, err)
		time.Sleep(2 * time.Second)
		if i == maxRetries {
			log.Fatalf("Redis接続に失敗しました: %v", err)
		}
	}

	// Redisのコマンドを受信するためのチャンネルを購読開始
	log.Printf("コマンドチャンネル '%s' の購読を開始", commandChannel)
	// サブスクライバーを作成
	// redisのサブスクライブチャンネルとは、誰かがメッセージをパブリッシュしたときにそのメッセージを受信するためのチャンネル
	pubsub:= rdb.Subscribe(ctx, commandChannel)
	// 関数終了時にサブスクライバーを閉じる

	// 購読状態の確認とチャンネルの取得
	ch := pubsub.Channel()

	return ch, rdb
}

// publishResult はコマンドの実行結果をRedisにパブリッシュする関数
// 引数にはコンテキスト、Redisクライアント、コマンドの実行結果を受け取る
func publishResult(ctx context.Context,rdb *redis.Client, result *CommandResult) error {
	// コマンドの実行結果をJSON形式に変換
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("JSON変換エラー: %w", err)
	}

	// Redisの結果チャンネルに送信
	log.Printf("結果を送信: %s", string(resultJSON))
	err = rdb.Publish(ctx, resultChannel, string(resultJSON)).Err()
	if err != nil {
		log.Printf("結果送信エラー: %v", err)
	} else {
		log.Println("結果送信完了")
	}

	return nil
}
