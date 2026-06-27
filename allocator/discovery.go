package main

import "context"

// PodDiscovery は terminal Pod の URL一覧を取得するインターフェース
// 開発時は StaticDiscovery、本番時は K8sDiscovery を使う
type PodDiscovery interface {
	ListPodURLs(ctx context.Context) ([]string, error)
}
