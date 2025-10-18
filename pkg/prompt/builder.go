package prompt

import (
	"bytes"
	"fmt"
	"sync"
	"text/template"
)

var parsedTemplateMap = make(map[string]*template.Template)
var mapMutex sync.RWMutex

// RegisterTemplate は、指定されたモード名に対してプロンプトテンプレート文字列を登録します。
func RegisterTemplate(mode string, templateString string) error {
	if mode == "" {
		return fmt.Errorf("モード名は空にできません")
	}
	if templateString == "" {
		return fmt.Errorf("テンプレート文字列は空にできません")
	}

	// 1. テンプレートの解析 (ここでコストの高い処理を実行)
	tmpl, err := template.New(mode).Parse(templateString)
	if err != nil {
		return fmt.Errorf("モード %s のプロンプトテンプレート解析エラー: %w", mode, err)
	}

	mapMutex.Lock()
	defer mapMutex.Unlock()

	if _, exists := parsedTemplateMap[mode]; exists {
		// 初期化段階での重複登録を許可し、上書きしてもエラーにしないように変更
		// 必要に応じてここでエラーを返すことも可能です。
		// return fmt.Errorf("モード %s のテンプレートは既に登録されています", mode)
	}

	// 2. 解析済みテンプレートをキャッシュ
	parsedTemplateMap[mode] = tmpl
	return nil
}

// GetParsedPromptByMode は指定されたモードに対応する解析済みテンプレートを取得します。
func getParsedPromptByMode(mode string) (*template.Template, error) {
	mapMutex.RLock()
	defer mapMutex.RUnlock()

	tmpl, ok := parsedTemplateMap[mode]
	if !ok {
		return nil, fmt.Errorf("未対応のモードです。テンプレートが登録されていません: %s", mode)
	}

	return tmpl, nil
}

// BuildFullPrompt はテンプレートを取得し、入力内容を埋め込んだ最終プロンプト文字列を構築します。
func BuildFullPrompt(inputText string, mode string) (string, error) {
	// 1. 解析済みテンプレートを取得 (キャッシュから取得するため高速)
	tmpl, err := getParsedPromptByMode(mode)
	if err != nil {
		return "", err
	}

	// 2. プロンプトにユーザーの入力テキストを埋め込む
	// テンプレートの変数名 (InputText) は固定とします。
	type InputData struct{ InputText string }

	// データの埋め込み
	data := InputData{InputText: inputText}
	var fullPrompt bytes.Buffer

	// Execute を実行するのみ
	if err := tmpl.Execute(&fullPrompt, data); err != nil {
		return "", fmt.Errorf("プロンプトへの入力埋め込みエラー (モード: %s): %w", mode, err)
	}

	return fullPrompt.String(), nil
}
