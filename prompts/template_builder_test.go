package prompts_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shouni/go-ai-client/v2/prompts"
)

// ----------------------------------------------------------------
// NewPromptBuilder のテスト
// ----------------------------------------------------------------

func TestNewPromptBuilder_Success(t *testing.T) {
	const templateName = "test_template"
	const validTemplate = "あなたはAIアシスタントです。\n入力内容: {{.Content}}"

	// 成功ケース: 有効なテンプレートをパースできること
	builder, err := prompts.NewPromptBuilder(templateName, validTemplate)
	require.NoError(t, err, "有効なテンプレートのパースでエラーが発生してはいけません")
	assert.NotNil(t, builder, "PromptBuilderインスタンスがnilであってはいけません")
}

func TestNewPromptBuilder_EmptyTemplate(t *testing.T) {
	// エラーケース 1: テンプレート内容が空の場合
	builder, err := prompts.NewPromptBuilder("empty_template", "")
	assert.Error(t, err, "空のテンプレート内容ではエラーが発生すべきです")
	assert.Contains(t, err.Error(), "プロンプトテンプレートの内容が空です", "エラーメッセージが期待通りではありません")
	assert.Nil(t, builder, "エラー発生時、PromptBuilderインスタンスはnilであるべきです")
}

func TestNewPromptBuilder_InvalidTemplateSyntax(t *testing.T) {
	const invalidTemplate = "不正なテンプレートです。{{.Content" // 閉じ括弧が不足

	// エラーケース 2: テンプレートの構文が不正な場合
	// ★ 修正: promptbuilder.NewPromptBuilder に変更
	builder, err := prompts.NewPromptBuilder("invalid_syntax", invalidTemplate)
	assert.Error(t, err, "不正なテンプレート構文でエラーが発生すべきです")
	assert.Contains(t, err.Error(), "プロンプトテンプレートの解析に失敗しました", "エラーメッセージが期待通りではありません")
	assert.Nil(t, builder, "エラー発生時、PromptBuilderインスタンスはnilであるべきです")
}

// ----------------------------------------------------------------
// Build のテスト
// ----------------------------------------------------------------

func TestPromptBuilder_Build_Success(t *testing.T) {
	const templateContent = "指示: 以下の内容に基づいて要約してください。\n\n内容:\n{{.Content}}\n"
	const inputContent = "今日は晴れでした。明日は雨の予報です。"

	// ★ 修正: promptbuilder.NewPromptBuilder に変更
	builder, err := prompts.NewPromptBuilder("build_test", templateContent)
	require.NoError(t, err)

	// ★ 修正: promptbuilder.TemplateData に変更
	data := prompts.TemplateData{Content: inputContent}

	// 成功ケース: テンプレートが正しくデータで埋め込まれること
	prompt, err := builder.Build(data)
	require.NoError(t, err, "Buildでエラーが発生してはいけません")

	expectedPrompt := fmt.Sprintf("指示: 以下の内容に基づいて要約してください。\n\n内容:\n%s\n", inputContent)
	assert.Equal(t, expectedPrompt, prompt, "生成されたプロンプトが期待値と一致しません")
}

func TestPromptBuilder_Build_EmptyData(t *testing.T) {
	const templateContent = "指示: {{.Content}} を評価してください。"

	builder, err := prompts.NewPromptBuilder("empty_data_test", templateContent)
	require.NoError(t, err)

	data := prompts.TemplateData{Content: ""}

	// エッジケース: 埋め込むデータが空文字列の場合でも、テンプレート実行は成功すること
	prompt, err := builder.Build(data)
	require.NoError(t, err, "データが空の場合でもBuildでエラーが発生してはいけません")

	expectedPrompt := "指示:  を評価してください。"
	assert.Equal(t, expectedPrompt, prompt, "空データでのプロンプト生成が期待値と一致しません")
}

func TestPromptBuilder_Build_MissingTemplateField(t *testing.T) {
	// TemplateDataに存在しないフィールド（例: .MissingField）をテンプレートが参照しているケース
	const templateName = "missing_field_test"
	const templateContent = "データ: {{.Content}} と {{.MissingField}} を使用します。"

	builder, err := prompts.NewPromptBuilder(templateName, templateContent)
	require.NoError(t, err)

	data := prompts.TemplateData{Content: "有効なデータ"}

	// 欠落フィールドの参照はデフォルトでエラーになることを期待する
	prompt, err := builder.Build(data)
	assert.Error(t, err, "欠落フィールドを参照した場合、エラーが発生すべきです")
	assert.Contains(t, err.Error(), "プロンプトの実行に失敗しました", "エラーメッセージに実行失敗の言及がありません")
	assert.Contains(t, err.Error(), "can't evaluate field MissingField", "エラーメッセージに欠落フィールドの言及がありません")
	assert.Empty(t, prompt, "エラー発生時、プロンプトは空文字列であるべきです")
}
