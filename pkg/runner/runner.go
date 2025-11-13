package runner

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/shouni/go-ai-client/v2/pkg/ai/gemini"
	"github.com/shouni/go-ai-client/v2/pkg/promptbuilder"
)

// PromptTemplateGetter は、モードに基づいてテンプレートを取得するインターフェースを定義します。
type PromptTemplateGetter interface {
	GetTemplate(mode string) (name string, content string, err error)
}

// TemplateGetterFunc は、関数を PromptTemplateGetter インターフェースに適合させるためのアダプタ型です。
type TemplateGetterFunc func(mode string) (name string, content string, err error)

// GetTemplate は PromptTemplateGetter インターフェースのメソッドを満たします。
func (f TemplateGetterFunc) GetTemplate(mode string) (name string, content string, err error) {
	return f(mode) // 内部でラップされた関数を呼び出す
}

// PromptBuilderConstructor は、プロンプトビルダーを生成する関数シグネチャを定義します。
type PromptBuilderConstructor func(name string, templateContent string) (*promptbuilder.PromptBuilder, error)

// Runner は AI 応答の生成と出力を管理するメインの実行構造体です。
// 依存関係を外部から注入（DI）します。
type Runner struct {
	Client             gemini.GenerativeModel // APIクライアント（インターフェース）
	TemplateGetter     PromptTemplateGetter
	BuilderConstructor PromptBuilderConstructor
	ModelName          string
	Timeout            time.Duration
}

// NewRunner は Runner の新しいインスタンスを作成します。
func NewRunner(
	client gemini.GenerativeModel,
	getter PromptTemplateGetter,
	constructor PromptBuilderConstructor,
	modelName string,
	timeout time.Duration,
) *Runner {
	return &Runner{
		Client:             client,
		TemplateGetter:     getter,
		BuilderConstructor: constructor,
		ModelName:          modelName,
		Timeout:            timeout,
	}
}

// BuildFullPrompt は、指定されたモードと入力コンテンツに基づいて
// 最終的なプロンプト文字列を構築します。
func (r *Runner) BuildFullPrompt(inputText string, mode string) (string, error) {
	// 1. テンプレートの取得
	templateName, templateContent, err := r.TemplateGetter.GetTemplate(mode)
	if err != nil {
		return "", err
	}

	// 2. PromptBuilder の初期化
	builder, err := r.BuilderConstructor(templateName, templateContent)
	if err != nil {
		return "", err
	}

	// 3. データの埋め込みとプロンプトの構築
	data := promptbuilder.TemplateData{
		Content: inputText,
	}

	finalPrompt, err := builder.Build(data)
	if err != nil {
		return "", fmt.Errorf("プロンプトの実行と構築に失敗しました: %w", err)
	}

	return finalPrompt, nil
}

// Run は、プロンプトを構築し、APIを呼び出し、結果を標準出力に出力する共通ロジックです。
func (r *Runner) Run(ctx context.Context, inputContent []byte, subcommandMode string) error {
	clientCtx, cancel := context.WithTimeout(ctx, r.Timeout)
	defer cancel()

	var finalPrompt string
	modeDisplay := subcommandMode
	inputText := string(inputContent)

	if subcommandMode == "generic" {
		finalPrompt = inputText
		modeDisplay = "テンプレートなし (generic)"
	} else {
		var err error
		finalPrompt, err = r.BuildFullPrompt(inputText, subcommandMode)
		if err != nil {
			return fmt.Errorf("failed to build full prompt (mode: %s): %w", subcommandMode, err)
		}
	}

	slog.Info("応答生成リクエスト送信", "model", r.ModelName, "mode", modeDisplay, "timeout", r.Timeout)
	fmt.Printf("モデル %s で応答を生成中 (モード: %s, Timeout: %d秒)...\n", r.ModelName, modeDisplay, int(r.Timeout.Seconds()))

	// API呼び出し
	resp, err := r.Client.GenerateContent(clientCtx, finalPrompt, r.ModelName)

	if err != nil {
		return fmt.Errorf("API処理中にエラーが発生しました: %w", err)
	}

	// 結果出力
	fmt.Printf("|| 応答 (モデル: %s, モード: %s) ||\n", r.ModelName, modeDisplay)
	fmt.Println(resp.Text)

	return nil
}
