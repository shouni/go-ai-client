package prompts

import (
	"fmt"
	"strings"
	"testing"
	"text/template"
)

// testTemplates は、テストで使用するためのテンプレートのモックデータです。
// テンプレート変数の参照を {{.DiffContent}} から {{.Content}} に変更します。
var testTemplates = map[string]string{
	"release":    "リリースレビューのプロンプト: {{.Content}}",
	"detail":     "詳細レビューのプロンプト（HTMLエスケープ確認）: {{.Content | html}}",
	"empty_mode": "",           // NewPromptBuilderでの空コンテンツエラー確認用
	"bad_syntax": "{{.Content", // NewPromptBuilderでの解析エラー確認用
}

// NewPromptBuilder_TestHelper は、テストのためにテンプレートマップを外部から注入するためのヘルパー関数です。
// 実際の NewPromptBuilder のロジックを再利用し、allTemplates の代わりに引数を使用します。
func NewPromptBuilder_TestHelper(templates map[string]string) (*PromptBuilder, error) {
	parsedTemplates := make(map[string]*template.Template)
	for mode, content := range templates {
		if content == "" {
			return nil, fmt.Errorf("プロンプトテンプレート '%s' の読み込みに失敗: 内容が空です", mode)
		}

		// 実際のNewPromptBuilderのロジックを模倣
		tmpl, err := template.New(mode).Parse(content)
		if err != nil {
			return nil, fmt.Errorf("テンプレート '%s' の解析に失敗しました: %w", mode, err)
		}
		parsedTemplates[mode] = tmpl
	}

	return &PromptBuilder{
		templates: parsedTemplates,
	}, nil
}

// TestNewPromptBuilder は NewPromptBuilder の初期化ロジックをテストします。
func TestNewPromptBuilder(t *testing.T) {
	// 1. 成功ケース
	t.Run("Success", func(t *testing.T) {
		builder, err := NewPromptBuilder_TestHelper(map[string]string{"valid1": "T1", "valid2": "T2"})
		if err != nil {
			t.Fatalf("NewPromptBuilder_TestHelper がエラーを返しました: %v", err)
		}
		if builder == nil {
			t.Fatal("NewPromptBuilder_TestHelper が nil を返しました")
		}
		if len(builder.templates) != 2 {
			t.Errorf("期待されるテンプレート数: 2, 実際: %d", len(builder.templates))
		}
	})

	// 2. テンプレート解析失敗ケース
	t.Run("ParseFailure", func(t *testing.T) {
		badTemplates := map[string]string{
			"fail": testTemplates["bad_syntax"], // 無効なテンプレート構文
		}
		_, err := NewPromptBuilder_TestHelper(badTemplates)
		if err == nil {
			t.Fatal("無効なテンプレート構文でエラーが期待されましたが、nilでした")
		}
		expectedErrorSubstring := "テンプレート 'fail' の解析に失敗しました"
		if !strings.Contains(err.Error(), expectedErrorSubstring) {
			t.Errorf("エラーメッセージが期待値と異なります。\n期待される部分文字列: '%s'\n実際のエラー: %v", expectedErrorSubstring, err)
		}
	})

	// 3. コンテンツ空（go:embed失敗をシミュレート）ケース
	t.Run("EmptyContentFailure", func(t *testing.T) {
		emptyTemplates := map[string]string{
			"empty": testTemplates["empty_mode"], // 空のコンテンツ
		}
		_, err := NewPromptBuilder_TestHelper(emptyTemplates)
		if err == nil {
			t.Fatal("空のテンプレートコンテンツでエラーが期待されましたが、nilでした")
		}
		expectedErrorSubstring := "プロンプトテンプレート 'empty' の読み込みに失敗"
		if !strings.Contains(err.Error(), expectedErrorSubstring) {
			t.Errorf("エラーメッセージが期待値と異なります。\n期待される部分文字列: '%s'\n実際のエラー: %v", expectedErrorSubstring, err)
		}
	})
}

// TestPromptBuilder_Build は Build メソッドのテンプレート実行ロジックをテストします。
func TestPromptBuilder_Build(t *testing.T) {
	// 成功するテンプレートのみを含むマップを定義 (エラーを起こすテンプレートを除外)
	cleanTemplates := map[string]string{
		"release": testTemplates["release"],
		"detail":  testTemplates["detail"],
	}

	// 事前準備: 正常に初期化された PromptBuilder
	builder, err := NewPromptBuilder_TestHelper(cleanTemplates)
	if err != nil {
		t.Fatalf("テストセットアップが失敗しました: %v", err)
	}

	// Contentフィールドを使用してテストデータを初期化します。
	testData := TemplateData{
		Content: "変更点\n- func main() { ... }", // ★ 修正点: DiffContent -> Content
	}

	// 1. 成功ケース (release)
	t.Run("Success_ReleaseMode", func(t *testing.T) {
		mode := "release"
		expected := "リリースレビューのプロンプト: 変更点\n- func main() { ... }"
		result, err := builder.Build(testData, mode)

		if err != nil {
			t.Fatalf("モード '%s' で Build がエラーを返しました: %v", mode, err)
		}
		if result != expected {
			t.Errorf("モード: %s\n期待される結果:\n%s\n実際の結果:\n%s", mode, expected, result)
		}
	})

	// 2. 成功ケース (detail - HTMLエスケープ確認)
	t.Run("Success_DetailMode_EscapeCheck", func(t *testing.T) {
		mode := "detail"
		// HTML特殊文字（<, >, &）を含むデータ
		dataWithHTML := TemplateData{
			Content: "i < 10 && j > 0", // ★ 修正点: DiffContent -> Content
		}
		// | html パイプラインによりエスケープされることを確認
		expected := "詳細レビューのプロンプト（HTMLエスケープ確認）: i &lt; 10 &amp;&amp; j &gt; 0"
		result, err := builder.Build(dataWithHTML, mode)

		if err != nil {
			t.Fatalf("モード '%s' で Build がエラーを返しました: %v", mode, err)
		}
		if result != expected {
			t.Errorf("モード: %s\n期待される結果:\n%s\n実際の結果:\n%s", mode, expected, result)
		}
	})

	// 3. 不明なモードケース
	t.Run("UnknownModeFailure", func(t *testing.T) {
		mode := "unknown"
		_, err := builder.Build(testData, mode)

		if err == nil {
			t.Fatal("不明なモードでエラーが期待されましたが、nilでした")
		}
		expectedErrorSubstring := fmt.Sprintf("不明なモードです: '%s'", mode)
		if !strings.Contains(err.Error(), expectedErrorSubstring) {
			t.Errorf("エラーメッセージが期待値と異なります。\n期待される部分文字列: '%s'\n実際のエラー: %v", expectedErrorSubstring, err)
		}
	})
}
