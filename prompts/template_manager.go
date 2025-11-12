package prompts

import (
	_ "embed"
	"fmt"
)

// --- テンプレートのリソース定義 (go:embed) ---
//
//go:embed prompt_solo.md
var soloPromptTemplate string

//go:embed prompt_dialogue.md
var dialoguePromptTemplate string

// GetTemplate は、モードに基づいて、テンプレート名とその内容を返します。
// エラーは、無効なモードが指定された場合に返されます。
func GetTemplate(mode string) (name string, content string, err error) {
	switch mode {
	case "solo":
		name = "solo"
		content = soloPromptTemplate
	case "dialogue":
		name = "dialogue"
		content = dialoguePromptTemplate
	default:
		// builderの堅牢性を高めるためにエラーを返す
		return "", "", fmt.Errorf("無効なモードが指定されました: '%s'。'dialogue' または 'solo' を選択してください。", mode)
	}

	// テンプレートの内容が空でないか（go:embedが失敗していないか）の基本的なチェックも追加できます
	if content == "" {
		return "", "", fmt.Errorf("モード '%s' に対応するプロンプトテンプレートの内容が空です。", mode)
	}

	return name, content, nil
}
