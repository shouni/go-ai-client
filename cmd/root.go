package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"go-ai-client/pkg/ai/gemini"
	"go-ai-client/pkg/prompt"
)

// グローバル定数
const separator = "=================================================="

// グローバル変数: コマンドラインフラグの値を保持
var (
	modelName string
	timeout   int
	mode      string
)

// rootCmd はアプリケーションのメインコマンドです
var rootCmd = &cobra.Command{
	Use:   "ai-client [テキストまたはファイル]",
	Short: "Google Gemini APIを利用した、ナレーションスクリプト生成CLI。",
	Long: `ai-client は、入力テキストを元に、指定されたモード（solo, dialogue）で
ナレーションスクリプトを生成するためにGemini APIを呼び出すCLIツールです。

利用例:
  ai-client "今日の天気は晴れです" -m solo
  cat input.txt | ai-client -m dialogue`,

	// 実行されるメインロジック (エラーハンドリングのため RunE を使用)
	RunE: func(cmd *cobra.Command, args []string) error {

		// 1. 入力内容の読み込み
		var inputContent []byte
		var err error

		if len(args) > 0 {
			// コマンドライン引数を入力として使用
			inputContent = []byte(strings.Join(args, " "))
		} else if cmd.InOrStdin() != os.Stdin {
			// パイプ（標準入力）から読み込み
			inputContent, err = io.ReadAll(cmd.InOrStdin())
			if err != nil {
				return fmt.Errorf("標準入力からの読み込みエラー: %w", err)
			}
		} else {
			return fmt.Errorf("致命的エラー: 処理するテキストがコマンドライン引数または標準入力から提供されていません。")
		}

		if len(inputContent) == 0 {
			return fmt.Errorf("致命的エラー: 入力内容が空です。")
		}

		// 2. APIキーの確認
		if os.Getenv("GEMINI_API_KEY") == "" && os.Getenv("GOOGLE_API_KEY") == "" {
			return fmt.Errorf("致命的エラー: GEMINI_API_KEY または GOOGLE_API_KEY 環境変数が設定されていません。")
		}

		// 3. タイムアウト設定とコンテキスト作成
		timeoutDuration := time.Duration(timeout) * time.Second
		ctx, cancel := context.WithTimeout(context.Background(), timeoutDuration)
		defer cancel()

		// 4. クライアントの初期化 (🚨 修正点 1, 2)
		client, err := gemini.NewClientFromEnv(ctx)
		if err != nil {
			return fmt.Errorf("Geminiクライアントの初期化に失敗しました: %w", err)
		}

		// 5. 応答の生成
		fmt.Printf("モデル %s でスクリプトを生成中 (モード: %s, Timeout: %d秒)...\n", modelName, mode, timeout)

		// 🚨 修正: GenerateContentの引数を新しいインターフェースに合わせる
		resp, err := client.GenerateContent(ctx, inputContent, mode, modelName)

		if err != nil {
			return fmt.Errorf("API処理中にエラーが発生しました: %w", err)
		}

		// 6. 結果の出力
		fmt.Println("\n" + separator)
		fmt.Printf("|| 応答 (モデル: %s, モード: %s) ||\n", modelName, mode)
		fmt.Println(separator)
		fmt.Println(resp.Text)
		fmt.Println(separator)

		return nil // 正常終了
	},

	// 引数検証のカスタムロジック
	Args: func(cmd *cobra.Command, args []string) error {
		// 標準入力がない場合（argsが空でないことを期待）
		if cmd.InOrStdin() == os.Stdin && len(args) == 0 {
			// 標準入力がパイプされていないことを確認するためのロジックは複雑なため、
			// ひとまず args が空かつパイプがない場合にエラーとする
			stat, _ := os.Stdin.Stat()
			isPiped := (stat.Mode() & os.ModeCharDevice) == 0

			if !isPiped && len(args) == 0 {
				return fmt.Errorf("エラー: 処理するテキストをコマンドライン引数として提供するか、標準入力からパイプしてください。")
			}
		}

		// モードフラグの検証
		if _, err := prompt.GetPromptByMode(mode); err != nil {
			return err
		}

		return nil
	},
}

// Execute はルートコマンドを実行します。
func Execute() error {
	return rootCmd.Execute()
}

// init() はアプリケーション起動時に自動的に実行され、フラグを設定します。
func init() {
	rootCmd.PersistentFlags().IntVarP(&timeout, "timeout", "t", 60, "APIリクエストのタイムアウト時間 (秒)")
	rootCmd.PersistentFlags().StringVarP(&modelName, "model", "m", "gemini-2.5-flash", "使用するGeminiモデル名 (例: gemini-2.5-pro)")
	rootCmd.PersistentFlags().StringVarP(&mode, "mode", "d", "solo", "生成するスクリプトのモード (solo, dialogue)")
}
