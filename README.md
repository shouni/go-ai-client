# go-ai-client

[![Language](https://img.shields.io/badge/Language-Go-blue)](https://golang.org/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://github.com/shouni/go-ai-client/blob/main/LICENSE)
[![GitHub tag (latest by date)](https://img.shields.io/github/v/tag/shouni/go-ai-client)](https://github.com/shouni/go-ai-client/tags)

## 🎯 概要

`go-ai-client` は、Go言語で実装された、ウェブページからノイズを除去し、**記事の本文や主要なコンテンツを正確に抽出**するためのCLIツール、およびコアパッケージ群です。特に、LLM（大規模言語モデル）に入力するためのクリーンで**構造化されたテキストデータを生成する**ことを目的としています。

-----

## 🚀 特徴

* **堅牢なHTTP処理 (GET/POST対応):** `context` を使用したタイムアウト制御に加え、**GETリクエストとJSON POSTリクエスト**の両方をサポートし、不安定なネットワーク環境や一時的なサーバーエラーに対応します。


-----

## ⚙️ CLIとしての利用

このプロジェクトは、`cmd/root.go` をエントリーポイントとする実行可能なCLIアプリケーションとして設計されています。

### ビルドと実行

プロジェクトルートで以下のコマンドを実行し、バイナリを生成します。

```bash
# バイナリのビルド
go build -o bin/ai-client
```

### 使用方法



**ヘルプメッセージ:**



## 📦 ライブラリ利用方法

主要な機能は

### 1\. インポート


### 2\. 変換 (`ai-client` の利用)



-----

## 🛠️ 開発者向け情報

### パッケージ構成

| ディレクトリ | パッケージ名 | 役割 |


-----

### 📜 ライセンス (License)

このプロジェクトは [MIT License](https://opensource.org/licenses/MIT) の下で公開されています。


