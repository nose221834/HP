FROM golang:1.26.1 AS go-builder
WORKDIR /app/server

# Goモジュールファイルをコピーして依存関係を取得
COPY server/go.mod server/go.sum ./
RUN go mod download

COPY server/. .

# terminal バイナリのビルド
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -o /app/terminal .

# help バイナリのビルド
WORKDIR /app/help
COPY help/. .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -o /cli/bin/help main.go

FROM debian:13 AS base-bin

RUN apt-get update && apt-get install -y --no-install-recommends bash-static busybox-static

# chownに必要なファイルをetcに配置
COPY ./etc /etc

COPY ./etc /tmp
COPY ./nonroot/ /home/nonroot/

# パーミッションと所有権の設定
RUN mkdir -p /tmp/profile.d && \
    chown -R nonroot:nonroot /home/nonroot && \
    chmod 755 /home/nonroot && \
    # /tmp以下のファイルの権限設定（nonrootユーザーに読み取り権限を付与）
    find /tmp -type f -exec chmod 644 {} \; && \
    find /tmp -type d -exec chmod 755 {} \; && \
    # profile.dディレクトリとスクリプトの権限設定
    # ディレクトリ自体はroot所有のまま、実行権限を付与
    chmod 755 /tmp/profile.d && \
    # profile.d内のスクリプトファイルに実行権限を付与（所有者はrootのまま）
    find /tmp/profile.d -type f -name "*.sh" -exec chmod 755 {} \; && \
    # /tmpディレクトリの所有権はrootのまま維持
    chown -R root:root /tmp && \
    # グループとその他のユーザーに読み取り権限を付与
    chmod -R g+r,o+r /tmp && \
    # profile.dディレクトリとその中のファイルに実行権限を付与（書き込み権限は付与しない）
    chmod -R g+x,o+x /tmp/profile.d

# 4) 最終イメージ
FROM gcr.io/distroless/static-debian13:nonroot

USER nonroot

COPY --from=base-bin /bin/bash-static /bin/bash
COPY --from=base-bin /bin/bash-static /bin/sh

# BusyBoxはファイル名で振る舞いを変える
COPY --from=base-bin /bin/busybox /bin/ls
COPY --from=base-bin /bin/busybox /bin/cat
COPY --from=base-bin /bin/busybox /bin/clear

# Goのバイナリを/binに配置
COPY --from=go-builder /cli/bin/help /bin/help
COPY --from=go-builder /app/terminal /usr/local/bin/terminal

# etcに配置するファイルをコピー
COPY --from=base-bin /tmp /etc

# ユーザ以下のファイルを追加
COPY --from=base-bin /home/nonroot/ /home/nonroot/

# アプリケーションの起動
CMD ["terminal"]
