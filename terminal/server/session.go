package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
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

	// 初期プロンプトを読み取る（非同期）
	go session.readInitialPrompt()

	return session, nil
}

// readInitialPrompt は初期プロンプトを読み取ってセッションを準備状態にする
func (s *Session) readInitialPrompt() {
	// 初期プロンプトが表示されるまで少し待機
	time.Sleep(100 * time.Millisecond)
	
	// 初期プロンプトを読み取る（バッファをクリア）
	buffer := make([]byte, 1024)
	s.Stdout.Read(buffer)
}

// ExecuteCommandInSession は継続的なシェルセッションでコマンドを実行
func (s *Session) ExecuteCommandInSession(command string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// コマンドをシェルに送信
	_, err := fmt.Fprintf(s.Stdin, "%s\n", command)
	if err != nil {
		return "", fmt.Errorf("コマンド送信エラー: %v", err)
	}

	// 出力を読み取る
	output, err := s.readOutput()
	if err != nil {
		return "", fmt.Errorf("出力読み取りエラー: %v", err)
	}

	return output, nil
}

// readOutput はシェルからの出力を読み取る
func (s *Session) readOutput() (string, error) {
	var output strings.Builder
	
	// 非ブロッキングで出力を読み取る
	done := make(chan bool)
	var readErr error
	
	go func() {
		defer close(done)
		
		// 標準出力と標準エラー出力を同時に読み取る
		stdoutDone := make(chan bool)
		stderrDone := make(chan bool)
		
		// 標準出力の読み取り
		go func() {
			defer close(stdoutDone)
			scanner := bufio.NewScanner(s.Stdout)
			promptFound := false
			
			for scanner.Scan() {
				line := scanner.Text()
				output.WriteString(line + "\n")
				
				// プロンプトが表示されたら読み取り終了
				if strings.Contains(line, "$ ") || strings.Contains(line, "# ") {
					promptFound = true
					break
				}
			}
			
			// プロンプトが見つからない場合は少し待機して再試行
			if !promptFound {
				time.Sleep(100 * time.Millisecond)
				// バッファに残っている出力を読み取る
				buffer := make([]byte, 1024)
				n, _ := s.Stdout.Read(buffer)
				if n > 0 {
					output.WriteString(string(buffer[:n]))
				}
			}
		}()
		
		// 標準エラー出力の読み取り
		go func() {
			defer close(stderrDone)
			scanner := bufio.NewScanner(s.Stderr)
			for scanner.Scan() {
				line := scanner.Text()
				output.WriteString(line + "\n")
			}
		}()
		
		// 両方の読み取りが完了するまで待機
		<-stdoutDone
		<-stderrDone
	}()
	
	// タイムアウト付きで待機
	select {
	case <-done:
		return output.String(), readErr
	case <-time.After(30 * time.Second): // 30秒タイムアウト
		return output.String(), fmt.Errorf("コマンド実行タイムアウト")
	}
}

// CloseSession はセッションを終了する
func (s *Session) CloseSession() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if s.Shell != nil && s.Shell.Process != nil {
		return s.Shell.Process.Kill()
	}
	return nil
}
