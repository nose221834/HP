package main

import (
	"fmt"
	"unicode/utf8"
)

func valivatePayload(payload *Payload) error {
	// コマンドが空の場合はエラー
	if payload.Command == "" {
		return fmt.Errorf("コマンドが空です")
	}

	// ブラックリストに含まれるコマンドをチェック
	if _, exists := BlacklistMap[payload.Command]; exists {
		return fmt.Errorf("このコマンドは実行できません: %s", payload.Command)
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
