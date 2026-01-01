package gemini

import (
	"context"
	"os"
	"strings"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// --- 初期化に関するテスト ---

func TestNewClient_InvalidAPIKey(t *testing.T) {
	ctx := context.Background()
	cfg := Config{APIKey: ""}

	t.Run("APIキーが空の場合にエラーを返すこと", func(t *testing.T) {
		_, err := NewClient(ctx, cfg)
		if err == nil {
			t.Error("FAIL: APIキーが空の場合、エラーが返されるべきです")
		}

		expectedError := "APIキーは必須です"
		if err != nil && !strings.Contains(err.Error(), expectedError) {
			t.Errorf("FAIL: 予期しないエラーメッセージ\n  got: %q\n  want (contains): %q", err.Error(), expectedError)
		}
	})
}

func TestNewClientFromEnv_MissingKey(t *testing.T) {
	ctx := context.Background()

	originalKey := os.Getenv("GEMINI_API_KEY")
	originalGoogleKey := os.Getenv("GOOGLE_API_KEY")
	os.Unsetenv("GEMINI_API_KEY")
	os.Unsetenv("GOOGLE_API_KEY")

	defer func() {
		os.Setenv("GEMINI_API_KEY", originalKey)
		os.Setenv("GOOGLE_API_KEY", originalGoogleKey)
	}()

	t.Run("環境変数がない場合にエラーを返すこと", func(t *testing.T) {
		_, err := NewClientFromEnv(ctx)
		if err == nil {
			t.Error("FAIL: 環境変数がない場合、エラーが返されるべきです")
		}

		// 指摘に基づき、strings.Contains を使用して検証方法を統一
		expectedError := "環境変数 GEMINI_API_KEY または GOOGLE_API_KEY が設定されていません"
		if err != nil && !strings.Contains(err.Error(), expectedError) {
			t.Errorf("FAIL: 予期しないエラーメッセージ\n  got: %q\n  want (contains): %q", err.Error(), expectedError)
		}
	})
}

// --- GenerateWithParts に関するテスト方針 ---

/*
  注意: GenerateWithParts の網羅的テストには、genai.Client が依存する
  ModelsService のインターフェースをモック化する必要があります。
  以下に、実装すべきテストケースの構造を定義します。
*/

// TODO: 本番環境でのモックライブラリ導入後、以下のテストを実装します。

func TestClient_GenerateWithParts_Mock(t *testing.T) {
	// 1. 正常系: 有効な []*genai.Part を渡し、生成結果が返却されること
	// 2. 異常系 (一時的エラー): codes.Unavailable を返し、リトライが走ることを検証
	// 3. 異常系 (永続的エラー): codes.InvalidArgument を返し、即座に終了することを検証
	// 4. 異常系 (ブロック): FinishReasonSafety によりブロックされ、APIResponseError が返ることを検証
}

// shouldRetry の単体テストを追加し、リトライロジックの妥当性を検証します
func TestShouldRetry(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"一時的エラー (Unavailable)", status.Error(codes.Unavailable, "service unavailable"), true},
		{"リソース不足 (ResourceExhausted)", status.Error(codes.ResourceExhausted, "quota exceeded"), true},
		{"内部エラー (Internal)", status.Error(codes.Internal, "internal server error"), true},
		{"永続的エラー (InvalidArgument)", status.Error(codes.InvalidArgument, "invalid prompt"), false},
		{"認証エラー (Unauthenticated)", status.Error(codes.Unauthenticated, "invalid key"), false},
		{"コンテキストキャンセル", context.Canceled, false},
		{"タイムアウト", context.DeadlineExceeded, false},
		// ------------------------------------------------------
		{"APIResponseError (ブロック)", &APIResponseError{msg: "blocked"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldRetry(tt.err); got != tt.want {
				t.Errorf("shouldRetry(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}
