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
)

// GenerativeModel は、このクライアントが提供する主要な生成操作を定義するインターフェース
type GenerativeModel interface {
	// GenerateContent は、プロンプトからテキストを生成します。
	GenerateContent(ctx context.Context, prompt string, modelName string) (*Response, error)
}

// Client は Gemini API との通信を管理する構造体です。GenerativeModel を実装しています。
type Client struct {
	client      *genai.Client
	temperature float32
	retryConfig retry.Config
}

// Config は Client を初期化するための設定項目です。
type Config struct {
	APIKey       string
	Temperature  *float32
	MaxRetries   uint64
	InitialDelay time.Duration // retry.Config.InitialInterval に対応
	MaxDelay     time.Duration // retry.Config.MaxInterval に対応
}

// ImageOptions は画像生成時の詳細なパラメータ（アスペクト比やシード値）を保持します。
type ImageOptions struct {
	AspectRatio string
	Seed        int32
}

// Response は Gemini API からのテキスト生成結果を保持する構造体です。
type Response struct {
	Text string
}

// NewClient は Config に基づいて Gemini クライアントを初期化します。
func NewClient(ctx context.Context, cfg Config) (*Client, error) {
	// 1. APIキーのバリデーション
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("APIキーは必須です。設定を確認してください")
	}

	// 2. 基底クライアントの作成
	clientConfig := &genai.ClientConfig{
		APIKey:  cfg.APIKey,
		Backend: genai.BackendGeminiAPI,
	}

	client, err := genai.NewClient(ctx, clientConfig)
	if err != nil {
		return nil, fmt.Errorf("Geminiクライアントの作成に失敗しました: %w", err)
	}

	// 3. 温度設定の初期化と検証
	temp := DefaultTemperature
	if cfg.Temperature != nil {
		if *cfg.Temperature < 0.0 || *cfg.Temperature > 1.0 {
			return nil, fmt.Errorf("温度設定は0.0から1.0の間である必要があります。入力値: %f", *cfg.Temperature)
		}
		temp = *cfg.Temperature
	}

	// 4. リトライ設定の構築
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

// NewClientFromEnv は環境変数からAPIキーを読み込んでクライアントを作成するヘルパー関数です。
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

// GenerateContent はリトライメカニズムを備えたテキスト生成リクエストを送信します。
func (c *Client) GenerateContent(ctx context.Context, finalPrompt string, modelName string) (*Response, error) {
	if finalPrompt == "" {
		return nil, errors.New("プロンプトが空です。入力を確認してください")
	}

	var responseText string
	contents := promptToContents(finalPrompt)
	config := &genai.GenerateContentConfig{
		Temperature: genai.Ptr(c.temperature),
	}

	// リトライ対象の操作を定義
	op := func() error {
		resp, err := c.client.Models.GenerateContent(ctx, modelName, contents, config)
		if err != nil {
			return err
		}

		extractedText, extractErr := extractTextFromResponse(resp)
		if extractErr != nil {
			return extractErr
		}

		responseText = extractedText
		return nil
	}

	// リトライ判定：API固有のエラー（ブロック等）はリトライしない
	shouldRetryFn := func(err error) bool {
		var apiErr *APIResponseError
		if errors.As(err, &apiErr) {
			return false
		}
		return shouldRetry(err)
	}

	err := retry.Do(ctx, c.retryConfig, fmt.Sprintf("Gemini API call to %s", modelName), op, shouldRetryFn)
	if err != nil {
		return nil, err
	}

	return &Response{Text: responseText}, nil
}

// GenerateWithParts はマルチモーダルなリクエスト（画像データ等を含む）をリトライ付きで送信します。
func (c *Client) GenerateWithParts(ctx context.Context, modelName string, parts []*genai.Part, opts ImageOptions) (*genai.GenerateContentResponse, error) {
	contents := []*genai.Content{{Role: "user", Parts: parts}}

	genConfig := &genai.GenerateContentConfig{
		Temperature:     genai.Ptr(c.temperature),
		TopP:            genai.Ptr(float32(0.95)),
		MaxOutputTokens: int32(8192),
		CandidateCount:  int32(1),
		Seed:            genai.Ptr(opts.Seed),
		ImageConfig: &genai.ImageConfig{
			AspectRatio: opts.AspectRatio,
		},
	}

	var finalResp *genai.GenerateContentResponse

	op := func() error {
		resp, err := c.client.Models.GenerateContent(ctx, modelName, contents, genConfig)
		if err != nil {
			return err
		}
		finalResp = resp
		return nil
	}

	// セーフティフィルタによるブロックなどはリトライ対象外
	shouldRetryWithFiltering := func(err error) bool {
		var apiErr *APIResponseError
		if errors.As(err, &apiErr) {
			return false
		}
		return shouldRetry(err)
	}

	err := retry.Do(ctx, c.retryConfig, fmt.Sprintf("Gemini Image API call to %s", modelName), op, shouldRetryWithFiltering)
	if err != nil {
		return nil, err
	}

	return finalResp, nil
}

// promptToContents は文字列プロンプトを SDK 用の Content 構造体に変換します。
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

// APIResponseError は生成成功後のフィルタリング等で発生したエラーを定義します。
type APIResponseError struct {
	msg string
}

func (e *APIResponseError) Error() string { return e.msg }

// shouldRetry は gRPC ステータスコードに基づき、エラーが一時的なものか判定します。
func shouldRetry(err error) bool {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	st, ok := status.FromError(err)
	if !ok {
		return false
	}

	switch st.Code() {
	case codes.DeadlineExceeded, codes.Unavailable, codes.ResourceExhausted, codes.Internal:
		return true // サーバー側の問題や一時的なリソース不足はリトライ対象
	default:
		return false
	}
}

// extractTextFromResponse はレスポンスからテキストを安全に抽出し、終了理由をチェックします。
func extractTextFromResponse(resp *genai.GenerateContentResponse) (string, error) {
	if resp == nil || len(resp.Candidates) == 0 {
		return "", &APIResponseError{msg: "Gemini APIから空のレスポンスが返されました"}
	}

	candidate := resp.Candidates[0]

	// ブロックされていないか確認
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
