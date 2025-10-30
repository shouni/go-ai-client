package main

import (
	"log"

	"github.com/shouni/go-ai-client/cmd"
	"github.com/shouni/go-cli-base"
)

// main は go-cli-base ライブラリを使用してCLIアプリケーションを初期化し、実行します。
// これにより、共通のCLI構造（ルートコマンド、フラグ、サブコマンド、エラーハンドリング）が抽象化され、
// アプリケーション固有のロジックに集中できるようになります。
func main() {
	log.SetFlags(0)

	// 1. ルートコマンドの基盤を作成 (go-cli-baseが共通フラグなどを設定)
	rootCmd := clibase.NewRootCmd("ai-client")

	// 2. アプリ固有の共通フラグをルートコマンドに追加
	cmd.InitFlags(rootCmd)

	// 3. go-cli-base の Execute を使ってCLIを実行
	// cmd パッケージで公開されたサブコマンド (PromptCmd, GenericCmd) を渡す
	// go-cli-base は、実行後のエラーハンドリング（Exit(1)）を担当
	clibase.Execute("ai-client", cmd.PromptCmd, cmd.GenericCmd)

	// Execute は内部で os.Exit を行うため、この後のコードは実行されない。
}
