package prompts

import (
	_ "embed"
)

// TemplateData はレビュープロンプトのテンプレートに渡すデータ構造です。
type TemplateData struct {
	Content string
}

var (
	//go:embed prompt_solo.md
	soloPromptTemplate string
	//go:embed prompt_dialogue.md
	dialoguePromptTemplate string
)

var (
	// allTemplates は、テンプレートのMAP
	allTemplates = map[string]string{
		"solo":     soloPromptTemplate,
		"dialogue": dialoguePromptTemplate,
	}
)
