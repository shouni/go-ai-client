# Go AI Client

[![Language](https://img.shields.io/badge/Language-Go-blue)](https://golang.org/)
[![Go Version](https://img.shields.io/github/go-mod/go-version/shouni/go-ai-client)](https://golang.org/)
[![GitHub tag (latest by date)](https://img.shields.io/github/v/tag/shouni/go-ai-client)](https://github.com/shouni/go-ai-client/tags)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## 🎯 概要: Gemini APIのためのテンプレートベースAIクライアント

Go AI Client は、Go言語で Generative AI（特に Google **Gemini API**）を簡単に利用するためのクライアントライブラリ、および**テンプレートベースのプロンプト生成**ユーティリティを提供します。

このパッケージを使うことで、開発者は複雑なAPIリクエストの構築や生のJSON/HTTP通信を意識することなく、GoらしいシンプルなインターフェースでAIモデルのパワーを活用できます。

-----

### ✨ 特徴

* **Gemini API クライアント:** Google Gemini APIとの基本的なやり取りを行うためのシンプルで使いやすいクライアントを提供します。**責務がAPI通信とリトライ処理、およびモデルパラメータの設定に限定**され、プロンプトの内容に依存しません。
* **モデルパラメータのサポート:** クライアントの初期化時に**温度 (`Temperature`)** を設定できるようになりました。これにより、生成されるコンテンツのランダム性（創造性）を制御できます。
* **堅牢なリトライ戦略:** ネットワークエラーや一時的なAPIエラーに対応するため、**リトライ回数、初期間隔、最大間隔**を細かく設定できる自動リトライロジック（`github.com/shouni/go-utils/retry`を使用）を内蔵しています。
* **柔軟なプロンプトモード:** `prompts`パッケージは、**`Builder`インターフェース**による抽象化を採用し、DIを容易にしました。また、テンプレートは**初期化時にすべてキャッシュ**されるようになり、ランタイムのパフォーマンスが向上しています。プロンプト構築のロジックはCLIツール側（`cmd`）に存在します。

-----

### 🚀 インストール

Goモジュールとしてプロジェクトに追加します。

```bash
go get github.com/shouni/go-ai-client/v2
```

### 🗝️ APIキーの設定

本ライブラリは、環境変数からGoogle Gemini APIキーを読み込むことを想定しています。

**`GEMINI_API_KEY`** または **`GOOGLE_API_KEY`** にAPIキーを設定してください。

```bash
export GEMINI_API_KEY="YOUR_API_KEY"
```

-----

### 💡 使用方法

このライブラリは、**`github.com/shouni/go-cli-base`** を活用した**クリーンなCLI構造**を採用しており、CLIツールとしてもすぐに利用できます。

#### 1\. CLIツールとしての使用 (推奨)

CLIツールとしてビルドし、コマンドラインで直接使用します。

```bash
# ビルド
go build -o bin/ai-client

# 実行例: テンプレートを使用し、モデル名とモードを指定
./bin/ai-client prompt "地球温暖化の主要な原因とその対策について、簡潔に説明してください。" \
    -d solo 
    
# 実行例 2: テンプレートを使用 (dialogue コマンド)
# dialogueモードで対話スクリプトを生成（モデル名も指定）
cat input.txt | ./bin/ai-client prompt -d dialogue -m gemini-2.5-flash

# 実行例 3: テンプレートを使用しない (generic コマンド)
# 自由なテキストをそのままプロンプトとして送信
./bin/ai-client generic "AI技術が社会に与えるポジティブな影響と、リスクについて議論してください。"

# 主要なフラグ (CLIで設定可能なフラグ):
# -d, --mode (promptコマンド専用): 生成するスクリプトのモード (solo, dialogue)
# -m, --model: 使用するGeminiモデル名
# -t, --timeout: 全体のリクエストタイムアウト時間 (秒)
```

> **重要な変更:** APIクライアントの初期化（DI）は、`prompt` または `generic` コマンドが実行された直前に行われます。認証情報（`GEMINI_API_KEY`）に問題がある場合、汎用的な「Runner初期化エラー」ではなく、「Geminiクライアントの初期化に失敗しました」という具体的なエラーメッセージが表示されるようになりました。

#### 2\. Goコード内でのクライアント使用と詳細設定

クライアントには**最終的なプロンプト文字列**を渡し、**クライアント初期化時**にモデルパラメータとリトライパラメータを設定します。

| `gemini.Config` フィールド | 役割 | CLIフラグ | デフォルト値 |
| :--- | :--- | :--- |:---|
| **`Temperature`** | モデルの応答温度 (0.0: 決定的 \~ 1.0: 創造的)。 | (CLIフラグなし) | `0.7` |
| **`MaxRetries`** | APIコールが失敗したときの最大リトライ回数。 | (CLIフラグなし) | `3` |
| **`InitialDelay`** | 指数バックオフの初期間隔 (`time.Duration`)。 | (CLIフラグなし) | **`30s`** |
| **`MaxDelay`** | 指数バックオフの最大間隔 (`time.Duration`)。 | (CLIフラグなし) | **`120s`** |

-----

### 📂 プロジェクト構造（`pkg`ディレクトリ）

| ディレクトリ/ファイル | 概要 |
| :--- | :--- |
| `pkg/ai/gemini` | Google Gemini API専用のクライアント実装。 |
| `pkg/ai/gemini/client.go` | **Gemini APIクライアント**のコアロジック。**API通信、リトライ、レスポンス処理、温度、およびリトライディレイ設定**を担う。 |
| `pkg/prompts` | **テンプレートベースのプロンプト構築ロジック**（`Builder`インターフェース、`PromptBuilder`）。|

-----

### 📜 ライセンス (License)

このプロジェクトは [MIT License](https://opensource.org/licenses/MIT) の下で公開されています。
