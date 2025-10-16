package prompt

import (
	"bytes"
	"fmt"
	"text/template"
)

// GetPromptByMode は指定されたモード（solo, dialogueなど）に対応する
// プロンプトテンプレート文字列を取得します。
// この関数はクライアントコード（Gemini）から呼び出されるため、公開（大文字）が必要です。
func GetPromptByMode(mode string) (string, error) {
	switch mode {
	case "solo":
		return `
あなたは熟練したナレーターです。以下の入力テキストを読み上げやすいように、一人の話者によるモノローグ形式のスクリプトに変換してください。
スクリプトの先頭には、[話者タグ]を付けてください。話者は「ずんだもん」を使ってください。

[入力テキスト]
{{.InputText}}
`, nil
	case "dialogue":
		return `
あなたは対話形式のスクリプトを生成するAIです。以下の入力テキストを元に、二人の話者（ずんだもん、めたん）による対話スクリプトに変換してください。
各行の先頭には、[話者タグ]を付けてください。

[入力テキスト]
{{.InputText}}
`, nil
	default:
		return "", fmt.Errorf("未対応のモードです: %s", mode)
	}
}

// BuildFullPrompt はテンプレートを取得し、入力内容を埋め込んだ最終プロンプト文字列を構築します。
// この関数は pkg/ai/gemini/client.go の GenerateScript メソッドから呼び出される公開関数です。
func BuildFullPrompt(inputContent []byte, mode string) (string, error) {
	// 1. プロンプトのテンプレートを取得
	promptTemplateString, err := GetPromptByMode(mode)
	if err != nil {
		return "", err
	}

	// 2. プロンプトにユーザーの入力テキストを埋め込む
	type InputData struct{ InputText string }

	// テンプレートの解析
	tmpl, err := template.New("narration_prompt").Parse(promptTemplateString)
	if err != nil {
		return "", fmt.Errorf("プロンプトテンプレートの解析エラー: %w", err)
	}

	// データの埋め込み
	data := InputData{InputText: string(inputContent)}
	var fullPrompt bytes.Buffer
	if err := tmpl.Execute(&fullPrompt, data); err != nil {
		return "", fmt.Errorf("プロンプトへの入力埋め込みエラー: %w", err)
	}

	return fullPrompt.String(), nil
}
