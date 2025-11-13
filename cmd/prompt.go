package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/shouni/go-ai-client/v2/pkg/ai/gemini"
	"github.com/shouni/go-ai-client/v2/prompts"
	"github.com/spf13/cobra"
)

// promptMode は 'prompt' サブコマンド固有のフラグ変数を定義
var promptMode string

// PromptCmd は 'prompt' サブコマンドのインスタンスです。（公開）
var promptCmd = NewPromptCmd()

// NewPromptCmd は 'prompt' コマンドを構築します。
func NewPromptCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "prompt [TEXT or pipe]",
		Short: "事前に登録されたプロンプトテンプレート（Solo/Dialogue）を使用してスクリプトを生成します。",
		Long: `このコマンドは、内部のプロンプトテンプレートを使用して、入力テキストを特定の役割（モード）を持つ
プロンプトに変換してからモデルに渡します。

利用例:
  ai-client prompt "Go言語の並行処理について" -d solo
  ai-client prompt "猫と魚の会話" -d dialogue
`,
		// コマンドの実行ロジックを外部関数に委譲
		RunE: executePromptCommand,
	}

	cmd.Flags().StringVarP(&promptMode, "mode", "d", "solo", "生成するスクリプトのモード (solo, dialogue)")

	return cmd
}

// executePromptCommand は 'prompt' サブコマンドの実際の実行ロジックを保持します。
func executePromptCommand(cmd *cobra.Command, args []string) error {
	commandCtx := cmd.Context()

	// 1. 入力内容の決定
	inputText, err := readInput(cmd, args)
	if err != nil {
		return err // readInput内で十分なエラーメッセージが出ていると想定
	}

	// 2. プロンプトの構築
	finalPrompt, err := buildPrompt(promptMode, inputText)
	if err != nil {
		return fmt.Errorf("プロンプトの構築に失敗しました: %w", err)
	}

	// 3. クライアント初期化と実行 (タイムアウト適用)
	client, err := gemini.NewClientFromEnv(commandCtx)
	if err != nil {
		return fmt.Errorf("AIクライアントの初期化に失敗しました: %w", err)
	}

	// タイムアウトコンテキストの適用 (Timeout グローバル変数を使用)
	clientCtx, cancel := context.WithTimeout(commandCtx, time.Duration(timeout)*time.Second)
	defer cancel()

	generateContent, err := client.GenerateContent(clientCtx, finalPrompt, modelName)
	if err != nil {
		return fmt.Errorf("AIコンテンツ生成中にエラーが発生しました: %w", err)
	}

	// 4. 結果の出力
	return GenerateAndOutput(commandCtx, generateContent.Text)
}

// buildPrompt はプロンプト構築のロジックを抽象化します。
func buildPrompt(mode string, inputText []byte) (string, error) {
	// a. テンプレートの取得
	name, content, err := prompts.GetTemplate(mode)
	if err != nil {
		// prompts.GetTemplate内で詳細なエラーメッセージが出ているため、そのまま返すか、
		// 呼び出し元がより分かりやすいようにラップします。
		return "", fmt.Errorf("テンプレート取得エラー: %w", err)
	}

	// b. ビルダーの初期化
	// 以前の改善で、NewPromptBuilder は prompts.PromptBuilder インターフェースを返すようになりました。
	builder, err := prompts.NewPromptBuilder(name, content)
	if err != nil {
		return "", fmt.Errorf("ビルダー初期化エラー: %w", err)
	}

	// c. データの埋め込みと実行
	data := prompts.TemplateData{
		Content: string(inputText),
	}

	finalPrompt, err := builder.Build(data)
	if err != nil {
		return "", fmt.Errorf("プロンプトの実行エラー: %w", err)
	}

	return finalPrompt, nil
}

func init() {
	promptCmd = NewPromptCmd()
	rootCmd.AddCommand(promptCmd)
}
