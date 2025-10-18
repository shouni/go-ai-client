# go-ai-client

[![Language](https://img.shields.io/badge/Language-Go-blue)](https://golang.org/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://github.com/shouni/go-ai-client/blob/main/LICENSE)
[![GitHub tag (latest by date)](https://img.shields.io/github/v/tag/shouni/go-ai-client)](https://github.com/shouni/go-ai-client/tags)

## 🎯 概要

`go-ai-client`は、Go言語でGenerative AI（特にGoogle **Gemini API**）を簡単に利用するためのクライアントライブラリおよび**テンプレートベースのプロンプト生成**ユーティリティを提供します。

### ✨ 特徴

* **Gemini API クライアント:** Google Gemini APIとの基本的なやり取りを行うためのシンプルで使いやすいクライアントを提供します。
* **リトライ戦略:** ネットワークエラーや一時的なAPIエラーに対応するため、自動リトライロジック（`go-web-exact/pkg/retry`を使用）を内蔵しています。
* **柔軟なプロンプトモード:** 特定タスク向けの**テンプレートベースのプロンプト**（`prompt`コマンド）と、自由なテキストをそのまま送る**汎用モード**（`generic`コマンド）をサポートします。

---

### 🚀 インストール

Goモジュールとしてプロジェクトに追加します。

```bash
go get https://github.com/shouni/go-ai-client
````

### 🗝️ APIキーの設定

本ライブラリは、環境変数からGoogle Gemini APIキーを読み込むことを想定しています。

**`GEMINI_API_KEY`** または **`GOOGLE_API_KEY`** にAPIキーを設定してください。

```bash
export GEMINI_API_KEY="YOUR_API_KEY"
```

-----

### 💡 使用方法

このライブラリのコア機能は、CLIツール (`cmd/root.go`) を通じて提供されています。

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
```

#### 2\. Goコード内でのクライアント使用

クライアントは、入力内容とモードを受け取り、内部でテンプレート処理の有無を判断します。

**【テンプレートを使用する場合】**

`mode` に登録済みのテンプレート名（`"solo"` や `"dialogue"`）を指定します。

**【テンプレートを使用しない場合（genericに相当）】**

`mode` に**空文字列 (`""`)** を指定することで、`inputContent` をそのままプロンプトとして API に送信できます。

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "go-ai-client/pkg/ai/gemini"
)

func main() {
    ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
    defer cancel()

    client, err := gemini.NewClientFromEnv(ctx)
    if err != nil {
       log.Fatalf("クライアント初期化エラー: %v", err)
    }

    // テンプレートを使用しない場合の例
    inputContent := []byte("Go言語でAPIクライアントを作成する利点について教えてください。")
    mode := "" // テンプレートを使用しないため、空文字列を指定
    model := "gemini-1.5-flash"
    
    // コンテンツ生成 (内部で mode="" を検知し、テンプレート処理をスキップ)
    response, err := client.GenerateContent(ctx, inputContent, mode, model)
    if err != nil {
       log.Fatalf("コンテンツ生成エラー: %v", err)
    }

    fmt.Println("--- 応答 ---")
    fmt.Println(response.Text)
}
```

-----

### 📂 プロジェクト構造（`pkg`ディレクトリ）

| ディレクトリ/ファイル | 概要 |
| :--- | :--- |
| `pkg/ai/gemini` | Google Gemini API専用のクライアント実装。 |
| `pkg/ai/gemini/client.go` | **Gemini APIクライアント**のコアロジック。`mode=""`を検知し、テンプレート処理をスキップする機能を含む。 |
| `pkg/prompt` | プロンプトのテンプレートおよび構築ロジック。 |
| `pkg/prompt/prompt.go` | **テンプレートベースのプロンプト定義**と、入力内容を埋め込む `BuildFullPrompt` 関数を提供。 |

-----

### 📜 ライセンス (License)

このプロジェクトは [MIT License](https://opensource.org/licenses/MIT) の下で公開されています。




