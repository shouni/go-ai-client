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
			// 1. SetupRunner の呼び出しを RunE の先頭に移動 (DIの実行)
			if err := SetupRunner(cmd.Context()); err != nil {
				return err // SetupRunnerでエラーが発生した場合、その具体的なエラーを返す
			}

			// 2. 入力内容の読み込み
			inputContent, err := readInput(cmd, args)
			if err != nil {
				return err
			}

			// 3. 実行と出力
			return GenerateAndOutput(cmd.Context(), inputContent, "")
		},
	}
	return cmd
}

func init() {
	genericCmd = NewGenericCmd()
	rootCmd.AddCommand(genericCmd)
}
