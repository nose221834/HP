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
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	commandChannel = "terminal:commands"
	resultChannel  = "terminal:results"
)

type CommandResult struct {
	Status    string `json:"status"`
	Command   string `json:"command"`
	Result    string `json:"result,omitempty"`
	Error     string `json:"error,omitempty"`
	Pwd       string `json:"pwd,omitempty"`
	SessionID string `json:"session_id,omitempty"`
}

type Session struct {
	ID         string
	CurrentDir string
	Shell      *exec.Cmd
	Stdin      *os.File
	Stdout     *os.File
	Stderr     *os.File
	mu         sync.Mutex
}

type SessionManager struct {
	sessions map[string]*Session
	mu       sync.RWMutex
}

func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[string]*Session),
	}
}

func (sm *SessionManager) GetSession(sessionID string) (*Session, error) {
	sm.mu.RLock()
	session, exists := sm.sessions[sessionID]
	sm.mu.RUnlock()

	if !exists {
		return sm.createSession(sessionID)
	}
	return session, nil
}

func (sm *SessionManager) createSession(sessionID string) (*Session, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// 二重チェック
	if session, exists := sm.sessions[sessionID]; exists {
		return session, nil
	}

	// 新しいシェルプロセスを作成
	shell := exec.Command("bash")
	shell.Env = append(os.Environ(), "TERM=xterm-256color")

	// パイプを作成
	stdin, err := shell.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("stdin pipe error: %v", err)
	}

	stdout, err := shell.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe error: %v", err)
	}

	stderr, err := shell.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("stderr pipe error: %v", err)
	}

	// シェルを開始
	if err := shell.Start(); err != nil {
		return nil, fmt.Errorf("shell start error: %v", err)
	}

	session := &Session{
		ID:         sessionID,
		CurrentDir: "/home/nonroot",
		Shell:      shell,
		Stdin:      stdin.(*os.File),
		Stdout:     stdout.(*os.File),
		Stderr:     stderr.(*os.File),
	}

	sm.sessions[sessionID] = session
	return session, nil
}

func (sm *SessionManager) CloseSession(sessionID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if session, exists := sm.sessions[sessionID]; exists {
		if session.Shell != nil && session.Shell.Process != nil {
			session.Shell.Process.Kill()
		}
		delete(sm.sessions, sessionID)
	}
}

var sessionManager = NewSessionManager()

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

		var commandData struct {
			Command   string `json:"command"`
			SessionID string `json:"session_id"`
		}

		if err := json.Unmarshal([]byte(msg.Payload), &commandData); err != nil {
			log.Printf("JSONパースエラー: %v", err)
			continue
		}

		// セッションIDが指定されていない場合は新規作成
		if commandData.SessionID == "" {
			commandData.SessionID = uuid.New().String()
		}

		// コマンドを実行
		result, err := executeCommand(commandData.Command, commandData.SessionID)
		if err != nil {
			log.Printf("コマンド実行エラー: %v", err)
			result = CommandResult{
				Status:    "error",
				Command:   commandData.Command,
				Error:     fmt.Sprintf("実行エラー: %v", err),
				SessionID: commandData.SessionID,
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

func executeCommand(cmd string, sessionID string) (CommandResult, error) {
	session, err := sessionManager.GetSession(sessionID)
	if err != nil {
		return CommandResult{
			Status:    "error",
			Command:   cmd,
			Error:     fmt.Sprintf("セッションエラー: %v", err),
			SessionID: sessionID,
		}, nil
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	// cdコマンドの特別処理
	parts := strings.Fields(cmd)
	if len(parts) > 0 && parts[0] == "cd" {
		if len(parts) == 1 {
			session.CurrentDir = "/home/nonroot"
			return CommandResult{
				Status:    "success",
				Command:   cmd,
				Result:    "",
				Pwd:       session.CurrentDir,
				SessionID: sessionID,
			}, nil
		}

		newDir := parts[1]
		if !strings.HasPrefix(newDir, "/") {
			newDir = filepath.Join(session.CurrentDir, newDir)
		}
		newDir = filepath.Clean(newDir)

		if _, err := os.Stat(newDir); os.IsNotExist(err) {
			return CommandResult{
				Status:    "error",
				Command:   cmd,
				Error:     fmt.Sprintf("ディレクトリが存在しません: %s", newDir),
				Pwd:       session.CurrentDir,
				SessionID: sessionID,
			}, nil
		}

		session.CurrentDir = newDir
		return CommandResult{
			Status:    "success",
			Command:   cmd,
			Result:    "",
			Pwd:       session.CurrentDir,
			SessionID: sessionID,
		}, nil
	}

	// 通常のコマンド実行
	cmdObj := exec.Command("bash", "-c", fmt.Sprintf("cd %s && %s", session.CurrentDir, cmd))
	output, err := cmdObj.CombinedOutput()
	outputStr := strings.TrimSpace(string(output))

	if err != nil {
		log.Printf("コマンド実行エラー: %v, 出力: %s", err, outputStr)
		return CommandResult{
			Status:    "error",
			Command:   cmd,
			Error:     fmt.Sprintf("コマンド実行エラー: %v", err),
			Result:    outputStr,
			Pwd:       session.CurrentDir,
			SessionID: sessionID,
		}, nil
	}

	result := CommandResult{
		Status:    "success",
		Command:   cmd,
		Result:    outputStr,
		Pwd:       session.CurrentDir,
		SessionID: sessionID,
	}
	log.Printf("コマンド実行成功: %+v", result)
	return result, nil
}
