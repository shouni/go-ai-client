package prompt

import (
	"bytes"
	"fmt"
	"sync"
	"text/template"
)

// templateMap は、モード名 (string) とそれに対応するプロンプトテンプレート文字列を格納するマップです。
var templateMap = make(map[string]string)
var mapMutex sync.RWMutex

// RegisterTemplate は、指定されたモード名に対してプロンプトテンプレート文字列を登録します。
// これは、アプリケーションの初期化段階（例: cmd/rootのinit関数）で呼び出されることを想定しています。
// 外部からのテンプレート注入を可能にし、promptパッケージの汎用性を高めます。
func RegisterTemplate(mode string, templateString string) error {
	if mode == "" {
		return fmt.Errorf("モード名は空にできません")
	}
	if templateString == "" {
		return fmt.Errorf("テンプレート文字列は空にできません")
	}

	mapMutex.Lock()
	defer mapMutex.Unlock()

	if _, exists := templateMap[mode]; exists {
		// 既に存在するテンプレートの上書きを許可するかどうかは設計次第ですが、
		// ここではエラーとして、重複登録を防ぎます。
		return fmt.Errorf("モード %s のテンプレートは既に登録されています", mode)
	}

	templateMap[mode] = templateString
	return nil
}

// GetPromptByMode は指定されたモードに対応するプロンプトテンプレート文字列を取得します。
// ハードコードされた switch 文の代わりに、登録されたマップを参照します。
// この関数はクライアントコード（Gemini）から呼び出されるため、公開が必要です。
func GetPromptByMode(mode string) (string, error) {
	mapMutex.RLock()
	defer mapMutex.RUnlock()

	templateString, ok := templateMap[mode]
	if !ok {
		return "", fmt.Errorf("未対応のモードです。テンプレートが登録されていません: %s", mode)
	}

	return templateString, nil
}

// BuildFullPrompt はテンプレートを取得し、入力内容を埋め込んだ最終プロンプト文字列を構築します。
// pkg/ai/gemini/client.go の GenerateScript メソッドから呼び出される公開関数です。
func BuildFullPrompt(inputContent []byte, mode string) (string, error) {
	// 1. プロンプトのテンプレートを取得 (マップから参照)
	promptTemplateString, err := GetPromptByMode(mode)
	if err != nil {
		return "", err
	}

	// 2. プロンプトにユーザーの入力テキストを埋め込む
	// テンプレートの変数名 (InputText) は固定とします。
	type InputData struct{ InputText string }

	// テンプレートの解析
	// Note: パフォーマンスのため、テンプレートは一度だけ解析し、キャッシュする方が望ましいですが、
	// ここではシンプルさを優先します。
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
