package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/shouni/go-ai-client/v2/pkg/ai/gemini"
	"github.com/shouni/go-utils/iohandler"
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
  # ファイルから読み込み、標準出力に出力
  ai-client generic -i input.txt

  # 直接テキストを渡し、ファイルに出力
  ai-client generic "量子コンピュータについて5行で解説せよ" -o output.txt`,

		RunE: func(cmd *cobra.Command, args []string) error {
			// 1. 入力内容の決定 (引数 > ファイル/stdin)
			var inputText string
			if len(args) > 0 {
				// コマンドライン引数が提供されている場合、それを優先
				inputText = args[0]
			} else {
				// フラグやパイプ入力を処理 (記憶している ReadInput を利用)
				content, err := iohandler.ReadInput("")
				if err != nil {
					return err
				}
				inputText = string(content)
			}

			if inputText == "" {
				return fmt.Errorf("処理するための入力テキストがありません。テキストを引数として渡すか、ファイル/標準入力から提供してください。")
			}

			// 2. クライアント初期化
			// 環境変数からクライアントを生成 (context は cmd.Context() を使用)
			client, err := gemini.NewClientFromEnv(cmd.Context())
			if err != nil {
				return fmt.Errorf("AIクライアントの初期化に失敗しました: %w", err)
			}

			// 3. タイムアウト設定とコンテンツ生成
			clientCtx, cancel := context.WithTimeout(cmd.Context(), time.Duration(Timeout)*time.Second)
			defer cancel()

			// Gemini APIを呼び出し
			content, err := client.GenerateContent(clientCtx, inputText, ModelName)
			if err != nil {
				return fmt.Errorf("AIコンテンツ生成中にエラーが発生しました: %w", err)
			}

			return GenerateAndOutput(cmd.Context(), content.Text)
		},
	}
	return cmd
}

func init() {
	// NewGenericCmdを呼び出す前に、genericCmdがnilでないことを確認するロジックは不要です
	// NewGenericCmdが必ず新しい*cobra.Commandを返すため、直接代入し、rootCmdに追加します。
	genericCmd = NewGenericCmd()
	rootCmd.AddCommand(genericCmd)
}
