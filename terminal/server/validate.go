package main

import (
	"fmt"
	"strings"
	"unicode/utf8"
)


func valivateCommand(cmd string) error {
	// ブラックリストに含まれるコマンドを定義
	var BlacklistMap = map[string]struct{}{
		"rm":       {},
		"shutdown": {},
	}

	// 空白で分割（例: "rm -rf /tmp" → ["rm", "-rf", "/tmp"]）
	parts := strings.Fields(cmd)

	// コマンドが空の場合はエラー
	if cmd == "" {
		return fmt.Errorf("コマンドが空です")
	}

	// 最初の要素だけを見る（実行されるコマンド名）
	baseCmd := parts[0]

	// ブラックリストに含まれるコマンドをチェック
	if _, exists := BlacklistMap[baseCmd]; exists {
		return fmt.Errorf("このコマンドは実行できません: %s", baseCmd)
	}

	return nil
}

func validateCommandResult(result *CommandResult) error {
	// セッションIDが空の場合はエラー
	if result.SessionID == "" {
		return fmt.Errorf("セッションIDが空です")
	}

	// 実行結果が異様に長い場合はエラー
	if len(result.Result) > 10000 {
		return fmt.Errorf("コマンドの実行結果が異常に長いです")
	}
	// 実行結果に不正な文字列が含まれている場合はエラー
	if !utf8.ValidString(result.Result) {
		return fmt.Errorf("コマンドの実行結果に不正な文字列が含まれています")
	}

	return nil
}
