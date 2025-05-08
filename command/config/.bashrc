export WORKDIR="/home/nonroot"
export PATH="/bin:$PATH"

# 自動 cd
cd "$WORKDIR" || true

# カラー定義
RED='\[\e[31m\]'
GREEN='\[\e[32m\]'
BLUE='\[\e[34m\]'
RESET='\[\e[0m\]'

# PS1 プロンプト設定（Git 情報なし）
PS1="${GREEN}\u@nose${BLUE}:\w${RESET}\$ "

alias help='/bin/help'