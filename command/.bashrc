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

enable -n help
enable -n echo
enable -n jobs
enable -n kill
enable -n logout
enable -n set
enable -n suspend
enable -n type
enable -n umask
enable -n wait
enable -n unalias
enable -n bind
enable -n builtin
enable -n command
enable -n declare
enable -n dirs
enable -n disown

alias help='/bin/help'