package help

import (
	"flag"
	"fmt"
)

func main() {
    helpFlag := flag.Bool("help", false, "ヘルプ情報を表示します。")
    flag.Parse()

    if *helpFlag {
        fmt.Println("このツールは、Goで作成された簡単なCLIアプリケーションの例です。")
        fmt.Println("使用方法:")
        fmt.Println("  -help    ヘルプ情報を表示します。")
        return
    }

    fmt.Println("コマンドが不正です。`-help` フラグを使用して使い方を確認してください。")
}
