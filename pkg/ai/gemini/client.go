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
	DefaultTemperature  float32 = 0.7
	DefaultMaxRetries           = 3
	DefaultInitialDelay         = 30 * time.Second
	DefaultMaxDelay             = 120 * time.Second

	DefaultTopP            float32 = 0.95
	DefaultMaxOutputTokens int32   = 4096
	DefaultCandidateCount  int32   = 1
)

type Response struct {
	Text        string
	RawResponse *genai.GenerateContentResponse
}

type GenerativeModel interface {
	GenerateContent(ctx context.Context, prompt string, modelName string) (*Response, error)
	GenerateWithParts(ctx context.Context, modelName string, parts []*genai.Part, opts ImageOptions) (*Response, error)
}

type Client struct {
	client      *genai.Client
	temperature float32
	retryConfig retry.Config
}

type Config struct {
	APIKey       string
	Temperature  *float32
	MaxRetries   uint64
	InitialDelay time.Duration
	MaxDelay     time.Duration
}

type ImageOptions struct {
	AspectRatio string
	Seed        *int32
}

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

func (c *Client) executeWithRetry(ctx context.Context, operationName string, op func() error, shouldRetryFn func(error) bool) error {
	return retry.Do(ctx, c.retryConfig, operationName, op, shouldRetryFn)
}

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

	// shouldRetry 内で APIResponseError を判定するようにしたため、シンプルに
	err := c.executeWithRetry(ctx, fmt.Sprintf("Gemini API call to %s", modelName), op, shouldRetry)
	if err != nil {
		return nil, err
	}

	return finalResp, nil
}

func (c *Client) GenerateWithParts(ctx context.Context, modelName string, parts []*genai.Part, opts ImageOptions) (*Response, error) {
	contents := []*genai.Content{{Role: "user", Parts: parts}}

	genConfig := &genai.GenerateContentConfig{
		Temperature:     genai.Ptr(c.temperature),
		TopP:            genai.Ptr(DefaultTopP),
		MaxOutputTokens: DefaultMaxOutputTokens,
		CandidateCount:  DefaultCandidateCount,
		Seed:            opts.Seed,
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

		// 指摘に基づき、エラーハンドリングを厳密化
		text, extractErr := extractTextFromResponse(resp)
		if extractErr != nil {
			var apiErr *APIResponseError
			// ブロックなどの致命的なエラー（APIResponseError）は即時返却
			if errors.As(extractErr, &apiErr) {
				return extractErr
			}
			// その他の非致命的な抽出エラーは無視して空文字を許容
		}

		finalResp = &Response{Text: text, RawResponse: resp}
		return nil
	}

	// shouldRetryFn をシンプルに
	err := c.executeWithRetry(ctx, fmt.Sprintf("Gemini Image API call to %s", modelName), op, shouldRetry)
	if err != nil {
		return nil, err
	}

	return finalResp, nil
}

func promptToContents(text string) []*genai.Content {
	return []*genai.Content{{Role: "user", Parts: []*genai.Part{{Text: text}}}}
}

type APIResponseError struct {
	msg string
}

func (e *APIResponseError) Error() string { return e.msg }

func shouldRetry(err error) bool {
	// APIResponseError はリトライしても解決しないため即座に false
	var apiErr *APIResponseError
	if errors.As(err, &apiErr) {
		return false
	}

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

func extractTextFromResponse(resp *genai.GenerateContentResponse) (string, error) {
	if resp == nil || len(resp.Candidates) == 0 {
		return "", &APIResponseError{msg: "Gemini APIから空のレスポンスが返されました"}
	}

	candidate := resp.Candidates[0]

	if candidate.FinishReason != genai.FinishReasonUnspecified && candidate.FinishReason != genai.FinishReasonStop {
		return "", &APIResponseError{msg: fmt.Sprintf("生成がブロックされました。理由: %v", candidate.FinishReason)}
	}

	if candidate.Content == nil || len(candidate.Content.Parts) == 0 {
		return "", &APIResponseError{msg: "レスポンスのコンテンツが空です"}
	}

	firstPart := candidate.Content.Parts[0]
	// テキストが存在しない場合は単なる抽出エラーとして扱う
	if firstPart.Text == "" {
		return "", fmt.Errorf("テキストデータが含まれていません")
	}

	return firstPart.Text, nil
}
