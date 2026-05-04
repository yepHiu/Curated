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

Curated は、Vue 3 フロントエンドと Go + SQLite バックエンドで構成されたローカルファーストのメディアライブラリアプリケーションです。現在のリポジトリには、Web ファースト構成、Windows 向けの配布パッケージ、トレイモード起動、メタデータスクレイピング、再生ワークフロー、キュレートフレーム管理、ゲームパッド操作、そして包括的な設定システムが実装されています。

全実装機能の一覧は [docs/features/2026-05-03-feature-inventory.md](docs/features/2026-05-03-feature-inventory.md) を参照してください。

正式な製品名は **Curated** です。リポジトリ名や npm パッケージ名には引き続き **`jav-shadcn`** が使われている場合があります。Go モジュール名は **`curated-backend`**、サーバーエントリポイントは **`backend/cmd/curated`** です。

## ハイライト

- **ローカルファースト** — Vue 3 SPA フロントエンド + Go HTTP API バックエンド + SQLite 永続化。
- **デュアルモード開発** — 実 API モード（フルバックエンド）と Mock モード（高速 UI 開発）を同一サービス層で利用可能。
- **包括的なライブラリ管理** — 仮想化ポスターグリッド、お気に入り、評価、タグ、俳優プロフィール、ゴミ箱/復元、ムービーノート、fsnotify ベースの自動スキャンとマルチルートライブラリパス。
- **ムービーインポート** — ドラッグ＆ドロップ、ファイル選択、フォルダ選択に対応し、進捗追跡と大容量ファイル向けレジューム可能チャンクアップロードを完備。
- **メタデータスクレイピング** — マルチプロバイダー対応、戦略設定（自動グローバル / 中国向け / カスタムチェーン / 指定）、プロバイダーヘルスチェック、ネットワーク診断向けの機械可読な障害カテゴリ。
- **再生** — HTML5 動画再生（Range ストリーミング）、レジューム再生、日次視聴統計、HLS セッション（remux/トランスコードパイプライン）、外部プレーヤー引き渡し、再生セッション診断。
- **ホームページ日次レコメンデーション** — UTC ベースの hero カルーセルとレコメンデーション行を SQLite に永続化し、クロスデバイスで一貫性を確保。重み付きサンプリング、クールダウン期間、出演者/スタジオ多様性バランシングを適用。
- **キュレートフレーム** — フレームキャプチャ、閲覧、タグ付け、フィルタリング、メタデータ埋め込みのマルチフォーマットエクスポート（JPG/WebP/PNG）。
- **俳優管理** — 俳優一覧、プロフィール詳細、ユーザータグ、外部リンク、同一オリジンアバターキャッシュ、非同期メタデータスクレイピング。
- **ゲームパッド操作** — Web Gamepad API による標準コントローラー（DualSense 含む）対応：グローバルフォーカス移動、ライブラリグリッド選択、プレイヤー操作。
- **Windows リリースパッケージング** — トレイモード起動、ローカルフロントエンド配信、Inno Setup インストーラー、ポータブル zip、FFmpeg バンドル、Windows ログイン自動起動、GitHub Releases ベースの更新チェックとインストーラー直接ダウンロード。
- **設定システム** — 包括的な設定 UI（概要、一般、ライブラリとストレージ、メタデータ、ネットワーク、キュレートフレーム、バージョン情報、メンテナンス）、ライブラリレベルの設定永続化、プロキシ設定、ログ管理。

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

- 大規模ライブラリ向け仮想化ポスターグリッド表示（URL ベースの選択状態）。
- 仮想化ポスターグリッドの標準ゲームパッド操作対応。
- お気に入り、評価（0-5）、ユーザータグ、メタデータタグ。
- マルチルートライブラリパス：追加、編集、削除、OS ファイルマネージャーで開く。
- ライブラリ整理（`organizeLibrary`）とゴミ箱/復元ワークフロー。
- ムービーノート/コメントの永続化。
- 俳優フィルター時の俳優プロフィールカード表示。
- キーワード、俳優、タグでの検索。

### スキャンとメタデータ

- 手動および自動スキャン（バックグラウンドタスク追跡）。
- fsnotify ベースのディレクトリ監視とデバウンス自動スキャン（`autoLibraryWatch`）。
- metatube-sdk-go によるメタデータスクレイピング（非同期タスク実行）。
- マルチプロバイダー対応、戦略設定：`auto-global`、`auto-cn-friendly`、`custom-chain`、`specified`。
- プロバイダーヘルスチェック（単一/全件）と障害カテゴリ。
- ムービースクレイピング成功時の俳優プロフィール自動補完（`autoActorProfileScrape`）。

### インポート

- トップバーからのムービーインポート：ドラッグ＆ドロップ、ファイル選択、フォルダ選択。
- ファイル単位の進捗追跡と失敗通知。
- 大容量ファイル向けレジューム可能チャンクアップロード（コミット/アボートライフサイクル）。
- 競合検出（既存のファイルは上書きされない）。
- デフォルトインポート先ライブラリパスの設定。

### 再生

- HTML5 動画再生（HTTP Range ストリーミング）。
- レジューム再生の永続化（Web API モードは SQLite、Mock モードは localStorage）。
- 再生ディスクリプタ抽象化：直接再生、remux、トランスコードパスを統一。
- HLS セッション対応（セッション診断、最近のセッション一覧）。
- 外部プレーヤー引き渡し（設定可能なブラウザプロトコルテンプレート、PotPlayer プリセット）。
- 日次視聴統計（設定 → 概要、91日間ウィンドウ）。
- プレイヤー統計オーバーレイ、タイムラインサムネイルプレビュー、キュレートフレームキャプチャ。
- ルートナビゲーションコンテキスト：タイムスタンプ（`?t=`）と戻り先（`?from=history`）。
- アクティブ再生時のサイドバー復帰。

### 俳優

- 俳優一覧：検索、タグフィルター、ソート、ページネーション。
- 俳優プロフィール詳細とメタデータ表示。
- ユーザータグ編集と外部リンク管理。
- 同一オリジンアバター配信（バックエンドキャッシュ）。
- 俳優メタデータの非同期スクレイピング。

### キュレートフレーム

- プレイヤーからのフレームキャプチャ。
- 閲覧：ページネーション、テキスト検索、タグ/俳優/ムービーでのフィルタリング。
- タグ編集とフレーム削除。
- 統計概要、タグ分類、俳優分類。
- JPG（EXIF）、WebP（EXIF）、PNG（iTXt）、ZIP 形式でのエクスポート（tags、schemaVersion、exportedAt、appName、appVersion メタデータ埋め込み）。
- エクスポート形式の設定（`curatedFrameExportFormat`）。

### ホームページとレコメンデーション

- UTC 日次レコメンデーションスナップショット（SQLite 永続化）。
- hero カルーセルとレコメンデーション行（クロスデバイス一貫性）。
- 非復元重み付きサンプリング（クールダウン期間と推薦回数減衰付き）。
- 出演者とスタジオの多様性バランシング。
- 強制リフレッシュ（hero 保持と現在の推薦除外に対応）。

### 設定

- 包括的な設定 UI：概要、一般、ライブラリとストレージ、メタデータ、ネットワーク、キュレートフレーム、バージョン情報、メンテナンス。
- `config/library-config.cfg` へのライブラリレベル設定永続化（アトミック書き込み）。
- プロキシ設定（JavBus および Google 接続テスト付き）。
- バックエンドログ：ディレクトリ、保持日数、レベルの設定。
- GitHub Releases ベースのアプリ更新チェック（サイドバーバッジとインストーラー直接ダウンロード）。
- Windows ログイン自動起動（`launchAtLogin`）。

### ゲームパッド操作

- Web Gamepad API による標準コントローラー対応（DualSense 含む）。
- グローバルフォーカス移動、ライブラリグリッド選択、プレイヤー操作。
- 大規模シークジャンプ、キュレートフレームキャプチャ、統計/操作レイヤー切り替え。
- ブラウザローカル設定トグル（localStorage 永続化）。

### パッケージングとリリース

- Windows リリースフロー：`pnpm release:publish`（Python CLI による統合）。
- トレイモード起動、`:8081` でのローカルフロントエンド配信。
- Inno Setup インストーラーとポータブル zip 配布。
- FFmpeg バンドルとリリースマニフェスト生成。
- パッケージ履歴台帳（`docs/ops/package-build-history.csv`）。

### 開発者体験

- デュアルモード開発：実 API モードと Mock モードの高速反復。
- フロントエンド：Vue 3 + TypeScript + Vite 8 + Tailwind CSS v4 + shadcn-vue。
- バックエンド：Go 1.25+ + SQLite (modernc) + Zap ロギング + クリーンアーキテクチャ。
- 国際化：English、简体中文、日本語（vue-i18n）。
- 開発用パフォーマンスモニターバー（dev ビルドのみ）。
- エラーバウンダリとクライアントリクエストタイムアウト。
- 全バックエンドドメインでの構造化エラーコード。

## 設定

実行時設定は、フロントエンド環境変数とバックエンド設定に分かれています。

### フロントエンド

- `VITE_USE_WEB_API=true`: 実バックエンドを使用
- `VITE_API_BASE_URL`: API ベース URL を上書き。未設定時、ローカル loopback の Web API 開発では大きなアップロードが Vite proxy を通らないよう開発 backend `:8080` へ直接接続し、release `:8081` の静的ホスティングやそれ以外のモードでは同一オリジンの `/api` を使用します
- `VITE_LOG_LEVEL`: ブラウザログレベルのデフォルト値

### バックエンド

バックエンドは JSON の主設定を読み込み、次のライブラリ設定ファイルをマージします。

- `config/library-config.cfg`

主なライブラリ設定項目:

- `organizeLibrary`
- `metadataMovieProvider`
- `metadataMovieStrategy`
- `defaultImportLibraryPathId`
- `autoLibraryWatch`
- `autoActorProfileScrape`
- `launchAtLogin`
- `curatedFrameExportFormat`（デフォルト `jpg`；指定可能：`jpg`、`webp`、`png`）
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
├── docs/                   # 概要は docs/README.md（reference / product / ops / plan 等）
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
- 現在のベースラインは `1.4.2` です。
- `pnpm release:*` は現在 `python scripts/release/release_cli.py` に統一されています。
- リリースフローでは、Windows 用ステージングディレクトリ、ポータブル zip、インストーラー実行ファイル、リリースマニフェストを生成します。
- パッケージ履歴台帳は `docs/ops/package-build-history.csv` に移行済みで、Excel / WPS 互換のため UTF-8 with BOM で保存されます。
- Windows リリースビルドはデフォルトでトレイモードになり、`frontend-dist/` が実行ファイルの横にあればフロントエンドをローカル配信できます。
- インストーラー自体は引き続き Inno Setup を使いますが、`.iss` テンプレートの描画と `ISCC.exe` 呼び出しは Python 側で行います。
- 設定画面から現在のユーザー向け Windows ログイン時起動を永続化できます。この自動起動はサイレントでトレイに入り、ブラウザは自動で開きません。

関連するリリース資料:

- [docs/plan/2026-03-31-production-packaging-and-config-strategy.md](docs/plan/2026-03-31-production-packaging-and-config-strategy.md)
- [docs/ops/package-build-history.csv](docs/ops/package-build-history.csv)
- [docs/ops/2026-04-02-package-build-history.md](docs/ops/2026-04-02-package-build-history.md)

## ドキュメント

- [API.md](API.md): 公開 HTTP API リファレンス
- [docs/features/2026-05-03-feature-inventory.md](docs/features/2026-05-03-feature-inventory.md): 全実装機能カタログ
- [docs/product/2026-03-20-jav-libary.md](docs/product/2026-03-20-jav-libary.md): 製品設計と目標アーキテクチャ
- [docs/reference/2026-03-20-project-memory.md](docs/reference/2026-03-20-project-memory.md): 実装事実と安定したプロジェクトメモリ
- [docs/reference/architecture-and-implementation.html](docs/reference/architecture-and-implementation.html): アーキテクチャ概要
- [docs/reference/2026-03-21-library-organize.md](docs/reference/2026-03-21-library-organize.md): ライブラリ整理メモ
- [docs/reference/2026-03-24-frontend-ui-spec.md](docs/reference/2026-03-24-frontend-ui-spec.md): フロントエンド UI 仕様

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
