<p align="center">
  <img src="icon/curated-title-nobg.png" alt="Curated" width="520" />
</p>

<p align="center">
  <a href="README.md">English</a> | <a href="README.zh-CN.md">简体中文</a> | 日本語
</p>

<p align="center">
  <img alt="Vue 3" src="https://img.shields.io/badge/Vue-3-42b883?style=flat-square&logo=vuedotjs&logoColor=white">
  <img alt="TypeScript 5.x" src="https://img.shields.io/badge/TypeScript-5.x-3178c6?style=flat-square&logo=typescript&logoColor=white">
  <img alt="Vite 8.x" src="https://img.shields.io/badge/Vite-8.x-646cff?style=flat-square&logo=vite&logoColor=white">
  <img alt="Go 1.25+" src="https://img.shields.io/badge/Go-1.25+-00add8?style=flat-square&logo=go&logoColor=white">
  <img alt="SQLite modernc" src="https://img.shields.io/badge/SQLite-modernc-003b57?style=flat-square&logo=sqlite&logoColor=white">
  <img alt="Tailwind CSS v4" src="https://img.shields.io/badge/Tailwind_CSS-v4-06b6d4?style=flat-square&logo=tailwindcss&logoColor=white">
  <img alt="shadcn-vue" src="https://img.shields.io/badge/shadcn--vue-ui-111111?style=flat-square">
  <img alt="Windows tray ready" src="https://img.shields.io/badge/Windows-tray_ready-0078d4?style=flat-square&logo=windows&logoColor=white">
</p>

# Curated

Curated は、Vue 3 フロントエンドと Go + SQLite バックエンドで構成されたローカルファーストのメディアライブラリアプリケーションです。現在のリポジトリには、Web ファースト構成、Windows 向けの配布パッケージ、トレイモード起動、メタデータスクレイピング、再生ワークフロー、そしてキュレートフレーム管理が実装されています。

正式な製品名は **Curated** です。リポジトリ名や npm パッケージ名には引き続き **`jav-shadcn`** が使われている場合があります。Go モジュール名は **`curated-backend`**、サーバーエントリポイントは **`backend/cmd/curated`** です。

## ハイライト

- Vue SPA フロントエンドと Go HTTP API バックエンドによるローカルファースト構成。
- 実 API モードと Mock モードの両方を備え、UI 開発を高速に進めやすい。
- ライブラリデータ、再生進捗、コメント、評価、キュレートフレームを SQLite に永続化。
- Web API モードでは、ホームの hero と「今日のおすすめ」に UTC 日次で切り替わる SQLite 永続スナップショットを使い、ブラウザや端末をまたいでも同じ結果を表示しつつ、直近の露出履歴ペナルティに加えて出演者とスタジオの偏りも抑え、連日の重複や同日の偏在をできるだけ減らします。
- Windows リリースではトレイモード起動、ローカル Web 配信、インストーラーパッケージ生成をサポート。
- 現在の Web フェーズでも、俳優メタデータ、キュレートフレームのエクスポート、再生セッション診断を利用可能。

## クイックスタート

### 必要環境

- **Node.js**: Vite 8 と互換性のある現行 LTS
- **pnpm**: このリポジトリは `pnpm-lock.yaml` を使用
- **Go**: `1.25.4+`

### バックエンドを起動する

```bash
cd backend
go run ./cmd/curated
```

開発時のデフォルト:

- HTTP アドレス: `:8080`
- ヘルス名: `curated-dev`

Windows 向け開発補助コマンド:

```bash
pnpm backend:build:dev
```

このコマンドは `backend/runtime/curated-dev.exe` を生成します。

### フロントエンドを起動する

```bash
pnpm install
pnpm dev
```

Vite の開発サーバーは通常 `http://localhost:5173` で起動します。

### 実 API モードと Mock モード

- リポジトリルートの `.env` に `VITE_USE_WEB_API=true` を設定すると、実バックエンド API を使用します。
- それ以外の値では Mock モードのまま動作します。
- Vite 開発サーバーは `/api` を `http://localhost:8080` にプロキシします。

## 機能

### ライブラリ

- 大規模ライブラリ向けの仮想化ポスターグリッド表示。
- お気に入り、評価、タグ付け、ライブラリ整理機能。
- 同じフロントエンドサービス層で実バックエンドと Mock アダプターの両方を利用可能。
- ホームの hero と「今日のおすすめ」は UTC 日付の切り替わりで自動再計算され、可能な限りバックエンド永続化済みの日次スナップショットを優先して使いながら、同じ日に出演者やスタジオが偏りすぎないようにも調整されます。

### 再生

- Web API モードで再生位置を永続化。
- 現在の再生パイプラインでは、ブラウザ再生、外部プレーヤーへの引き渡し、HLS セッションをサポート。
- 直再生、remux、トランスコードの判断理由を説明するための詳細な再生診断情報を提供。

### 俳優

- 俳優一覧、プロフィール読み込み、ユーザータグ編集に対応。
- 俳優アバターはバックエンドでキャッシュされ、同一オリジンで配信される。

### キュレートフレーム

- フレームキャプチャ、閲覧、タグ付け、フィルタリング、エクスポートをサポート。
- メタデータ付きの WebP / PNG エクスポートに対応。

### パッケージング

- Windows 向けのリリースフローを提供。
- リリース時はトレイモードで起動し、ビルド済みフロントエンドをローカル配信できる。

## 設定

実行時設定は、フロントエンド環境変数とバックエンド設定に分かれています。

### フロントエンド

- `VITE_USE_WEB_API=true`: 実バックエンドを使用
- `VITE_API_BASE_URL`: API ベース URL を上書き
- `VITE_LOG_LEVEL`: ブラウザログレベルのデフォルト値

### バックエンド

バックエンドは JSON の主設定を読み込み、次のライブラリ設定ファイルをマージします。

- `config/library-config.cfg`

主なライブラリ設定項目:

- `organizeLibrary`
- `metadataMovieProvider`
- `metadataMovieStrategy`
- `autoLibraryWatch`
- `launchAtLogin`
- `proxy`
- バックエンドログの保存先と保持設定

  空の `logDir` は「ファイルログを無効化」ではなく「既定のログ保存先を使う」意味です:
  release ビルドは `LOCALAPPDATA\\Curated\\logs`、開発時は `backend/runtime/logs` を使います。

リリースビルドでは、設定で上書きしない限りデフォルトで `:8081` を使用します。

## API

Curated は、ライブラリ、再生、俳優、設定、キュレートフレーム向けの Go HTTP API を提供します。

完全なエンドポイント一覧は [API.md](API.md) を参照してください。

## リポジトリ構成

```text
.
├── src/                    # Vue SPA: 画面、UI、ドメインコンポーネント、API クライアント、アダプター
├── backend/
│   ├── cmd/curated/        # バックエンドエントリポイント
│   └── internal/           # app、config、storage、server、scanner、scraper、tasks、desktop
├── config/                 # ライブラリ実行設定
├── docs/                   # 製品ノート、計画文書、UI 仕様、アーキテクチャ文書
├── icon/                   # ブランドデザインのソースアセット
└── package.json            # pnpm スクリプトと依存関係
```

## リリースとパッケージング

推奨リリース入口:

```powershell
pnpm release:publish
```

主なポイント:

- 本番パッケージのバージョンは `scripts/release/version.json` で一元管理されます。
- 現在のベースラインは `1.1.0` です。
- `pnpm release:*` は現在 `python scripts/release/release_cli.py` に統一されています。
- リリースフローでは、Windows 用ステージングディレクトリ、ポータブル zip、インストーラー実行ファイル、リリースマニフェストを生成します。
- パッケージ履歴台帳は `docs/package-build-history.csv` に移行済みで、Excel / WPS 互換のため UTF-8 with BOM で保存されます。
- Windows リリースビルドはデフォルトでトレイモードになり、`frontend-dist/` が実行ファイルの横にあればフロントエンドをローカル配信できます。
- インストーラー自体は引き続き Inno Setup を使いますが、`.iss` テンプレートの描画と `ISCC.exe` 呼び出しは Python 側で行います。
- 設定画面から現在のユーザー向け Windows ログイン時起動を永続化できます。この自動起動はサイレントでトレイに入り、ブラウザは自動で開きません。

関連するリリース資料:

- [docs/plan/2026-03-31-production-packaging-and-config-strategy.md](docs/plan/2026-03-31-production-packaging-and-config-strategy.md)
- [docs/package-build-history.csv](docs/package-build-history.csv)
- [docs/2026-04-02-package-build-history.md](docs/2026-04-02-package-build-history.md)

## ドキュメント

- [API.md](API.md): 公開 HTTP API リファレンス
- [docs/2026-03-20-jav-libary.md](docs/2026-03-20-jav-libary.md): 製品設計と目標アーキテクチャ
- [docs/2026-03-20-project-memory.md](docs/2026-03-20-project-memory.md): 実装事実と安定したプロジェクトメモリ
- [docs/architecture-and-implementation.html](docs/architecture-and-implementation.html): アーキテクチャ概要
- [docs/2026-03-21-library-organize.md](docs/2026-03-21-library-organize.md): ライブラリ整理メモ
- [docs/2026-03-24-frontend-ui-spec.md](docs/2026-03-24-frontend-ui-spec.md): フロントエンド UI 仕様

## 補足

- 現在のリポジトリは **Web-first** 実装フェーズです。
- Electron と mpv は今後の方向性であり、現時点でこのリポジトリに同梱される機能ではありません。
- `docs/film-scanner/` は主に参照資料とフィクスチャを保持しており、本番モジュール構成そのものではありません。

## Root Directory Policy

- `videos_test/` はローカルのテスト用フィクスチャディレクトリとしてリポジトリ直下に固定で残します。
- `config/` はライブラリ実行設定のためにリポジトリ直下に残し、`backend/internal/config` へ統合しません。
- `backend/runtime/` は開発時の実行生成物を置く許可済みディレクトリです。
- 新しいローカル専用の一時状態は `.workspace/` を優先します。
- Go ビルドキャッシュはリポジトリ内に作成しません。release スクリプトはバックエンド build cache にシステムの一時ディレクトリを使うようになりました。
