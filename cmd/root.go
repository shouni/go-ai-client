package cmd

import (
	clibase "github.com/shouni/go-cli-base"
	"github.com/spf13/cobra"
)

// グローバルなフラグ変数（PersistentFlagsで設定される）
var (
	modelName string
	timeout   int
)

// --- サブコマンド（他のファイルで定義され、ここで利用される） ---
// 外部ファイルで定義された公開変数を利用することを想定
var genericCmd *cobra.Command
var promptCmd *cobra.Command

// NewGenericCmd と NewPromptCmd がどこかで実行され、genericCmd と promptCmd に値が設定されていることを前提とします。
func init() {
	// 依存関係を初期化
	genericCmd = NewGenericCmd()
	promptCmd = NewPromptCmd()
}

// addAppPersistentFlags は、アプリケーション全体で利用可能な永続フラグを追加します。
// clibase.Execute に渡されます。
func addAppPersistentFlags(rootCmd *cobra.Command) {
	rootCmd.PersistentFlags().IntVarP(&timeout, "timeout", "t", 60, "APIリクエストのタイムアウト時間 (秒)")
	rootCmd.PersistentFlags().StringVarP(&modelName, "model", "m", "gemini-2.5-flash", "使用するGeminiモデル名")
}

// --- メイン実行関数 ---

// Execute は、CLIアプリケーションのエントリポイントです。
// アプリケーション固有のサブコマンドとカスタマイズ関数をルートコマンドに追加し、実行します。
func Execute() {
	// clibase.Execute を使用して、アプリケーションの実行に必要なすべてを設定し、実行します。
	clibase.Execute(
		"go-ai-client", // アプリケーション名
		addAppPersistentFlags,
		initAppPreRunE,
		genericCmd,
		promptCmd,
	)
}
