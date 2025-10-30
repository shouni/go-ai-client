package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/shouni/go-ai-client/pkg/ai/gemini"
	"github.com/shouni/go-ai-client/pkg/prompt"
	"github.com/spf13/cobra"
)

// --- グローバル設定（フラグ変数） ---

const separator = "=============================================="

// 公開（大文字）に変更
var (
	ModelName string
	Timeout   int
)

// --- ユーティリティ関数（全コマンドで共有） ---

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

// checkAPIKey は、APIキー環境変数が設定されているかを確認します。
func checkAPIKey() error {
	if os.Getenv("GEMINI_API_KEY") == "" && os.Getenv("GOOGLE_API_KEY") == "" {
		return fmt.Errorf("致命的エラー: GEMINI_API_KEY または GOOGLE_API_KEY 環境変数が設定されていません。")
	}
	return nil
}

// GenerateAndOutput は、Gemini APIを呼び出し、結果を標準出力に出力する共通ロジックです。（公開）
func GenerateAndOutput(ctx context.Context, inputContent []byte, subcommandMode, modelName string) error {
	// 1. クライアントの初期化
	clientCtx, cancel := context.WithTimeout(ctx, time.Duration(Timeout)*time.Second)
	defer cancel()

	client, err := gemini.NewClientFromEnv(clientCtx)
	if err != nil {
		return fmt.Errorf("Geminiクライアントの初期化に失敗しました: %w", err)
	}

	var finalPrompt string
	modeDisplay := subcommandMode
	inputText := string(inputContent)

	// 2. プロンプトの構築ロジック
	if subcommandMode == "generic" {
		finalPrompt = inputText
		modeDisplay = "テンプレートなし (generic)"
	} else {
		finalPrompt, err = prompt.BuildFullPrompt(inputText, subcommandMode)
		if err != nil {
			return fmt.Errorf("failed to build full prompt (mode: %s): %w", subcommandMode, err)
		}
	}

	fmt.Printf("モデル %s で応答を生成中 (モード: %s, Timeout: %d秒)...\n", modelName, modeDisplay, Timeout)

	// 3. 応答の生成
	resp, err := client.GenerateContent(clientCtx, finalPrompt, modelName)

	if err != nil {
		return fmt.Errorf("API処理中にエラーが発生しました: %w", err)
	}

	// 4. 結果の出力
	fmt.Println("\n" + separator)
	fmt.Printf("|| 応答 (モデル: %s, モード: %s) ||\n", modelName, modeDisplay)
	fmt.Println(separator)
	fmt.Println(resp.Text)
	fmt.Println(separator)

	return nil
}

// InitFlags は、go-cli-base がルートコマンドを生成した後、永続フラグを追加するために呼び出されます。（公開）
func InitFlags(rootCmd *cobra.Command) {
	rootCmd.PersistentFlags().IntVarP(&Timeout, "timeout", "t", 60, "APIリクエストのタイムアウト時間 (秒)")
	rootCmd.PersistentFlags().StringVarP(&ModelName, "model", "m", "gemini-2.5-flash", "使用するGeminiモデル名")
}
