package cmd

import (
	"context"
	_ "embed"
	"errors"

	"github.com/spf13/cobra"
)

// promptCmd固有のフラグ変数を定義
var promptMode string

// PromptCmd は 'prompt' サブコマンドのインスタンスです。（公開）
var PromptCmd = NewPromptCmd()

func NewPromptCmd() *cobra.Command { // 関数名をNewPromptCmdに変更
	cmd := &cobra.Command{
		Use:   "prompt [テキストまたはファイル]",
		Short: "事前に登録されたプロンプトテンプレート（Solo/Dialogue）を使用してスクリプトを生成します。",
		// ... (Longの説明は省略)

		RunE: func(cmd *cobra.Command, args []string) error {
			// 1. 入力内容の読み込みとAPIキー確認
			inputContent, err := readInput(cmd, args)
			if err != nil {
				return err
			}
			if err := checkAPIKey(); err != nil {
				return err
			}

			// ★ 修正: テンプレートモードでの入力必須チェックを追加
			if len(inputContent) == 0 {
				return errors.New("テンプレートモード (prompt) は、処理するための入力テキストを必要とします。コマンドライン引数または標準入力で提供してください。")
			}

			// 2. 実行と出力 (共通ロジックを使用)
			return GenerateAndOutput(context.Background(), inputContent, promptMode, ModelName)
		},

		Args: cobra.ArbitraryArgs,
	}

	// promptCmd のみに 'mode' フラグを設定
	cmd.Flags().StringVarP(&promptMode, "mode", "d", "solo", "生成するスクリプトのモード (solo, dialogue)")

	return cmd
}

// init はパッケージ初期化時にテンプレートを登録します。
func init() {
}
