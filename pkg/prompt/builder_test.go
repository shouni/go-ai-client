package prompt

import (
	"strings"
	"testing"
)

// NOTE: このテストは、builder.go内の GetPromptByMode がテンプレート文字列をハードコードしていることを前提としています。

// TestGetPromptByMode は、モードに応じたテンプレートが正しく取得されるか、および未対応モードでエラーが返されるかを検証します。
func TestGetPromptByMode(t *testing.T) {
	tests := []struct {
		name              string
		mode              string
		wantErr           bool
		expectedSubstring string // テンプレートが正しく切り替わっていることを証明する固有の文字列
	}{
		{
			name:              "Soloモードの取得成功",
			mode:              "solo",
			wantErr:           false,
			expectedSubstring: "一人の話者によるモノローグ形式のスクリプトに変換",
		},
		{
			name:              "Dialogueモードの取得成功",
			mode:              "dialogue",
			wantErr:           false,
			expectedSubstring: "二人の話者（ずんだもん、めたん）による対話スクリプト",
		},
		{
			name:    "未対応モードでエラー",
			mode:    "unsupported_mode",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := GetPromptByMode(tt.mode)

			if (err != nil) != tt.wantErr {
				t.Fatalf("FAIL: GetPromptByMode(%s) エラー状態が期待値と不一致\n  got error: %v, want error: %v", tt.mode, err, tt.wantErr)
			}

			// エラーがない場合のみ、内容を検証
			if !tt.wantErr && !strings.Contains(result, tt.expectedSubstring) {
				t.Errorf("FAIL: プロンプト内容にモード固有の文字列が含まれていません。\n  Got: %q\n  Want Substring: %q", result, tt.expectedSubstring)
			}
		})
	}
}

// TestBuildFullPrompt_Success は、テンプレートへの入力埋め込みとモードの選択が成功するかテストします。
func TestBuildFullPrompt_Success(t *testing.T) {
	rawInput := "AIは人間の生活を豊かにします。"
	testInput := []byte(rawInput)
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

// TestBuildFullPrompt_ErrorPropagation は、GetPromptByModeで発生したエラーが伝播するかテストします。
func TestBuildFullPrompt_ErrorPropagation(t *testing.T) {
	testInput := []byte("dummy")
	invalidMode := "unsupported_mode"

	t.Run("無効なモードの場合にエラーを返すこと", func(t *testing.T) {
		finalPrompt, err := BuildFullPrompt(testInput, invalidMode)

		if err == nil {
			t.Fatal("FAIL: 無効なモードの場合、エラーが返されるべきです")
		}

		// エラーが GetPromptByMode から正しく伝播しているか検証
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
