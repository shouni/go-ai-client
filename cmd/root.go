package cmd

import (
	"fmt"
	"io"
	"strings"

	clibase "github.com/shouni/go-cli-base"
	"github.com/spf13/cobra"
)

// 公開（大文字）に変更 - Persistent Flags
var (
	ModelName string
	Timeout   int
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
	// PersistentPreRunE で初期設定のみを実行し、RunnerのDIはサブコマンドのRunEに任せる
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// 1. 基本設定 (ログ、APIキーチェック) のみ実行
		// SetupRunner の呼び出しは削除されました。
		return initAppPreRunE(cmd, args)
	},
}

func addAppPersistentFlags(rootCmd *cobra.Command) {
	rootCmd.PersistentFlags().IntVarP(&Timeout, "timeout", "t", 60, "APIリクエストのタイムアウト時間 (秒)")
	rootCmd.PersistentFlags().StringVarP(&ModelName, "model", "m", "gemini-2.5-flash", "使用するGeminiモデル名")
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

// --- 共通ユーティリティ関数（Rootに近いためここに配置） ---

// readInput は、コマンドライン引数または標準入力からテキストを読み込みます。
func readInput(cmd *cobra.Command, args []string) ([]byte, error) {
	if len(args) > 0 {
		return []byte(strings.Join(args, " ")), nil
	}
	// cmd.InOrStdin() を使用して標準入力から読み込み
	input, err := io.ReadAll(cmd.InOrStdin())
	if err != nil {
		return nil, fmt.Errorf("標準入力からの読み込みエラー: %w", err)
	}
	if len(input) == 0 {
		return nil, fmt.Errorf("致命的エラー: 処理するテキストがコマンドライン引数または標準入力から提供されていません。")
	}
	return input, nil
}
