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

Curated は、Vue 3 フロントエンドと Go + SQLite バックエンドで構成されたローカルファーストのメディアライブラリアプリケーションです。現在のリポジトリでは、共有 Web UI と HTTP サービス境界を Electron デスクトップシェルで提供し、Windows 向け配布、トレイライフサイクル、メタデータスクレイピング、再生、キュレートフレーム、ゲームパッド操作、包括的な設定を実装しています。

全実装機能の一覧は [docs/features/2026-05-03-feature-inventory.md](docs/features/2026-05-03-feature-inventory.md) を参照してください。

正式な製品名は **Curated** です。リポジトリ名や npm パッケージ名には引き続き **`jav-shadcn`** が使われている場合があります。Go モジュール名は **`curated-backend`**、サーバーエントリポイントは **`backend/cmd/curated`** です。

## ハイライト

- **ローカルファースト** — Vue 3 SPA フロントエンド + Go HTTP API バックエンド + SQLite 永続化。
- **デュアルモード開発** — 実 API モード（フルバックエンド）と Mock モード（高速 UI 開発）を同一サービス層で利用可能。
- **包括的なライブラリ管理** — 仮想化ポスターグリッド、お気に入り、評価、タグ、俳優プロフィール、ゴミ箱/復元、ムービーノート、fsnotify ベースの自動スキャンとマルチルートライブラリパス。
- **再起動後も復旧可能なムービーインポート** — ドラッグ＆ドロップ、ファイル選択、フォルダ選択、進捗追跡に対応し、大容量ファイルのレジューム情報を SQLite に永続化してバックエンド再起動後も復旧します。
- **ストレージ存在確認** — Windows の外付けドライブ利用を優先し、設定済みライブラリルートのオフライン状態やボリューム変更を検出して、起動時アラート、通知センター、スキャン/インポートのブロック、手動再バインドを提供。
- **検証可能なバックアップ** — `VACUUM INTO` による一貫した SQLite スナップショット、任意のライブラリ設定、SHA-256 マニフェスト、SQLite 整合性検証、設定画面からの作成・検証・復元プリフライト、オフライン原子的復元、ロールバックコピー保持。
- **監査可能なパス移行** — ドライブ文字、マウントポイント、Windows→Unix プレフィックス変更向けのオフライン dry-run/apply CLI。パスセグメント一致、移行先/競合検査、自動検証バックアップ、単一トランザクション更新、ストレージ binding リセット、永続監査記録を提供します。
- **ライブラリヘルスと上限付き修復** — Settings → Maintenance から SQLite、ストレージルート、ソースファイル、アセット、メタデータ、孤立ユーザー状態、インポート一時領域を読み取り専用で診断します。finding ID は安定し、オフラインストレージを誤削除扱いしません。欠落/失敗メタデータの再試行は明示確認・タスク追跡・再起動中断の記録・件数上限付きで、孤立状態または厳密に限定された一時領域の削除は最新 finding の再検証と監査を必須とし、最終動画ファイルは削除しません。
- **保存済みビュー** — 未視聴/視聴途中/完了、ローカル評価、出演者/タグ/メーカー、解像度、検索、並び順、相対追加期間を再利用できる条件として保存・適用・改名・並べ替え・更新・削除できます。Web API は SQLite、Mock は専用 localStorage に保存し、選択中作品や再生位置は定義に含めません。
- **メタデータスクレイピング** — マルチプロバイダー対応、戦略設定（自動グローバル / 中国向け / カスタムチェーン / 指定）、プロバイダーヘルスチェック、ネットワーク診断向けの機械可読な障害カテゴリ。
- **再生** — HTML5 動画再生（Range ストリーミング）、レジューム再生、日次視聴統計、HLS セッション（remux/トランスコードパイプライン）、外部プレーヤー引き渡し、再生セッション診断。
- **説明可能なホームページレコメンデーション** — UTC ベースの hero と推薦スナップショットを SQLite に永続化し、各推薦に検証可能な理由を付与します。ローカルの明示的フィードバックとして「興味なし」、期限付きスヌーズ、出演者/スタジオ/タグの重み低下、即時取り消し、一括管理を提供し、単なる更新操作はフィードバックを作成しません。
- **パーソナル分析** — 遅延読み込みのローカル専用画面で、過去 30/90/365 日または全期間の視聴時間、視聴開始数、現在の完了数/率、ローカル評価、上限付きの canonical 出演者/スタジオ/タグ順位を表示します。Web API は SQLite 集計だけを返し、Mock は同じサービス境界内で計算します。分母がない値を誤解を招く `0%` として表示しません。
- **キュレートフレーム** — フレームキャプチャ、閲覧、タグ付け、フィルタリング、メタデータ埋め込みのマルチフォーマットエクスポート（JPG/WebP/PNG）。
- **俳優 ID 管理** — 俳優一覧、プロフィール詳細、ユーザータグ、外部リンク、同一オリジンアバターキャッシュ、非同期メタデータスクレイピングに加え、Unicode 正規化 alias と「読み取り専用プレビュー + 明示確認」によるトランザクション統合を提供し、関連付けと監査履歴を保持します。
- **ゲームパッド操作** — Web Gamepad API による標準コントローラー（DualSense 含む）対応：グローバルフォーカス移動、ライブラリグリッド選択、プレイヤー操作。
- **Windows リリースパッケージング** — インストール後の入口を Electron デスクトップアプリにし、Inno Setup インストーラー、ポータブル zip、FFmpeg バンドル、リリースマニフェスト、Windows ログイン自動起動、GitHub Releases ベースの更新チェックとインストーラー直接ダウンロードを備えています。
- **Electron デスクトップシェル** — リポジトリ内の Electron main process が Go HTTP バックエンドを起動または再利用し、開発時は Vite フロントエンドも起動または再利用します。Curated アイコンとトレイで既存の Web UI を読み込み、ウィンドウを閉じるとトレイに隠れます。業務 REST API は IPC に置き換えず、preload はネイティブディレクトリ選択だけを公開します。パッケージ版では `Curated.exe` が Electron シェルで、Go バックエンドは `resources/app/curated.exe` に配置されます。
- **設定システム** — 包括的な設定 UI（概要、一般、動画の保存先、メタデータ、ネットワーク、キュレートフレーム、バージョン情報、メンテナンス）、ライブラリレベルの設定永続化、プロキシ設定、ログ管理。

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

- HTTP アドレス: `127.0.0.1:8080`（ローカル loopback のみ）
- ヘルス名: `curated-dev`

Windows 向け開発補助コマンド:

```bash
pnpm backend:build:dev
```

このコマンドは `backend/runtime/curated-dev.exe` を生成します。

### バックアップ、検証、復元

Web API モードでは、Settings -> Maintenance からパッケージの作成と即時検証、既存パッケージの検証、復元プリフライトを実行できます。パスはバックエンド実行マシン上の絶対パスです。オンライン復元ボタンは意図的に提供せず、実際の復元はオフラインメンテナンス操作のままです。

メンテナンスコマンドは `backend/` から実行します。データベースパスをカスタム main config で指定している場合は `-config path/to/config.json` も渡してください。

```powershell
go run ./cmd/curated -maintenance backup-create -backup-path C:\Backups\curated.curated-backup
go run ./cmd/curated -maintenance backup-verify -backup-path C:\Backups\curated.curated-backup
go run ./cmd/curated -maintenance backup-preflight -backup-path C:\Backups\curated.curated-backup
```

作成と検証は現在のデータを置き換えません。復元はオフライン専用です。Curated を完全終了し、成功した preflight と警告を確認してから明示的に承認します。

```powershell
go run ./cmd/curated -maintenance backup-restore -backup-path C:\Backups\curated.curated-backup -confirm-restore
```

初期フォーマットには一貫した SQLite スナップショットと、存在する場合の `library-config.cfg` が含まれます。メディアソースとユーザーアセットは含まれません。manifest はサイズ、SHA-256、アプリ識別、スコープ、適用済み migration を記録します。検証は全ファイルと SQLite `quick_check` / `foreign_key_check` を確認し、復元は未知の将来 migration や容量不足を拒否して原子的に置換し、旧データベース/設定を `.pre-restore-*` として保持します。

### 保存済みパスを移行する

ドライブ文字、マウントポイント、ライブラリルートが変わった場合は、Curated を完全終了してから読み取り専用 plan を実行します。移行元と移行先は絶対 Windows、UNC、Unix パスである必要があります。照合はパスセグメント単位なので、`D:\Media` が `D:\Media2` に誤一致することはありません。

```powershell
go run ./cmd/curated -maintenance path-migrate-plan -path-from D:\Media -path-to E:\Media
```

apply には未使用のバックアップ先と明示的な確認が必要です。Curated は単一トランザクション更新を開始する前に、移行前パッケージを作成して検証します。

```powershell
go run ./cmd/curated -maintenance path-migrate-apply -path-from D:\Media -path-to E:\Media -backup-path D:\Backups\before-path-migration.curated-backup -confirm-path-migration
```

ホワイトリストは `library_paths.path`、`movies.location`、`scan_items.path`、`media_assets.local_path`、`actors.avatar_local_path`、`library_path_storage_bindings.root_path`、`app_update_status.downloaded_file_path` のみに限定され、自由文や URL は書き換えません。移行先競合、型不一致、欠落は apply をブロックします。現在の OS で移行先を検査できない意図的なクロスプラットフォーム移行では、`-allow-missing-paths` が明示的なリスク承認になります。旧プレフィックスのストレージ binding は削除され、次回起動時に新しいボリュームを再検出・再 binding します。成功時は SQLite 整合性検査を行い、`path_migration_audits` をパス変更と同じトランザクションで保存します。

### フロントエンドを起動する

```bash
pnpm install
pnpm dev
```

Vite の開発サーバーは通常 `http://localhost:5173` で起動します。

### Electron シェル MVP の起動

```powershell
pnpm dev:electron
```

このコマンドは `backend/runtime/curated-dev.exe` と `electron-dist/` をビルドし、Go バックエンドを `-mode http` で起動または再利用します。その後 `/api/health` を待ってから `http://127.0.0.1:5173` の Vite フロントエンドを起動または再利用し、Curated アイコン付きの BrowserWindow でその URL を開きます。Electron が起動する Vite には `VITE_USE_WEB_API=true` が渡され、API は Electron 管理のバックエンドへ向きます。パッケージ版ではインストールされた `Curated.exe` が Electron シェルで、同梱 Go バックエンドは `resources/app/curated.exe` にあり、Electron は `http://127.0.0.1:8081` でバックエンドがホストする静的 UI を読み込みます。ウィンドウを閉じるとトレイに隠れ、バックエンドと Web エントリは動作を続けます。トレイメニューから Curated の再表示、ブラウザでの Web UI 表示、Settings の表示、アプリの終了ができます。業務 API は引き続き HTTP を使い、preload は `window.javLibrary.pickDirectory()` だけを公開して既存のフォルダ選択フローから Electron のネイティブディレクトリダイアログを使えるようにします。

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
- セッション、ファイル、チャンク範囲を SQLite に永続化し、バックエンド再起動後に元のタスクを復元して中断したコミットを調整。
- 期限切れセッションと厳密に限定した孤立ステージングを監査付き janitor で清掃。対象ストレージがオフラインなら延期し、最終配置済みファイルは削除しません。
- 競合検出（既存のファイルは上書きされない）。
- デフォルトインポート先ライブラリパスの設定。
- デフォルトのインポート先ドライブがオフライン、またはバインド済みボリュームと一致しない場合は、ストレージ警告を出してインポートをブロック。

### 再生

- HTML5 動画再生（HTTP Range ストリーミング）。
- レジューム再生の永続化（Web API モードは SQLite、Mock モードは localStorage）。
- 再生ディスクリプタ抽象化：直接再生、remux、トランスコードパスを統一。
- HLS セッション対応（セッション診断、最近のセッション一覧）。ブラウザにネイティブ HLS がない場合は公式 `hls.js/light` をオンデマンドで読み込み、実行時 CDN は不要です。
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
- canonical 俳優 ID は NFKC・大小文字 folding・空白 folding を使用し、旧名はプロフィール、検索、ライブラリフィルター、スクレイピング、メタデータ取り込みで canonical actor に解決されます。
- 俳優詳細ページの統合ワークベンチは影響と profile 競合を読み取り専用で確認し、stale token を検証して単一トランザクションで適用します。監査は照会可能です。Web API は SQLite migrations `0035`/`0036`、Mock は `curated-actor-merges-v1` を使用します。

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
- 各推薦の真実な reason code をスナップショットに永続化し、フロントエンドは翻訳のみを担当します。
- 「興味なし」、1～365 日のスヌーズ、出演者/スタジオ/タグの上限付き重み低下、取り消し、ローカルフィードバック管理に対応します。
- 推薦の更新自体は負のフィードバックを作成しません。

### パーソナル分析

- 遅延読み込みの `/insights` 画面で、過去 30 日、90 日、365 日、全期間のローカル暦日範囲を選べます。
- 視聴時間、視聴開始作品数、現在の進捗が 90% に達した完了数/率、現在のローカル評価数と平均を、分母の定義とともに表示します。
- canonical 出演者、スタジオ、重複排除タグの順位は上限付き `full-per-entity` 帰属です。項目間の合計は仕様上 100% を超える場合があります。
- Web API はバックエンド集計のみを返し、Mock は `LibraryService` の内側でローカル視聴時間/進捗を読みます。分母がない指標は偽の `0%` ではなく未算出表示になります。

### 設定

- 包括的な設定 UI：概要、一般、動画の保存先、メタデータ、ネットワーク、キュレートフレーム、バージョン情報、メンテナンス。
- Web API モードのメンテナンス画面は、PIN で保護されたバックアップ作成、パッケージ検証、オフライン復元プリフライトを提供します。Mock モードでは実ファイルシステム操作を無効化します。
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
- インストール後の本番入口：`Curated.exe` は Electron デスクトップシェルです。release Go バックエンドは `resources/app/curated.exe` として同梱され、Electron が所有する場合は `-mode http` で起動します。
- トレイ常駐起動、loopback `127.0.0.1:8081` でのローカルフロントエンド配信。
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

`config/library-config.example.cfg` は、リリースパッケージで使用するバージョン管理済みの
サニタイズ済みサンプルです。端末固有の `library-config.cfg` は配布物にコピーされません。

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

開発ビルドとリリースビルドは、それぞれローカル専用の `127.0.0.1:8080` と `127.0.0.1:8081` を既定で使用します。スタンドアロンサーバーを LAN に公開する場合は、まず loopback モードでアプリ PIN を設定し、メインランタイム JSON 設定で非 loopback の `httpAddr` と `"lanEnabled": true` の両方を明示してください（例: `{"httpAddr":"0.0.0.0:8081","lanEnabled":true}`）。LAN の明示的な有効化または PIN 初期化がない場合、Curated は非 loopback リスナーの起動を拒否します。ブラウザー CORS は同一オリジン、loopback 開発 Origin、およびメイン設定の `corsAllowedOrigins` に正確に列挙した追加 Origin のみに制限され、任意の credentialed Origin は反映しません。無効な PIN 設定/解除が 5 回連続すると、`Retry-After` 付きの指数バックオフが適用されます。Curated はローカルと LAN で共通のグローバル PIN ロックを使用し、設定画面では現在のデバイス、IP、ブラウザー、最終利用時刻を確認して、個別または他のすべての永久信頼デバイスを確認付きで取り消せます。組み込みフロントエンドは引き続き同一オリジンの `/api` を使用します。

## API

Curated は、ライブラリ、再生、俳優 ID と統合、設定、ストレージ存在確認、キュレートフレーム向けの Go HTTP API を提供します。

完全なエンドポイント一覧は [API.md](API.md) を参照してください。

ライブラリストレージの存在確認は `/api/library/paths/storage-status` 配下のエンドポイントで、オフラインまたはボリューム不一致のライブラリパスを検出します。現在の実装は Windows を優先し、macOS と Linux は基本的なパスプローブを使う将来の対応対象です。

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
- 現在のベースラインは `1.4.7` です。
- `pnpm release:*` は現在 `python scripts/release/release_cli.py` に統一されています。
- `pnpm release:publish` は Vue フロントエンド、release Go バックエンド、Electron main process をビルドしてから成果物を組み立てます。
- リリースフローでは、Windows 用 Electron ステージングディレクトリ、ポータブル zip、インストーラー実行ファイル、リリースマニフェストを生成します。
- 組み立てられたアプリは Electron runtime を `release/Curated` にコピーし、`electron.exe` を `Curated.exe` にリネームし、`resources/app/package.json` を書き込み、`electron-dist/`、`frontend-dist/`、Go バックエンド `curated.exe` を `resources/app/` 配下に配置します。
- パッケージ履歴台帳は `docs/ops/package-build-history.csv` に移行済みで、Excel / WPS 互換のため UTF-8 with BOM で保存されます。
- インストーラー自体は引き続き Inno Setup を使いますが、`.iss` テンプレートの描画と `ISCC.exe` 呼び出しは Python 側で行います。ショートカットとインストール完了後の起動入口は `{app}\Curated.exe` を指します。
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
- Electron は現在 `electron/` 配下の最小デスクトップシェルとして存在し、トレイライフサイクル管理と狭いネイティブディレクトリ選択 preload ブリッジを備えています。より深い IPC ブリッジ、mpv/プロセス制御、広範なネイティブファイルブリッジ、コントローラーのハードウェア統合は今後の方向性です。
- `docs/film-scanner/` は主に参照資料とフィクスチャを保持しており、本番モジュール構成そのものではありません。

## Root Directory Policy

- `videos_test/` はローカルのテスト用フィクスチャディレクトリとしてリポジトリ直下に固定で残します。
- `config/` はライブラリ実行設定のためにリポジトリ直下に残し、`backend/internal/config` へ統合しません。
- `backend/runtime/` は開発時の実行生成物を置く許可済みディレクトリです。
- 新しいローカル専用の一時状態は `.workspace/` を優先します。
- Go ビルドキャッシュはリポジトリ内に作成しません。release スクリプトはバックエンド build cache にシステムの一時ディレクトリを使うようになりました。
