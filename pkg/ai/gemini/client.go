package gemini

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/shouni/go-utils/retry"
	"google.golang.org/genai"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	// DefaultTemperature は、モデルの応答温度のデフォルト値（0.0〜1.0）
	DefaultTemperature float32 = 0.7
	// DefaultMaxRetries は、API呼び出し失敗時の最大リトライ回数
	DefaultMaxRetries = 3
	// DefaultInitialDelay は、指数バックオフの最初の待機時間
	DefaultInitialDelay = 30 * time.Second
	// DefaultMaxDelay は、指数バックオフの最大待機時間
	DefaultMaxDelay = 120 * time.Second

	// デフォルトの生成パラメータ（マジックナンバーの排除）
	DefaultTopP            float32 = 0.95
	DefaultMaxOutputTokens int32   = 4096 // 安全性とコストを考慮した適正値
	DefaultCandidateCount  int32   = 1
)

// Response は Gemini API からの生成結果を抽象化した構造体です。
type Response struct {
	Text        string
	RawResponse *genai.GenerateContentResponse // SDK固有のデータが必要な場合に備え保持
}

// GenerativeModel は、このパッケージが提供する生成操作を定義するインターフェースです。
type GenerativeModel interface {
	// GenerateContent は、テキストプロンプトからコンテンツを生成します。
	GenerateContent(ctx context.Context, prompt string, modelName string) (*Response, error)
	// GenerateWithParts は、マルチモーダル入力からコンテンツを生成します。
	GenerateWithParts(ctx context.Context, modelName string, parts []*genai.Part, opts ImageOptions) (*Response, error)
}

// Client は Gemini API との通信を管理し、リトライ機能を備えています。
type Client struct {
	client      *genai.Client
	temperature float32
	retryConfig retry.Config
}

// Config は Client の初期化設定です。
type Config struct {
	APIKey       string
	Temperature  *float32
	MaxRetries   uint64
	InitialDelay time.Duration
	MaxDelay     time.Duration
}

// ImageOptions は画像生成を含むリクエストの詳細パラメータを保持します。
type ImageOptions struct {
	AspectRatio string
	Seed        *int32 // nil（未指定）と 0（固定シード）を区別するためポインタ型
}

// NewClient は Config に基づいて Client を初期化します。
func NewClient(ctx context.Context, cfg Config) (*Client, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("APIキーは必須です。設定を確認してください")
	}

	clientConfig := &genai.ClientConfig{
		APIKey:  cfg.APIKey,
		Backend: genai.BackendGeminiAPI,
	}

	client, err := genai.NewClient(ctx, clientConfig)
	if err != nil {
		return nil, fmt.Errorf("Geminiクライアントの作成に失敗しました: %w", err)
	}

	temp := DefaultTemperature
	if cfg.Temperature != nil {
		if *cfg.Temperature < 0.0 || *cfg.Temperature > 1.0 {
			return nil, fmt.Errorf("温度設定は0.0から1.0の間である必要があります。入力値: %f", *cfg.Temperature)
		}
		temp = *cfg.Temperature
	}

	// リトライ設定の構築
	retryCfg := retry.DefaultConfig()
	if cfg.MaxRetries > 0 {
		retryCfg.MaxRetries = cfg.MaxRetries
	} else {
		retryCfg.MaxRetries = DefaultMaxRetries
	}

	if cfg.InitialDelay > 0 {
		retryCfg.InitialInterval = cfg.InitialDelay
	} else {
		retryCfg.InitialInterval = DefaultInitialDelay
	}

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

// NewClientFromEnv は環境変数から API キーを取得して Client を作成します。
func NewClientFromEnv(ctx context.Context) (*Client, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("GOOGLE_API_KEY")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("環境変数 GEMINI_API_KEY または GOOGLE_API_KEY が設定されていません")
	}

	return NewClient(ctx, Config{APIKey: apiKey})
}

// executeWithRetry は、共通のリトライロジックを実行するヘルパーです（DRY原則の適用）。
func (c *Client) executeWithRetry(ctx context.Context, operationName string, op func() error, shouldRetryFn func(error) bool) error {
	return retry.Do(
		ctx,
		c.retryConfig,
		operationName,
		op,
		shouldRetryFn,
	)
}

// GenerateContent はテキスト生成を実行し、抽象化された Response を返します。
func (c *Client) GenerateContent(ctx context.Context, finalPrompt string, modelName string) (*Response, error) {
	if finalPrompt == "" {
		return nil, errors.New("プロンプトが空です。入力を確認してください")
	}

	var finalResp *Response
	contents := promptToContents(finalPrompt)
	config := &genai.GenerateContentConfig{
		Temperature: genai.Ptr(c.temperature),
	}

	op := func() error {
		resp, err := c.client.Models.GenerateContent(ctx, modelName, contents, config)
		if err != nil {
			return err
		}
		text, extractErr := extractTextFromResponse(resp)
		if extractErr != nil {
			return extractErr
		}
		finalResp = &Response{Text: text, RawResponse: resp}
		return nil
	}

	shouldRetryFn := func(err error) bool {
		var apiErr *APIResponseError
		return !errors.As(err, &apiErr) && shouldRetry(err)
	}

	err := c.executeWithRetry(ctx, fmt.Sprintf("Gemini API call to %s", modelName), op, shouldRetryFn)
	if err != nil {
		return nil, err
	}

	return finalResp, nil
}

// GenerateWithParts は画像データ等を含むリクエストを送信し、抽象化された Response を返します。
func (c *Client) GenerateWithParts(ctx context.Context, modelName string, parts []*genai.Part, opts ImageOptions) (*Response, error) {
	contents := []*genai.Content{{Role: "user", Parts: parts}}

	genConfig := &genai.GenerateContentConfig{
		Temperature:     genai.Ptr(c.temperature),
		TopP:            genai.Ptr(DefaultTopP),
		MaxOutputTokens: DefaultMaxOutputTokens,
		CandidateCount:  DefaultCandidateCount,
		Seed:            opts.Seed, // ポインタ型により nil ならSDK側でデフォルト扱い
		ImageConfig: &genai.ImageConfig{
			AspectRatio: opts.AspectRatio,
		},
	}

	var finalResp *Response
	op := func() error {
		resp, err := c.client.Models.GenerateContent(ctx, modelName, contents, genConfig)
		if err != nil {
			return err
		}
		// テキストが含まれる場合は抽出し、含まれない場合は RawResponse のみを保持
		text, _ := extractTextFromResponse(resp)
		finalResp = &Response{Text: text, RawResponse: resp}
		return nil
	}

	shouldRetryFn := func(err error) bool {
		var apiErr *APIResponseError
		return !errors.As(err, &apiErr) && shouldRetry(err)
	}

	err := c.executeWithRetry(ctx, fmt.Sprintf("Gemini Image API call to %s", modelName), op, shouldRetryFn)
	if err != nil {
		return nil, err
	}

	return finalResp, nil
}

// promptToContents は文字列プロンプトを SDK 形式の Content に変換します。
func promptToContents(text string) []*genai.Content {
	return []*genai.Content{
		{
			Role: "user",
			Parts: []*genai.Part{
				{Text: text},
			},
		},
	}
}

// APIResponseError は API 応答成功後のビジネスロジックエラー（ブロック等）を表します。
type APIResponseError struct {
	msg string
}

func (e *APIResponseError) Error() string { return e.msg }

// shouldRetry は gRPC エラーコードからリトライの可否を判定します。
func shouldRetry(err error) bool {
	// コンテキスト関連のエラーは上位で制御されるべき、またはリトライしても成功しないため除外
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	st, ok := status.FromError(err)
	if !ok {
		return false
	}

	switch st.Code() {
	case codes.DeadlineExceeded, codes.Unavailable, codes.ResourceExhausted, codes.Internal:
		return true
	default:
		return false
	}
}

// extractTextFromResponse はレスポンスからテキストを安全に抽出し、安全フィルタ等を検証します。
func extractTextFromResponse(resp *genai.GenerateContentResponse) (string, error) {
	if resp == nil || len(resp.Candidates) == 0 {
		return "", &APIResponseError{msg: "Gemini APIから空のレスポンスが返されました"}
	}

	candidate := resp.Candidates[0]

	// FinishReason の確認（ブロックされていないか）
	if candidate.FinishReason != genai.FinishReasonUnspecified && candidate.FinishReason != genai.FinishReasonStop {
		return "", &APIResponseError{msg: fmt.Sprintf("生成がブロックされました。理由: %v", candidate.FinishReason)}
	}

	if candidate.Content == nil || len(candidate.Content.Parts) == 0 {
		return "", &APIResponseError{msg: "レスポンスのコンテンツが空です"}
	}

	firstPart := candidate.Content.Parts[0]
	if firstPart.Text == "" {
		return "", &APIResponseError{msg: "テキストデータが含まれていません"}
	}

	return firstPart.Text, nil
}
