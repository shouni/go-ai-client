# go-ai-client

[![Language](https://img.shields.io/badge/Language-Go-blue)](https://golang.org/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://github.com/shouni/go-ai-client/blob/main/LICENSE)
[![GitHub tag (latest by date)](https://img.shields.io/github/v/tag/shouni/go-web-exact)](https://github.com/shouni/go-ai-client/tags)

## 🎯 概要

`go-ai-client`は、Go言語でGenerative AI（特にGoogle **Gemini API**）を簡単に利用するためのクライアントライブラリおよびプロンプト構築ユーティリティを提供します。

### ✨ 特徴

* **Gemini API クライアント:** Google Gemini APIとの基本的なやり取り（テキスト生成など）を行うためのシンプルで使いやすいクライアントを提供します。
* **プロンプトビルダー:** 複雑なプロンプトや、システム命令、チャット履歴などを含むコンテキストを簡単に構築するためのユーティリティを提供します。
* **モジュール化:** クライアント実装（`ai`）とプロンプトの構成（`prompt`）が分離されており、柔軟なAIアプリケーション開発をサポートします。

-----

### 🚀 インストール

Goモジュールとしてプロジェクトに追加します。

```bash
go get github.com/shouni/go-ai-client
```

### 🗝️ APIキーの設定

本ライブラリは、環境変数からGoogle Gemini APIキーを読み込むことを想定しています。

**`GEMINI_API_KEY`** にAPIキーを設定してください。

```bash
export GEMINI_API_KEY="YOUR_API_KEY"

```

-----

### 💡 使用方法

#### 1\. AIクライアントの初期化と使用

`pkg/ai/gemini/client.go` を利用して、Gemini APIにアクセスします。

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/shouni/go-ai-client/pkg/ai/gemini"
)

func main() {
	client := gemini.NewClient() // 環境変数からAPIキーを自動で取得

	prompt := "Go言語について簡単に説明してください。"
	
	// テキスト生成
	response, err := client.GenerateText(context.Background(), prompt, "gemini-2.5-flash")
	if err != nil {
		log.Fatalf("テキスト生成エラー: %v", err)
	}

	fmt.Println("--- 応答 ---")
	fmt.Println(response.Text)
}
```

#### 2\. プロンプトビルダーの使用

`pkg/prompt/builder.go` を利用して、より構造化されたプロンプトを作成します。

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/shouni/go-ai-client/pkg/ai/gemini"
	"github.com/shouni/go-ai-client/pkg/prompt"
)

func main() {
	// プロンプトの構築
	builder := prompt.NewBuilder()
	builder.SetSystemInstruction("あなたは親切でプロフェッショナルなアシスタントです。")
	builder.AddUserMessage("今日の天気予報を教えてください。")
	
	// 構築したプロンプトを取得 (クライアントが要求する形式に変換)
	geminiPrompt := builder.Build() 

	// クライアントで実行
	client := gemini.NewClient()
	response, err := client.GenerateContent(context.Background(), geminiPrompt, "gemini-2.5-flash")
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
| `pkg/ai` | AIプロバイダーとのインターフェースおよびクライアントの実装を含むパッケージ。 |
| `pkg/ai/gemini` | Google Gemini API専用のクライアント実装。 |
| `pkg/ai/gemini/client.go` | **Gemini APIクライアント**のコアロジック。テキスト生成などのAPIコールを提供。|
| `pkg/prompt` | プロンプトの構築や管理に関するユーティリティを含むパッケージ。 |
| `pkg/prompt/builder.go` | **プロンプトビルダー**の実装。システム命令やメッセージ履歴を簡単に構成するためのメソッドを提供。 |


-----

### 📜 ライセンス (License)

このプロジェクトは [MIT License](https://opensource.org/licenses/MIT) の下で公開されています。



