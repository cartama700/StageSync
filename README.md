# StageSync

> **リズムゲーム バックエンド サーバエンジニアの日常業務を Go で再現したポートフォリオ**

[![CI](https://github.com/cartama700/StageSync/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/cartama700/StageSync/actions/workflows/ci.yml)
[![Go Version](https://img.shields.io/badge/go-1.26-00ADD8?logo=go)](./go.mod)
[![License](https://img.shields.io/badge/license-portfolio--only-lightgrey)](#ライセンス)

**言語 / 언어**: [日本語 (本ファイル)](./README.md) · [한국어](./README.ko.md)
**SSOT**: [`docs/MISSION.md`](./docs/MISSION.md) · [`docs/PLAN.md`](./docs/PLAN.md) · [`docs/STATUS.md`](./docs/STATUS.md) · [`docs/API.md`](./docs/API.md) · [`CHANGELOG.md`](./CHANGELOG.md)

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

### 実装済み (Phase 0-16 · v3 MVP スコープ)

| 領域 | 採用技術 |
|---|---|
| ルーティング | `chi` + HTTP/2 cleartext (h2c) — [ADR-0003](./docs/adr/0003-h2c-for-websocket-coexistence.md) |
| アーキテクチャ | handler → service → repository 3 層 + `Mount(r)` + consumer-defined interface |
| 設定 | `internal/config` で環境変数集約 + バリデーション |
| ランタイム | Graceful shutdown (`SIGTERM` → readiness drain → `srv.Shutdown`) · request timeout · request-scoped slog (`request_id` 伝搬) |
| DB (RDBMS) | Aurora MySQL (`sqlc` + `goose` + 원자 TX) · inmem ↔ MySQL 切替 — [ADR-0002](./docs/adr/0002-sqlc-over-orm.md) |
| DB (KV) | Redis ZSET (ランキング) · `REDIS_ADDR` graceful degrade (inmem fallback) |
| ゲームドメイン | プロフィール · ガチャ (10-roll TX + 天井 80 回) · イベント (時間 derived 状態 + 原子 UPSERT) · ランキング (ZSET + 本人 ±N) |
| バリデーション | `go-playground/validator/v10` + カスタムエラー型階層 (`apperror`) |
| エラー処理 | `fmt.Errorf("%w")` · `errors.Is` / `errors.As` · sentinel errors |
| 観測性 | Prometheus `/metrics` (Histogram + Gauge + Go runtime collector) · `/debug/pprof/*` · access log (`request_id`) |
| テスト | `testify/require` · table-driven · `t.Parallel()` · httptest E2E · `go-sqlmock` · `miniredis` · race detector |
| 静的解析 | `.golangci.yml` v2 (errcheck, staticcheck, revive, gocritic, bodyclose 等) |
| CI | GitHub Actions (test + lint + Docker build + benchmark) |
| デプロイ | Multi-target Dockerfile (distroless/static) + docker-compose 3 profile + K8s manifest ([`deploy/k8s/`](./deploy/k8s/)) + readiness gate |
| 負荷試験 | Locust cluster シナリオ ([`deploy/locust/`](./deploy/locust/)) + `cmd/bots` WebSocket (even/herd/cluster × N) |

### ボーナス軸 — リアルタイム通信

- `coder/websocket` + protobuf binary frame
- `sync.RWMutex` + map による thread-safe Room 管理
- AOI フィルタ + `sync.Pool` 最適化トグル (**約 2.5 倍高速化 · 0 allocs/op** — 実測は [`docs/BENCHMARKS.md`](./docs/BENCHMARKS.md))
- `cmd/bots` WebSocket 負荷シミュレータ (`even` / `herd` / `cluster` シナリオ · N 並列)

### 提出後に追加予定 (v0.7 障害シナリオラボ)

**Phase 19 — HP 同時減算デッドロックラボ**: `SELECT ... FOR UPDATE` による直列ロック でデッドロックを再現 → ユーザー別パーティションキューで直列化 → ベンチ比較 3 段。
面接期間中に追加予定 — "最近の更新" として面接でアピール。

### v3 で除外した Phase (提出スコープ外)

Phase 8 (メール) · Phase 10-11 (非同期バッチ · Write-Behind) · Phase 12 (Spanner) · Phase 15 (Terraform GKE) · Phase 17 (AI Ops LLM) · Phase 20-22 (その他の障害ラボ) — 理由は [`docs/PLAN.md`](./docs/PLAN.md) "スコープ再編記録" 参照。

---

## Phase 進捗 (v3 MVP スコープ)

| マイルストーン | Phase | 状態 |
|---|---|---|
| ボーナス軸 | 0, A, B | ✅ 3/3 |
| v0.1 基盤 | 1-4 | ✅ 4/4 |
| v0.2 ドメイン | 5, 6, 7 | ✅ 3/3 (ガチャ · イベント · ランキング) |
| v0.3 運用 lite | 9 | ✅ 1/1 (Histogram + pprof) |
| v0.5 デプロイ lite | 13, 14 | ✅ 2/2 (Docker profiles + K8s manifest) |
| v0.6 仕上げ | 16, 18 | ✅ 2/2 (Locust + README/ドキュメント完了) |
| v0.7 障害ラボ (提出後) | 19 | ⏳ 0/1 (面接期間中に追加予定) |

**総合: 15/15 MVP ✅ 完了** — 詳細ロードマップ + 除外 Phase の理由: [`docs/PLAN.md`](./docs/PLAN.md) · 現状スナップショット: [`docs/STATUS.md`](./docs/STATUS.md)

---

## クイックスタート

### 🐳 Docker Compose — 30 秒で REST + MySQL + Redis 起動 (**推奨**)

Docker さえ入っていれば OS 問わず即実行:

```bash
docker compose up --build           # server + mysql + redis
# 別端末で:
curl -X POST http://localhost:5050/api/profile \
     -H "Content-Type: application/json" \
     -d '{"id":"p1","name":"sekai"}'
curl http://localhost:5050/api/profile/p1
curl http://localhost:5050/metrics          # Prometheus scrape
```

**外部依存なし (inmem only)**:
```bash
docker compose --profile inmem up server-inmem --build
```

**負荷シミュレーション付き (server + mysql + redis + bots)**:
```bash
docker compose --profile load up --build    # + bots-cluster + bots-herd が自動で server に接続
curl http://localhost:5050/metrics | grep stagesync_room_connected_players
```

**Locust で REST 負荷試験 (別プロセス)**:
```bash
pip install locust
locust -f deploy/locust/locustfile.py --host http://localhost:5050 \
       --headless -u 500 -r 10 -t 1m --html=locust_report.html
```
詳細: [`deploy/locust/README.md`](./deploy/locust/README.md) · 結果テンプレート: [`docs/BENCHMARKS.md`](./docs/BENCHMARKS.md).

**Kubernetes デプロイ (manifest dry-run)**:
```bash
kubectl apply --dry-run=client -f deploy/k8s/
```
readiness gate · HPA · graceful drain 設定は [`deploy/k8s/README.md`](./deploy/k8s/README.md).

Makefile ショートカット: `make compose-up` / `make compose-inmem` / `make compose-load` / `make compose-down`.
環境変数は [`.env.example`](./.env.example) 参照 (`LISTEN_ADDR` · `LOG_LEVEL` · `SHUTDOWN_TIMEOUT` · `REQUEST_TIMEOUT` · `MYSQL_DSN` · `REDIS_ADDR`).

### ネイティブ実行 (ソース修正 · 開発者向け)

一括ツールチェーンセットアップ (macOS は bash, Windows は PowerShell):

```bash
./scripts/setup.sh     # macOS: Homebrew で Go · protoc · sqlc · goose · golangci-lint
./scripts/setup.ps1    # Windows: Chocolatey で同等セット
```

そのあと:

```bash
make run               # inmem モード
make run-mysql         # MySQL 接続 (make dev-up で事前に MySQL 起動)
```

### ボーナス軸 — WebSocket 負荷シミュレーション

```bash
make run                                    # 別端末
go run ./cmd/bots -n=50 -scenario=herd      # 50 botsが原点付近へ群れ集合
go run ./cmd/bots -n=100 -scenario=even     # 100 botsがマップ全体に均等分散
curl -X POST http://localhost:5050/api/optimize \
     -H "Content-Type: application/json" -d '{"on":true}'
curl http://localhost:5050/metrics | grep stagesync_
```

### テスト · 静的解析

```bash
make test              # go test -race ./...
make bench             # AOI ベンチマーク
```

API 詳細は [`docs/API.md`](./docs/API.md) を参照。

---

## ディレクトリ構造

```
StageSync/
├── cmd/
│   ├── server/          REST + WebSocket サーバ
│   └── bots/            WebSocket 負荷シミュレータ (even/herd/cluster)
├── api/proto/roompb/    protobuf スキーマ + 生成コード
├── internal/
│   ├── config/          環境変数ベース設定 + バリデーション
│   ├── domain/          純粋ドメインオブジェクト (profile, gacha, event, ranking)
│   ├── service/         ビジネスロジック (profile, gacha, event, ranking, aoi)
│   ├── persistence/
│   │   ├── inmem/       メモリ実装 (開発 · テスト · Redis fallback)
│   │   ├── mysql/       sqlc + goose + schema + queries
│   │   └── redis/       ランキング ZSET (miniredis テスト)
│   ├── endpoint/        HTTP ハンドラ + ミドルウェア (Mount パターン · Prometheus Histogram · pprof)
│   ├── apperror/        エラー型階層 + HTTP マッピング
│   ├── room/            WebSocket Room 状態 (ボーナス軸)
│   └── lifecycle/       ランタイムフラグ (最適化トグル · readiness gate)
├── docs/
│   ├── MISSION.md · PLAN.md · STATUS.md
│   ├── API.md · BENCHMARKS.md · CHANGELOG はルート
│   ├── adr/             Architecture Decision Records
│   └── demo/            デモ GIF · スクリーンショット
├── scripts/setup.sh     一括環境セットアップ
├── .github/workflows/   CI パイプライン (test + lint + docker build + bench)
├── deploy/
│   ├── k8s/             namespace · deployment · service · hpa · configmap
│   └── locust/          cluster シナリオ + README
├── Dockerfile           multi-target (server + bots, distroless/static)
├── docker-compose.yml   server + MySQL + Redis + bots (profile 別)
├── .env.example
├── CHANGELOG.md
├── Makefile · sqlc.yaml · .golangci.yml · go.mod
```

---

## 関連ドキュメント

- [**docs/MISSION.md**](./docs/MISSION.md) — プロジェクトミッション · 求人マッピング · 5 軸フレーム
- [**docs/PLAN.md**](./docs/PLAN.md) — Phase ロードマップ (v3) · 除外 Phase の理由 · 学習トラッカー
- [**docs/STATUS.md**](./docs/STATUS.md) — 現状スナップショット (PLAN ↔ リポジトリ対照)
- [**docs/API.md**](./docs/API.md) — エンドポイント仕様 (SSOT)
- [**docs/BENCHMARKS.md**](./docs/BENCHMARKS.md) — AOI + Locust 実測
- [**docs/adr/**](./docs/adr/) — 主要な技術選択の記録 (chi · sqlc · h2c)
- [**CHANGELOG.md**](./CHANGELOG.md) — 完了済み変更の履歴
- [**deploy/k8s/README.md**](./deploy/k8s/README.md) — K8s デプロイ手順 · readiness drain 挙動
- [**deploy/locust/README.md**](./deploy/locust/README.md) — 負荷試験の実行方法
- [**README.ko.md**](./README.ko.md) — 한국어 版

---

## ライセンス

個人ポートフォリオ用 · 商用再利用制限。
