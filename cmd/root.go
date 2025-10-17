package cmd

import (
	"context"
	_ "embed" // go:embed のためにアンダースコアインポート
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/shouni/go-ai-client/pkg/ai/gemini"
	"github.com/shouni/go-ai-client/pkg/prompt"
	"github.com/spf13/cobra"
)

// --- グローバル設定（フラグ変数） ---

const separator = "=============================================="

var (
	modelName string
	timeout   int
)

// --- ルートコマンド定義 ---

// rootCmd はアプリケーションのメインコマンドです
var rootCmd = &cobra.Command{
	Use:   "ai-client",
	Short: "Google Gemini APIを利用したCLIインターフェース。",
	Long: `ai-client は、Gemini APIを利用してテキスト生成やスクリプト生成を行うCLIです。
サブコマンド (prompt または generic) を使って実行モードを指定してください。`,
}

// --- ユーティリティ関数（全コマンドで共有） ---

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

// checkAPIKey は、APIキー環境変数が設定されているかを確認します。
func checkAPIKey() error {
	if os.Getenv("GEMINI_API_KEY") == "" && os.Getenv("GOOGLE_API_KEY") == "" {
		return fmt.Errorf("致命的エラー: GEMINI_API_KEY または GOOGLE_API_KEY 環境変数が設定されていません。")
	}
	return nil
}

// generateAndOutput は、Gemini APIを呼び出し、結果を標準出力に出力する共通ロジックです。
// mode パラメータは、promptCmdではテンプレートモード、genericCmdでは "generic" という固定文字列が入ります。
func generateAndOutput(ctx context.Context, inputContent []byte, mode, modelName string) error {
	// 1. クライアントの初期化
	client, err := gemini.NewClientFromEnv(ctx)
	if err != nil {
		return fmt.Errorf("Geminiクライアントの初期化に失敗しました: %w", err)
	}

	// 2. 応答の生成
	fmt.Printf("モデル %s で応答を生成中 (モード: %s, Timeout: %d秒)...\n", modelName, mode, timeout)

	resp, err := client.GenerateContent(ctx, inputContent, mode, modelName)

	if err != nil {
		return fmt.Errorf("API処理中にエラーが発生しました: %w", err)
	}

	// 3. 結果の出力
	fmt.Println("\n" + separator)
	fmt.Printf("|| 応答 (モデル: %s, モード: %s) ||\n", modelName, mode)
	fmt.Println(separator)
	fmt.Println(resp.Text)
	fmt.Println(separator)

	return nil
}

// Execute はルートコマンドを実行します。
func Execute() error {
	return rootCmd.Execute()
}

// init() はアプリケーション起動時に自動的に実行され、フラグとプロンプトテンプレートを設定します。
func init() {
	// ルートコマンドに PersistentFlags (全サブコマンドで共通) を設定
	rootCmd.PersistentFlags().IntVarP(&timeout, "timeout", "t", 60, "APIリクエストのタイムアウト時間 (秒)")
	rootCmd.PersistentFlags().StringVarP(&modelName, "model", "m", "gemini-2.5-flash", "使用するGeminiモデル名")

	// サブコマンドの追加 (他ファイルで定義されたコマンドをここで登録)
	rootCmd.AddCommand(newPromptCmd())
	rootCmd.AddCommand(newGenericCmd())

	// 埋め込まれた string 変数を使って prompt パッケージに登録する
	registerPromptTemplates()
}

// registerPromptTemplates は、埋め込まれた string 変数からテンプレートを読み込み、pkg/prompt に登録します。
func registerPromptTemplates() {
	// ユーティリティ関数: エラー発生時にエラーメッセージを出力し、終了コード1でプロセスを終了する
	safeExit := func(msg string) {
		fmt.Fprintf(os.Stderr, "クリティカルエラー (起動時): %s\n", msg)
		os.Exit(1)
	}

	// 1. Soloモードのテンプレート登録
	if ZundamonSoloPrompt == "" {
		safeExit("ソロテンプレートの埋め込みが失敗しているか、ファイルが空です。")
	}
	if err := prompt.RegisterTemplate("solo", ZundamonSoloPrompt); err != nil {
		safeExit(fmt.Sprintf("ソロテンプレートの登録に失敗: %v", err))
	}

	// 2. Dialogueモードのテンプレート登録
	if ZundaMetanDialoguePrompt == "" {
		safeExit("対話テンプレートの埋め込みが失敗しているか、ファイルが空です。")
	}
	if err := prompt.RegisterTemplate("dialogue", ZundaMetanDialoguePrompt); err != nil {
		safeExit(fmt.Sprintf("対話テンプレートの登録に失敗: %v", err))
	}
}
