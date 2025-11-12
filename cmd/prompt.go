package cmd

import (
	"errors"

	"github.com/spf13/cobra"
)

// promptMode は 'prompt' サブコマンド固有のフラグ変数を定義
var promptMode string

// PromptCmd は 'prompt' サブコマンドのインスタンスです。（公開）
var PromptCmd = NewPromptCmd()

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
			// 1. 入力内容の読み込み
			inputContent, err := readInput(cmd, args)
			if err != nil {
				return err
			}

			if len(inputContent) == 0 {
				return errors.New("致命的エラー: テンプレートモード (prompt) は、処理するための入力テキストを必要とします。コマンドライン引数または標準入力で提供してください。")
			}

			// モードフラグの必須チェック (MarkFlagRequiredで定義済みだが、念のため)
			if promptMode == "" {
				return errors.New("致命的エラー: 'mode' フラグ (-d) が必須です。")
			}

			// 2. 実行と出力 (共通ロジックを使用)
			// Contextは cmd.Context() を使用
			// GenerateAndOutput と ModelName は cmd_core.go で定義された共通要素
			return GenerateAndOutput(cmd.Context(), inputContent, promptMode, ModelName)
		},

		Args: cobra.ArbitraryArgs,
	}

	// promptCmd のみに 'mode' フラグを設定
	cmd.Flags().StringVarP(&promptMode, "mode", "d", "solo", "生成するスクリプトのモード (solo, dialogue)")
	cmd.MarkFlagRequired("mode")

	return cmd
}
