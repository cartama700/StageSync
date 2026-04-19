# StageSync

> **리듬게임 백엔드 서버 엔지니어의 일상 업무를 Go 로 재현한 포트폴리오**

[![CI](https://github.com/cartama700/StageSync/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/cartama700/StageSync/actions/workflows/ci.yml)
[![Go Version](https://img.shields.io/badge/go-1.26-00ADD8?logo=go)](./go.mod)
[![License](https://img.shields.io/badge/license-portfolio--only-lightgrey)](#라이선스)

**언어 / 言語**: [한국어 (현재 파일)](./README.ko.md) · [日本語](./README.md)
**SSOT**: [`docs/MISSION.md`](./docs/MISSION.md) · [`docs/PLAN.md`](./docs/PLAN.md) · [`docs/STATUS.md`](./docs/STATUS.md) · [`docs/API.md`](./docs/API.md) · [`CHANGELOG.md`](./CHANGELOG.md)

---

## ✨ 하이라이트

- **MVP (Phase 0-18) 을 4 일간 완료** + 제출 후 **보안 · 장애 대응 4 건** 추가 구현 (JWT Auth · Idempotency · Rate Limit · Phase 19 데드락 랩)
- **REST API 총 20 개 엔드포인트** (프로필 · 가챠 · 이벤트 · 랭킹 · 배틀) + **WebSocket 실시간 통신 (보너스축)**
- **가챠 확률 엔진** — **10 연 가챠 원자 트랜잭션** + **80 연 천장** · **10,000 샘플 분포 오차 ±5% 이내 검증**
- **Redis ZSET 랭킹** — `REDIS_ADDR` 미설정 시 인메모리로 **graceful fallback**
- **보안 3 층** — JWT HS256 인증 + `Idempotency-Key` 캐시 (Redis `SET NX`) + Token Bucket Rate Limit (per identity)
- **Phase 19 HP 데드락 랩** — `SELECT ... FOR UPDATE` 행 락 경합 재현 → playerID 별 단일 워커로 Go 레벨 직렬화 → `maxInFlight == 1` 테스트 증명 + `cmd/battlebench` 실측 ([실측 가이드](./docs/BENCHMARKS.md#phase-19--hp-同時減算デッドロック-ラボ))
- **Prometheus Histogram** (method × chi RoutePattern × status) + `/debug/pprof/*`
- **K8s readiness gate** — `SIGTERM` 수신 시 `/health/ready` 가 503 → drain 5 초 → `srv.Shutdown()`
- **AOI 최적화** — Naive vs Pooled **약 2.48× 고속화 · 0 allocs/op**
- **테스트 238 PASS** · `go vet` · `golangci-lint v2` · `-race` 모두 green
- **`docker compose up --build`** 한 줄로 MySQL + Redis + server 가 30 초 내 기동

---

## 🚀 퀵스타트

### Docker Compose 로 즉시 기동 (권장)

```bash
docker compose up --build           # server + mysql + redis
# 다른 터미널:
curl -X POST http://localhost:5050/api/profile \
     -H "Content-Type: application/json" \
     -d '{"id":"p1","name":"sekai"}'
curl http://localhost:5050/api/profile/p1
curl http://localhost:5050/metrics          # Prometheus scrape
```

**프로파일별**:

| 명령 | 기동 컴포넌트 | 용도 |
|---|---|---|
| `docker compose up --build` | server + MySQL + Redis | 일반 리뷰용 (권장) |
| `docker compose --profile inmem up server-inmem --build` | server (inmem only) | 외부 의존 없이 동작 확인 |
| `docker compose --profile load up --build` | + bots-cluster + bots-herd | 부하 시뮬 포함 |

Makefile 단축: `make compose-up` / `make compose-inmem` / `make compose-load` / `make compose-down`.

### Locust 부하 테스트 (별 프로세스)

```bash
pip install locust
locust -f deploy/locust/locustfile.py --host http://localhost:5050 \
       --headless -u 500 -r 10 -t 1m --html=locust_report.html
```

상세: [`deploy/locust/README.md`](./deploy/locust/README.md) · 결과 템플릿: [`docs/BENCHMARKS.md`](./docs/BENCHMARKS.md)

### Phase 19 데드락 랩 실측 (MySQL 필요)

```bash
# MySQL 기동 + 서버 1 회 실행 (goose 마이그레이션 자동)
docker compose up mysql -d
make run-mysql

# v1-naive (FOR UPDATE) — 락 경합 재현
make battle-bench-naive

# v2-queue (Go 레벨 직렬화) — lock wait 에러 0 기대
make battle-bench-queue
```

### Kubernetes 매니페스트 검증 (실 클러스터 불필요)

```bash
kubectl apply --dry-run=client -f deploy/k8s/
```

readiness gate · HPA · graceful drain 설정: [`deploy/k8s/README.md`](./deploy/k8s/README.md)

### 네이티브 실행 (소스 수정 · 개발자용)

툴체인 일괄 설치 (macOS 는 bash, Windows 는 PowerShell):

```bash
./scripts/setup.sh     # macOS  — Homebrew 로 Go · protoc · sqlc · goose · golangci-lint
./scripts/setup.ps1    # Windows — Chocolatey 로 동등한 셋

make run               # inmem 모드
make run-mysql         # MySQL 접속 (make dev-up 으로 MySQL 선 기동)
make test              # go test -race ./...
make bench             # AOI 벤치마크
```

환경변수는 [`.env.example`](./.env.example), API 상세는 [`docs/API.md`](./docs/API.md) 참조.

---

## 개요

株式会社 Colorful Palette 서버사이드 엔지니어 공고 [BA-09-04a](https://hrmos.co/pages/colorfulpalette/jobs/BA-09-04a) 를 타겟으로, 실제 일상 업무 (**REST API · DB 설계 · 비동기 배치 · 운영**) 를 상정하여 Go 로 구현한 포트폴리오.

**설계 방침**: 공고에 명시된 기술 스택 (Aurora MySQL · Redis · Docker · GKE · Locust) 을 실운용 레벨로 동작시키는 것을 최우선. Mercari / CyberAgent 등 **일본 Go 커뮤니티의 베스트 프랙티스** (`context.Context` 전파 · consumer-defined interface · `fmt.Errorf("%w")` 에러 래핑 · 테이블 주도 테스트) 를 전체에 반영.

**참고한 실 프로덕션 아키텍처** (Project Sekai):

- 실시간 통신: **Diarkis** (Go / GKE · TCP·UDP·RUDP) — 외부 미들웨어라 본 포트폴리오 스코프 밖
- REST 레이어: **Spring Boot (Java) / Go** — **본 포트폴리오의 주축**
- 데이터 스토어: Aurora MySQL + Spanner + Redis

REST API 를 주축으로, 추가 요건 (보너스 과제) 으로 WebSocket 실시간 통신도 구현한 2 층 구조.

---

## 기술 스택

### 구현 완료 (v3 MVP · Phase 0-18 + 제출후 추가 · Phase 19)

| 영역 | 채택 기술 |
|---|---|
| 라우팅 | `chi` + HTTP/2 cleartext (h2c) — [ADR-0003](./docs/adr/0003-h2c-for-websocket-coexistence.md) |
| 아키텍처 | handler → service → repository 3 층 · `Mount(r)` 패턴 · consumer-defined interface |
| 설정 관리 | `internal/config` 로 환경변수 집약 + 검증 |
| 런타임 | Graceful shutdown (`SIGTERM` → readiness drain → `srv.Shutdown`) · request timeout · request-scoped slog (`request_id` 전파) |
| DB (RDBMS) | Aurora MySQL · `sqlc` + `goose` + **원자 트랜잭션** · inmem ↔ MySQL 전환 — [ADR-0002](./docs/adr/0002-sqlc-over-orm.md) |
| DB (KV) | Redis ZSET (랭킹) · `Idempotency-Key` 캐시 · `REDIS_ADDR` 미설정 시 인메모리 fallback |
| 게임 도메인 | 프로필 · 가챠 (10 연 원자 처리 + 80 연 천장) · 이벤트 (시간 경과 derived 상태 + 원자 UPSERT) · 랭킹 (ZSET + 본인 ±N) · **배틀 (Phase 19 데드락 랩)** |
| **보안** | JWT HS256 인증 (`/api/auth/login` + `RequireAuth` 미들웨어) + Token Bucket Rate Limit (per identity) + `Idempotency-Key` 중복 차단 |
| 검증 | `go-playground/validator/v10` + 커스텀 에러 타입 계층 (`apperror`) |
| 에러 처리 | `fmt.Errorf("%w")` · `errors.Is` / `errors.As` · Sentinel Errors |
| **관측성** | Prometheus `/metrics` (Histogram + Gauge + Go runtime collector) · `/debug/pprof/*` · access log (`request_id`) |
| 테스트 | `testify/require` · 테이블 주도 테스트 · `t.Parallel()` · `httptest` E2E · `go-sqlmock` · `miniredis` · race detector · **238 PASS** |
| 정적 분석 | `.golangci.yml` v2 (errcheck · staticcheck · revive · gocritic · bodyclose 등) |
| CI | GitHub Actions (test + lint + Docker build + benchmark) |
| 배포 | Multi-target Dockerfile (distroless/static) + docker-compose 3 profile + K8s manifest ([`deploy/k8s/`](./deploy/k8s/)) + readiness gate |
| 부하 테스트 | Locust cluster 시나리오 ([`deploy/locust/`](./deploy/locust/)) + `cmd/bots` WebSocket (even/herd/cluster × N) + `cmd/battlebench` (Phase 19 실측 CLI) |

### 추가 구현 (보너스 과제) — 실시간 통신

- `coder/websocket` + protobuf 바이너리 프레임 통신
- `sync.RWMutex` + `map` 기반 thread-safe Room 관리
- AOI 필터 + `sync.Pool` 최적화 토글 (**약 2.48 배 고속화 · 0 allocs/op** — 실측: [`docs/BENCHMARKS.md`](./docs/BENCHMARKS.md))
- `cmd/bots` WebSocket 부하 시뮬레이터 (`even` / `herd` / `cluster` · N 병렬)

### 추가 구현 (TRADEOFFS 대응) — 제출 후 보안 · 장애 대응 강화

| 항목 | 구현 내용 | 관련 PR |
|---|---|---|
| **JWT 인증 미들웨어** | HS256 Issuer + Validator + ctx helpers · `/api/auth/login` · `RequireAuth` 가 `/api/gacha/*` 보호 · `AUTH_SECRET` 미설정 시 pass-through (개발 호환) | #5 |
| **Idempotency-Key** | `Idempotency-Key` 헤더 기반 캐시 · Redis `SET NX EX` (프로덕션) ↔ 인메모리 (개발) · GET/HEAD pass-through | #6 |
| **Rate Limit** | Token Bucket per identity (auth player → XFF → RemoteAddr) · TTL sweep + `golang.org/x/time/rate` · 429 + `Retry-After` | #6 |
| **Phase 19 HP 데드락 랩** | `SELECT ... FOR UPDATE` 로 행 락 경합 재현 (v1-naive) → playerID 별 단일 워커로 Go 레벨 직렬화 (v2-queue) → `maxInFlight == 1` 테스트 증명 · `cmd/battlebench` 로 실측 | #7 |

### 제출 스코프 밖 (v3 에서 제외)

Phase 8 (메일) · Phase 10-11 (비동기 배치 · Write-Behind) · Phase 12 (Cloud Spanner) · Phase 15 (Terraform GKE) · Phase 17 (AI Ops LLM) · Phase 20-22 (기타 장애 검증) — 제외 이유는 [`docs/PLAN.md`](./docs/PLAN.md) "스코프 재편 기록" 참조.
아키텍처의 기존 한계 + 면접 대응 로직은 [`docs/TRADEOFFS.md`](./docs/TRADEOFFS.md) 에 정리.

---

## Phase 진행 현황

| 마일스톤 | Phase | 상태 |
|---|---|---|
| 보너스축 | 0, A, B | ✅ 3/3 (chi + h2c · WebSocket Room · AOI 최적화) |
| v0.1 기반 | 1-4 | ✅ 4/4 (clean architecture · MySQL + sqlc · validation · test CI) |
| v0.2 도메인 | 5, 6, 7 | ✅ 3/3 (가챠 · 이벤트 · 랭킹) |
| v0.3 운영 lite | 9 | ✅ 1/1 (Histogram + pprof) |
| v0.5 배포 lite | 13, 14 | ✅ 2/2 (Docker profiles + K8s manifest) |
| v0.6 마감 | 16, 18 | ✅ 2/2 (Locust + 문서 완료) |
| **TRADEOFFS 대응 (제출 후)** | — | ✅ **JWT Auth (#5) · Idempotency + Rate Limit (#6)** |
| **v0.7 장애 검증 (제출 후)** | 19 | ✅ **HP 데드락 랩 (#7) — v1-naive + v2-queue 완료** |

**총합: 15/15 MVP ✅ 완료 + 제출후 강화 3 건 (인증 · 레이트 리밋 · 데드락 랩) ✅**

상세 로드맵 + 제외 Phase 이유: [`docs/PLAN.md`](./docs/PLAN.md) · 현재 스냅샷: [`docs/STATUS.md`](./docs/STATUS.md)

---

## 디렉토리 구조

```
StageSync/
├── cmd/
│   ├── server/                 REST + WebSocket 서버
│   ├── bots/                   WebSocket 부하 시뮬 (even/herd/cluster)
│   └── battlebench/            Phase 19 HP 데드락 랩 실측 CLI
├── api/proto/roompb/           protobuf 스키마 + 생성 코드
├── internal/
│   ├── auth/                   JWT HS256 Issuer + Validator + ctx helpers
│   ├── config/                 환경변수 기반 설정 + 검증
│   ├── domain/                 순수 도메인 객체 (profile · gacha · event · ranking · battle)
│   ├── service/                비즈니스 로직 (+ aoi · battle)
│   ├── persistence/
│   │   ├── inmem/              메모리 구현 (개발 · 테스트 · Redis fallback)
│   │   ├── mysql/              sqlc + goose + schema + queries + battle repo
│   │   └── redis/              랭킹 ZSET (miniredis 테스트)
│   ├── idempotency/            Idempotency-Key 캐시 (Redis SET NX / inmem)
│   ├── ratelimit/              Token Bucket per identity (golang.org/x/time/rate)
│   ├── endpoint/               HTTP 핸들러 + 미들웨어 (Mount · Auth · Idempotency · RateLimit · Histogram · pprof)
│   ├── apperror/               에러 타입 계층 + HTTP 매핑
│   ├── room/                   WebSocket Room 상태 (보너스축)
│   └── lifecycle/              런타임 플래그 (최적화 토글 · readiness gate)
├── docs/
│   ├── MISSION.md / PLAN.md / STATUS.md           설계 · 로드맵 · 현황
│   ├── API.md / BENCHMARKS.md                     엔드포인트 사양 · 실측
│   ├── PITCH.md / SUBMISSION_CHECKLIST.md         면접 피치 · 제출 체크리스트
│   ├── TRADEOFFS.md                               기존 한계와 면접 대응 논리
│   ├── PORTFOLIO_SCENARIOS.md                     장애 시나리오 랩 (Phase 19)
│   ├── adr/                                       Architecture Decision Records
│   └── demo/                                      데모 GIF · 스크린샷
├── deploy/
│   ├── k8s/                    namespace · configmap · deployment · service · hpa
│   └── locust/                 cluster 시나리오 (3:2:1 task 가중치)
├── scripts/                    setup.sh (macOS) · setup.ps1 (Windows)
├── .github/workflows/          CI 파이프라인 (test + lint + docker build + bench)
├── Dockerfile                  multi-target (server + bots, distroless/static)
├── docker-compose.yml          server + MySQL + Redis + bots (profile 별)
├── .env.example · CHANGELOG.md
└── Makefile · sqlc.yaml · .golangci.yml · go.mod
```

---

## 작성자에 대해 (응모에 있어서)

> **먼저, 솔직한 배경을 말씀드립니다.**

본 프로젝트는 오랜 시간 다듬어진 「장인의 대표작」이 아니라, 저 자신의 단기간 캐치업 역량을 증명하는 **「성장의 궤적을 보여주는 포트폴리오」** 입니다. 새로운 언어를 빠르게 습득한 프로세스와 일본 개발 커뮤니티의 문화 · 베스트 프랙티스를 깊이 존중하며 적응하려는 자세를 전달하는 것을 제 1 목표로 하고 있습니다.

| 항목 | 현황과 접근 |
|---|---|
| 실무 경험 | **C# / .NET 수 년의 백엔드 경험** — Go · Java 에 의한 본격 개발은 이번이 **첫 도전**. |
| 일본어 능력 | **N2 ~ N3 수준** — 공고 요건인 N1 을 향해 현재도 학습 중. (에이전트 경유로 면담 기회를 얻었고, 본 포트폴리오가 제 기술력과 열의의 판단 재료가 되었으면 합니다.) |
| 개발 스타일 | **AI 코딩 지원 (Claude Code 등) 적극 활용** — 2026 년 시점의 최신 개발 생산성과, 미지의 기술에 대한 압도적 학습 속도를 가시화하는 시도. |

**본 포트폴리오를 통해 증명하고 싶은 것**:
새로운 기술 · 환경에 대한 **「압도적인 학습 속도」**, 표면적 구현에 그치지 않는 아키텍처 설계의 **「기술적 깊이」**, 그리고 일본 문화 · 업계 표준에 대한 **「적응력과 리스펙트」** — 이 3 점입니다.

---

## 관련 문서

| 문서 | 내용 |
|---|---|
| [`docs/MISSION.md`](./docs/MISSION.md) | 프로젝트 미션 · 공고 매핑 · 5 축 프레임 |
| [`docs/PLAN.md`](./docs/PLAN.md) | Phase 로드맵 (v3) · 제외 Phase 이유 · 학습 트래커 |
| [`docs/STATUS.md`](./docs/STATUS.md) | 현재 스냅샷 (PLAN ↔ 리포지토리 대조) |
| [`docs/API.md`](./docs/API.md) | 엔드포인트 사양 (SSOT, 20 엔드포인트) |
| [`docs/BENCHMARKS.md`](./docs/BENCHMARKS.md) | AOI + Locust + Phase 19 실측 |
| [`docs/adr/`](./docs/adr/) | 주요 기술 선택 기록 (chi · sqlc · h2c) |
| [`docs/PITCH.md`](./docs/PITCH.md) | 면접 피치 스크립트 (30 초 / 2 분 / 5 분) |
| [`docs/TRADEOFFS.md`](./docs/TRADEOFFS.md) | 기존 한계 · 의도적 scope-out · 면접 대응 로직 |
| [`CHANGELOG.md`](./CHANGELOG.md) | v0.1 릴리즈 변경 이력 |
| [`deploy/k8s/README.md`](./deploy/k8s/README.md) | K8s 배포 절차 · readiness drain 동작 |
| [`deploy/locust/README.md`](./deploy/locust/README.md) | 부하 테스트 실행 방법 |
| [`README.md`](./README.md) | 日本語 版 |

---

## 라이선스

개인 포트폴리오 용도. 상업 이용 · 재배포는 제한 (요상담).
