package main

import (
	"fmt"
	"log"
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
	// 継続的なシェルセッションでコマンドを実行
	output, err := session.ExecuteCommandInSession(cmd)
	if err != nil {
		return CommandResult{
			Status:    "error",
			Command:   cmd,
			Error:     fmt.Sprintf("コマンド実行エラー: %v", err),
			SessionID: sessionID,
		}, nil
	}

	// 現在のディレクトリとユーザー名を取得（プロンプトから解析）
	pwd, username, prompt := parsePromptInfo(output)

	return CommandResult{
		Status:    "success",
		Command:   cmd,
		Result:    output,
		Pwd:       pwd,
		Username:  username,
		Prompt:    prompt,
		SessionID: sessionID,
	}, nil
}

// parsePromptInfo は出力からプロンプト情報を解析する
func parsePromptInfo(output string) (pwd, username, prompt string) {
	lines := strings.Split(output, "\n")
	
	// 最後の行からプロンプト情報を取得
	for i := len(lines) - 1; i >= 0; i-- {
		line := lines[i]
		if strings.Contains(line, "$ ") || strings.Contains(line, "# ") {
			prompt = line
			// プロンプトの形式: username@hostname:path$
			parts := strings.Split(line, "@")
			if len(parts) >= 2 {
				username = parts[0]
				// パス部分を抽出
				pathParts := strings.Split(parts[1], ":")
				if len(pathParts) >= 2 {
					pwd = pathParts[1]
					// $記号を除去
					pwd = strings.TrimSuffix(pwd, "$")
					pwd = strings.TrimSuffix(pwd, "#")
				}
			}
			break
		}
	}
	
	return pwd, username, prompt
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
