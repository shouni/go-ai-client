package gemini

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"google.golang.org/genai"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/shouni/go-web-exact/pkg/retry"
)

const (
	// DefaultTemperature デフォルトの温度 (0.0 から 1.0 の範囲で、通常 0.0 が決定論的、1.0 が創造的)
	DefaultTemperature float32 = 0.7
	// DefaultMaxRetries デフォルトのリトライ回数
	DefaultMaxRetries = 3
	// DefaultInitialDelay デフォルトの指数バックオフの初期間隔
	DefaultInitialDelay = 60 * time.Second
	// DefaultMaxDelay デフォルトの指数バックオフの最大間隔
	DefaultMaxDelay = 300 * time.Second
)

// GenerativeModel is the interface that defines the core operations this client provides.
type GenerativeModel interface {
	GenerateContent(ctx context.Context, prompt string, modelName string) (*Response, error)
}

// Client manages communication with the Gemini API. It implements the GenerativeModel interface.
type Client struct {
	client      *genai.Client
	temperature float32
	retryConfig retry.Config
}

// Config defines the configuration for initializing the Client.
type Config struct {
	APIKey       string
	Temperature  *float32
	MaxRetries   uint64
	InitialDelay time.Duration // retry.Config.InitialInterval に対応
	MaxDelay     time.Duration // retry.Config.MaxInterval に対応
}

// Response holds the Gemini API result.
type Response struct {
	Text string
}

// NewClient initializes a Client struct.
func NewClient(ctx context.Context, cfg Config) (*Client, error) {

	// 1. APIキーのバリデーション
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("APIKey is required for Gemini client initialization")
	}

	// 2. 基底クライアントの作成
	clientConfig := &genai.ClientConfig{
		APIKey: cfg.APIKey,
	}

	client, err := genai.NewClient(ctx, clientConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}

	// 3. 温度設定の初期化と検証
	temp := DefaultTemperature
	if cfg.Temperature != nil {
		// float32として検証
		if *cfg.Temperature < 0.0 || *cfg.Temperature > 1.0 {
			return nil, fmt.Errorf("temperature must be between 0.0 and 1.0, got %f", *cfg.Temperature)
		}
		temp = *cfg.Temperature
	}

	// 4. リトライ設定の初期化と反映
	retryCfg := retry.DefaultConfig()

	// MaxRetries の反映
	if cfg.MaxRetries > 0 {
		retryCfg.MaxRetries = cfg.MaxRetries
	} else {
		retryCfg.MaxRetries = DefaultMaxRetries
	}

	// InitialDelay (InitialInterval) の反映
	if cfg.InitialDelay > 0 {
		retryCfg.InitialInterval = cfg.InitialDelay
	} else {
		retryCfg.InitialInterval = DefaultInitialDelay
	}

	// MaxDelay (MaxInterval) の反映
	if cfg.MaxDelay > 0 {
		retryCfg.MaxInterval = cfg.MaxDelay
	} else {
		retryCfg.MaxInterval = DefaultMaxDelay
	}

	return &Client{
		client:      client,
		temperature: temp,
		retryConfig: retryCfg,
	}, nil
}

// NewClientFromEnv is a helper function that creates a client using the API key from the environment variable.
func NewClientFromEnv(ctx context.Context) (*Client, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("GOOGLE_API_KEY") // GOOGLE_API_KEY もサポート
	}
	if apiKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY or GOOGLE_API_KEY environment variable is not set")
	}

	cfg := Config{
		APIKey: apiKey,
	}

	return NewClient(ctx, cfg)
}

// GenerateContent sends a prompt to the Gemini model with a retry mechanism.
func (c *Client) GenerateContent(ctx context.Context, finalPrompt string, modelName string) (*Response, error) {

	if finalPrompt == "" {
		return nil, errors.New("prompt content cannot be empty")
	}

	var responseText string
	// 文字列から Content 構造体を構築
	contents := promptToContents(finalPrompt)

	// Temperatureには*float32のポインタが必要なため、Clientのfloat32値をポインタに変換
	tempPtr := &c.temperature

	// API呼び出しパラメータの構築: genai.GenerateContentConfigを使用
	config := &genai.GenerateContentConfig{
		Temperature: tempPtr, // *float32型を渡す
	}

	// 1. API呼び出しとレスポンス処理を行う操作関数
	op := func() error {
		// GenerateContentに設定（config）を渡す
		resp, err := c.client.Models.GenerateContent(ctx, modelName, contents, config)

		if err != nil {
			return err // API呼び出し自体のエラー
		}

		// レスポンスからテキストを抽出（ブロックエラーもここで処理）
		extractedText, extractErr := extractTextFromResponse(resp)
		if extractErr != nil {
			return extractErr // APIResponseError を返す
		}

		responseText = extractedText
		return nil
	}

	// 2. shouldRetryFn: API固有の一時的エラー判定ロジック
	shouldRetryFn := func(err error) bool {
		var apiErr *APIResponseError
		if errors.As(err, &apiErr) {
			return false // APIResponseError (ブロックなど) は永続エラー
		}
		// API呼び出しエラーの場合のみ、Gemini固有の判定ロジックを適用
		return shouldRetry(err)
	}

	// 3. 汎用リトライサービスを利用して操作を実行
	err := retry.Do(
		ctx,
		c.retryConfig,
		fmt.Sprintf("Gemini API call to %s", modelName),
		op,
		shouldRetryFn,
	)

	if err != nil {
		return nil, err
	}

	return &Response{Text: responseText}, nil
}

// promptToContents converts a simple string prompt to the genai.Content required by the SDK.
func promptToContents(text string) []*genai.Content {
	// 単一のユーザーメッセージとしてコンテンツをラップ
	return []*genai.Content{
		{
			Role: "user",
			Parts: []*genai.Part{
				{Text: text},
			},
		},
	}
}

// APIResponseError is an error that occurred after a successful API call but during response processing (e.g., content blocking).
type APIResponseError struct {
	msg string
}

func (e *APIResponseError) Error() string { return e.msg }

// shouldRetry determines if an error is transient and should be retried (based on gRPC error codes).
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

// extractTextFromResponse safely extracts text from a successful API response.
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
