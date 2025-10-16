package gemini

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/shouni/go-web-exact/pkg/retry"
	"go-ai-client/pkg/prompt"

	"google.golang.org/genai"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	// クライアントのデフォルトタイムアウト（リトライロジックとは別個に、単一リクエストの最大時間を設定）
	DefaultRequestTimeout = 30 * time.Second
)

// GenerativeModel は、このクライアントが提供する主要な操作を定義するインターフェースです。
type GenerativeModel interface {
	GenerateScript(ctx context.Context, inputContent []byte, mode string) (string, error)
}

// Client はGemini APIとの通信を管理します。GenerativeModel インターフェースを満たします。
type Client struct {
	client    *genai.Client
	modelName string
	// ★ 変更点2: 汎用リトライ設定を保持
	retryConfig retry.Config
}

// Config は Client を初期化するための設定を定義します。
type Config struct {
	APIKey    string
	ModelName string
	// ★ 変更点3: 最大リトライ回数を設定項目に追加
	MaxRetries uint64
}

// NewClient はConfig構造体とcontextを受け取り、Clientを初期化します。
func NewClient(ctx context.Context, cfg Config) (*Client, error) {

	// 1. APIキーのバリデーション
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("APIKey is required for Gemini client initialization")
	}

	// 2. クライアントの作成
	clientConfig := &genai.ClientConfig{
		APIKey: cfg.APIKey,
		// 単一リクエストのタイムアウトを設定
		Timeout: DefaultRequestTimeout,
	}

	client, err := genai.NewClient(ctx, clientConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}

	// 3. リトライ設定の初期化
	retryCfg := retry.DefaultConfig()
	if cfg.MaxRetries > 0 {
		retryCfg.MaxRetries = cfg.MaxRetries
	}

	return &Client{
		client:      client,
		modelName:   cfg.ModelName,
		retryConfig: retryCfg,
	}, nil
}

// NewClientFromEnv は環境変数からAPIキーを取得し、NewClientを呼び出すヘルパー関数です。
func NewClientFromEnv(ctx context.Context, modelName string) (GenerativeModel, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY environment variable is not set")
	}

	cfg := Config{
		APIKey:    apiKey,
		ModelName: modelName,
		// 環境変数からの作成時にデフォルトリトライ回数を設定
		MaxRetries: retry.DefaultMaxRetries,
	}

	return NewClient(ctx, cfg)
}

// GenerateScript はナレーションスクリプトを生成します。
func (c *Client) GenerateScript(ctx context.Context, inputContent []byte, mode string) (string, error) {

	finalPrompt, err := prompt.BuildFullPrompt(inputContent, mode)
	if err != nil {
		return "", fmt.Errorf("failed to build prompt: %w", err)
	}

	return c.callGenerateContent(ctx, finalPrompt)
}

// callGenerateContent はリトライロジックを適用してAPIを呼び出します。
func (c *Client) callGenerateContent(ctx context.Context, finalPrompt string) (string, error) {
	var text string

	// 1. API呼び出しとレスポンス処理を行う操作関数 (retry.Operation型に準拠)
	op := func() error {
		contents := []*genai.Content{
			{Role: "user", Parts: []*genai.Part{{Text: finalPrompt}}},
		}

		resp, err := c.client.Models.GenerateContent(ctx, c.modelName, contents, nil)

		if err != nil {
			return err // API呼び出し自体のエラー（ネットワークやgRPCエラー）
		}

		// レスポンスからテキストを抽出（ブロックエラーもここで処理）
		extractedText, extractErr := extractTextFromResponse(resp)
		if extractErr != nil {
			return extractErr
		}

		text = extractedText
		return nil
	}

	// 2. shouldRetryFn: API固有の一時的エラー判定ロジック (retry.ShouldRetryFunc型に準拠)
	shouldRetryFn := func(err error) bool {
		// レスポンス処理エラー（コンテンツブロックなど）はリトライすべきでない
		if errors.As(err, &APIResponseError{}) {
			return false
		}
		// API呼び出しエラーの場合のみ、Gemini固有の判定ロジックを適用
		return shouldRetry(err)
	}

	// 3. 汎用リトライサービスを利用して操作を実行
	err := retry.Do(
		ctx,
		c.retryConfig,
		fmt.Sprintf("Gemini API call to %s", c.modelName),
		op,
		shouldRetryFn,
	)

	if err != nil {
		return "", err
	}

	return text, nil
}

// APIResponseError は、API呼び出しは成功したがレスポンス処理で問題が発生したエラーです。
type APIResponseError struct {
	msg string
}

func (e *APIResponseError) Error() string { return e.msg }

// shouldRetry はエラーがリトライ可能かどうかを判定します。（gRPCエラーコードに基づく）
func shouldRetry(err error) bool {
	// コンテキストエラー（ユーザーによるキャンセルやタイムアウト）はリトライ対象
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	st, ok := status.FromError(err)
	if !ok {
		return false // gRPCステータスではない場合はリトライしない
	}

	// リトライすべきgRPCステータスコード
	switch st.Code() {
	case codes.DeadlineExceeded, codes.Unavailable, codes.ResourceExhausted, codes.Internal:
		// サーバー側の問題や一時的なリソース不足
		return true
	case codes.Unauthenticated, codes.InvalidArgument, codes.NotFound, codes.PermissionDenied:
		// 認証失敗、不正な引数など、リトライしても解決しない永続的なエラー
		return false
	default:
		return false
	}
}

// extractTextFromResponse は、成功したAPIレスポンスからテキストを安全に抽出します。
func extractTextFromResponse(resp *genai.GenerateContentResponse) (string, error) {
	if resp == nil || len(resp.Candidates) == 0 {
		return "", &APIResponseError{msg: "Gemini APIから空または無効なレスポンスが返されました"}
	}

	candidate := resp.Candidates[0]

	// 安全性チェック: レスポンスがブロックされていないか確認
	if candidate.FinishReason != genai.FinishReasonUnspecified && candidate.FinishReason != genai.FinishReasonStop {
		return "", &APIResponseError{msg: fmt.Sprintf("APIレスポンスがブロックされたか、途中で終了しました。理由: %v", candidate.FinishReason)}
	}

	// コンテンツの有無をチェック
	if candidate.Content == nil || len(candidate.Content.Parts) == 0 {
		return "", &APIResponseError{msg: "Gemini レスポンスのコンテンツが空です"}
	}

	firstPart := candidate.Content.Parts[0]

	// Textフィールドの値をチェック
	if firstPart.Text == "" {
		return "", &APIResponseError{msg: "APIは非テキスト形式の応答を返したか、テキストフィールドが空です"}
	}

	return firstPart.Text, nil
}
