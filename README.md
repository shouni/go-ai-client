# ✨ Go AI Client

[![Language](https://img.shields.io/badge/Language-Go-blue)](https://golang.org/)
[![Go Version](https://img.shields.io/github/go-mod/go-version/shouni/go-ai-client)](https://golang.org/)
[![GitHub tag (latest by date)](https://img.shields.io/github/v/tag/shouni/go-ai-client)](https://github.com/shouni/go-ai-client/tags)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## 🎯 概要: Gemini APIのためのテンプレートベースAIクライアント

Go AI Client は、Go言語で Google **Gemini API** を利用するためのクライアントライブラリと、**テンプレートベースのプロンプト生成**ユーティリティを提供します。

本プロジェクトは、**AI駆動のコンテンツ制作ワークフロー**を支える基盤として設計されています。クリーンなアーキテクチャに基づき、ビジネスロジックと外部API通信を厳密に分離しています。

-----

## 💎 特徴と設計思想

* ### 🤖 堅牢かつ多機能なAIクライアント (`pkg/ai/gemini`)

* **マルチモーダル対応:** テキストだけでなく、画像データを含む複数の要素 (`genai.Part`) を組み合わせたリクエストに対応しています。
* **インターフェースの抽象化:** SDK固有のレスポンスを抽象化した `Response` 型で統一。呼び出し側は内部実装の詳細を意識せずに結果を利用できます。
* **高度なリトライ戦略:** `github.com/shouni/go-utils/retry` による指数バックオフを内蔵。一時的な通信エラーは自動復旧しつつ、**セーフティフィルタによるブロック**などは即時検知してリトライを停止する賢いロジックを備えています。
* **決定論的な制御:** シード値 (`Seed`) をポインタで管理。完全なランダム生成と、シード固定による再現性のある生成を明確に使い分けられます。


* ### 📝 柔軟なプロンプト管理 (`pkg/prompts`)

* **DI対応の抽象化:** `Builder` インターフェースにより、プロンプト構築ロジックの差し替えやテストが容易です。
* **テンプレートキャッシュ:** 起動時にテンプレートをキャッシュし、ランタイムのオーバーヘッドを最小化しています。

-----

## 🚀 インストール

```bash
go get github.com/shouni/go-ai-client/v2
```

### 🗝️ APIキーの設定

環境変数 **`GEMINI_API_KEY`** または **`GOOGLE_API_KEY`** を設定してください。

```bash
export GEMINI_API_KEY="YOUR_API_KEY"
```

-----

## 💡 使用方法

### 1. CLIツールとしての使用

```bash
# 実行例: テンプレートモード (solo) でテキスト生成
./bin/ai-client prompt "ずんだ餅の魅力を熱く語ってなのだ。" -d solo 
    
# 実行例: パイプを利用した対話形式の生成
cat script.txt | ./bin/ai-client prompt -d dialogue -m gemini-2.5-flash

```

### 2. Goコード内でのクライアント使用

クライアントは、テキスト生成用の `GenerateContent` と、画像を含むマルチモーダル用の `GenerateWithParts` を提供します。

```go
// クライアントの初期化
client, _ := gemini.NewClientFromEnv(ctx)

// マルチモーダル入力（画像＋テキスト）の例
parts := []*genai.Part{
    genai.NewPartFromText("このキャラクターの表情に合わせたセリフを考えて"),
    genai.NewPartFromData(imageData, "image/png"),
}

// 画像生成用オプション (アスペクト比、シード値固定)
opts := gemini.ImageOptions{
    AspectRatio: "16:9",
    Seed:        genai.Ptr(int32(12345)), 
}

resp, err := client.GenerateWithParts(ctx, "gemini-1.5-flash", parts, opts)
if err != nil {
    log.Fatal(err)
}
fmt.Println(resp.Text)

```

| 設定項目 (`gemini.Config`) | 役割 | デフォルト値 |
| --- | --- | --- |
| **`Temperature`** | 応答の創造性 (0.0 ~ 1.0) | `0.7` |
| **`MaxOutputTokens`** | 最大出力トークン数 (安全性向上のため制限) | `4096` |
| **`TopP`** | 生成の多様性制御 | `0.95` |
| **`MaxRetries`** | 最大リトライ回数 | `3` |
| **`InitialDelay`** | リトライ開始時の待機時間 | `30s` |

----

## 📂 プロジェクト構造

| ディレクトリ | 役割 |
| --- | --- |
| `cmd/` | **I/O層**: CLIエントリーポイント、フラグ解析、DIコンテナの構築。 |
| `pkg/ai/gemini` | **外部層**: Gemini APIとの通信、リトライ、決定論的パラメータ管理。 |
| `pkg/prompts` | **ロジック層**: プロンプトテンプレートの管理、データ埋め込み、モード切り替え。 |

### 📜 ライセンス (License)

このプロジェクトは [MIT License](https://opensource.org/licenses/MIT) の下で公開されています。
