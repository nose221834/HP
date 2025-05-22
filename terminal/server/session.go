package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// グローバルなセッションマネージャーインスタンス
var sessionManager = NewSessionManager()

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

	// 現在のユーザー名を取得
	whoamiCmd := exec.Command("whoami")
	username, err := whoamiCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("whoami error: %v", err)
	}

	// 新しいbashシェルプロセスを作成
	shell := exec.Command("bash", "-l")
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

	session := &Session{
		ID:          sessionID,
		CurrentDir:  "/home/nonroot", // デフォルトの作業ディレクトリ
		PreviousDir: "/home/nonroot", // 初期値は現在のディレクトリと同じ
		Username:    strings.TrimSpace(string(username)), // ユーザー名を設定
		Shell:       shell,
		Stdin:       stdin.(*os.File),
		Stdout:      stdout.(*os.File),
		Stderr:      stderr.(*os.File),
	}

	// セッションをマップに登録
	sm.sessions[sessionID] = session
	return session, nil
}
