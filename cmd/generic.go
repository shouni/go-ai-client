package cmd

import (
	"github.com/shouni/go-ai-client/v2/pkg/ai/gemini"
	"github.com/spf13/cobra"
)

// GenericCmd は 'generic' サブコマンドのインスタンスです。（公開）
var genericCmd = NewGenericCmd()

// NewGenericCmd は 'generic' コマンドを構築します。
func NewGenericCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generic [TEXT or pipe]",
		Short: "プロンプトテンプレートを使用せず、入力テキストをそのままモデルに渡します。",
		Long: `このコマンドは、モデルに特別な役割を与えず、通常のチャットや要約を目的とします。
'mode' フラグは無視されます。

利用例:
  ai-client generic "量子コンピュータについて5行で解説せよ"`,

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

			content, err := client.GenerateContent(cmd.Context(), inputText, ModelName)
			if err != nil {
				return err
			}

			return GenerateAndOutput(cmd.Context(), content.Text)
		},
	}
	return cmd
}

func init() {
	genericCmd = NewGenericCmd()
	rootCmd.AddCommand(genericCmd)
}
