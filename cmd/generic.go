package cmd

import (
	"context"
	"time"

	"github.com/spf13/cobra"
)

// newGenericCmd は 'generic' サブコマンドを構築して返します。
func newGenericCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generic [テキストまたはファイル]",
		Short: "プロンプトテンプレートを使用せず、入力テキストをそのままモデルに渡します。",
		Long: `このコマンドは、モデルに特別な役割を与えず、通常のチャットや要約を目的とします。
'mode' フラグは無視されます。

利用例:
  ai-client generic "量子コンピュータについて5行で解説せよ"`,

		RunE: func(cmd *cobra.Command, args []string) error {
			// 1. 入力内容の読み込みとAPIキー確認
			inputContent, err := readInput(cmd, args)
			if err != nil {
				return err
			}
			if err := checkAPIKey(); err != nil {
				return err
			}

			// 2. タイムアウト設定とコンテキスト作成 (root.goで定義されたグローバル変数を使用)
			timeoutDuration := time.Duration(timeout) * time.Second
			ctx, cancel := context.WithTimeout(context.Background(), timeoutDuration)
			defer cancel()

			// 3. 実行と出力 ("generic" モードはテンプレートを使用しないことを示す)
			// root.go の generateAndOutput にモード名として "generic" を渡す
			return generateAndOutput(ctx, inputContent, "generic", modelName)
		},
	}
	return cmd
}
