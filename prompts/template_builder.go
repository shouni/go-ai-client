package prompts

import (
	"fmt"
	"strings"
	"text/template"
)

// Builder は、最終的なAIプロンプトを構築する契約です。
// より具体的な実装名と区別するため、インターフェース名を Builder に変更することも検討できます。
type Builder interface {
	Build(data TemplateData, mode string) (string, error) // 慣習に合わせ引数順序を調整
}

// PromptBuilder は Builder インターフェースを実装します。
type PromptBuilder struct {
	templates map[string]*template.Template
}

// NewPromptBuilder は PromptBuilder を初期化し、すべてのテンプレートを一度パースしてキャッシュします。
func NewPromptBuilder() (*PromptBuilder, error) {
	parsedTemplates := make(map[string]*template.Template)
	for mode, content := range allTemplates {
		if content == "" {
			return nil, fmt.Errorf("プロンプトテンプレート '%s' (go:embed) の読み込みに失敗: 内容が空です", mode)
		}

		tmpl, err := template.New(mode).Parse(content)
		if err != nil {
			// エラーメッセージをより詳細に
			return nil, fmt.Errorf("テンプレート '%s' の解析に失敗しました: %w", mode, err)
		}
		parsedTemplates[mode] = tmpl
	}

	return &PromptBuilder{
		templates: parsedTemplates,
	}, nil
}

// Build は、TemplateDataを埋め込み、要求されたモードに応じて適切なテンプレートを実行します。
func (b *PromptBuilder) Build(data TemplateData, mode string) (string, error) {
	tmpl, ok := b.templates[mode]
	if !ok {
		return "", fmt.Errorf("不明なモードです: '%s'", mode)
	}

	var sb strings.Builder
	// テンプレートの実行
	if err := tmpl.Execute(&sb, data); err != nil {
		return "", fmt.Errorf("プロンプトテンプレート '%s' の実行に失敗しました: %w", mode, err)
	}

	return sb.String(), nil
}
