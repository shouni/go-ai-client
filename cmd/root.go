package cmd

import (
	clibase "github.com/shouni/go-cli-base"
	"github.com/spf13/cobra"
)

var (
	modelName string
	timeout   int
)

// --- CLI定義 ---

// rootCmd は、このアプリケーションのメインとなるコマンドです。
var rootCmd = &cobra.Command{
	Use:   "go-ai-client",
	Short: "Gemini APIのためのテンプレートベースAIクライアント",
	Long:  `Go言語で Generative AI（特に Google Gemini API）を簡単に利用するためのクライアントライブラリ、およびテンプレートベースのプロンプト生成ユーティリティを提供します。`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// 1. 基本設定 (ログ、APIキーチェック) のみ実行
		// SetupRunner の呼び出しは削除されました。
		return initAppPreRunE(cmd, args)
	},
}

func addAppPersistentFlags(rootCmd *cobra.Command) {
	rootCmd.PersistentFlags().IntVarP(&timeout, "timeout", "t", 60, "APIリクエストのタイムアウト時間 (秒)")
	rootCmd.PersistentFlags().StringVarP(&modelName, "model", "m", "gemini-2.5-flash", "使用するGeminiモデル名")
}

func Execute() {
	clibase.Execute(
		"go-ai-client",
		addAppPersistentFlags,
		initAppPreRunE,
		genericCmd,
		promptCmd,
	)
}
