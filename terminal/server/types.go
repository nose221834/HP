package main

import (
	"os"
	"os/exec"
	"sync"
)

// redisからのメッセージを受信するための
type Payload struct {
	Command     string `json:"command"`     // コマンド
	SessionID   string `json:"session_id"`  // セッションID
}

// Session は、各クライアントのシェルセッションを管理する構造体
// 各セッションは独自のシェルプロセスと入出力パイプを持つ
// これにより、クライアントごとに独立したシェル環境を提供
type Session struct {
	ID            string      // セッションの一意識別子（UUID）
	CurrentDir    string      // 現在の作業ディレクトリ（cdコマンドで変更可能）
	PreviousDir   string      // 直前の作業ディレクトリ（cd -コマンド用）
	Username      string      // 現在のユーザー名
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
	sessions map[string]*Session 	// セッションIDをキーとするセッションマップ
	mu       sync.RWMutex       	// セッションマップの排他制御用ミューテックス
}

// CommandResult は、コマンド実行の結果を表す構造体
// Redisを通じてクライアントに返される形式
// 各フィールドはJSONとしてシリアライズされる
type CommandResult struct {
	Status    string `json:"status"`    			// 実行結果のステータス（success/error）
	Command   string `json:"command"`   			// 実行されたコマンド
	Result    string `json:"result,omitempty"`    	// コマンドの出力結果（エラー時は空）
	Error     string `json:"error,omitempty"`     	// エラーメッセージ（エラー時のみ）
	Pwd       string `json:"pwd,omitempty"`       	// 現在の作業ディレクトリ
	Username  string `json:"username,omitempty"`  	// 現在のユーザー名
	SessionID string `json:"session_id,omitempty"` 	// セッション識別子（クライアント識別用）
	Prompt    string `json:"prompt,omitempty"`    	// 現在のプロンプト（ターミナル状態）
}
