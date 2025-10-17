package cmd

import (
	"context"
	_ "embed" // go:embed のためにアンダースコアインポート
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/shouni/go-ai-client/pkg/ai/gemini"
	"github.com/shouni/go-ai-client/pkg/prompt"
	"github.com/spf13/cobra"
)

// プロンプトテンプレートをstring変数に直接埋め込む
// ファイルは cmd/prompt/ に配置されていることを想定
//
//go:embed prompt/zundamon_solo.md
var ZundamonSoloPrompt string

//go:embed prompt/zundametan_dialogue.md
var ZundaMetanDialoguePrompt string

// グローバル定数
const separator = "=============================================="

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
  ai-client "今日の天気は晴れです" -d solo
  cat input.txt | ./bin/ai-client -d dialogue`,

	RunE: func(cmd *cobra.Command, args []string) error {

		// 1. 入力内容の読み込み
		var inputContent []byte
		var err error

		if len(args) > 0 {
			// コマンドライン引数を入力として使用
			inputContent = []byte(strings.Join(args, " "))
		} else {
			// コマンドライン引数がない場合、標準入力から読み込みを試みる
			inputContent, err = io.ReadAll(cmd.InOrStdin())
			if err != nil {
				return fmt.Errorf("標準入力からの読み込みエラー: %w", err)
			}
		}

		if len(inputContent) == 0 {
			return fmt.Errorf("致命的エラー: 処理するテキストがコマンドライン引数または標準入力から提供されていません。")
		}

		// 2. APIキーの確認
		if os.Getenv("GEMINI_API_KEY") == "" && os.Getenv("GOOGLE_API_KEY") == "" {
			return fmt.Errorf("致命的エラー: GEMINI_API_KEY または GOOGLE_API_KEY 環境変数が設定されていません。")
		}

		// 3. タイムアウト設定とコンテキスト作成
		timeoutDuration := time.Duration(timeout) * time.Second
		ctx, cancel := context.WithTimeout(context.Background(), timeoutDuration)
		defer cancel()

		// 4. クライアントの初期化
		client, err := gemini.NewClientFromEnv(ctx)
		if err != nil {
			return fmt.Errorf("Geminiクライアントの初期化に失敗しました: %w", err)
		}

		// 5. 応答の生成
		fmt.Printf("モデル %s でスクリプトを生成中 (モード: %s, Timeout: %d秒)...\n", modelName, mode, timeout)

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

	Args: func(cmd *cobra.Command, args []string) error {
		// モードフラグの検証 (テンプレートが登録されているかを確認)
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

// init() はアプリケーション起動時に自動的に実行され、フラグとプロンプトテンプレートを設定します。
func init() {
	// フラグの設定
	rootCmd.PersistentFlags().IntVarP(&timeout, "timeout", "t", 60, "APIリクエストのタイムアウト時間 (秒)")
	rootCmd.PersistentFlags().StringVarP(&modelName, "model", "m", "gemini-2.5-flash", "使用するGeminiモデル名 (例: gemini-2.5-flash, gemini-2.5-pro)")
	rootCmd.PersistentFlags().StringVarP(&mode, "mode", "d", "solo", "生成するスクリプトのモード (solo, dialogue) -d はdialogueの略")

	// 埋め込まれた string 変数を使って prompt パッケージに登録する
	registerPromptTemplates()
}

// registerPromptTemplates は、埋め込まれた string 変数からテンプレートを読み込み、pkg/prompt に登録します。
func registerPromptTemplates() {

	// ユーティリティ関数: エラー発生時にエラーメッセージを出力し、終了コード1でプロセスを終了する
	safeExit := func(msg string) {
		fmt.Fprintf(os.Stderr, "クリティカルエラー (起動時): %s\n", msg)
		os.Exit(1)
	}

	// 1. Soloモードのテンプレート登録
	if ZundamonSoloPrompt == "" {
		safeExit("ソロテンプレート (ZundamonSoloPrompt) の埋め込みが失敗しているか、ファイルが空です。")
	}
	if err := prompt.RegisterTemplate("solo", ZundamonSoloPrompt); err != nil {
		safeExit(fmt.Sprintf("ソロテンプレートの登録に失敗: %v", err))
	}

	// 2. Dialogueモードのテンプレート登録
	if ZundaMetanDialoguePrompt == "" {
		safeExit("対話テンプレート (ZundaMetanDialoguePrompt) の埋め込みが失敗しているか、ファイルが空です。")
	}
	if err := prompt.RegisterTemplate("dialogue", ZundaMetanDialoguePrompt); err != nil {
		safeExit(fmt.Sprintf("対話テンプレートの登録に失敗: %v", err))
	}
}
