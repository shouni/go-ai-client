# Go AI Client

[![Language](https://img.shields.io/badge/Language-Go-blue)](https://golang.org/)
[![Go Version](https://img.shields.io/github/go-mod/go-version/shouni/go-ai-client)](https://golang.org/)
[![GitHub tag (latest by date)](https://img.shields.io/github/v/tag/shouni/go-ai-client)](https://github.com/shouni/go-ai-client/tags)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## 🎯 概要: Gemini APIのためのテンプレートベースAIクライアント

Go AI Client は、Go言語で Generative AI（特に Google **Gemini API**）を簡単に利用するためのクライアントライブラリ、および**テンプレートベースのプロンプト生成**ユーティリティを提供します。

このパッケージを使うことで、開発者は複雑なAPIリクエストの構築や生のJSON/HTTP通信を意識することなく、GoらしいシンプルなインターフェースでAIモデルのパワーを活用できます。

### ✨ 特徴

* **Gemini API クライアント:** Google Gemini APIとの基本的なやり取りを行うためのシンプルで使いやすいクライアントを提供します。**責務がAPI通信とリトライ処理、およびモデルパラメータの設定に限定**され、プロンプトの内容に依存しません。
* **モデルパラメータのサポート (New!):** クライアントの初期化時に**温度 (`Temperature`)** を設定できるようになりました。これにより、生成されるコンテンツのランダム性（創造性）を制御できます。
* **リトライ戦略:** ネットワークエラーや一時的なAPIエラーに対応するため、自動リトライロジック（`go-web-exact/pkg/retry`を使用）を内蔵しています。
* **柔軟なプロンプトモード:** 特定タスク向けの**テンプレートベースのプロンプト構築機能**（`pkg/prompt`）と、自由なテキストをそのまま送る**汎用モード**をサポートします。プロンプト構築のロジックはCLIツール側（`cmd`）に存在します。

-----

### 🚀 インストール

Goモジュールとしてプロジェクトに追加します。

```bash
go get https://github.com/shouni/go-ai-client
```

### 🗝️ APIキーの設定

本ライブラリは、環境変数からGoogle Gemini APIキーを読み込むことを想定しています。

**`GEMINI_API_KEY`** または **`GOOGLE_API_KEY`** にAPIキーを設定してください。

```bash
export GEMINI_API_KEY="YOUR_API_KEY"
```

-----

### 💡 使用方法

このライブラリのコア機能は、CLIツール (`cmd/root.go`) を通じて提供されています。（CLI側の対応は別途必要ですが、クライアント機能として記載します。）

#### 1\. CLIツールとしての使用 (推奨)

CLIツールとしてビルドし、コマンドラインで直接使用します。

```bash
# ビルド
go build -o bin/ai-client

# 実行例 1: テンプレートを使用 (prompt コマンド)
# soloモードでナレーションスクリプトを生成
./bin/ai-client prompt "地球温暖化の主要な原因とその対策について、簡潔に説明してください。" -d solo 

# 実行例 2: テンプレートを使用 (dialogue コマンド)
# dialogueモードで対話スクリプトを生成（モデル名も指定）
cat input.txt | ./bin/ai-client prompt -d dialogue -m gemini-2.5-flash

# 実行例 3: テンプレートを使用しない (generic コマンド)
# 自由なテキストをそのままプロンプトとして送信
./bin/ai-client generic "AI技術が社会に与えるポジティブな影響と、リスクについて議論してください。"

# 主要なフラグ:
# -d, --mode (promptコマンド専用): 生成するスクリプトのモード (solo, dialogue)
# -m, --model: 使用するGeminiモデル名
# -t, --timeout: 全体のリクエストタイムアウト時間 (秒)
# -T, --temperature (New!): モデルの応答温度 (0.0: 決定的, 1.0: 創造的)
```

#### 2\. Goコード内でのクライアント使用

クライアントは**最終的なプロンプト文字列**を受け取ります。テンプレートを使用する場合は、**呼び出し元**で `prompt.BuildFullPrompt` を使ってプロンプトを構築する必要があります。

**【テンプレートを使用する/しないにかかわらず共通】**

クライアントには、テンプレート処理を経た**最終的なプロンプト文字列**を渡します。また、**クライアント初期化時**に温度を設定できます。

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"
    "math"

    "go-ai-client/pkg/ai/gemini"
    "go-ai-client/pkg/prompt" 
)

func main() {
    ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
    defer cancel()

    // 新しい温度設定を定義
    tempValue := float32(0.9) // 創造性を高めるために高めに設定
    
    // クライアント設定を定義
    cfg := gemini.Config{
        APIKey: os.Getenv("GEMINI_API_KEY"), // 環境変数から取得
        // 温度を設定
        Temperature: &tempValue, 
    }

    client, err := gemini.NewClient(ctx, cfg) // NewClientを使用
    if err != nil {
       log.Fatalf("クライアント初期化エラー: %v", err)
    }
    
    // ----------------------------------------------------------------
    // 1. テンプレートを使用する場合 (例: solo モード)
    // ----------------------------------------------------------------
    rawInput := "Go言語でAPIクライアントを作成する利点について教えてください。" 
    mode := "solo" 
    
    finalPrompt, err := prompt.BuildFullPrompt(rawInput, mode) 
    if err != nil {
        log.Fatalf("プロンプト構築エラー: %v", err)
    }
    
    model := "gemini-2.5-flash"
    
    // クライアントは最終プロンプト文字列のみを受け取る
    response, err := client.GenerateContent(ctx, finalPrompt, model)
    if err != nil {
       log.Fatalf("コンテンツ生成エラー: %v", err)
    }

    fmt.Println("--- 応答 ---")
    fmt.Printf("設定温度: %.1f\n", *cfg.Temperature)
    fmt.Println(response.Text)
    
    // ----------------------------------------------------------------
    // 2. テンプレートを使用しない場合
    // ----------------------------------------------------------------
    // client.GenerateContent(ctx, "自由なテキストプロンプト", "モデル名") で直接実行可能
}
```

-----

### 📂 プロジェクト構造（`pkg`ディレクトリ）

| ディレクトリ/ファイル | 概要 |
| :--- | :--- |
| `pkg/ai/gemini` | Google Gemini API専用のクライアント実装。 |
| `pkg/ai/gemini/client.go` | **Gemini APIクライアント**のコアロジック。**API通信、リトライ、レスポンス処理、および温度設定**を担う。 |
| `pkg/prompt` | プロンプトのテンプレートおよび構築ロジック。 |
| `pkg/prompt/prompt.go` | **テンプレートベースのプロンプト定義**と、入力内容を埋め込む `BuildFullPrompt` 関数を提供。テンプレートは事前解析され、キャッシュされる。 |

-----

### 📜 ライセンス (License)

このプロジェクトは [MIT License](https://opensource.org/licenses/MIT) の下で公開されています。
