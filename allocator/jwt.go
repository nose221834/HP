package main

import (
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// ─── JWT のペイロード定義 ──────────────────────────────────────────
// 標準クレームに加えて、接続先PodのURLを埋め込む

type terminalClaims struct {
	PodURL string `json:"pod_url"` // 接続先Pod の内部URL
	jwt.RegisteredClaims
}

// ─── 秘密鍵の取得 ─────────────────────────────────────────────────
// 環境変数 JWT_SECRET から取得する
// allocator と terminal Pod の両方が同じ値を持つ必要がある

func jwtSecret() []byte {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		// 開発時のデフォルト（本番では必ず環境変数を設定すること）
		secret = "dev-secret-change-in-production"
	}
	return []byte(secret)
}

// ─── JWT 発行 ──────────────────────────────────────────────────────
// /allocate が空きPodを見つけたときに呼ぶ
// podURL: Pod の内部URL（例: http://terminal-0.terminal-svc:8080）

func issueToken(podURL string) (string, error) {
	now := time.Now()
	claims := terminalClaims{
		PodURL: podURL,
		RegisteredClaims: jwt.RegisteredClaims{
			// 発行時刻
			IssuedAt: jwt.NewNumericDate(now),
			// 有効期限: 5分
			// /allocate でJWTを受け取ってから5分以内にWS接続する必要がある
			ExpiresAt: jwt.NewNumericDate(now.Add(5 * time.Minute)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret())
}

// ─── JWT 検証 ──────────────────────────────────────────────────────
// /ws がWSアップグレード前に呼ぶ
// 有効なJWTならPodのURLを返す

func verifyToken(tokenString string) (string, error) {
	token, err := jwt.ParseWithClaims(
		tokenString,
		&terminalClaims{},
		func(t *jwt.Token) (any, error) {
			// 署名アルゴリズムの確認（HS256以外は拒否）
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("unexpected signing method")
			}
			return jwtSecret(), nil
		},
	)
	if err != nil {
		return "", err
	}

	claims, ok := token.Claims.(*terminalClaims)
	if !ok || !token.Valid {
		return "", errors.New("invalid token")
	}

	return claims.PodURL, nil
}
