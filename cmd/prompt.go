package cmd

import (
	"context"
	_ "embed"
	"time"

	"github.com/shouni/go-ai-client/pkg/prompt"
	"github.com/spf13/cobra"
)

// promptCmd固有のフラグ変数を定義
var promptMode string

// --- 埋め込みプロンプト ---

//go:embed prompt/zundamon_solo.md
var ZundamonSoloPrompt string

//go:embed prompt/zundametan_dialogue.md
var ZundaMetanDialoguePrompt string

// newPromptCmd は 'prompt' サブコマンドを構築して返します。
func newPromptCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "prompt [テキストまたはファイル]",
		Short: "事前に登録されたプロンプトテンプレート（Solo/Dialogue）を使用してスクリプトを生成します。",
		Long: `このコマンドは、ずんだもんやめたんなどのキャラクターに特化したテンプレートを使用して、
入力テキストをナレーションスクリプト形式に変換します。

利用例:
  ai-client prompt "今日の天気は晴れです" -d solo`,

		RunE: func(cmd *cobra.Command, args []string) error {
			// 1. 入力内容の読み込みとAPIキー確認
			inputContent, err := readInput(cmd, args)
			if err != nil {
				return err
			}
			if err := checkAPIKey(); err != nil {
				return err
			}

			// 2. モードフラグの検証
			if _, err := prompt.GetPromptByMode(promptMode); err != nil {
				return err
			}

			// 3. タイムアウト設定とコンテキスト作成 (root.goで定義されたグローバル変数を使用)
			timeoutDuration := time.Duration(timeout) * time.Second
			ctx, cancel := context.WithTimeout(context.Background(), timeoutDuration)
			defer cancel()

			// 4. 実行と出力 (root.goの共通ロジックを使用)
			return generateAndOutput(ctx, inputContent, promptMode, modelName)
		},

		Args: func(cmd *cobra.Command, args []string) error {
			// RunEの前に、モードフラグの検証を行う
			if _, err := prompt.GetPromptByMode(promptMode); err != nil {
				return err
			}
			return nil
		},
	}

	// promptCmd のみに 'mode' フラグを設定
	cmd.Flags().StringVarP(&promptMode, "mode", "d", "solo", "生成するスクリプトのモード (solo, dialogue)")

	return cmd
}
