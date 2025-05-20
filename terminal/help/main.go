package main

import (
	"fmt"
)

func main() {
    fmt.Println("[Command Help] This is a command help message.")
	fmt.Println("Usage: command [options]")
	fmt.Println("Options:")
	fmt.Println("  help     この画面を表示します")
	fmt.Println("  profile  プロファイルを表示します")
	fmt.Println("  cat      ファイルを表示します && 猫が出てきます")
	fmt.Println("  sl       slが出てきます")
}
