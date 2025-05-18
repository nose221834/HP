# 作業ディレクトリを環境変数で定義
export WORKDIR="$HOME"

# ログイン時に必ずホームディレクトリに移動
cd "$WORKDIR" || exit
export PATH="/bin:$PATH"
