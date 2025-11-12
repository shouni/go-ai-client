package cmd

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/shouni/go-ai-client/v2/pkg/ai/gemini"
	clibase "github.com/shouni/go-cli-base"
	"github.com/spf13/cobra"
)

// 公開（大文字）に変更
var (
	ModelName string
	Timeout   int
)

const separator = "=============================================="

// clientKey は context.Context に格納するための非公開キー
type clientKey struct{}

// rootCmd は、このアプリケーションのメインとなるコマンドです。
var rootCmd = &cobra.Command{
	Use:   "go-ai-client",
	Short: "Gemini APIのためのテンプレートベースAIクライアント",
	Long:  `Go言語で Generative AI（特に Google Gemini API）を簡単に利用するためのクライアントライブラリ、およびテンプレートベースのプロンプト生成ユーティリティを提供します。`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return initAppPreRunE(cmd, args)
	},
}

// checkAPIKey は、APIキー環境変数が設定されているかを確認します。
func checkAPIKey() error {
	if os.Getenv("GEMINI_API_KEY") == "" && os.Getenv("GOOGLE_API_KEY") == "" {
		return fmt.Errorf("致命的エラー: GEMINI_API_KEY または GOOGLE_API_KEY 環境変数が設定されていません。")
	}
	return nil
}

func initAppPreRunE(cmd *cobra.Command, args []string) error {

	logLevel := slog.LevelInfo
	if clibase.Flags.Verbose {
		logLevel = slog.LevelDebug
	}
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: logLevel,
	})
	slog.SetDefault(slog.New(handler))

	err := checkAPIKey()
	if err != nil {
		slog.Error("🚨 APIKeyの取得に失敗しました", "error", err)
		return fmt.Errorf("APIKeyの取得に失敗しました: %w", err)
	}

	slog.Info("アプリケーション設定初期化完了")
	return nil
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
		PromptCmd,
	)
}

func init() {
	//
}

// --- 共通ユーティリティ関数（すべてのサブコマンドで使用） ---

// readInput は、コマンドライン引数または標準入力からテキストを読み込みます。
func readInput(cmd *cobra.Command, args []string) ([]byte, error) {
	if len(args) > 0 {
		return []byte(strings.Join(args, " ")), nil
	}
	input, err := io.ReadAll(cmd.InOrStdin())
	if err != nil {
		return nil, fmt.Errorf("標準入力からの読み込みエラー: %w", err)
	}
	if len(input) == 0 {
		return nil, fmt.Errorf("致命的エラー: 処理するテキストがコマンドライン引数または標準入力から提供されていません。")
	}
	return input, nil
}

// GenerateAndOutput は、Gemini APIを呼び出し、結果を標準出力に出力する共通ロジックです。（公開）
func GenerateAndOutput(ctx context.Context, inputContent []byte, subcommandMode, modelName string) error {
	clientCtx, cancel := context.WithTimeout(ctx, time.Duration(Timeout)*time.Second)
	defer cancel()

	client, err := gemini.NewClientFromEnv(clientCtx)
	if err != nil {
		return fmt.Errorf("Geminiクライアントの初期化に失敗しました: %w", err)
	}

	var finalPrompt string
	modeDisplay := subcommandMode
	inputText := string(inputContent)

	if subcommandMode == "generic" {
		finalPrompt = inputText
		modeDisplay = "テンプレートなし (generic)"
	} else {
		finalPrompt, err = promptbuilder.Build(inputText, subcommandMode)
		if err != nil {
			return fmt.Errorf("failed to build full prompt (mode: %s): %w", subcommandMode, err)
		}
	}

	slog.Info("応答生成リクエスト送信", "model", modelName, "mode", modeDisplay, "timeout", Timeout)
	fmt.Printf("モデル %s で応答を生成中 (モード: %s, Timeout: %d秒)...\n", modelName, modeDisplay, Timeout)

	resp, err := client.GenerateContent(clientCtx, finalPrompt, modelName)

	if err != nil {
		return fmt.Errorf("API処理中にエラーが発生しました: %w", err)
	}

	fmt.Println("\n" + separator)
	fmt.Printf("|| 応答 (モデル: %s, モード: %s) ||\n", modelName, modeDisplay)
	fmt.Println(separator)
	fmt.Println(resp.Text)
	fmt.Println(separator)

	return nil
}
