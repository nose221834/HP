package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

// cdコマンドの特殊な処理を行う関数
// 引数としてセッションとコマンドの分割結果を受け取る
func executeCD(session *Session, parts []string,sessionID string,cmd string) (CommandResult, error) {
	// 引数が1つの場合はデフォルトディレクトリに移動(cdのみ)
	if len(parts) == 1 {
		// cdのみの場合はデフォルトディレクトリに移動
		session.PreviousDir = session.CurrentDir // 現在のディレクトリを保存
		session.CurrentDir = "/home/nonroot"
		return CommandResult{
			Status:    "success",
			Command:   cmd,
			Result:    "",
			Pwd:       session.CurrentDir,
			Username:  session.Username,  // ユーザー名を結果に含める
			SessionID: sessionID,
		}, nil
	}

	// cd - の特別処理
	if parts[1] == "-" {
		// 直前のディレクトリが空の場合はエラー
		if session.PreviousDir == "" {
			return CommandResult{
				Status:    "error",
				Command:   cmd,
				Error:     "直前のディレクトリがありません",
				Pwd:       session.CurrentDir,
				Username:  session.Username,  // ユーザー名を結果に含める
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
			Username:  session.Username,
			SessionID: sessionID,
		}, nil
	}

	// cd ~ の特別処理
	if strings.HasPrefix(parts[1], "~") {
		// ユーザーのホームディレクトリに移動
		homeDir := os.Getenv("HOME")
		if homeDir == "" {
			return CommandResult{
				Status:    "error",
				Command:   cmd,
				Error:     "ホームディレクトリが取得できません",
				Pwd:       session.CurrentDir,
				Username:  session.Username,  // ユーザー名を結果に含める
				SessionID: sessionID,
			}, nil
		}
		// ~をホームディレクトリに置き換え
		parts[1] = strings.Replace(parts[1], "~", homeDir, 1)
	}

	// 通常のcdコマンド処理
	// 相対パスの場合は現在のディレクトリからの相対パスに変換
	if !strings.HasPrefix(parts[1], "/") {
		parts[1] = filepath.Join(session.CurrentDir, parts[1])
	}
	// パスの正規化（..や.の解決）
	newDir := filepath.Clean(parts[1])

	// ディレクトリの存在確認
	fileInfo, err := os.Stat(newDir)
	if err != nil {
		return CommandResult{
			Status:    "error",
			Command:   cmd,
			Error:     fmt.Sprintf("ディレクトリが存在しません: %s", newDir),
			Pwd:       session.CurrentDir,
			Username:  session.Username,  // ユーザー名を結果に含める
			SessionID: sessionID,
		}, nil
	}

	// 入手した情報がディレクトリかどうかを確認
	if !fileInfo.IsDir() {
		return CommandResult{
			Status:    "error",
			Command:   cmd,
			Error:     fmt.Sprintf("ディレクトリが存在しません: %s", newDir),
			Pwd:       session.CurrentDir,
			Username:  session.Username,  // ユーザー名を結果に含める
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
		Username:  session.Username,  // ユーザー名を結果に含める
		SessionID: sessionID,
	}, nil
}

// 通常のコマンドを実行する関数
// 引数としてセッションとコマンドの分割結果を受け取る
func executeNormalCommand(session *Session,sessionID string,cmd string) (CommandResult, error) {
		cmdObj := exec.Command("bash","-l", "-c", fmt.Sprintf("cd %s && %s", session.CurrentDir, cmd))
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
			Username:  session.Username,  // ユーザー名を結果に含める
			SessionID: sessionID,
		}, nil
	}

	// 成功時の結果を返却
	result := CommandResult{
		Status:    "success",
		Command:   cmd,
		Result:    outputStr,
		Pwd:       session.CurrentDir,
		Username:  session.Username,  // ユーザー名を結果に含める
		SessionID: sessionID,
	}
	log.Printf("コマンド実行成功: %+v", result)
	return result, nil
}


// executeCommand は、指定されたコマンドを実行し、結果を返す
// cdコマンドは特別に処理され、セッションの現在ディレクトリを更新
// その他のコマンドは、セッションの現在ディレクトリで実行される
func executeCommand(cmd string, sessionID string) (CommandResult, error) {

	// セッションIDが指定されていない場合は新規作成
	if sessionID == "" {
		sessionID = uuid.New().String()
		log.Printf("新規セッションIDを生成: %s", sessionID)
	}

	// セッションの取得
	session, err := sessionManager.GetSession(sessionID)
	if err != nil {
		return CommandResult{
			Status:    "error",
			Command:   cmd,
			Error:     "セッションエラー: " + err.Error(),
			SessionID: sessionID,
		}, nil
	}

	// セッションの排他制御
	session.mu.Lock()
	defer session.mu.Unlock()

	// コマンドを空白で分割して解析
	parts := strings.Fields(cmd)

	// cdコマンドの処理
	if len(parts) > 0 && parts[0] == "cd" {
		return executeCD(session, parts, sessionID, cmd)
	}

	// 通常のコマンド実行
	// 現在のディレクトリでコマンドを実行
	return executeNormalCommand(session, sessionID, cmd)
}
