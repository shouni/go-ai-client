package gemini

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"google.golang.org/genai"
)

// uploadToFileAPI はデータをアップロードし、Active状態になるまでポーリングするのだ。
// 戻り値として、File APIでのURI、削除時に使用する名前、およびエラーを返すのだ。
func (c *Client) uploadToFileAPI(ctx context.Context, data []byte, mimeType string) (string, string, error) {
	reader := bytes.NewReader(data)
	uploadCfg := &genai.UploadFileConfig{
		MIMEType:    mimeType,
		DisplayName: fmt.Sprintf("gemini-auto-%d", time.Now().UnixNano()),
	}

	// 1. ファイルをアップロードするのだ
	file, err := c.client.Files.Upload(ctx, reader, uploadCfg)
	if err != nil {
		return "", "", fmt.Errorf("file upload failed: %w", err)
	}

	// 2. Active状態になるまでポーリング待機するのだ
	ticker := time.NewTicker(filePollingInterval)
	defer ticker.Stop()

	// 無限ループを防ぐためのタイムアウト設定なのだ
	timeout := time.After(filePollingTimeout)

	for {
		select {
		case <-ctx.Done():
			// 呼び出し元がキャンセルされた場合
			return "", "", ctx.Err()

		case <-timeout:
			// File API側の処理が遅延してタイムアウトした場合、非同期でクリーンアップを試みるのだ
			// 元の ctx は切れている可能性があるため、Background を使用するのだ
			go func(fileName string) {
				_, _ = c.client.Files.Delete(context.Background(), fileName, &genai.DeleteFileConfig{})
			}(file.Name)
			return "", "", fmt.Errorf("file processing timed out after %v", filePollingTimeout)

		case <-ticker.C:
			// 現在の状態を取得するのだ
			currentFile, err := c.client.Files.Get(ctx, file.Name, &genai.GetFileConfig{})
			if err != nil {
				return "", "", fmt.Errorf("failed to get file status: %w", err)
			}

			switch currentFile.State {
			case genai.FileStateActive:
				// 利用可能になったのだ！
				return currentFile.URI, currentFile.Name, nil
			case genai.FileStateFailed:
				// 何らかの理由で処理が失敗した場合
				return "", "", errors.New("File API processing failed on server side")
			case genai.FileStateProcessing:
				// まだ処理中なので次のループへ行くのだ
				slog.DebugContext(ctx, "File API processing...", "name", file.Name)
				continue
			default:
				// 未定義の状態などの場合
				slog.WarnContext(ctx, "Unknown file state received", "state", currentFile.State)
			}
		}
	}
}
