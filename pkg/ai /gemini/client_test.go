package gemini

import (
	"context"
	"fmt"
	"os"
	"testing"
)

// NOTE: 実際のAPI通信を行う GenerateScript のテストは、
// ネットワークアクセスが必要なため「統合テスト」として扱い、
// ユニットテストでは主に NewClient/NewClientFromEnv の初期化ロジックをテストします。

// TestNewClient_InvalidAPIKey は、APIキーが空の場合に NewClient がエラーを返すかテストします。
func TestNewClient_InvalidAPIKey(t *testing.T) {
	ctx := context.Background()

	// 1. 設定構造体で APIKey を空にする
	cfg := Config{
		APIKey:    "",
		ModelName: "gemini-2.5-flash",
	}

	t.Run("APIキーが空の場合にエラーを返すこと", func(t *testing.T) {
		_, err := NewClient(ctx, cfg)

		if err == nil {
			t.Error("FAIL: APIキーが空の場合、エラーが返されるべきです")
		}

		// 期待されるエラーメッセージの検証
		expectedError := "APIKey is required for Gemini client initialization"
		if err.Error() != expectedError {
			t.Errorf("FAIL: 予期しないエラーメッセージ\n  got: %q\n  want: %q", err.Error(), expectedError)
		}
	})
}

// TestNewClientFromEnv_MissingKey は、環境変数がない場合に NewClientFromEnv がエラーを返すかテストします。
func TestNewClientFromEnv_MissingKey(t *testing.T) {
	ctx := context.Background()

	// テスト対象の環境変数名
	envVarName := "GEMINI_API_KEY"

	// -------------------------------------------------------------------------
	// 重要な処理: 環境変数テストのためのセットアップとクリーンアップ
	// テスト前に GEMINI_API_KEY が設定されていた場合、値を保存し、テスト後に元に戻す
	originalKey := os.Getenv(envVarName)
	os.Unsetenv(envVarName)
	defer func() {
		if originalKey != "" {
			os.Setenv(envVarName, originalKey)
		}
	}()
	// -------------------------------------------------------------------------

	t.Run("環境変数がない場合にエラーを返すこと", func(t *testing.T) {
		_, err := NewClientFromEnv(ctx, "gemini-2.5-flash")

		if err == nil {
			t.Error("FAIL: 環境変数がない場合、エラーが返されるべきです")
		}

		// 期待されるエラーメッセージの検証
		expectedError := fmt.Sprintf("%s environment variable is not set", envVarName)
		if err.Error() != expectedError {
			t.Errorf("FAIL: 予期しないエラーメッセージ\n  got: %q\n  want: %q", err.Error(), expectedError)
		}
	})
}

// TestNewClientFromEnv_Success は、環境変数がある場合にクライアントが成功裏に初期化されるかテストします。
// NOTE: 実際のAPI接続テストはコストがかかるため、ここではクライアントオブジェクトがnilでないことのみを確認します。
func TestNewClientFromEnv_Success(t *testing.T) {
	ctx := context.Background()

	// -------------------------------------------------------------------------
	// 重要な処理: 環境変数テストのためのセットアップとクリーンアップ
	// ダミーキーを設定
	originalKey := os.Getenv("GEMINI_API_KEY")
	os.Setenv("GEMINI_API_KEY", "DUMMY_API_KEY_FOR_TEST") // ダミーキーを設定
	defer func() {
		// テスト後に元の環境変数に戻す
		if originalKey != "" {
			os.Setenv("GEMINI_API_KEY", originalKey)
		} else {
			os.Unsetenv("GEMINI_API_KEY")
		}
	}()
	// -------------------------------------------------------------------------

	t.Run("環境変数がある場合に成功すること", func(t *testing.T) {
		client, err := NewClientFromEnv(ctx, "gemini-2.5-flash")

		if err != nil {
			t.Fatalf("FAIL: 環境変数が設定されているにも関わらず初期化に失敗しました: %v", err)
		}

		if client == nil {
			t.Fatal("FAIL: クライアントオブジェクトが nil です")
		}
	})
}
