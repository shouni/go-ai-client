package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/shouni/go-ai-client/v2/pkg/ai/gemini"
	"github.com/shouni/go-ai-client/v2/pkg/promptbuilder"
	"github.com/shouni/go-ai-client/v2/pkg/runner"
	"github.com/shouni/go-ai-client/v2/prompts"
)

// 実行ロジックをRunnerに委譲するため、Runnerのインスタンスを保持
var aiRunner *runner.Runner

// SetRunner は、RunnerのインスタンスをDIするためのセッターです。
func SetRunner(r *runner.Runner) {
	aiRunner = r
}

// SetupRunner は、コマンド実行に必要な全ての依存関係を構築し、グローバル変数 (aiRunner) にDIします。
func SetupRunner(ctx context.Context) error {
	// 既に設定済みであればスキップ
	if aiRunner != nil {
		return nil
	}

	// 1. Gemini Client の初期化とエラー処理の集約
	client, err := gemini.NewClientFromEnv(ctx)
	if err != nil {
		slog.Error("🚨 Geminiクライアント初期化失敗", "error", err)
		return fmt.Errorf("Geminiクライアントの初期化に失敗しました。認証情報（GEMINI_API_KEYなど）を確認してください: %w", err)
	}

	// 2. Runner のインスタンス構築とDI（Timeout変数を直接利用）
	r := runner.NewRunner(
		client,
		runner.TemplateGetterFunc(prompts.GetTemplate),
		promptbuilder.NewPromptBuilder,
		ModelName,
		time.Duration(Timeout)*time.Second, // Timeoutフラグを直接Durationに変換
	)

	SetRunner(r) // DIの完了
	return nil
}
