package gemini

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/shouni/go-utils/retry"
	"golang.org/x/sync/errgroup"
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
	// fileAPITransferThreshold は、インラインデータをFile APIへ自動転送する際のデータサイズの閾値 (512KB) です。
	fileAPITransferThreshold = 512 * 1024
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

// uploadToInternalFileAPI はバイナリデータを Gemini File API にアップロードし、URI を返す
func (c *Client) uploadToFileAPI(ctx context.Context, data []byte, mimeType string) (string, error) {
	// io.Reader に変換
	reader := bytes.NewReader(data)

	// アップロード設定
	// DisplayName は任意だけど、管理しやすいようにタイムスタンプを入れる
	uploadCfg := &genai.UploadFileConfig{
		MIMEType:    mimeType,
		DisplayName: fmt.Sprintf("auto-upload-%d", time.Now().UnixNano()),
	}

	// SDK の Files.Upload メソッドを呼び出す
	file, err := c.client.Files.Upload(ctx, reader, uploadCfg)
	if err != nil {
		return "", fmt.Errorf("failed to upload to File API: %w", err)
	}

	// 4. アップロードされたファイルの URI を返す
	// 例: https://generativelanguage.googleapis.com/v1beta/files/xxxx
	slog.InfoContext(ctx, "巨大データを File API へ自動退避したのだ", "uri", file.URI, "size", len(data))
	return file.URI, nil
}

// GenerateWithParts は画像データなどの Parts を含むリクエストを処理する
func (c *Client) GenerateWithParts(ctx context.Context, modelName string, parts []*genai.Part, opts ImageOptions) (*Response, error) {
	processedParts := make([]*genai.Part, len(parts))
	copy(processedParts, parts) // 元のパーツをコピー

	eg, gCtx := errgroup.WithContext(ctx)

	for i, p := range parts {
		if p.InlineData != nil && len(p.InlineData.Data) > fileAPITransferThreshold {
			i := i
			p := p

			eg.Go(func() error {
				slog.InfoContext(gCtx, "巨大なインラインデータを検知。File APIへ自動転送します。", "size", len(p.InlineData.Data))
				fileURI, err := c.uploadToFileAPI(gCtx, p.InlineData.Data, p.InlineData.MIMEType)
				if err != nil {
					// エラーをラップして返すのみに留める
					return fmt.Errorf("failed to upload large inline data to File API: %w", err)
				}
				processedParts[i] = &genai.Part{FileData: &genai.FileData{FileURI: fileURI}}
				return nil
			})
		}
	}

	// 並列アップロード処理中にコンテキストのキャンセルなどが発生した場合のエラーを処理します。
	if err := eg.Wait(); err != nil {
		// 呼び出し元で一元的にエラーログを出力
		slog.ErrorContext(ctx, "File APIへの並列アップロード中にエラーが発生しました。", "error", err)
		return nil, fmt.Errorf("failed during parallel file upload: %w", err)
	}

	contents := []*genai.Content{{Role: "user", Parts: processedParts}}
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

	// Partsの中からテキストを検索する
	for _, part := range candidate.Content.Parts {
		if part.Text != "" {
			return part.Text, nil
		}
	}

	// テキスト部分が見つからなかった場合
	return "", nil
}
