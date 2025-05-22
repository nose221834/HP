# 基本的なシェル操作に必要なコマンドを有効化（最初に実行）
enable .  # sourceコマンドとして必要
enable :  # 基本的なシェル構文
enable [  # testコマンドとして必要
enable alias  # エイリアス設定に必要
enable cd  # ディレクトリ移動に必要
enable -n echo  # 出力に必要
enable -n exit  # シェル終了に必要
enable export  # 環境変数設定に必要
enable pwd  # 現在のディレクトリ表示に必要
enable times  # 基本的なシェル機能

# 危険なコマンドを無効化
enable -n bg
enable -n bind
enable -n break
enable -n builtin
enable -n caller
enable -n command
enable -n compgen
enable -n complete
enable -n compopt
enable -n continue
enable -n declare
enable -n dirs
enable -n disown
enable -n eval
enable -n exec
enable -n false
enable -n fc
enable -n fg
enable -n getopts
enable -n hash
enable -n help
enable -n history
enable -n jobs
enable -n kill
enable -n let
enable -n local
enable -n logout
enable -n mapfile
enable -n popd
enable -n printf
enable -n pushd
enable -n read
enable -n readarray
enable -n readonly
enable -n return
enable -n set
enable -n shift
enable -n shopt
enable -n source
enable -n suspend
enable -n test
enable -n trap
enable -n true
enable -n type
enable -n typeset
enable -n ulimit
enable -n umask
enable -n unalias
enable -n wait

# 最後にenableコマンド自体を無効化
enable -n enable

# 許可されているコマンド：
# - . (source): スクリプトの読み込みに必要
# - : (コロン): 基本的なシェル構文
# - [ (test): 条件テストに必要
# - alias: コマンドのエイリアス設定
# - cd: ディレクトリ移動
# - echo: 出力表示
# - exit: シェル終了
# - export: 環境変数設定
# - pwd: 現在のディレクトリ表示
# - times: 基本的なシェル機能