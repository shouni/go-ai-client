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

	DefaultTopP           float32 = 0.95
	DefaultCandidateCount int32   = 1
)

// Response は Gemini API からの応答を統一して扱うための構造体なのだ。
type Response struct {
	Text        string                         // 抽出されたテキスト内容
	RawResponse *genai.GenerateContentResponse // SDK生のレスポンス（画像データ抽出用などに保持）
}

// GenerativeModel は、テキストやマルチモーダル（画像含む）生成の振る舞いを定義するインターフェースなのだ。
type GenerativeModel interface {
	// GenerateContent はテキストプロンプトを送信して応答を得るのだ。
	GenerateContent(ctx context.Context, prompt string, modelName string) (*Response, error)
	// GenerateWithParts は画像などのマルチモーダルパーツを送信して応答を得るのだ。
	GenerateWithParts(ctx context.Context, modelName string, parts []*genai.Part, opts ImageOptions) (*Response, error)
}

// Client は Gemini API との実際の通信を担い、リトライ機能を管理する実体なのだ。
type Client struct {
	client      *genai.Client
	temperature float32
	retryConfig retry.Config
}

// Config はクライアントを初期化するための設定項目なのだ。
type Config struct {
	APIKey       string        // Google AI SDK の API キー
	Temperature  *float32      // 生成の多様性（0.0〜1.0）
	MaxRetries   uint64        // 失敗時の最大リトライ回数
	InitialDelay time.Duration // 指数バックオフの開始待機時間
	MaxDelay     time.Duration // 指数バックオフの最大待機時間
}

// ImageOptions は画像生成時に渡すオプション情報なのだ。
type ImageOptions struct {
	AspectRatio string // アスペクト比（"16:9", "1:1" など）
	Seed        *int32 // 生成を固定するためのシード値
}

// NewClient は設定を基に新しい Gemini クライアントを生成するのだ。
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

// NewClientFromEnv は環境変数（GEMINI_API_KEY等）から設定を読み取って初期化するのだ。
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

// executeWithRetry は指定された操作をリトライ設定に従って実行する内部関数なのだ。
func (c *Client) executeWithRetry(ctx context.Context, operationName string, op func() error, shouldRetryFn func(error) bool) error {
	return retry.Do(ctx, c.retryConfig, operationName, op, shouldRetryFn)
}

// GenerateContent は純粋なテキストプロンプトからコンテンツを生成するのだ。
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

	// 失敗時にリトライ判定を呼び出しつつ実行するのだ
	err := c.executeWithRetry(ctx, fmt.Sprintf("Gemini API call to %s", modelName), op, shouldRetry)
	if err != nil {
		return nil, err
	}

	return finalResp, nil
}

// GenerateWithParts は画像データなどの Parts を含むリクエストを処理するのだ。
func (c *Client) GenerateWithParts(ctx context.Context, modelName string, parts []*genai.Part, opts ImageOptions) (*Response, error) {
	contents := []*genai.Content{{Role: "user", Parts: parts}}

	genConfig := &genai.GenerateContentConfig{
		Temperature:    genai.Ptr(c.temperature),
		TopP:           genai.Ptr(DefaultTopP),
		CandidateCount: DefaultCandidateCount,
		Seed:           opts.Seed,
	}

	// アスペクト比が指定されている場合のみ ImageConfig をセットする（未対応モデルでのエラー防止なのだ）
	if opts.AspectRatio != "" {
		genConfig.ImageConfig = &genai.ImageConfig{
			AspectRatio: opts.AspectRatio,
		}
	}

	var finalResp *Response
	op := func() error {
		resp, err := c.client.Models.GenerateContent(ctx, modelName, contents, genConfig)
		if err != nil {
			return err
		}

		// 画像生成の場合、テキストが空でもエラーにせず正常終了とするのだ
		text, extractErr := extractTextFromResponse(resp)
		if extractErr != nil {
			return extractErr
		}

		finalResp = &Response{Text: text, RawResponse: resp}
		return nil
	}

	err := c.executeWithRetry(ctx, fmt.Sprintf("Gemini Image API call to %s", modelName), op, shouldRetry)
	if err != nil {
		return nil, err
	}

	return finalResp, nil
}

// promptToContents は文字列を SDK が受け取れる Content 構造に変換するのだ。
func promptToContents(text string) []*genai.Content {
	return []*genai.Content{{Role: "user", Parts: []*genai.Part{{Text: text}}}}
}

// APIResponseError は生成ブロックや空レスポンスなど、通信成功後の論理的なエラーを示すのだ。
type APIResponseError struct {
	msg string
}

func (e *APIResponseError) Error() string { return e.msg }

// shouldRetry は発生したエラーがリトライで解決可能かどうかを判定するのだ。
func shouldRetry(err error) bool {
	// 規約違反（ブロック）などはリトライしても無駄なので即座に諦めるのだ
	var apiErr *APIResponseError
	if errors.As(err, &apiErr) {
		return false
	}

	// キャンセルやタイムアウト（上位管理）もリトライ対象外なのだ
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	// gRPC のステータスコードを元に、一時的な障害のみリトライを許可するのだ
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

// extractTextFromResponse はレスポンスからテキストを安全に抽出し、異常な終了理由がないか確認するのだ。
func extractTextFromResponse(resp *genai.GenerateContentResponse) (string, error) {
	if resp == nil || len(resp.Candidates) == 0 {
		return "", &APIResponseError{msg: "Gemini APIから空のレスポンスが返されました"}
	}

	candidate := resp.Candidates[0]

	// FinishReason が正常（指定なし or 停止）以外ならブロックとみなすのだ
	if candidate.FinishReason != genai.FinishReasonUnspecified && candidate.FinishReason != genai.FinishReasonStop {
		return "", &APIResponseError{msg: fmt.Sprintf("生成がブロックされました。理由: %v", candidate.FinishReason)}
	}

	// 画像生成の場合、Content 自体が空でもエラーにせず続行させるのだ（画像データは別途取得可能）
	if candidate.Content == nil || len(candidate.Content.Parts) == 0 {
		return "", nil
	}

	firstPart := candidate.Content.Parts[0]

	// テキストが存在すればそれを返し、なければ空文字を返すのだ（エラーにはしない！）
	return firstPart.Text, nil
}
