# 1) 共通ビルダー：BusyBox(static) を準備
FROM debian AS base-builder

# busybox-static をインストール
RUN apt-get update && apt-get install -y \
    build-essential wget \
    libncurses5-dev libreadline-dev libssl-dev texinfo \
    bison flex gettext busybox-static \
    && rm -rf /var/lib/apt/lists/*

# Bash を静的リンクでビルド
RUN wget http://ftp.gnu.org/gnu/bash/bash-5.2.tar.gz \
    && tar xzf bash-5.2.tar.gz \
    && cd bash-5.2 \
    && ./configure --without-bash-malloc --enable-static \
    CFLAGS='-static' LDFLAGS='-static' \
    && make \
    && strip bash

# 独自ディレクトリに busybox 本体と必要なリンクを作成
RUN mkdir /bb \
    && cp /bin/busybox /bb/ \
    && cd /bb \
    && ln -s busybox ls \
    && ln -s busybox cat \
    && ln -s busybox clear \
    && ln -s busybox whoami

# 2) Go バイナリビルダー
FROM golang:1.23.1 AS go-builder
# 作業ディレクトリを server に設定
WORKDIR /app/server

# Goモジュールファイルをコピーして依存関係を取得
COPY server/go.mod server/go.sum ./
RUN go mod download

# ソースコード全体を server ディレクトリにコピー
COPY server/. .

# ビルド引数の定義とデフォルト値
ARG REDIS_HOST=redis
ARG REDIS_PORT=6379
ARG REDIS_PASSWORD=password
ARG REDIS_DB=0

# 実行時環境変数（任意で使用）
ENV REDIS_HOST=${REDIS_HOST} \
    REDIS_PORT=${REDIS_PORT} \
    REDIS_PASSWORD=${REDIS_PASSWORD} \
    REDIS_DB=${REDIS_DB}

# terminal バイナリのビルド
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags "\
    -X main.redisHost=$REDIS_HOST \
    -X main.redisPort=$REDIS_PORT \
    -X main.redisPass=$REDIS_PASSWORD \
    -X main.redisDB=$REDIS_DB" \
    -o /app/terminal .

# help バイナリのビルド（helpディレクトリがある前提）
WORKDIR /app/help
COPY help/. .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -o /cli/bin/help main.go

# 3) パーミッション設定用の一時的なステージ
FROM debian:bullseye-slim AS permission-setter

# chownに必要なファイルをetcに配置
COPY ./etc /etc

# 必要なファイルをコピー
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
FROM gcr.io/distroless/static-debian11:nonroot

USER nonroot

# bashのバイナリを/binに配置
COPY --from=base-builder /bash-5.2/bash /bin/bash

# RUNコマンドを実行するために、bashを/bin/shにリンク
COPY --from=base-builder /bash-5.2/bash   /bin/sh

# BusyBox バイナリとリンクのみを /bin に配置
COPY --from=base-builder /bb/ls       /bin/ls
COPY --from=base-builder /bb/cat      /bin/cat
COPY --from=base-builder /bb/clear    /bin/clear
COPY --from=base-builder /bb/whoami   /bin/whoami

# Goのバイナリを/binに配置
COPY --from=go-builder /cli/bin/help /bin/help
COPY --from=go-builder /app/terminal /usr/local/bin/terminal

# etcに配置するファイルをコピー
COPY --from=permission-setter /tmp /etc

# ユーザ以下のファイルを追加
COPY --from=permission-setter /home/nonroot/ /home/nonroot/

# アプリケーションの起動
CMD ["terminal"]
