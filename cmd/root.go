package cmd

import (
	"fmt"
	"log/slog"
	"os"

	clibase "github.com/shouni/go-cli-base"
	"github.com/spf13/cobra"
)

// 公開（大文字）に変更
var (
	ModelName string
	Timeout   int
)

// clientKey は context.Context に httpkit.Client を格納・取得するための非公開キー
// (以前のコードにあった httpkit の依存は、今回のコードにはないため省略しますが、
// 以前の記憶に基づき、ここでは context.Context の設定のみを残します。)
type clientKey struct{}

// rootCmd は、このアプリケーションのメインとなるコマンドです。
var rootCmd = &cobra.Command{
	Use:   "go-ai-client", // プロジェクト名に合わせて修正
	Short: "Gemini APIのためのテンプレートベースAIクライアント",
	Long:  `Go言語で Generative AI（特に Google Gemini API）を簡単に利用するためのクライアントライブラリ、およびテンプレートベースのプロンプト生成ユーティリティを提供します。`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
	// PersistentPreRunE は clibase.Execute の引数として渡されるため、定義のみ残します。
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return initAppPreRunE(cmd, args)
	},
}

// checkAPIKey は、APIキー環境変数が設定されているかを確認します。
func checkAPIKey() error {
	// 以前記憶したロジック (GEMINI_API_KEY または GOOGLE_API_KEY を確認)
	if os.Getenv("GEMINI_API_KEY") == "" && os.Getenv("GOOGLE_API_KEY") == "" {
		return fmt.Errorf("致命的エラー: GEMINI_API_KEY または GOOGLE_API_KEY 環境変数が設定されていません。")
	}
	return nil
}

func initAppPreRunE(cmd *cobra.Command, args []string) error {

	// 1. slog ハンドラの設定
	logLevel := slog.LevelInfo
	if clibase.Flags.Verbose {
		logLevel = slog.LevelDebug
	}
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: logLevel,
	})
	slog.SetDefault(slog.New(handler))

	// 2. APIキーチェック
	err := checkAPIKey()
	if err != nil {
		slog.Error("🚨 APIKeyの取得に失敗しました", "error", err)
		return fmt.Errorf("APIKeyの取得に失敗しました: %w", err)
	}

	// 3. タイムアウト設定をコンテキストに格納するなどのロジックを追加可能
	// (今回は httpkit.Client の初期化ロジックがないため、Contextへの格納は省略)

	slog.Info("アプリケーション設定初期化完了")
	return nil
}

// addAppPersistentFlags は、アプリケーション固有の永続フラグをルートコマンドに追加します。
// フラグの定義をこの関数内に移動させます。
func addAppPersistentFlags(rootCmd *cobra.Command) {
	// Timeout と ModelName にバインド
	rootCmd.PersistentFlags().IntVarP(&Timeout, "timeout", "t", 60, "APIリクエストのタイムアウト時間 (秒)")
	rootCmd.PersistentFlags().StringVarP(&ModelName, "model", "m", "gemini-2.5-flash", "使用するGeminiモデル名")
}

// Execute は、clibase.Execute を使用してルートコマンドの構築と実行を委譲します。
func Execute() {
	clibase.Execute(
		"go-ai-client", // プロジェクト名に合わせて修正
		addAppPersistentFlags,
		initAppPreRunE,
		genericCmd,
		PromptCmd,
	)
}

// サブコマンドの仮定義 (Execute 関数で参照されるため)
/*
var genericCmd = &cobra.Command{Use: "generic", Short: "自由なテキストをGemini APIに送信します。"}
var PromptCmd = &cobra.Command{Use: "prompt", Short: "テンプレートを使用してプロンプトを構築し、Gemini APIに送信します。"}
*/

// init は、パッケージロード時に実行される Go の組み込み関数です。
// 引数を受け取れないため、以前ご提示いただいたコードは修正が必要です。
func init() {
	// フラグの追加ロジックは addAppPersistentFlags に移動しました。
}
