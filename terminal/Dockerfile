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

# 2) Go バイナリビルダー（もし不要なら省略可）
FROM golang:1.23.4 AS go-builder
WORKDIR /cli
COPY ./help ./help
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -o /cli/bin/help ./help/main.go

# 3) 最終イメージ：scratch
FROM scratch AS runner

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

# groupとpasswdをコピー
COPY ./etc /etc

# 非rootユーザを作成
USER nonroot

# (オプション) .bashrc など
COPY ./config/ /home/nonroot/

ENTRYPOINT ["/bin/bash","-l"]
