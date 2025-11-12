package builder

import (
	"github.com/shouni/go-ai-client/v2/pkg/ai/gemini"
	"github.com/shouni/go-ai-client/v2/pkg/promptbuilder"
)

// GeminiServiceDependencies は、Geminiクライアントとプロンプトビルダを保持する構造体です。
type GeminiServiceDependencies struct {
	GeminiClient  gemini.GenerativeModel
	PromptBuilder *promptbuilder.PromptBuilder
}
