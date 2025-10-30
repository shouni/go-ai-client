package cmd

import (
	"context"

	"github.com/spf13/cobra"
)

// GenericCmd は 'generic' サブコマンドのインスタンスです。（公開）
var GenericCmd = NewGenericCmd()

func NewGenericCmd() *cobra.Command { // 関数名をNewGenericCmdに変更
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

			// 2. 実行と出力 (共通ロジックを使用)
			// ModelName は cmd.common.go で定義された公開変数を使用
			return GenerateAndOutput(context.Background(), inputContent, "generic", ModelName)
		},
	}
	return cmd
}
