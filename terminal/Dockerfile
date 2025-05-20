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
    && ln -s busybox clear

# 2) Go バイナリビルダー
FROM golang:1.21 AS go-builder
WORKDIR /app

# Goモジュールの初期化と依存関係のインストール
COPY go.mod go.sum ./
RUN go mod download

# ソースコードのコピーとビルド
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /app/terminal

RUN go build -o /cli/bin/help ./help/main.go

# 3) 最終イメージ
FROM gcr.io/distroless/static-debian11:nonroot

# bashのバイナリを/binに配置
COPY --from=base-builder /bash-5.2/bash /bin/bash

# RUNコマンドを実行するために、bashを/bin/shにリンク
COPY --from=base-builder /bash-5.2/bash   /bin/sh

# BusyBox バイナリとリンクのみを /bin に配置
COPY --from=base-builder /bb/ls       /bin/ls
COPY --from=base-builder /bb/cat      /bin/cat
COPY --from=base-builder /bb/clear    /bin/clear

# Goのバイナリを/binに配置
COPY --from=go-builder /cli/bin/help /bin/help
COPY --from=go-builder /app/terminal /usr/local/bin/terminal

# groupとpasswdをコピー
COPY ./etc /etc
COPY ./config/ /home/nonroot/


# アプリケーションの起動
CMD ["terminal"]
