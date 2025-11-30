# ✨ Go AI Client: テンプレートベース Gemini クライアントライブラリ

[![Language](https://img.shields.io/badge/Language-Go-blue)](https://golang.org/)
[![Go Version](https://img.shields.io/github/go-mod/go-version/shouni/go-ai-client)](https://golang.org/)
[![GitHub tag (latest by date)](https://img.shields.io/github/v/tag/shouni/go-ai-client)](https://github.com/shouni/go-ai-client/tags)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## 🎯 概要: Gemini APIのためのテンプレートベースAIクライアント

Go AI Client は、Go言語で Google **Gemini API** を利用するためのクライアントライブラリと、**テンプレートベースのプロンプト生成**ユーティリティを提供します。

開発者は、複雑なAPIリクエストの構築や生の通信を意識することなく、GoらしいシンプルなインターフェースでAIモデルのパワーを活用できます。この設計は、アプリケーションの**責務の分離**（変換、I/O、AI通信）を重視したクリーンなアーキテクチャに基づいています。

-----

## 💎 特徴と設計思想

このライブラリは、単なるAPIラッパーではなく、堅牢性と柔軟性を念頭に置いて設計されています。

* ### 🤖 堅牢なAIクライアント (`pkg/ai/gemini`)

    * **責務の限定:** クライアントの責務は、**API通信**、**リトライ処理**、**モデルパラメータの設定**のみに限定され、プロンプトの内容とは独立しています。
    * **モデルパラメータ:** クライアント初期化時に**温度 (`Temperature`)** を設定でき、生成されるコンテンツのランダム性（創造性）を制御できます。
    * **自動リトライ戦略:** ネットワークエラーや一時的なAPIエラーに対応するため、`github.com/shouni/go-utils/retry` を利用した**指数バックオフ**付きの自動リトライロジックを内蔵しています。(**最大リトライ回数、初期間隔、最大間隔**を設定可能)。

* ### 📝 柔軟なプロンプト管理 (`pkg/prompts`)

    * **DI対応の抽象化:** プロンプト構築ロジックは、**`Builder`インターフェース**によって抽象化されており、依存性注入（DI）を容易にしています。
    * **テンプレートキャッシュ:** テンプレートファイル（例: `prompt_dialogue.md`, `prompt_solo.md`）は、**初期化時にすべてキャッシュ**され、ランタイムのパフォーマンスが向上しています。

-----

## 🚀 インストール

Goモジュールとしてプロジェクトに追加します。

```bash
go get github.com/shouni/go-ai-client/v2
```

### 🗝️ APIキーの設定

本ライブラリは、標準的なGoの慣習に従い、環境変数からGoogle Gemini APIキーを読み込みます。

**`GEMINI_API_KEY`** または **`GOOGLE_API_KEY`** にAPIキーを設定してください。

```bash
export GEMINI_API_KEY="YOUR_API_KEY"
```

-----

## 💡 使用方法

このライブラリは、**CLIツール**の形で利用することを推奨していますが、Goコード内でのクライアント利用も容易です。

### 1\. CLIツールとしての使用 (推奨)

`prompt` または `generic` コマンドでAIを利用できます。

```bash
# ビルド
go build -o bin/ai-client

# 実行例 1: テンプレートを使用し、モードを指定
./bin/ai-client prompt "地球温暖化の主要な原因とその対策について、簡潔に説明してください。" \
    -d solo 
    
# 実行例 2: パイプとテンプレートを使用 (dialogueモード)
# input.txt の内容をプロンプトデータとして使用
cat input.txt | ./bin/ai-client prompt -d dialogue -m gemini-2.5-flash

# 実行例 3: テンプレートを使用しない (generic コマンド)
# 自由なテキストをそのままプロンプトとして送信
./bin/ai-client generic "AI技術が社会に与えるポジティブな影響と、リスクについて議論してください。"
```

| 主要なCLIフラグ | 役割 | 関連パッケージ |
| :--- | :--- | :--- |
| **`-d, --mode`** | `prompt`コマンド専用のテンプレートモード (`solo`, `dialogue`) | `pkg/prompts` |
| **`-m, --model`** | 使用するGeminiモデル名 (`gemini-2.5-flash`など) | `pkg/ai/gemini` |
| **`-t, --timeout`** | 全体のリクエストタイムアウト時間 (秒) | `pkg/ai/gemini` |

### 2\. Goコード内でのクライアント使用

クライアントは、**最終的なプロンプト文字列**を受け取ります。リトライや温度設定は、クライアント初期化時に行います。

| `gemini.Config` フィールド | 役割 | デフォルト値 |
| :--- | :--- | :--- |
| **`Temperature`** | モデルの応答温度 (0.0: 決定的 \~ 1.0: 創造的)。 | `0.7` |
| **`MaxRetries`** | APIコールが失敗したときの最大リトライ回数。 | `3` |
| **`InitialDelay`** | 指数バックオフの初期間隔 (`time.Duration`)。 | **`30s`** |
| **`MaxDelay`** | 指数バックオフの最大間隔 (`time.Duration`)。 | **`120s`** |

-----

## 📂 プロジェクト構造 (現在の構成に基づく)

Goプロジェクトの関心事を分離するため、以下の構造を採用しています。

| ディレクトリ/ファイル | 責務の範囲 |
| :--- | :--- |
| `cmd/` | **I/O層**: CLIのエントリーポイント。フラグの解析、依存関係の構築、結果の標準出力への表示。 |
| `pkg/ai/gemini` | **外部層**: Google Gemini APIとの通信ロジック、リトライ、タイムアウト管理。 |
| `pkg/prompts` | **ビジネスロジック層**: プロンプトテンプレートの管理、データ埋め込み (`TemplateData`)、モードに応じたプロンプト構築。 |

-----

### 📜 ライセンス (License)

このプロジェクトは [MIT License](https://opensource.org/licenses/MIT) の下で公開されています。
