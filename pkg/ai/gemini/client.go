package gemini

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sync"

	"github.com/shouni/go-utils/retry"
	"golang.org/x/sync/errgroup"
	"google.golang.org/genai"
)

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

	retryCfg := retry.DefaultConfig()
	if cfg.MaxRetries > 0 {
		retryCfg.MaxRetries = cfg.MaxRetries
	} else {
		retryCfg.MaxRetries = DefaultMaxRetries
	}

	retryCfg.InitialInterval = DefaultInitialDelay
	if cfg.InitialDelay > 0 {
		retryCfg.InitialInterval = cfg.InitialDelay
	}

	retryCfg.MaxInterval = DefaultMaxDelay
	if cfg.MaxDelay > 0 {
		retryCfg.MaxInterval = cfg.MaxDelay
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
	contents := []*genai.Content{{Role: "user", Parts: []*genai.Part{{Text: finalPrompt}}}}
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

	err := c.executeWithRetry(ctx, fmt.Sprintf("Gemini API call to %s", modelName), op, shouldRetry)
	if err != nil {
		return nil, err
	}

	return finalResp, nil
}

// GenerateWithParts はマルチモーダルパーツを処理し、巨大なデータは自動的に File API へ退避するのだ。
func (c *Client) GenerateWithParts(ctx context.Context, modelName string, parts []*genai.Part, opts ImageOptions) (*Response, error) {
	processedParts := make([]*genai.Part, len(parts))
	copy(processedParts, parts)

	eg, gCtx := errgroup.WithContext(ctx)
	var (
		mu            sync.Mutex
		uploadedFiles []string
	)

	for i, p := range parts {
		if p.InlineData != nil && len(p.InlineData.Data) > fileAPITransferThreshold {
			i, p := i, p
			eg.Go(func() error {
				slog.InfoContext(gCtx, "巨大データを検知。File APIへ自動転送するのだ", "size", len(p.InlineData.Data))
				fileURI, fileName, err := c.uploadToFileAPI(gCtx, p.InlineData.Data, p.InlineData.MIMEType)
				if err != nil {
					return err
				}
				// インデックスが独立しているためここは安全なのだ
				processedParts[i] = &genai.Part{FileData: &genai.FileData{FileURI: fileURI}}

				// ★ 共有スライスへの append を Mutex で保護するのだ
				mu.Lock()
				uploadedFiles = append(uploadedFiles, fileName)
				mu.Unlock()

				return nil
			})
		}
	}

	// 並列アップロードの完了を待機するのだ
	if err := eg.Wait(); err != nil {
		slog.ErrorContext(ctx, "File APIへの並列アップロード中にエラーが発生しました", "error", err)
		return nil, fmt.Errorf("file upload failed: %w", err)
	}

	// 生成処理の完了後（または失敗時）、一時ファイルを一括削除するのだ
	defer func() {
		for _, name := range uploadedFiles {
			if _, err := c.client.Files.Delete(ctx, name, &genai.DeleteFileConfig{}); err != nil {
				slog.WarnContext(ctx, "File API クリーンアップ失敗", "name", name, "error", err)
			}
		}
	}()

	// --- AIへのリクエスト組み立て ---
	contents := []*genai.Content{{Role: "user", Parts: processedParts}}
	genConfig := &genai.GenerateContentConfig{
		Temperature:    genai.Ptr(c.temperature),
		TopP:           genai.Ptr(DefaultTopP),
		CandidateCount: DefaultCandidateCount,
		Seed:           opts.Seed,
		SafetySettings: opts.SafetySettings,
	}

	if opts.SystemPrompt != "" {
		genConfig.SystemInstruction = &genai.Content{
			Parts: []*genai.Part{{Text: opts.SystemPrompt}},
		}
	}

	if opts.AspectRatio != "" {
		genConfig.ImageConfig = &genai.ImageConfig{AspectRatio: opts.AspectRatio}
	}

	var finalResp *Response
	op := func() error {
		resp, err := c.client.Models.GenerateContent(ctx, modelName, contents, genConfig)
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

	// 指数バックオフ付きのリトライ実行なのだ
	err := c.executeWithRetry(ctx, fmt.Sprintf("Gemini Image API call to %s", modelName), op, shouldRetry)
	if err != nil {
		return nil, err
	}

	return finalResp, nil
}
