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

// Redisのチャンネル名定義
// コマンド受信用と結果送信用の2つのチャンネルを使用
const (
	commandChannel = "terminal:commands" // クライアントからのコマンドを受信するチャンネル
	resultChannel  = "terminal:results"  // コマンド実行結果を送信するチャンネル
)

// CommandResult は、コマンド実行の結果を表す構造体
// Redisを通じてクライアントに返される形式
// 各フィールドはJSONとしてシリアライズされる
type CommandResult struct {
	Status    string `json:"status"`    // 実行結果のステータス（success/error）
	Command   string `json:"command"`   // 実行されたコマンド
	Result    string `json:"result,omitempty"`    // コマンドの出力結果（エラー時は空）
	Error     string `json:"error,omitempty"`     // エラーメッセージ（エラー時のみ）
	Pwd       string `json:"pwd,omitempty"`       // 現在の作業ディレクトリ
	SessionID string `json:"session_id,omitempty"` // セッション識別子（クライアント識別用）
}

// Session は、各クライアントのシェルセッションを管理する構造体
// 各セッションは独自のシェルプロセスと入出力パイプを持つ
// これにより、クライアントごとに独立したシェル環境を提供
type Session struct {
	ID            string      // セッションの一意識別子（UUID）
	CurrentDir    string      // 現在の作業ディレクトリ（cdコマンドで変更可能）
	PreviousDir   string      // 直前の作業ディレクトリ（cd -コマンド用）
	Shell         *exec.Cmd   // 実行中のシェルプロセス（bash）
	Stdin         *os.File    // 標準入力パイプ（コマンド入力用）
	Stdout        *os.File    // 標準出力パイプ（コマンド出力用）
	Stderr        *os.File    // 標準エラー出力パイプ（エラー出力用）
	mu            sync.Mutex  // セッション操作の排他制御用ミューテックス（同時実行制御）
}

// SessionManager は、複数のセッションを管理する構造体
// セッションの作成、取得、終了を担当
// スレッドセーフな操作を保証するため、RWMutexで保護
type SessionManager struct {
	sessions map[string]*Session // セッションIDをキーとするセッションマップ
	mu       sync.RWMutex       // セッションマップの排他制御用ミューテックス
}

// NewSessionManager は新しいSessionManagerインスタンスを作成
// セッションマップを初期化して返す
func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[string]*Session),
	}
}

// GetSession は指定されたIDのセッションを取得
// セッションが存在しない場合は新規作成
// 読み取りロックを使用して並行アクセスを最適化
func (sm *SessionManager) GetSession(sessionID string) (*Session, error) {
	// 読み取りロックでセッションの存在確認
	sm.mu.RLock()
	session, exists := sm.sessions[sessionID]
	sm.mu.RUnlock()

	if !exists {
		// セッションが存在しない場合は新規作成
		return sm.createSession(sessionID)
	}
	return session, nil
}

// createSession は新しいシェルセッションを作成
// シェルプロセスの起動と入出力パイプの設定を行う
// 二重チェックロックパターンを使用して並行性を制御
func (sm *SessionManager) createSession(sessionID string) (*Session, error) {
	// 書き込みロックを取得
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// 二重チェック：ロック取得後に再度存在確認
	// これにより、並行して作成された場合の重複を防止
	if session, exists := sm.sessions[sessionID]; exists {
		return session, nil
	}

	// 新しいbashシェルプロセスを作成
	shell := exec.Command("bash")
	// ターミナルエミュレーションの設定
	shell.Env = append(os.Environ(), "TERM=xterm-256color")

	// 入出力パイプの設定
	// 各パイプはos.File型にキャストして使用
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

	// シェルプロセスを開始
	if err := shell.Start(); err != nil {
		return nil, fmt.Errorf("shell start error: %v", err)
	}

	// 新しいセッションを作成
	session := &Session{
		ID:          sessionID,
		CurrentDir:  "/home/nonroot", // デフォルトの作業ディレクトリ
		PreviousDir: "/home/nonroot", // 初期値は現在のディレクトリと同じ
		Shell:       shell,
		Stdin:       stdin.(*os.File),
		Stdout:      stdout.(*os.File),
		Stderr:      stderr.(*os.File),
	}

	// セッションをマップに登録
	sm.sessions[sessionID] = session
	return session, nil
}

// CloseSession は指定されたIDのセッションを終了
// シェルプロセスの終了とセッションの削除を行う
func (sm *SessionManager) CloseSession(sessionID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if session, exists := sm.sessions[sessionID]; exists {
		// シェルプロセスが存在する場合は強制終了
		if session.Shell != nil && session.Shell.Process != nil {
			session.Shell.Process.Kill()
		}
		// セッションをマップから削除
		delete(sm.sessions, sessionID)
	}
}

// グローバルなセッションマネージャーインスタンス
var sessionManager = NewSessionManager()

// main はアプリケーションのエントリーポイント
// Redisとの接続確立とコマンド処理ループを開始
func main() {
	// Redisクライアントの設定
	// コマンドの受信と結果の送信に使用
	redisOpts := &redis.Options{
		Addr:     "redis:6379",  // Redisサーバーのアドレス
		Password: "password",    // Redisのパスワード
		DB:       0,            // 使用するDB番号
	}
	log.Printf("Redis接続設定: %+v", redisOpts)

	// Redisクライアントの作成
	rdb := redis.NewClient(redisOpts)
	ctx := context.Background()

	// Redisの接続確認
	// 最大5回まで再接続を試行
	for i := 0; i < 5; i++ {
		err := rdb.Ping(ctx).Err()
		if err == nil {
			log.Println("Redis接続成功")
			break
		}
		log.Printf("Redis接続試行 %d/5 失敗: %v", i+1, err)
		if i < 4 {
			time.Sleep(time.Second * 2) // 2秒待機して再試行
		} else {
			log.Fatalf("Redis接続に失敗しました: %v", err)
		}
	}

	// コマンドチャンネルの購読開始
	log.Printf("コマンドチャンネル '%s' の購読を開始", commandChannel)
	pubsub := rdb.Subscribe(ctx, commandChannel)
	defer pubsub.Close()

	// 購読状態の確認とチャンネルの取得
	ch := pubsub.Channel()
	log.Println("チャンネル購読準備完了")

	// メッセージ受信ループ
	// Redisのコマンドチャンネルからメッセージを受信し続ける
	for msg := range ch {
		log.Printf("メッセージを受信: %s", msg.Payload)

		// 受信したJSONメッセージをパース
		var commandData struct {
			Command   string `json:"command"`    // 実行するコマンド
			SessionID string `json:"session_id"` // セッションID
		}

		if err := json.Unmarshal([]byte(msg.Payload), &commandData); err != nil {
			log.Printf("JSONパースエラー: %v, ペイロード: %s", err, msg.Payload)
			continue
		}

		// コマンドが空の場合はスキップ
		if commandData.Command == "" {
			log.Printf("空のコマンドを受信: %s", msg.Payload)
			continue
		}

		// セッションIDが指定されていない場合は新規作成
		if commandData.SessionID == "" {
			commandData.SessionID = uuid.New().String()
			log.Printf("新規セッションIDを生成: %s", commandData.SessionID)
		}

		// コマンドを実行し、結果を取得
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

		// 結果をRedisの結果チャンネルに送信
		log.Printf("結果を送信: %s", string(jsonResult))
		err = rdb.Publish(ctx, resultChannel, string(jsonResult)).Err()
		if err != nil {
			log.Printf("結果送信エラー: %v", err)
		} else {
			log.Println("結果送信完了")
		}
	}
}

// executeCommand は、指定されたコマンドを実行し、結果を返す
// cdコマンドは特別に処理され、セッションの現在ディレクトリを更新
// その他のコマンドは、セッションの現在ディレクトリで実行される
func executeCommand(cmd string, sessionID string) (CommandResult, error) {
	// セッションの取得（存在しない場合は新規作成）
	session, err := sessionManager.GetSession(sessionID)
	if err != nil {
		return CommandResult{
			Status:    "error",
			Command:   cmd,
			Error:     fmt.Sprintf("セッションエラー: %v", err),
			SessionID: sessionID,
		}, nil
	}

	// セッションの排他制御
	session.mu.Lock()
	defer session.mu.Unlock()

	// コマンドを空白で分割して解析
	parts := strings.Fields(cmd)
	if len(parts) > 0 && parts[0] == "cd" {
		// cdコマンドの特別処理
		if len(parts) == 1 {
			// cdのみの場合はデフォルトディレクトリに移動
			session.PreviousDir = session.CurrentDir // 現在のディレクトリを保存
			session.CurrentDir = "/home/nonroot"
			return CommandResult{
				Status:    "success",
				Command:   cmd,
				Result:    "",
				Pwd:       session.CurrentDir,
				SessionID: sessionID,
			}, nil
		}

		// cd - の特別処理
		if parts[1] == "-" {
			if session.PreviousDir == "" {
				return CommandResult{
					Status:    "error",
					Command:   cmd,
					Error:     "直前のディレクトリがありません",
					Pwd:       session.CurrentDir,
					SessionID: sessionID,
				}, nil
			}
			// 現在のディレクトリと直前のディレクトリを入れ替え
			session.PreviousDir, session.CurrentDir = session.CurrentDir, session.PreviousDir
			return CommandResult{
				Status:    "success",
				Command:   cmd,
				Result:    "",
				Pwd:       session.CurrentDir,
				SessionID: sessionID,
			}, nil
		}

		// 通常のcdコマンド処理
		// 相対パスの場合は現在のディレクトリからの相対パスに変換
		if !strings.HasPrefix(parts[1], "/") {
			parts[1] = filepath.Join(session.CurrentDir, parts[1])
		}
		// パスの正規化（..や.の解決）
		newDir := filepath.Clean(parts[1])

		// ディレクトリの存在確認
		if _, err := os.Stat(newDir); os.IsNotExist(err) {
			return CommandResult{
				Status:    "error",
				Command:   cmd,
				Error:     fmt.Sprintf("ディレクトリが存在しません: %s", newDir),
				Pwd:       session.CurrentDir,
				SessionID: sessionID,
			}, nil
		}

		// ディレクトリの変更
		session.PreviousDir = session.CurrentDir // 現在のディレクトリを保存
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
	// 現在のディレクトリでコマンドを実行
	cmdObj := exec.Command("bash", "-c", fmt.Sprintf("cd %s && %s", session.CurrentDir, cmd))
	output, err := cmdObj.CombinedOutput()
	outputStr := strings.TrimSpace(string(output))

	if err != nil {
		// エラー発生時の処理
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

	// 成功時の結果を返却
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
