package prompts

import (
	_ "embed"
	"fmt"
	"strings"
	"text/template"
)

// ----------------------------------------------------------------
// テンプレート構造体
// ----------------------------------------------------------------

// TemplateData はプロンプトのテンプレートに渡すデータ構造です。
type TemplateData struct {
	Content string
}

// ----------------------------------------------------------------
// ビルダー実装
// ----------------------------------------------------------------

// PromptBuilder はレビュープロンプトの構成を管理します。
type PromptBuilder struct {
	// 差分を埋め込むための text/template を保持します
	tmpl *template.Template
}

// NewPromptBuilder は PromptBuilder を初期化します。
// テンプレート文字列を受け取り、それをパースして *template.Template を保持します。
// name はテンプレートの名前であり、主にデバッグやエラーメッセージの識別に利用されます。
func NewPromptBuilder(name string, templateContent string) (*PromptBuilder, error) {
	if templateContent == "" {
		return nil, fmt.Errorf("プロンプトテンプレートの内容が空です")
	}

	tmpl, err := template.New(name).Parse(templateContent)
	if err != nil {
		return nil, fmt.Errorf("プロンプトテンプレートの解析に失敗しました: %w", err)
	}
	return &PromptBuilder{tmpl: tmpl}, nil
}

// Build は ReviewTemplateData を埋め込み、Geminiへ送るための最終的なプロンプト文字列を完成させます。
func (b *PromptBuilder) Build(data TemplateData) (string, error) {
	if b.tmpl == nil {
		return "", fmt.Errorf("プロンプトテンプレートが適切に初期化されていません。NewRPromptBuilderが正しく呼び出されたか確認してください")
	}

	var sb strings.Builder
	if err := b.tmpl.Execute(&sb, data); err != nil {
		return "", fmt.Errorf("プロンプトの実行に失敗しました: %w", err)
	}

	return sb.String(), nil
}
