# StageSync

> **リズムゲーム バックエンド サーバエンジニアの日常業務を Go で再現したポートフォリオ**

**言語 / 언어**: [日本語 (本ファイル)](./README.md) · [한국어](./README.ko.md)
**SSOT**: [`docs/MISSION.md`](./docs/MISSION.md) · [`docs/PLAN.md`](./docs/PLAN.md)

---

## 概要

株式会社 Colorful Palette サーバサイドエンジニア求人 [BA-09-04a](https://hrmos.co/pages/colorfulpalette/jobs/BA-09-04a) を対象に、実際の日常業務 — **REST API · DB · 非同期バッチ · 運用** — を Go で実装したポートフォリオ。

求人明記スタック (**Aurora MySQL · Cloud Spanner · Redis · GKE · Terraform · Locust**) に完全整合させ、Mercari / CyberAgent グループ基準の日本 Go 業界慣行を反映。

**Project Sekai の実プロダクション構造を模倣**:
- リアルタイム通信: **Diarkis** (Go / GKE / TCP·UDP·RUDP) — 外部ミドルウェア
- REST レイヤ: **Spring Boot (Java/Go)** — 本ポートフォリオの **主軸**
- データストア: Aurora MySQL + Spanner + Redis

二層構造 (REST 主軸 + WebSocket ボーナス軸) で両方を実装。

---

## 作者について (正直に)

本プロジェクトは「職人の代表作」ではなく **「成長ポートフォリオ」** である。

- **Go · Java は今回が初の実務挑戦** (C#/.NET 実務経験数年)
- **日本語 N2 ~ N3** (求人は N1 必須 — エージェント経由でコンタクト成立、本ポートフォリオが決め手)
- **AI ツール (Claude Code) を積極活用** — 2026 年現在の開発生産性 · 学習速度を実証
- 証明したい点: **学習速度** · **技術的深さ** · **日本業界慣行への整合意志**

---

## 技術スタック

### 実装済み (Phase 0-4 + ボーナス軸)

| 領域 | 採用技術 |
|---|---|
| ルーティング | `chi` + HTTP/2 cleartext (h2c) |
| アーキテクチャ | handler → service → repository 3 層 + `Mount(r)` パターン |
| DB | Aurora MySQL (`sqlx` + `sqlc` + `goose`) · inmem ↔ MySQL 切替 |
| バリデーション | `go-playground/validator/v10` + カスタムエラー型階層 (`apperror`) |
| エラー処理 | `fmt.Errorf("%w")` · `errors.Is` / `errors.As` · sentinel errors |
| テスト | `testify/require` · table-driven · `t.Parallel()` · httptest E2E · race detector |
| 静的解析 | `.golangci.yml` (errcheck, staticcheck, revive, gocritic, bodyclose 等) |
| CI | GitHub Actions (test + lint + benchmark) |

### ボーナス軸 — リアルタイム通信

- `coder/websocket` + protobuf binary frame
- `sync.RWMutex` + map による thread-safe Room 管理
- AOI フィルタ + `sync.Pool` 最適化トグル (**1.5 倍高速化 · 0 allocs/op**)
- `cmd/bots` WebSocket 負荷シミュレータ

### 実装予定 (Phase 5-18)

ガチャ · イベント · ランキング · メール (ゲームドメイン API) → Prometheus · pprof · 非同期バッチ · Write-Behind → Cloud Spanner デュアルストア → Docker Compose · Kubernetes + HPA · Terraform GKE → Locust + k6 負荷テスト → AI Ops アシスタント

---

## Phase 進捗

| マイルストーン | Phase | 状態 |
|---|---|---|
| ボーナス軸 | 0, A, B | ✅ 3/3 |
| v0.1 基盤 | 1-4 | ✅ 4/4 |
| v0.2 ドメイン | 5-8 | ⏳ 0/4 |
| v0.3 運用 | 9-11 | ⏳ 0/3 |
| v0.4 データ | 12 | ⏳ 0/1 |
| v0.5 デプロイ | 13-15 | ⏳ 0/3 |
| v0.6 仕上げ | 16-18 | ⏳ 0/3 |

詳細: [`docs/PLAN.md`](./docs/PLAN.md)

---

## クイックスタート

### 一括環境セットアップ (macOS · idempotent)

```bash
./scripts/setup.sh
```

Homebrew さえあれば Go · protoc · sqlc · goose · golangci-lint · Colima · Docker までワンショットで準備。

### 基本動作 (Docker 不要 · inmem モード)

```bash
make run
curl -X POST http://localhost:5050/api/profile \
     -H "Content-Type: application/json" \
     -d '{"id":"p1","name":"sekai"}'
curl http://localhost:5050/api/profile/p1
```

### MySQL 実接続 (Docker 必要)

```bash
make dev-up          # Colima + MySQL をオンデマンド起動
make run-mysql       # サーバ起動 (goose が自動マイグレーション実行)
# ... 動作確認 ...
make dev-down        # 終了時に両方停止 (バッテリー節約)
```

### ボーナス軸 — WebSocket リアルタイム

```bash
make run
go run ./cmd/bots -player=p1 -tick=200
# 別ターミナル:
curl http://localhost:5050/api/metrics
curl -X POST http://localhost:5050/api/optimize \
     -H "Content-Type: application/json" \
     -d '{"on":true}'
```

### テスト · 静的解析

```bash
make test            # go test -race ./...
make bench           # AOI ベンチマーク
make                 # Makefile のヘルプ (デフォルトターゲット)
```

---

## ディレクトリ構造

```
StageSync/
├── cmd/
│   ├── server/          REST + WebSocket サーバ
│   └── bots/            WebSocket 負荷シミュレータ
├── api/proto/roompb/    protobuf スキーマ + 生成コード
├── internal/
│   ├── domain/          純粋ドメインオブジェクト
│   ├── service/         ビジネスロジック (profile, aoi)
│   ├── persistence/
│   │   ├── inmem/       メモリ実装 (開発 · テスト)
│   │   └── mysql/       sqlc + goose + schema + queries
│   ├── endpoint/        HTTP ハンドラ (Mount パターン)
│   ├── apperror/        エラー型階層 + HTTP マッピング
│   ├── room/            WebSocket Room 状態 (ボーナス軸)
│   └── lifecycle/       ランタイムフラグ
├── docs/                ミッション · プラン · アーカイブ
├── scripts/setup.sh     一括環境セットアップ
├── .github/workflows/   CI パイプライン
├── Makefile
├── sqlc.yaml
├── .golangci.yml
└── go.mod
```

---

## 関連ドキュメント

- [**docs/MISSION.md**](./docs/MISSION.md) — プロジェクトミッション · 求人マッピング · 5 軸フレーム
- [**docs/PLAN.md**](./docs/PLAN.md) — Phase 0-18 ロードマップ · 依存関係 · 学習トラッカー
- [**README.ko.md**](./README.ko.md) — 한국어 版

---

## ライセンス

個人ポートフォリオ用 · 商用再利用制限。
