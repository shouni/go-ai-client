package prompts

import (
	"bytes"
	_ "embed"
	"fmt"
	"text/template"
)

// ----------------------------------------------------------------
// インターフェース定義
// ----------------------------------------------------------------

// TemplateData はプロンプトのテンプレートに渡すデータ構造です。
type TemplateData struct {
	Content string
}

// PromptBuilder は、テンプレートデータから最終的なプロンプト文字列を生成する責務を定義します。
// これにより、具体的な実装（text/templateなど）から利用側を分離できます。
type PromptBuilder interface {
	Build(data TemplateData) (string, error)
}

// ----------------------------------------------------------------
// ビルダー実装
// ----------------------------------------------------------------

// textPromptBuilder は text/template を使用した PromptBuilder の具体的な実装です。
type textPromptBuilder struct {
	tmpl *template.Template
}

// NewPromptBuilder は PromptBuilder インターフェースを実装する新しいインスタンスを初期化します。
// name はテンプレートの名前であり、主にデバッグやエラーメッセージの識別に利用されます。
func NewPromptBuilder(name string, templateContent string) (PromptBuilder, error) {
	if templateContent == "" {
		// テンプレート名もエラーに含めると、利用側でのデバッグが容易になります。
		return nil, fmt.Errorf("プロンプトテンプレート '%s' の内容が空です", name)
	}

	// Option: .Funcs() などでカスタム関数を登録することもできますが、今回はシンプルに保ちます。
	tmpl, err := template.New(name).Parse(templateContent)
	if err != nil {
		return nil, fmt.Errorf("プロンプトテンプレート '%s' の解析に失敗しました: %w", name, err)
	}
	// テンプレートの検証を助けるため、具体的な実装構造体（textPromptBuilder）のポインタを返します。
	return &textPromptBuilder{tmpl: tmpl}, nil
}

// Build は TemplateData を埋め込み、最終的なプロンプト文字列を完成させます。
func (b *textPromptBuilder) Build(data TemplateData) (string, error) {
	// strings.Builder の代わりに bytes.Buffer を使用。これは io.Writer を実装しており、
	// text/template の Execute に渡す標準的な方法です。
	var buf bytes.Buffer
	if err := b.tmpl.Execute(&buf, data); err != nil {
		// tmpl.Name() を利用してエラーを具体化することも可能です
		return "", fmt.Errorf("プロンプトテンプレート '%s' の実行に失敗しました: %w", b.tmpl.Name(), err)
	}

	return buf.String(), nil
}
