package prompt

import (
	"os"
	"strings"
	"testing"
)

// --- テスト設定: アプリケーションの init() をエミュレート ---

// テスト用のダミーテンプレート定義
const testSoloTemplate = `
一人の話者によるモノローグ形式のスクリプトに変換
[入力テキスト]
{{.InputText}}
`

const testDialogueTemplate = `
二人の話者（ずんだもん、めたん）による対話スクリプトに変換
[入力テキスト]
{{.InputText}}
`

// TestMain 関数: 全テスト実行前に一度だけ実行され、テンプレートを登録する
func TestMain(m *testing.M) {
	// テンプレートをテスト開始前に登録する
	// RegisterTemplate はテンプレートの解析とキャッシュも行うようになりました。
	if err := RegisterTemplate("solo", testSoloTemplate); err != nil {
		// 登録失敗は致命的エラー
		panic("Solo テンプレートのテスト登録に失敗: " + err.Error())
	}
	if err := RegisterTemplate("dialogue", testDialogueTemplate); err != nil {
		// 登録失敗は致命的エラー
		panic("Dialogue テンプレートのテスト登録に失敗: " + err.Error())
	}

	// テストを実行
	code := m.Run()

	// 終了コードでプロセスを終了
	os.Exit(code)
}

// --- TestGetPromptByMode は非公開関数になったため削除（または、非公開関数をテストしたい場合は getParsedPromptByMode のテストに置き換え） ---

// TestBuildFullPrompt_Success は、テンプレートへの入力埋め込みとモードの選択が成功するかテストします。
func TestBuildFullPrompt_Success(t *testing.T) {
	rawInput := "AIは人間の生活を豊かにします。"
	// ★ 修正: テスト入力の型を string に変更
	testInput := rawInput

	// builder.goのテンプレート内のタグに合わせて期待値を設定
	expectedInputLine := "[入力テキスト]\n" + rawInput

	tests := []struct {
		name string
		mode string
		// テンプレートがモードに応じて切り替わっているかを確認する文字列
		modeSpecificCheck string
	}{
		{
			name:              "Soloモード: テンプレートの切り替え検証",
			mode:              "solo",
			modeSpecificCheck: "一人の話者によるモノローグ形式のスクリプトに変換",
		},
		{
			name:              "Dialogueモード: テンプレートの切り替え検証",
			mode:              "dialogue",
			modeSpecificCheck: "二人の話者（ずんだもん、めたん）による対話スクリプトに変換",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// ★ 修正: BuildFullPrompt に string を渡す
			finalPrompt, err := BuildFullPrompt(testInput, tt.mode)

			if err != nil {
				t.Fatalf("FAIL: BuildFullPrompt(%s)に失敗しました: %v", tt.mode, err)
			}

			// 1. 入力テキストが正しく埋め込まれているか検証 (共通)
			if !strings.Contains(finalPrompt, expectedInputLine) {
				t.Errorf("FAIL: 入力テキストの埋め込みエラー。\n  Got: %q\n  Expected Substring: %q", finalPrompt, expectedInputLine)
			}

			// 2. テンプレートがモードに応じて切り替わっているか検証 (モード固有)
			if !strings.Contains(finalPrompt, tt.modeSpecificCheck) {
				t.Errorf("FAIL: モード固有のテンプレートが使われていません。\n  Mode: %s\n  Expected Substring: %q", tt.mode, tt.modeSpecificCheck)
			}
		})
	}
}

// TestBuildFullPrompt_ErrorPropagation は、無効なモードでエラーが返されるかテストします。
func TestBuildFullPrompt_ErrorPropagation(t *testing.T) {
	// ★ 修正: テスト入力の型を string に変更
	testInput := "dummy input"
	invalidMode := "unsupported_mode"

	t.Run("無効なモードの場合にエラーを返すこと", func(t *testing.T) {
		// ★ 修正: BuildFullPrompt に string を渡す
		finalPrompt, err := BuildFullPrompt(testInput, invalidMode)

		if err == nil {
			t.Fatal("FAIL: 無効なモードの場合、エラーが返されるべきです")
		}

		// エラーが GetParsedPromptByMode から正しく伝播しているか検証
		expectedErrorSubstring := "未対応のモードです"
		if !strings.Contains(err.Error(), expectedErrorSubstring) {
			t.Errorf("FAIL: 予期しないエラーメッセージ\n  got: %q\n  want substring: %q", err.Error(), expectedErrorSubstring)
		}

		// エラー発生時、プロンプトは空であることを確認
		if finalPrompt != "" {
			t.Errorf("FAIL: エラー発生時、プロンプトは空であるべきです。\n  got: %q", finalPrompt)
		}
	})
}
