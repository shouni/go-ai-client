package cmd

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"os"

	"github.com/shouni/go-ai-client/pkg/prompt"
	"github.com/spf13/cobra"
)

// promptCmd固有のフラグ変数を定義
var promptMode string

// --- 埋め込みプロンプト --- (省略)

//go:embed prompt/zundamon_solo.md
var ZundamonSoloPrompt string

//go:embed prompt/zundametan_dialogue.md
var ZundaMetanDialoguePrompt string

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
	failOnInit := func(msg string, err error) {
		fmt.Fprintf(os.Stderr, "エラー: CLI初期化に失敗しました: %s\n", msg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "詳細: %v\n", err)
		}
		os.Exit(1) // 非ゼロの終了コードで終了
	}

	// 1. Soloモードのテンプレート登録
	if ZundamonSoloPrompt == "" {
		failOnInit("ソロテンプレートの埋め込みが失敗しているか、ファイルが空です。", nil)
	}
	if err := prompt.RegisterTemplate("solo", ZundamonSoloPrompt); err != nil {
		failOnInit("ソロテンプレートの登録に失敗", err)
	}

	// 2. Dialogueモードのテンプレート登録
	if ZundaMetanDialoguePrompt == "" {
		failOnInit("対話テンプレートの埋め込みが失敗しているか、ファイルが空です。", nil)
	}
	if err := prompt.RegisterTemplate("dialogue", ZundaMetanDialoguePrompt); err != nil {
		failOnInit("対話テンプレートの登録に失敗", err)
	}
}
