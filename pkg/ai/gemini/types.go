package gemini

import (
	"context"
	"time"

	"github.com/shouni/go-utils/retry"
	"google.golang.org/genai"
)

const (
	DefaultTemperature  float32 = 0.7
	DefaultMaxRetries           = 3
	DefaultInitialDelay         = 30 * time.Second
	DefaultMaxDelay             = 120 * time.Second

	DefaultTopP              float32 = 0.95
	DefaultCandidateCount    int32   = 1
	fileAPITransferThreshold         = 512 * 1024
	filePollingInterval              = 2 * time.Second
	filePollingTimeout               = 60 * time.Second
)

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
	AspectRatio    string
	Seed           *int32
	SystemPrompt   string
	SafetySettings []*genai.SafetySetting
}

type Response struct {
	Text        string
	RawResponse *genai.GenerateContentResponse
}
