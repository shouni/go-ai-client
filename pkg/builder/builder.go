package builder

import (
	"context"
	"fmt"

	"github.com/shouni/go-ai-client/v2/pkg/ai/gemini"
	"github.com/shouni/go-ai-client/v2/pkg/promptbuilder"
)

// GeminiServiceDependencies は、Geminiクライアントとプロンプトビルダを保持する構造体です。
type GeminiServiceDependencies struct {
	GeminiClient  gemini.GenerativeModel
	PromptBuilder *promptbuilder.PromptBuilder
}

// BuildGeminiClient は、環境変数からAPIキーを使用して Gemini クライアントを初期化します。
func BuildGeminiClient(ctx context.Context) (gemini.GenerativeModel, error) {
	// gemini.NewClientFromEnv (以前記憶した関数) を利用
	client, err := gemini.NewClientFromEnv(ctx)
	if err != nil {
		return nil, fmt.Errorf("Geminiクライアントの初期化エラー: %w", err)
	}
	return client, nil
}

// BuildPromptBuilder は、指定されたテンプレート名と内容を使用して PromptBuilder を初期化します。
func BuildPromptBuilder(templateName string, templateContent string) (*promptbuilder.PromptBuilder, error) {
	// promptbuilder.NewPromptBuilder (以前記憶した関数) を利用
	builder, err := promptbuilder.NewPromptBuilder(templateName, templateContent)
	if err != nil {
		return nil, fmt.Errorf("PromptBuilderの初期化エラー: %w", err)
	}
	return builder, nil
}

// BuildGeminiServiceDependencies は、個別のビルドメソッドを呼び出して依存関係全体を構築します。
func BuildGeminiServiceDependencies(
	ctx context.Context,
	templateName string,
	templateContent string,
) (*GeminiServiceDependencies, error) {
	// 1. Gemini クライアントの構築
	geminiClient, err := BuildGeminiClient(ctx)
	if err != nil {
		return nil, err
	}

	// 2. PromptBuilder の構築
	promptBuilder, err := BuildPromptBuilder(templateName, templateContent)
	if err != nil {
		return nil, err
	}

	// 3. 依存関係をまとめて注入
	return &GeminiServiceDependencies{
		GeminiClient:  geminiClient,
		PromptBuilder: promptBuilder,
	}, nil
}
