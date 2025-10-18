package cmd

import (
	"context"
	_ "embed" // go:embed のためにアンダースコアインポート
	"fmt"
	"os"

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

			// 2. コンテキスト作成
			// タイムアウト設定は generateAndOutput 内で行われるため、ここでは基本コンテキストを使用
			ctx := context.Background()
			// タイムアウト設定は generateAndOutput 内で行われますが、ここでは一応定義 (冗長)
			// timeoutDuration := time.Duration(timeout) * time.Second
			// ctx, cancel := context.WithTimeout(context.Background(), timeoutDuration)
			// defer cancel()

			// 3. 実行と出力 (root.goの共通ロジックを使用)
			return generateAndOutput(ctx, inputContent, promptMode, modelName)
		},

		Args: func(cmd *cobra.Command, args []string) error {
			// ★ 修正: テンプレートの検証を BuildFullPrompt に任せるため、ここでは引数の有無のみチェック。
			// テンプレートが登録されているかのチェックは init() で登録エラーがないか確認済みのため、
			// ここでは省略するか、RunEで BuildFullPrompt のエラーに任せるのがシンプル。
			// ただし、入力が必須でない場合はこのチェックも不要です。

			// テンプレートモードでは入力コンテンツの有無を RunE がチェックします。
			// ここではカスタムロジックを削除し、CobraのデフォルトArgs処理に任せる。
			return nil
		},
	}

	// promptCmd のみに 'mode' フラグを設定
	cmd.Flags().StringVarP(&promptMode, "mode", "d", "solo", "生成するスクリプトのモード (solo, dialogue)")

	return cmd
}

// init はパッケージ初期化時にテンプレートを登録します。
func init() {
	// 開発者向け: この panic は、go:embed の設定ミスなど、ビルド時の致命的な問題を検出するためのものです。
	safePanic := func(msg string) {
		// エラーメッセージをstderrに出力してから panic
		fmt.Fprintf(os.Stderr, "クリティカルエラー (prompt init): %s\n", msg)
		panic(msg)
	}

	// 1. Soloモードのテンプレート登録
	if ZundamonSoloPrompt == "" {
		safePanic("ソロテンプレートの埋め込みが失敗しているか、ファイルが空です。")
	}
	if err := prompt.RegisterTemplate("solo", ZundamonSoloPrompt); err != nil {
		safePanic(fmt.Sprintf("ソロテンプレートの登録に失敗: %v", err))
	}

	// 2. Dialogueモードのテンプレート登録
	if ZundaMetanDialoguePrompt == "" {
		safePanic("対話テンプレートの埋め込みが失敗しているか、ファイルが空です。")
	}
	if err := prompt.RegisterTemplate("dialogue", ZundaMetanDialoguePrompt); err != nil {
		safePanic(fmt.Sprintf("対話テンプレートの登録に失敗: %v", err))
	}
}
