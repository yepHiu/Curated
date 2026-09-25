<p align="center">
  <img src="icon/curated-wordmark.png" alt="Curated" width="520" />
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

キャプチャのプレビュー・再試行・取り消し、ソースフレーム、GIF/MP4/WebM クリップに対応しています。[操作ガイド](docs/guide.md#curated-capture-and-inspection)を参照してください。

Curated はローカルファーストのメディアライブラリです。Vue 3 フロントエンド、Go + SQLite バックエンド、Electron デスクトップシェルで構成されます。正式な製品名は **Curated** です。リポジトリ名や npm パッケージ名には引き続き **`jav-shadcn`** が使われている場合があります。

この README は短い公開エントリです。セットアップ、設定、パッケージング、および `docs/` 内の各文書への案内は **[docs/guide.md](docs/guide.md)** を参照してください。HTTP API リファレンスは **[API.md](API.md)** です。

## ダウンロード

公式の Windows パッケージは **[GitHub Releases](https://github.com/yepHiu/Curated/releases)** にあります。[最新リリース](https://github.com/yepHiu/Curated/releases/latest) を使ってください。

- コミック / 写真ライブラリ **Beta**：設定 → ライブラリ → 漫画ライブラリ / 写真ライブラリで個別に有効化すると設定を表示します。写真ライブラリではお気に入り、タグ追加、索引削除の一括操作ができます。[利用ガイド](docs/guide.md#comic-and-photo-library-beta)。
- **インストーラ（推奨）：** `Curated-Setup-<version>.exe`
- **ポータブル：** `Curated-<version>-windows-x64.zip`

インストール済みのアプリは、設定 → システム → アプリ情報と更新から新しいインストーラを確認・ダウンロードできます。

## ハイライト

- ブラウザープラグイン連携は「設定 → アクセスと接続 → ネットワークと端末」で有効にできます。認証情報は不要です。追加元のページは保存され、詳細に JAVDB などのサイト名リンクで表示されます。連携をオフにしても既存のウィッシュリストは保持されます。[使い方](docs/guide.md#wishlist)。

- ローカルファーストの Vue 3 SPA、Go HTTP API、SQLite、Windows 向け Electron トレイシェル。
- デュアルモード開発：実 Web API と Mock UI を同一サービス層で利用。
- コンテンツ右側の Agent パネルと AI 設定。文脈の自動整理と会話要約の永続保存に対応し、作品の記録を参照して表示前に検証します。ツール呼び出しは標準で無制限、任意の上限を設定でき、書き込みの確認待ちと保存済みを区別します。モデルのコンテキスト容量とプリセットを設定でき、権限、実測トークン、所要時間、監査も管理できます。[手引き](docs/guide.md#ai-settings-and-governance)を参照してください。
- ライブラリ閲覧、スクレイピング、インポート、再生、キュレートフレーム、俳優アイデンティティ、おすすめ、個人インサイト。
- 独立した HLS セッション、再開位置の指定、回数制限付きエラー回復。詳しくは[再生ガイド](docs/guide.md#playback-recovery)をご覧ください。
- 任意の PIN ロック、検証可能なバックアップ、パス移行、ライブラリヘルス修復。
- GitHub Releases に基づく更新確認とインストーラダウンロード。配布物には FFmpeg とローカル `hls.js` を同梱。

## クイックスタート

要件：Vite 8 互換の現行 Node.js LTS、**pnpm**、Go `1.25.4+`。

```bash
cd backend && go run ./cmd/curated
```

```bash
pnpm install
pnpm dev
```

- バックエンド既定値：`http://127.0.0.1:8080`（loopback のみ）、ヘルス名 `curated-dev`。
- フロントエンド既定値：`http://localhost:5173`。
- ルート `.env` で `VITE_USE_WEB_API=true` を設定すると実 API、それ以外は Mock。
- デスクトップシェル：`pnpm dev:electron`。
- Windows 開発バイナリ：`pnpm backend:build:dev` → `backend/runtime/curated-dev.exe`。

バックアップ、復元、パス移行、設定キー、リリース手順は [docs/guide.md](docs/guide.md) を参照してください。

## ドキュメント

| 文書 | 内容 |
| --- | --- |
| [docs/guide.md](docs/guide.md) | 詳細ハンドブックと文書索引 |
| [API.md](API.md) | 公開 HTTP API リファレンス |
| [docs/features/2026-05-03-feature-inventory.md](docs/features/2026-05-03-feature-inventory.md) | 実装済み / 目標機能カタログ |
| [docs/README.md](docs/README.md) | `docs/` 配下の使い方 |

## リポジトリ構成

```text
src/        Vue SPA
backend/    Go モジュール curated-backend（`cmd/curated`）
electron/   デスクトップシェル MVP
config/     ライブラリ実行時設定（リポジトリルートに残す）
docs/       ハンドブック、規範、製品、運用、計画、PRD
icon/       ブランド原資料（wordmark / appicon / mark）
```

ソース、テスト、必要な素材、すべての文書とプロトタイプを Git に保持し、ローカル生成物は[リポジトリ管理方針](docs/guide.md#repository-tracking-policy)に従います。

ビルドサイズは通常の増加を許容し、急増を警告し、重大な上限超過のみ停止します。[レポートと基準](docs/guide.md#build-size-monitoring)を参照してください。

## 注意

- 現在の段階は Web 優先 + 最小 Electron シェルです。より深い IPC、mpv、広範なネイティブ橋渡しは今後の目標です。
- `docs/film-scanner/` は参考資料であり、本番モジュールツリーではありません。

## このプロジェクトを手伝ったモデル

後続の大規模言語モデルがこのリポジトリの作業を引き継ぐ場合は、以下に名前を追加してください。

- Composer 2.5
- DeepSeek V4
- GPT 5.5
- GPT 5.6 Terra sol
- Grok 4.6
