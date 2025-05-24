package main

import (
	"encoding/json"
	"fmt"
)

func parsePayload(rawPayload string) (*Payload, error) {
	// Payload構造体を初期化
	var payload Payload

	// jsonのパースで失敗したときのエラーハンドリング
	if err := json.Unmarshal([]byte(rawPayload), &payload); err != nil {
		return nil, fmt.Errorf("JSONパースエラー: %w (payload: %s)", err, rawPayload)
	}

	return &payload, nil
}

