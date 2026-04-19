# StageSync

> **リズムゲームのバックエンドサーバーの日常業務を Go で再現したポートフォリオ**

[![CI](https://github.com/cartama700/StageSync/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/cartama700/StageSync/actions/workflows/ci.yml)
[![Go Version](https://img.shields.io/badge/go-1.26-00ADD8?logo=go)](./go.mod)
[![License](https://img.shields.io/badge/license-portfolio--only-lightgrey)](#ライセンス)

**言語**: [日本語 (本ファイル)](./README.md) · [한국어](./README.ko.md)

---

## ✨ ハイライト

- **MVP (Phase 0-18) を 4 日間で完了** + 提出後に **セキュリティ · 障害対応の 4 件** を追加実装 (JWT Auth · Idempotency · Rate Limit · Phase 19 デッドロックラボ)
- **REST API 全 20 エンドポイント** (プロフィール · ガチャ · イベント · ランキング · バトル) + **WebSocket リアルタイム通信 (ボーナス課題)**
- **ガチャ確率エンジン** — **10 連ガチャのアトミックなトランザクション** + **80 連天井** · **10,000 サンプルで分布誤差 ±5% 以内を検証**
- **Redis ZSET ランキング** — `REDIS_ADDR` 未設定時はインメモリ動作へ**フォールバック (Fallback)**
- **セキュリティ 3 層** — JWT HS256 認証 + `Idempotency-Key` キャッシュ (Redis `SET NX`) + Token Bucket Rate Limit (per identity)
- **Phase 19 HP デッドロックラボ** — `SELECT ... FOR UPDATE` による行ロック競合を再現 → playerID 別単一ワーカーで Go レベル直列化 → `maxInFlight == 1` でテスト検証 + `cmd/battlebench` で実測 ([実測ガイド](./docs/BENCHMARKS.md#phase-19--hp-同時減算デッドロック-ラボ))
- **Prometheus Histogram** (method × chi RoutePattern × status) + `/debug/pprof/*`
- **K8s readiness gate** — `SIGTERM` 受信で `/health/ready` を 503 に切替 → drain 5 秒 → `srv.Shutdown()`
- **AOI 最適化** — Naive vs Pooled 比較で **約 2.48 倍の高速化 · 0 allocs/op**
- **テスト 238 PASS** · `go vet` · `golangci-lint v2` · `-race` 全て green
- **`docker compose up --build`** コマンド一つで MySQL + Redis + server が 30 秒で起動

---

## 🚀 クイックスタート

### Docker Compose で即起動 (推奨)

```bash
docker compose up --build           # server + mysql + redis
# 別ターミナルで実行:
curl -X POST http://localhost:5050/api/profile \
     -H "Content-Type: application/json" \
     -d '{"id":"p1","name":"sekai"}'
curl http://localhost:5050/api/profile/p1
curl http://localhost:5050/metrics          # Prometheus scrape
```

**実行プロファイル別**:

| コマンド | 起動コンポーネント | 用途 |
|---|---|---|
| `docker compose up --build` | server + MySQL + Redis | 通常レビュー用 (推奨) |
| `docker compose --profile inmem up server-inmem --build` | server (inmem only) | 外部依存なしでの動作確認 |
| `docker compose --profile load up --build` | + bots-cluster + bots-herd | 負荷シミュレーション込み |

Makefile ショートカット: `make compose-up` / `make compose-inmem` / `make compose-load` / `make compose-down`

### Locust 負荷試験 (別プロセス)

```bash
pip install locust
locust -f deploy/locust/locustfile.py --host http://localhost:5050 \
       --headless -u 500 -r 10 -t 1m --html=locust_report.html
```

詳細: [`deploy/locust/README.md`](./deploy/locust/README.md) · 結果テンプレート: [`docs/BENCHMARKS.md`](./docs/BENCHMARKS.md)

### Kubernetes マニフェスト検証 (実クラスタ不要)

```bash
kubectl apply --dry-run=client -f deploy/k8s/
```

readiness gate · HPA · graceful drain の挙動は [`deploy/k8s/README.md`](./deploy/k8s/README.md) に記載しています。

### ローカル環境での直接実行 (ソース修正 · 開発者向け)

ツールチェーンの一括セットアップ (macOS は bash, Windows は PowerShell):

```bash
./scripts/setup.sh     # macOS  — Homebrew で Go · protoc · sqlc · goose · golangci-lint をインストール
./scripts/setup.ps1    # Windows — Chocolatey で同等セットをインストール

make run               # inmem モードで起動
make run-mysql         # MySQL 接続モード (make dev-up で事前に MySQL を起動)
make test              # go test -race ./...
make bench             # AOI ベンチマーク実行
```

環境変数は [`.env.example`](./.env.example)、API 詳細は [`docs/API.md`](./docs/API.md) を参照してください。

---

## 概要

株式会社 Colorful Palette サーバサイドエンジニア求人 [BA-09-04a](https://hrmos.co/pages/colorfulpalette/jobs/BA-09-04a) を対象とし、実際の日常業務 (**REST API · DB 設計 · 非同期バッチ · 運用**) を想定して Go で実装したポートフォリオです。

**設計方針**: 求人票に明記された技術スタック (Aurora MySQL · Redis · Docker · GKE · Locust) を実運用レベルで稼働させることを最優先としています。また、メルカリやサイバーエージェントなど、**日本の Go コミュニティにおけるベストプラクティス** (`context.Context` の伝搬 · consumer-defined interface · `fmt.Errorf("%w")` によるエラーラッピング · テーブル駆動テスト) を全体に反映しています。

**参考にした実プロダクションのアーキテクチャ** (Project Sekai):

- リアルタイム通信: **Diarkis** (Go / GKE · TCP·UDP·RUDP) — 外部ミドルウェアのため本ポートフォリオのスコープ外
- REST レイヤ: **Spring Boot (Java) / Go** — **本ポートフォリオの主軸**
- データストア: Aurora MySQL + Spanner + Redis

REST API を主軸としつつ、追加要件 (ボーナス課題) として WebSocket を用いたリアルタイム通信も実装した二層構造となっています。

---

## 技術スタック

### 実装済み (v3 MVP · Phase 0-18 + 提出後追加 · Phase 19)

| 領域 | 採用技術 |
|---|---|
| ルーティング | `chi` + HTTP/2 cleartext (h2c) — [ADR-0003](./docs/adr/0003-h2c-for-websocket-coexistence.md) |
| アーキテクチャ | handler → service → repository 3 層 · `Mount(r)` パターン · consumer-defined interface |
| 設定管理 | `internal/config` で環境変数集約 + バリデーション |
| ランタイム | Graceful shutdown (`SIGTERM` → readiness drain → `srv.Shutdown`) · request timeout · request-scoped slog (`request_id` 伝搬) |
| DB (RDBMS) | Aurora MySQL · `sqlc` + `goose` + **アトミックなトランザクション** · inmem ↔ MySQL 切替 — [ADR-0002](./docs/adr/0002-sqlc-over-orm.md) |
| DB (KV) | Redis ZSET (ランキング) · `Idempotency-Key` キャッシュ · `REDIS_ADDR` 未設定時はインメモリへフォールバック |
| ゲームドメイン | プロフィール · ガチャ (10 連ガチャのアトミック処理 + 80 連天井) · イベント (時間経過に依存する状態管理 + アトミックな UPSERT) · ランキング (ZSET + 自身の前後 ±N 件) · **バトル (Phase 19 デッドロックラボ)** |
| **セキュリティ** | JWT HS256 認証 (`/api/auth/login` + `RequireAuth` ミドルウェア) + Token Bucket Rate Limit (per identity) + `Idempotency-Key` 中複 차단 |
| バリデーション | `go-playground/validator/v10` + カスタムエラー型階層 (`apperror`) |
| エラー処理 | `fmt.Errorf("%w")` · `errors.Is` / `errors.As` · Sentinel Errors (定義済みエラー) |
| **オブザーバビリティ** | Prometheus `/metrics` (Histogram + Gauge + Go runtime collector) · `/debug/pprof/*` · access log (`request_id`) |
| テスト | `testify/require` · テーブル駆動テスト · `t.Parallel()` · `httptest` E2E · `go-sqlmock` · `miniredis` · race detector · **238 PASS** |
| 静的解析 | `.golangci.yml` v2 (errcheck · staticcheck · revive · gocritic · bodyclose 等) |
| CI | GitHub Actions (test + lint + Docker build + benchmark) |
| デプロイ | Multi-target Dockerfile (distroless/static) + docker-compose 3 profile + K8s manifest ([`deploy/k8s/`](./deploy/k8s/)) + readiness gate |
| 負荷試験 | Locust cluster シナリオ ([`deploy/locust/`](./deploy/locust/)) + `cmd/bots` WebSocket (even/herd/cluster × N 並列) + `cmd/battlebench` (Phase 19 実測 CLI) |

### 追加実装 (ボーナス課題) — リアルタイム通信

- `coder/websocket` + protobuf バイナリフレーム通信
- `sync.RWMutex` + `map` による thread-safe な Room 管理
- AOI フィルタ + `sync.Pool` による最適化トグル (**約 2.48 倍の高速化 · 0 allocs/op** — 実測: [`docs/BENCHMARKS.md`](./docs/BENCHMARKS.md))
- `cmd/bots` による WebSocket 負荷シミュレータ (`even` / `herd` / `cluster` · N 並列)

### 追加実装 (TRADEOFFS 対応) — 提出後のセキュリティ · 障害対応強化

| 項目 | 実装内容 | 関連 PR |
|---|---|---|
| **JWT 認証ミドルウェア** | HS256 Issuer + Validator + ctx helpers · `/api/auth/login` · `RequireAuth` が `/api/gacha/*` を保護 · `AUTH_SECRET` 未設定時は pass-through (開発互換) | #5 |
| **Idempotency-Key** | `Idempotency-Key` ヘッダベースのキャッシュ · Redis `SET NX EX` (本番) ↔ インメモリ (開発) · GET/HEAD は pass-through | #6 |
| **Rate Limit** | Token Bucket per identity (auth player → XFF → RemoteAddr) · TTL sweep + `golang.org/x/time/rate` · 429 + `Retry-After` | #6 |
| **Phase 19 HP デッドロックラボ** | `SELECT ... FOR UPDATE` で行ロック競合を再現 (v1-naive) → playerID 別単一ワーカーで Go レベル直列化 (v2-queue) → `maxInFlight == 1` でテスト検証 · `cmd/battlebench` で実測 | #7 |

### 提出スコープ外 (v3 で除外)

Phase 8 (メール) · Phase 10-11 (非同期バッチ · Write-Behind) · Phase 12 (Cloud Spanner) · Phase 15 (Terraform GKE) · Phase 17 (AI Ops LLM) · Phase 20-22 (その他の障害検証) — 除外理由は [`docs/PLAN.md`](./docs/PLAN.md) の「スコープ再編記録」をご参照ください。
アーキテクチャの既知の限界と面接対応ロジックは [`docs/TRADEOFFS.md`](./docs/TRADEOFFS.md) にまとめています。

---

## Phase 進捗

| マイルストーン | Phase | 状態 |
|---|---|---|
| 追加要件 (リアルタイム) | 0, A, B | ✅ 3/3 (chi + h2c · WebSocket Room · AOI 最適化) |
| v0.1 基盤構築 | 1-4 | ✅ 4/4 (clean architecture · MySQL + sqlc · validation · test CI) |
| v0.2 ドメイン実装 | 5, 6, 7 | ✅ 3/3 (ガチャ · イベント · ランキング) |
| v0.3 運用系 lite | 9 | ✅ 1/1 (Histogram + pprof) |
| v0.5 デプロイ lite | 13, 14 | ✅ 2/2 (Docker profiles + K8s manifest) |
| v0.6 最終仕上げ | 16, 18 | ✅ 2/2 (Locust + ドキュメント完了) |
| **TRADEOFFS 対応 (提出後)** | — | ✅ **JWT Auth (#5) · Idempotency + Rate Limit (#6)** |
| **v0.7 障害検証 (提出後)** | 19 | ✅ **HP デッドロックラボ (#7) — v1-naive + v2-queue 完了** |

**総合: 15/15 MVP ✅ 完了 + 提出後強化 3 件 (認証 · レート制限 · デッドロックラボ) ✅**

詳細ロードマップ + 除外 Phase の理由: [`docs/PLAN.md`](./docs/PLAN.md) · 現状のスナップショット: [`docs/STATUS.md`](./docs/STATUS.md)

---

## ディレクトリ構造

```
StageSync/
├── cmd/
│   ├── server/                 REST + WebSocket サーバ
│   ├── bots/                   WebSocket 負荷シミュレータ (even/herd/cluster)
│   └── battlebench/            Phase 19 HP デッドロックラボ 実測 CLI
├── api/proto/roompb/           protobuf スキーマ + 生成コード
├── internal/
│   ├── auth/                   JWT HS256 Issuer + Validator + ctx helpers
│   ├── config/                 環境変数ベース設定 + バリデーション
│   ├── domain/                 純粋ドメインオブジェクト (profile · gacha · event · ranking · battle)
│   ├── service/                ビジネスロジック (+ aoi · battle)
│   ├── persistence/
│   │   ├── inmem/              メモリ実装 (開発 · テスト · Redis fallback)
│   │   ├── mysql/              sqlc + goose + schema + queries + battle repo
│   │   └── redis/              ランキング ZSET (miniredis テスト)
│   ├── idempotency/            Idempotency-Key キャッシュ (Redis SET NX / inmem)
│   ├── ratelimit/              Token Bucket per identity (golang.org/x/time/rate)
│   ├── endpoint/               HTTP ハンドラ + ミドルウェア (Mount · Auth · Idempotency · RateLimit · Histogram · pprof)
│   ├── apperror/               エラー型階層 + HTTP マッピング
│   ├── room/                   WebSocket Room 状態 (ボーナス課題)
│   └── lifecycle/              ランタイムフラグ (最適化トグル · readiness gate)
├── docs/
│   ├── MISSION.md / PLAN.md / STATUS.md           設計 · ロードマップ · 現状
│   ├── API.md / BENCHMARKS.md                     エンドポイント仕様 · 実測
│   ├── PITCH.md / SUBMISSION_CHECKLIST.md         面接ピッチ · 提出チェックリスト
│   ├── TRADEOFFS.md                               既知の限界と面接対応ロジック
│   ├── PORTFOLIO_SCENARIOS.md                     障害シナリオラボ (Phase 19)
│   ├── adr/                                       Architecture Decision Records
│   └── demo/                                      デモ GIF · スクリーンショット
├── deploy/
│   ├── k8s/                    namespace · configmap · deployment · service · hpa
│   └── locust/                 cluster シナリオ (3:2:1 タスク重み)
├── scripts/                    setup.sh (macOS) · setup.ps1 (Windows)
├── .github/workflows/          CI パイプライン (test + lint + docker build + bench)
├── Dockerfile                  multi-target (server + bots, distroless/static)
├── docker-compose.yml          server + MySQL + Redis + bots (profile 別)
├── .env.example · CHANGELOG.md
└── Makefile · sqlc.yaml · .golangci.yml · go.mod
```

---

## 作者について (応募にあたって)

> **まず初めに、率直な背景をお伝えしておきます。**

本プロジェクトは、長年かけて作り込まれた「熟練職人の代表作」というよりも、私自身の短期間でのキャッチアップ力を証明する **「成長の軌跡を示すポートフォリオ」** です。新しい言語を素早く習得したプロセスと、日本の開発コミュニティの文化・ベストプラクティスを深く尊重し、適応しようとする姿勢をお伝えすることを第一の目標としています。

| 項目 | 現状とアプローチ |
|---|---|
| 実務経験 | **C# / .NET での数年にわたるバックエンド経験** — Go 言語や Java による本格的な開発は今回が **初の挑戦** となります。 |
| 日本語能力 | **N2 ~ N3 レベル相当** — 求人要件である N1 に向けて現在も学習中です。(エージェント様経由で面談の機会をいただいており、本ポートフォリオが私の技術力と熱意の判断材料となれば幸いです) |
| 開発スタイル | **AI コーディング支援 (Claude Code 等) を積極活用** — 2026 年時点の最新の開発生産性と、未知の技術に対する圧倒的な学習速度を可視化する試みです。 |

**本ポートフォリオを通じて証明したいこと**:
新しい技術・環境への **「圧倒的な学習速度」**、表面的な実装にとどまらないアーキテクチャ設計における **「技術的な深さ」**、そして日本のカルチャー・業界標準への **「適応力とリスペクト」** — この 3 点です。

---

## 関連ドキュメント

| ドキュメント | 内容 |
|---|---|
| [`docs/MISSION.md`](./docs/MISSION.md) | プロジェクトミッション · 求人マッピング · 5 軸フレーム |
| [`docs/PLAN.md`](./docs/PLAN.md) | Phase ロードマップ (v3) · 除外 Phase の理由 · 学習トラッカー |
| [`docs/STATUS.md`](./docs/STATUS.md) | 現状スナップショット (PLAN ↔ リポジトリ対照) |
| [`docs/API.md`](./docs/API.md) | エンドポイント仕様 (SSOT, 19 エンドポイント) |
| [`docs/BENCHMARKS.md`](./docs/BENCHMARKS.md) | AOI 実測 + Locust 結果テンプレート |
| [`docs/adr/`](./docs/adr/) | 主要な技術選択の記録 (chi · sqlc · h2c) |
| [`docs/PITCH.md`](./docs/PITCH.md) | 面接ピッチスクリプト (30 秒 / 2 分 / 5 分) |
| [`docs/TRADEOFFS.md`](./docs/TRADEOFFS.md) | 既知の限界 · 意図的スコープアウト · 面接対応論理 |
| [`CHANGELOG.md`](./CHANGELOG.md) | v0.1 リリース変更履歴 |
| [`deploy/k8s/README.md`](./deploy/k8s/README.md) | K8s デプロイ手順 · readiness drain 挙動 |
| [`deploy/locust/README.md`](./deploy/locust/README.md) | 負荷試験の実行方法 |
| [`README.ko.md`](./README.ko.md) | 한국어 版 |

---

## ライセンス

個人ポートフォリオとしての用途に限ります。商用利用および再配布は制限させていただきます (要相談)。
