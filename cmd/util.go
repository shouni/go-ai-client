package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	clibase "github.com/shouni/go-cli-base"
	"github.com/spf13/cobra"
)

// GenerateAndOutput は、RunnerのRunメソッドを呼び出すように変更します。
// ModelName は Runner 内部で保持されるため、引数から削除されました。
func GenerateAndOutput(ctx context.Context, inputContent []byte, subcommandMode string) error {
	// RunnerのインスタンスがDIされていることを確認
	if aiRunner == nil {
		return fmt.Errorf("内部エラー: AI Runnerが適切に初期化されていません。SetupRunnerが呼び出されましたか？")
	}
	// Runnerに処理を委譲
	return aiRunner.Run(ctx, inputContent, subcommandMode)
}

// checkAPIKey は、APIキー環境変数が設定されているかを確認します。
func checkAPIKey() error {
	if os.Getenv("GEMINI_API_KEY") == "" && os.Getenv("GOOGLE_API_KEY") == "" {
		return fmt.Errorf("致命的エラー: GEMINI_API_KEY または GOOGLE_API_KEY 環境変数が設定されていません。")
	}
	return nil
}

// initAppPreRunE は、ログレベル設定とAPIキーチェックを実行します。
func initAppPreRunE(cmd *cobra.Command, args []string) error {
	// ログレベル設定
	logLevel := slog.LevelInfo
	if clibase.Flags.Verbose {
		logLevel = slog.LevelDebug
	}
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: logLevel,
	})
	slog.SetDefault(slog.New(handler))

	// APIキーチェック
	err := checkAPIKey()
	if err != nil {
		slog.Error("🚨 APIKeyの取得に失敗しました", "error", err)
		return fmt.Errorf("APIKeyの取得に失敗しました: %w", err)
	}

	slog.Info("アプリケーション設定初期化完了")
	return nil
}
