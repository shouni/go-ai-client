package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/shouni/go-ai-client/v2/pkg/ai/gemini"
	"github.com/spf13/cobra"
)

// NewGenericCmd は 'generic' コマンドを構築します。
func NewGenericCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generic [TEXT or pipe]",
		Short: "プロンプトテンプレートを使用せず、入力テキストをそのままモデルに渡します。",
		Long: `このコマンドは、モデルに特別な役割を与えず、通常のチャットや要約を目的とします。
'mode' フラグは無視されます。

利用例:
  # ファイルから読み込み、標準出力に出力
  ai-client generic -i input.txt`,

		// 実行ロジックを外部関数に委譲
		RunE: executeGenericCommand,
	}
	return cmd
}

// executeGenericCommand は 'generic' サブコマンドの実際の実行ロジックを保持します。
func executeGenericCommand(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// 1. 入力内容の決定
	// readInputは []byte, error を返す
	inputText, err := readInput(cmd, args)
	if err != nil {
		return err // readInput内で十分なエラーメッセージが出ていると想定
	}

	// 2. クライアント初期化
	// 環境変数からクライアントを生成
	client, err := gemini.NewClientFromEnv(ctx)
	if err != nil {
		return fmt.Errorf("AIクライアントの初期化に失敗しました: %w", err)
	}

	// 3. タイムアウト設定とコンテンツ生成
	// commandCtx を使用し、処理全体にタイムアウトを適用
	commandCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	// Gemini APIを呼び出し
	// inputTextは []byte なので、string() にキャストして渡す
	generateContent, err := client.GenerateContent(commandCtx, string(inputText), modelName)
	if err != nil {
		return fmt.Errorf("AIコンテンツ生成中にエラーが発生しました: %w", err)
	}

	// 4. 結果の出力
	return GenerateAndOutput(ctx, generateContent.Text)
}
