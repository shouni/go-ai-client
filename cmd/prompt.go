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
		RunE: func(cmd *cobra.Command, args []string) error {
			// 1. 入力内容の決定
			inputText, err := readInput(cmd, args)
			if err != nil {
				return err
			}

			// 2. プロンプトの構築 (元のロジックを流用)
			name, content, err := prompts.GetTemplate(promptMode)
			if err != nil {
				// エラーをより詳しくラップ
				return fmt.Errorf("プロンプトテンプレート '%s' の取得に失敗しました: %w", promptMode, err)
			}

			builder, err := prompts.NewPromptBuilder(name, content)
			if err != nil {
				return fmt.Errorf("プロンプトビルダーの初期化に失敗しました: %w", err)
			}

			data := prompts.TemplateData{
				Content: string(inputText),
			}

			finalPrompt, err := builder.Build(data)
			if err != nil {
				return fmt.Errorf("最終プロンプトの構築に失敗しました: %w", err)
			}

			// 3. クライアント初期化と実行
			client, err := gemini.NewClientFromEnv(cmd.Context())
			if err != nil {
				return fmt.Errorf("AIクライアントの初期化に失敗しました: %w", err)
			}

			// タイムアウトコンテキストの適用 (Timeout グローバル変数を使用)
			clientCtx, cancel := context.WithTimeout(cmd.Context(), time.Duration(timeout)*time.Second)
			defer cancel()

			generateContent, err := client.GenerateContent(clientCtx, finalPrompt, modelName)
			if err != nil {
				return fmt.Errorf("AIコンテンツ生成中にエラーが発生しました: %w", err)
			}

			// 4. 結果の出力 (iohandler.WriteOutput を利用)
			return GenerateAndOutput(cmd.Context(), generateContent.Text)
		},
	}

	cmd.Flags().StringVarP(&promptMode, "mode", "d", "solo", "生成するスクリプトのモード (solo, dialogue)")

	return cmd
}

func init() {
	promptCmd = NewPromptCmd()
	rootCmd.AddCommand(promptCmd)
}
