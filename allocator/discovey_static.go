package main

import (
	"context"
	"fmt"
	"os"
	"strings"
)

// StaticDiscovery は環境変数 TERMINAL_URLS からPodのURL一覧を読む
// 開発時（docker compose）での使用を想定
//
// 設定例:
//   TERMINAL_URLS=http://terminal:8080
//   TERMINAL_URLS=http://terminal-1:8080,http://terminal-2:8080

type StaticDiscovery struct {
	urls []string
}

func NewStaticDiscovery() (*StaticDiscovery, error) {
	raw := os.Getenv("TERMINAL_URLS")
	if raw == "" {
		return nil, fmt.Errorf("TERMINAL_URLS is required for static discovery mode")
	}

	// カンマ区切りで複数URLを受け付ける
	urls := strings.Split(raw, ",")
	for i, u := range urls {
		urls[i] = strings.TrimSpace(u)
	}

	return &StaticDiscovery{urls: urls}, nil
}

func (s *StaticDiscovery) ListPodURLs(ctx context.Context) ([]string, error) {
	return s.urls, nil
}
