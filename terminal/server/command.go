package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// CommandChain は、&&や||で繋がれたコマンドのチェーンを表す
type CommandChain struct {
	Commands []string
	Operators []string // "&&" または "||"
	Results  []CommandResult
}

// parseCommandChain は、&&や||で繋がれたコマンドを解析して分割する
func parseCommandChain(cmd string) *CommandChain {
	// まず&&で分割
	andParts := strings.Split(cmd, "&&")
	var commands []string
	var operators []string

	log.Printf("DEBUG: 元のコマンド: %s", cmd)
	log.Printf("DEBUG: &&で分割: %v", andParts)

	for i, part := range andParts {
		// 各部分を||で分割
		orParts := strings.Split(part, "||")
		log.Printf("DEBUG: 部分%dを||で分割: %v", i, orParts)

		for j, orPart := range orParts {
			trimmedPart := strings.TrimSpace(orPart)
			if trimmedPart != "" {
				commands = append(commands, trimmedPart)

				// 演算子を決定
				if i > 0 && j == 0 {
					// &&で分割された部分の最初は&&演算子
					operators = append(operators, "&&")
				} else if j > 0 {
					// ||で分割された部分の2番目以降は||演算子
					operators = append(operators, "||")
				}
			}
		}
		}

	log.Printf("DEBUG: 最終的なコマンド: %v", commands)
	log.Printf("DEBUG: 最終的な演算子: %v", operators)

	return &CommandChain{
		Commands: commands,
		Operators: operators,
		Results:  make([]CommandResult, 0, len(commands)),
	}
}

// executeCommandChain は、&&や||で繋がれたコマンドを順次実行する
func executeCommandChain(chain *CommandChain, session *Session, sessionID string) (CommandResult, error) {
	var finalResult CommandResult
	var combinedOutput strings.Builder
	var hasError bool
	var shouldContinue bool = true
	var skipUntilOr bool = false

	for i, command := range chain.Commands {
		if command == "" {
			continue
		}

		log.Printf("DEBUG: コマンド%d実行前 - shouldContinue: %v, skipUntilOr: %v, コマンド: %s", i, shouldContinue, skipUntilOr, command)

		// 前のコマンドの結果により実行をスキップするかチェック
		if !shouldContinue {
			// 前のコマンドの結果によりスキップ
			log.Printf("DEBUG: コマンド%dをスキップ", i)

			// skipUntilOrがtrueの場合、次の演算子が||ならshouldContinueをtrueに設定
			if skipUntilOr && i < len(chain.Operators) && chain.Operators[i] == "||" {
				shouldContinue = true
				skipUntilOr = false
				log.Printf("DEBUG: ||演算子検出 - shouldContinueをtrueに設定、skipUntilOrをfalseに設定")
			}

			continue
		}
		// 個別のコマンドを実行
		result, err := executeSingleCommand(command, session, sessionID, true)
		if err != nil {
			log.Printf("コマンドチェーン実行エラー: %v", err)
			return CommandResult{
				Status:    "error",
				Command:   buildCommandString(chain),
				Error:     fmt.Sprintf("コマンドチェーン実行エラー: %v", err),
				SessionID: sessionID,
			}, nil
		}

		log.Printf("DEBUG: コマンド%d実行結果 - Status: %s, Error: %s", i, result.Status, result.Error)

		// 結果を保存
		chain.Results = append(chain.Results, result)

		// 出力を結合
		if result.Result != "" {
			combinedOutput.WriteString(result.Result)
			if !strings.HasSuffix(result.Result, "\n") {
				combinedOutput.WriteString("\n")
			}
		}

		// 演算子に基づいて次のコマンドを実行するか決定
		if i < len(chain.Operators) {
			operator := chain.Operators[i]
			log.Printf("DEBUG: 演算子%d: %s", i, operator)

			if operator == "&&" {
				// &&の場合：エラーが発生したら次のコマンドをスキップ
				if result.Status == "error" {
					shouldContinue = false
					skipUntilOr = true
					hasError = true
					finalResult = result
					log.Printf("DEBUG: &&でエラー - shouldContinueをfalseに設定、skipUntilOrをtrueに設定")
				} else {
					shouldContinue = true
					skipUntilOr = false
					log.Printf("DEBUG: &&で成功 - shouldContinueをtrueに設定、skipUntilOrをfalseに設定")
				}
			} else if operator == "||" {
				// ||の場合：成功したら次のコマンドをスキップ
				if result.Status == "success" {
					shouldContinue = false
					skipUntilOr = false
					finalResult = result
					log.Printf("DEBUG: ||で成功 - shouldContinueをfalseに設定")
				} else {
					// ||で失敗した場合、次のコマンドを実行
					shouldContinue = true
					skipUntilOr = false
					log.Printf("DEBUG: ||で失敗 - shouldContinueをtrueに設定")
				}
			}
		}

		// 最後のコマンドの結果を最終結果として使用
		finalResult = result
	}

	// 最終結果を更新
	finalResult.Command = buildCommandString(chain)
	finalResult.Result = combinedOutput.String()

	if hasError {
		finalResult.Status = "error"
	} else {
		finalResult.Status = "success"
	}

	return finalResult, nil
}

// buildCommandString は、コマンドチェーンを元の文字列形式に復元する
func buildCommandString(chain *CommandChain) string {
	if len(chain.Commands) == 0 {
		return ""
	}

	var result strings.Builder
	result.WriteString(chain.Commands[0])

	for i, operator := range chain.Operators {
		if i+1 < len(chain.Commands) {
			result.WriteString(" ")
			result.WriteString(operator)
			result.WriteString(" ")
			result.WriteString(chain.Commands[i+1])
		}
	}

	return result.String()
}

// executeSingleCommand は、単一のコマンドを実行する（既存のexecuteCommandの機能を分割）
func executeSingleCommand(cmd string, session *Session, sessionID string, isChainPart bool) (CommandResult, error) {
	// コマンドを空白で分割して解析
	parts := strings.Fields(cmd)

	// cdコマンドの処理（コマンドチェーンの一部でない場合のみ特別処理）
	if len(parts) > 0 && parts[0] == "cd" && !isChainPart {
		return executeCD(session, parts, sessionID, cmd)
	}

	// 通常のコマンド実行
	return executeNormalCommand(session, sessionID, cmd)
}

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

	// 終了コードを取得
	var exitCode int
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		} else {
			exitCode = -1
		}
	} else {
		exitCode = 0
	}

	// 実際のシェルの動作に合わせる：
	// - 終了コードが0の場合は成功
	// - 終了コードが0以外でも出力がある場合は成功（エラーメッセージなど）
	// - 終了コードが0以外で出力がない場合は失敗（falseコマンドなど）
	if exitCode == 0 {
		// 成功時の結果を返却
		result := CommandResult{
			Status:    "success",
			Command:   cmd,
			Result:    outputStr,
			Pwd:       session.CurrentDir,
			Username:  session.Username,
			SessionID: sessionID,
		}
		log.Printf("コマンド実行成功: %+v", result)
		return result, nil
	} else if outputStr != "" {
		// 終了コードが0以外でも出力がある場合は成功として扱う
		result := CommandResult{
			Status:    "success",
			Command:   cmd,
			Result:    outputStr,
			Pwd:       session.CurrentDir,
			Username:  session.Username,
			SessionID: sessionID,
		}
		log.Printf("コマンド実行成功（終了コード%d）: %+v", exitCode, result)
		return result, nil
	} else {
		// 終了コードが0以外で出力がない場合は失敗
		log.Printf("コマンド実行失敗（終了コード%d）: %s", exitCode, cmd)
		return CommandResult{
			Status:    "error",
			Command:   cmd,
			Error:     fmt.Sprintf("コマンド実行失敗（終了コード: %d）", exitCode),
			Result:    outputStr,
			Pwd:       session.CurrentDir,
			Username:  session.Username,
			SessionID: sessionID,
		}, nil
	}
}


// executeCommand は、指定されたコマンドを実行し、結果を返す
// cdコマンドは特別に処理され、セッションの現在ディレクトリを更新
// その他のコマンドは、セッションの現在ディレクトリで実行される
// &&や||で繋がれたコマンドチェーンも対応
func executeCommand(cmd string, sessionID string) (CommandResult, error) {

	// セッションIDの検証（APIから送信されたセッションIDが空でないことを確認）
	if sessionID == "" {
		log.Printf("セキュリティ警告: 空のセッションIDでコマンド実行試行")
		return CommandResult{
			Status:    "error",
			Command:   cmd,
			Error:     "Invalid session ID: Session ID is required",
			SessionID: "",
		}, nil
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

	// &&や||が含まれているかチェック
	if strings.Contains(cmd, "&&") || strings.Contains(cmd, "||") {
		// コマンドチェーンを解析
		chain := parseCommandChain(cmd)
		log.Printf("コマンドチェーン検出: %v (演算子: %v)", chain.Commands, chain.Operators)

		// コマンドチェーンを実行
		return executeCommandChain(chain, session, sessionID)
	}

	// 単一コマンドの実行
	return executeSingleCommand(cmd, session, sessionID, false)
}
