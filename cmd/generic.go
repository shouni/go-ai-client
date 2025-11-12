package cmd

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/shouni/go-ai-client/pkg/prompt"
	"github.com/spf13/cobra"
)

// GenericCmd は 'generic' サブコマンドのインスタンスです。（公開）
var genericCmd = NewGenericCmd()

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

// readInput は、コマンドライン引数または標準入力からテキストを読み込みます。
func readInput(cmd *cobra.Command, args []string) ([]byte, error) {
	if len(args) > 0 {
		return []byte(strings.Join(args, " ")), nil
	}
	input, err := io.ReadAll(cmd.InOrStdin())
	if err != nil {
		return nil, fmt.Errorf("標準入力からの読み込みエラー: %w", err)
	}
	if len(input) == 0 {
		return nil, fmt.Errorf("致命的エラー: 処理するテキストがコマンドライン引数または標準入力から提供されていません。")
	}
	return input, nil
}

// GenerateAndOutput は、Gemini APIを呼び出し、結果を標準出力に出力する共通ロジックです。（公開）
func GenerateAndOutput(ctx context.Context, inputContent []byte, subcommandMode, modelName string) error {
	// 1. クライアントの初期化
	clientCtx, cancel := context.WithTimeout(ctx, time.Duration(Timeout)*time.Second)
	defer cancel()

	client, err := gemini.NewClientFromEnv(clientCtx)
	if err != nil {
		return fmt.Errorf("Geminiクライアントの初期化に失敗しました: %w", err)
	}

	var finalPrompt string
	modeDisplay := subcommandMode
	inputText := string(inputContent)

	// 2. プロンプトの構築ロジック
	if subcommandMode == "generic" {
		finalPrompt = inputText
		modeDisplay = "テンプレートなし (generic)"
	} else {
		finalPrompt, err = prompt.BuildFullPrompt(inputText, subcommandMode)
		if err != nil {
			return fmt.Errorf("failed to build full prompt (mode: %s): %w", subcommandMode, err)
		}
	}

	fmt.Printf("モデル %s で応答を生成中 (モード: %s, Timeout: %d秒)...\n", modelName, modeDisplay, Timeout)

	// 3. 応答の生成
	resp, err := client.GenerateContent(clientCtx, finalPrompt, modelName)

	if err != nil {
		return fmt.Errorf("API処理中にエラーが発生しました: %w", err)
	}

	// 4. 結果の出力
	fmt.Println("\n" + separator)
	fmt.Printf("|| 応答 (モデル: %s, モード: %s) ||\n", modelName, modeDisplay)
	fmt.Println(separator)
	fmt.Println(resp.Text)
	fmt.Println(separator)

	return nil
}
