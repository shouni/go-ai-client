package cmd

import (
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
			inputContent, err := readInput(cmd, args)
			if err != nil {
				return err
			}

			return GenerateAndOutput(cmd.Context(), inputContent, "generic")
		},
	}
	return cmd
}
