package cmd

import (
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

			// 入力内容の読み込み
			inputContent, err := readInput(cmd, args)
			if err != nil {
				return err
			}
			inputText := string(inputContent)

			client, err := gemini.NewClientFromEnv(cmd.Context())
			if err != nil {
				return err
			}

			name, content, err := prompts.GetTemplate(promptMode)
			if err != nil {
				return err
			}

			builder, err := prompts.NewPromptBuilder(name, content)
			if err != nil {
				return err
			}

			data := prompts.TemplateData{
				Content: inputText,
			}

			finalPrompt, err := builder.Build(data)
			if err != nil {
				return err
			}

			generateContent, err := client.GenerateContent(cmd.Context(), finalPrompt, ModelName)
			if err != nil {
				return err
			}

			// 3. 実行と出力
			return GenerateAndOutput(cmd.Context(), generateContent.Text)
		},

		Args: cobra.ArbitraryArgs,
	}

	// promptCmd のみに 'mode' フラグを設定
	cmd.Flags().StringVarP(&promptMode, "mode", "d", "solo", "生成するスクリプトのモード (solo, dialogue)")
	cmd.MarkFlagRequired("mode")

	return cmd
}

func init() {
	promptCmd = NewPromptCmd()
	rootCmd.AddCommand(promptCmd)
}
