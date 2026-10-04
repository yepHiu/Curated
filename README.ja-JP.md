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

「作品」と「FC2」は別々に閲覧でき、ストレージとタグを共有します。同じ品番の複数ファイルは一枚のポスターにまとまり、パート選択と個別の再生位置に対応します。[FC2 と複数ファイルの案内](docs/guide.md#fc2-and-multipart-movies)をご覧ください。

AI タグ整理は整理済み・未整理・要更新を集計し、問題のある作品はスキップして後で確認でき、既定では未整理作品のみ処理します。変更作品の更新、単体・選択範囲・全体の再整理と一般的な題材競合の自動修正に対応します。[操作ガイド](docs/guide.md#ai-user-tags-and-homepage-topics)を参照してください。

Server は暗い背景、Desktop はオフホワイト（`#F5F6F8`）の背景のアイコンを使用し、中央の図柄は共通です。[ブランドガイド](docs/reference/curated-brand-guidelines.md)と[アセット一覧](icon/brand/index.html)を参照してください。

Desktop の設定またはトレイからサーバー接続を保存・管理・切り替えできます。[接続ガイド](docs/guide.md#desktop-server-connections)を参照してください。

キャプチャのプレビュー・再試行・取り消し、ソースフレーム、GIF/MP4/WebM クリップに対応しています。[操作ガイド](docs/guide.md#curated-capture-and-inspection)を参照してください。

Curated はローカルファーストのメディアライブラリです。Vue 3 フロントエンド、Go + SQLite バックエンド、Electron デスクトップシェルで構成されます。正式な製品名は **Curated** です。リポジトリ名や npm パッケージ名には引き続き **`jav-shadcn`** が使われている場合があります。

この README は短い公開エントリです。セットアップ、設定、パッケージング、および `docs/` 内の各文書への案内は **[docs/guide.md](docs/guide.md)** を参照してください。HTTP API リファレンスは **[API.md](API.md)** です。

## ダウンロード

次期 Windows インストーラー（Server 1.7.4 / Desktop 0.2.2）のソースは、専用のインストール先ページ、バージョンと上書きの確認、実行中のみの終了確認に対応しています。[インストール手順](docs/guide.md#windows-installation-flow)を参照してください。公開リリースは別途行います。

Full/Server 1.7.3、Desktop 0.2.1 では Windows インストーラの簡体字中国語対応と更新前の自動終了を追加しました。Full はバックアップ検証後に旧一体型をアンインストールします。Windows/Mac CD 検証を通過し、公開済みです。対応範囲と Windows 検証は[移行ガイド](docs/guide.md#migrating-an-old-all-in-one-installation)を参照してください。

今後は **Desktop / Server の単体パッケージのみ**を **[GitHub Releases](https://github.com/yepHiu/Curated/releases)** で公開します。Release Notes には両モジュールの更新有無を記載し、変更のないモジュールのバージョンは据え置きます。

- コミック / 写真ライブラリ **Beta**：設定 → ライブラリ → 漫画ライブラリ / 写真ライブラリで個別に有効化すると設定を表示します。写真ライブラリではお気に入り、タグ追加、索引削除の一括操作ができます。[利用ガイド](docs/guide.md#comic-and-photo-library-beta)。
- **Windows インストーラ：** `Curated-Server-Setup-<version>-windows-x64.exe` と `Curated-Desktop-Setup-<version>-windows-x64.exe`。各単体 ZIP もあります。
- **Apple Silicon Desktop：** `Curated-Desktop-<version>-macos-arm64.dmg` / `.zip`。

新しい Full は作成しません。Release は **Curated YYYYMMDD**、同日中の追加分は **-2**、**-3** と命名し、更新のある Server／Desktop を同じ Release に含めます。各コンポーネントのバージョンと更新確認は独立します。最初の日付付き Release 公開後、Latest もこの方式に移行します。旧一体型の移行には [Full 1.7.3](https://github.com/yepHiu/Curated/releases/tag/full-v1.7.3) を使用します。

## ハイライト

- ブラウザープラグイン連携は「設定 → アクセスと接続 → ネットワークと端末」で有効にできます。認証情報は不要です。更新済みプラグインはウィッシュリスト状態を同期し、追加済みを緑色で表示します。追加元のページは保存され、詳細に JAVDB などのサイト名リンクで表示されます。連携をオフにしても既存のウィッシュリストは保持されます。[使い方](docs/guide.md#wishlist)。

- ローカルファーストの Vue 3 SPA、Go HTTP API、SQLite、Windows 向け Electron トレイシェル。
- macOS デスクトップでは、ネイティブのウインドウボタンを統合したタイトルバーを使用します。
- デュアルモード開発：実 Web API と Mock UI を同一サービス層で利用。
- コンテンツ右側の Agent パネルと AI 設定。文脈の自動整理と会話要約の永続保存に対応し、作品の記録を参照して表示前に検証します。ツール呼び出しは標準で無制限、任意の上限を設定でき、書き込みの確認待ちと保存済みを区別します。モデルのコンテキスト容量とプリセットを設定でき、権限、実測トークン、所要時間、監査も管理できます。[手引き](docs/guide.md#ai-settings-and-governance)を参照してください。
- ライブラリ閲覧、スクレイピング、インポート、再生、キュレートフレーム、俳優アイデンティティ、おすすめ、個人インサイト。
- 俳優の検索、残す人物情報の選択、統合後のプロフィール確認に対応。[俳優の統合](docs/guide.md#merging-duplicate-actors)を参照してください。
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

バックアップ、復元、パス移行、設定キー、リリース手順は [docs/guide.md](docs/guide.md) を参照してください。CD は Windows Server / Desktop の単体パッケージと Apple Silicon Mac Desktop をビルドし、バッチタグによる公開と手動のドラフト実行に対応しています。[リリース操作](docs/guide.md#8-release-and-packaging)を参照してください。

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
- リモートの保存先は読み取り専用です。設定は Server 側のパソコンで行い、アップロードには Server の保存先を使用します。[ガイド](docs/guide.md#remote-storage-directory-access)を参照。
- リモートでは端末の設定を利用できます。全体設定・セキュリティ・AI 管理・バックアップは Server 側で操作します。[設定ガイド](docs/guide.md#remote-settings-access)を参照。
- Server ログは既定フォルダーを使用し、同じ PC の Desktop から開けます。[ログ設定](docs/guide.md#server-logging)を参照。
- リモート接続時は Desktop の更新のみを提供し、Server のインストール操作は確認済みのローカル接続に限定します。Desktop は現在手動ダウンロード方式です。[更新ガイド](docs/guide.md#update-target-and-remote-connections)を参照。
- Server と Desktop の独立した 3 要素のバージョンを表示し、Server のビルドタイムスタンプも別途保持します。コンポーネント命名規則は実装済みで、分離インストーラーと移行は未実装です。[ガイド](docs/guide.md#desktop-version-and-component-release-planning)を参照。

## このプロジェクトを手伝ったモデル

後続の大規模言語モデルがこのリポジトリの作業を引き継ぐ場合は、以下に名前を追加してください。

- Composer 2.5
- DeepSeek V4
- GPT 5.5
- GPT 5.6 Terra sol
- Grok 4.6

Server identity and LAN discovery, Desktop device settings, upgrade compatibility and release packaging: [guide](docs/guide.md).
