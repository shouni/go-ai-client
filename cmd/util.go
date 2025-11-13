package cmd

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"

	clibase "github.com/shouni/go-cli-base"
	"github.com/shouni/go-utils/iohandler"
	"github.com/spf13/cobra"
)

// セパレータの定数定義
const (
	separatorHeavy = "=============================================="
	separatorLight = "----------------------------------------------"
)

// readInput は、コマンドライン引数、ファイルフラグ、標準入力の順序で
func readInput(cmd *cobra.Command, args []string) ([]byte, error) {
	// 1. コマンドライン引数からの読み込みを優先 (パイプ処理との混同を避けるため)
	if len(args) > 0 {
		// 読み込み元を標準エラー出力で通知
		fmt.Fprintf(cmd.ErrOrStderr(), "コマンドライン引数から読み込み中...\n")
		// 引数をスペース区切りで結合して返す
		return []byte(strings.Join(args, " ")), nil
	}

	// 2. 標準入力からの読み込み
	// cmd.InOrStdin() を使用して標準入力から読み込み
	fmt.Fprintf(cmd.ErrOrStderr(), "標準入力 (stdin) から読み込み中...\n")

	input, err := io.ReadAll(cmd.InOrStdin())
	if err != nil {
		// io.ReadAll のエラーは通常、リソースの切断など致命的な問題
		return nil, fmt.Errorf("標準入力からの読み込みに失敗しました: %w", err)
	}

	// 3. 空入力のチェック
	if len(bytes.TrimSpace(input)) == 0 {
		// バイトスライスをトリムして、空白や改行のみでないか確認
		// 致命的エラーではなく、適切な使い方を促すメッセージにする
		return nil, fmt.Errorf("入力エラー: 処理するテキストが提供されていません。\n\n使用法:\n1. コマンド引数として直接指定: `yourcommand \"テキスト内容\"`\n2. 標準入力としてパイプで渡す: `cat input.txt | yourcommand`")
	}

	return input, nil
}

// GenerateAndOutput は、RunnerのRunメソッドを呼び出し、結果として得られた
// AIの応答内容を標準出力に出力し、メタ情報を付加します。
func GenerateAndOutput(ctx context.Context, outputContent string) error {
	// 全ての出力を一つの文字列に組み立てる
	var sb strings.Builder

	// 応答の開始セパレータとヘッダー (定数を使用)
	sb.WriteString("\n" + separatorHeavy)
	sb.WriteString("\n🤖 AIモデルからの応答:")
	sb.WriteString("\n" + separatorHeavy + "\n")

	// AIの応答本文
	sb.WriteString(outputContent)

	// 応答の終了セパレータとメタ情報 (定数を使用)
	sb.WriteString("\n\n" + separatorLight)

	// メタ情報
	sb.WriteString(fmt.Sprintf("\nModel: %s", modelName))
	//	sb.WriteString(fmt.Sprintf("\n実行モード: %s", displayMode))
	sb.WriteString(fmt.Sprintf("\n出力処理時刻: %s", time.Now().Format("2006-01-02 15:04:05")))

	// 終了セパレータ
	sb.WriteString("\n" + separatorLight + "\n")

	return iohandler.WriteOutputString("", sb.String()) // 第一引数の空文字列は標準出力を意味する
}

// checkAPIKey、initAppPreRunE 関数は変更なし

// checkAPIKey は、APIキー環境変数が設定されているかを確認します。
func checkAPIKey() error {
	if os.Getenv("GEMINI_API_KEY") == "" && os.Getenv("GOOGLE_API_KEY") == "" {
		return fmt.Errorf("致命的エラー: GEMINI_API_KEY または GOOGLE_API_KEY 環境変数が設定されていません。")
	}
	return nil
}

// initAppPreRunE は、ログレベル設定とAPIキーチェックを実行します。
func initAppPreRunE(cmd *cobra.Command, args []string) error {
	// ログレベル設定
	logLevel := slog.LevelInfo
	if clibase.Flags.Verbose {
		logLevel = slog.LevelDebug
	}
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: logLevel,
	})
	slog.SetDefault(slog.New(handler))

	// APIキーチェック
	err := checkAPIKey()
	if err != nil {
		slog.Error("🚨 APIKeyの取得に失敗しました", "error", err)
		return fmt.Errorf("APIKeyの取得に失敗しました: %w", err)
	}

	slog.Info("アプリケーション設定初期化完了")
	return nil
}
