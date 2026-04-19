# StageSync

> **리듬게임 백엔드 서버 엔지니어의 일상 업무를 Go 로 재현한 포트폴리오**

[![CI](https://github.com/cartama700/StageSync/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/cartama700/StageSync/actions/workflows/ci.yml)
[![Go Version](https://img.shields.io/badge/go-1.26-00ADD8?logo=go)](./go.mod)
[![License](https://img.shields.io/badge/license-portfolio--only-lightgrey)](#라이선스)

**언어 / 言語**: [한국어 (현재 파일)](./README.ko.md) · [日本語](./README.md)
**SSOT**: [`docs/MISSION.md`](./docs/MISSION.md) · [`docs/PLAN.md`](./docs/PLAN.md) · [`docs/STATUS.md`](./docs/STATUS.md) · [`docs/API.md`](./docs/API.md) · [`CHANGELOG.md`](./CHANGELOG.md)

---

## 개요

株式会社 Colorful Palette 서버사이드 엔지니어 공고 [BA-09-04a](https://hrmos.co/pages/colorfulpalette/jobs/BA-09-04a) 를 타겟으로, 실제 일상 업무 — **REST API · DB · 비동기 배치 · 운영** — 을 Go 로 구현한 포트폴리오.

공고 명시 스택 (**Aurora MySQL · Cloud Spanner · Redis · GKE · Terraform · Locust**) 에 완전 정렬시키고, Mercari / CyberAgent 그룹 기준의 일본 Go 업계 관행을 반영.

**Project Sekai 의 실 프로덕션 구조 모사**:
- 실시간 통신: **Diarkis** (Go / GKE / TCP·UDP·RUDP) — 외부 미들웨어
- REST 레이어: **Spring Boot (Java/Go)** — 본 포트폴리오의 **주축**
- 데이터 스토어: Aurora MySQL + Spanner + Redis

2 층 구조 (REST 주축 + WebSocket 보너스축) 양쪽 모두 구현.

---

## 작성자에 대해 (정직하게)

본 프로젝트는 "장인의 대표작" 이 아니라 **"성장 포트폴리오"** 다.

- **Go · Java 는 이번이 첫 실무 도전** (C#/.NET 실무 경력 수 년)
- **일본어 N2 ~ N3** (공고는 N1 필수 — 에이전트 경유 컨택 성립, 본 포트폴리오가 결정타)
- **AI 툴 (Claude Code) 적극 활용** — 2026 년 현재의 개발 생산성 · 학습 속도 실증
- 증명하고 싶은 것: **학습 속도** · **기술적 깊이** · **일본 업계 관행에 정합하려는 의지**

---

## 기술 스택

### 구현 완료 (Phase 0-16 · v3 MVP 스코프)

| 영역 | 채택 기술 |
|---|---|
| 라우팅 | `chi` + HTTP/2 cleartext (h2c) — [ADR-0003](./docs/adr/0003-h2c-for-websocket-coexistence.md) |
| 아키텍처 | handler → service → repository 3 층 + `Mount(r)` + consumer-defined interface |
| 설정 | `internal/config` 환경변수 집약 + 유효성 검증 |
| 런타임 | Graceful shutdown (`SIGTERM` → readiness drain → `srv.Shutdown`) · request timeout · request-scoped slog (`request_id` 전파) |
| DB (RDBMS) | Aurora MySQL (`sqlc` + `goose` + 원자 TX) · inmem ↔ MySQL 전환 — [ADR-0002](./docs/adr/0002-sqlc-over-orm.md) |
| DB (KV) | Redis ZSET (랭킹) · `REDIS_ADDR` graceful degrade (inmem fallback) |
| 게임 도메인 | 프로필 · 가챠 (10-roll TX + 천장 80 회) · 이벤트 (시간 derived 상태 + 원자 UPSERT) · 랭킹 (ZSET + 본인 ±N) |
| 검증 | `go-playground/validator/v10` + 커스텀 에러 타입 계층 (`apperror`) |
| 에러 처리 | `fmt.Errorf("%w")` · `errors.Is` / `errors.As` · sentinel errors |
| 관측성 | Prometheus `/metrics` (Histogram + Gauge + Go runtime collector) · `/debug/pprof/*` · access log (`request_id`) |
| 테스트 | `testify/require` · table-driven · `t.Parallel()` · httptest E2E · `go-sqlmock` · `miniredis` · race detector |
| 정적 분석 | `.golangci.yml` v2 (errcheck, staticcheck, revive, gocritic, bodyclose 등) |
| CI | GitHub Actions (test + lint + Docker build + benchmark) |
| 배포 | Multi-target Dockerfile (distroless/static) + docker-compose 3 profile + K8s manifest ([`deploy/k8s/`](./deploy/k8s/)) + readiness gate |
| 부하 테스트 | Locust cluster 시나리오 ([`deploy/locust/`](./deploy/locust/)) + `cmd/bots` WebSocket (even/herd/cluster × N) |

### 보너스축 — 실시간 통신

- `coder/websocket` + protobuf binary frame
- `sync.RWMutex` + map 기반 thread-safe Room 관리
- AOI 필터 + `sync.Pool` 최적화 토글 (**약 2.5 배 고속화 · 0 allocs/op** — 실측은 [`docs/BENCHMARKS.md`](./docs/BENCHMARKS.md))
- `cmd/bots` WebSocket 부하 시뮬레이터 (`even` / `herd` / `cluster` 시나리오 · N 병렬)

### 제출 후 추가 예정 (v0.7 장애 시나리오 랩)

**Phase 19 — HP 동시 차감 데드락 랩**: `SELECT ... FOR UPDATE` 기반 직렬 잠금으로 데드락 재현 → 유저별 파티션 큐로 직렬화 → 벤치 비교 3 단.
면접 기간 중 추가 예정 — "최근 업데이트" 로 면접에서 어필.

### v3 에서 제외한 Phase (제출 스코프 밖)

Phase 8 (메일) · Phase 10-11 (비동기 배치 · Write-Behind) · Phase 12 (Spanner) · Phase 15 (Terraform GKE) · Phase 17 (AI Ops LLM) · Phase 20-22 (기타 장애 랩) — 이유는 [`docs/PLAN.md`](./docs/PLAN.md) "스코프 재편 기록" 참조.

---

## Phase 진행 현황 (v3 MVP 스코프)

| 마일스톤 | Phase | 상태 |
|---|---|---|
| 보너스축 | 0, A, B | ✅ 3/3 |
| v0.1 기반 | 1-4 | ✅ 4/4 |
| v0.2 도메인 | 5, 6, 7 | ✅ 3/3 (가챠 · 이벤트 · 랭킹) |
| v0.3 운영 lite | 9 | ✅ 1/1 (Histogram + pprof) |
| v0.5 배포 lite | 13, 14 | ✅ 2/2 (Docker profiles + K8s manifest) |
| v0.6 마감 | 16, 18 | ✅ 2/2 (Locust + README/문서 완료) |
| v0.7 장애 랩 (제출 후) | 19 | ⏳ 0/1 (면접 기간 중 추가 예정) |

**총합: 15/15 MVP ✅ 완료** — 상세 로드맵 + 제외 Phase 이유: [`docs/PLAN.md`](./docs/PLAN.md) · 현재 스냅샷: [`docs/STATUS.md`](./docs/STATUS.md)

---

## 퀵스타트

### 🐳 Docker Compose — 30 초 내 REST + MySQL + Redis 기동 (**권장**)

Docker 만 깔려 있으면 OS 무관 즉시 실행:

```bash
docker compose up --build           # server + mysql + redis
# 다른 터미널:
curl -X POST http://localhost:5050/api/profile \
     -H "Content-Type: application/json" \
     -d '{"id":"p1","name":"sekai"}'
curl http://localhost:5050/api/profile/p1
curl http://localhost:5050/metrics          # Prometheus scrape
```

**외부 의존 없음 (inmem only)**:
```bash
docker compose --profile inmem up server-inmem --build
```

**부하 시뮬 포함 (server + mysql + redis + bots)**:
```bash
docker compose --profile load up --build    # + bots-cluster + bots-herd 가 자동으로 server 에 접속
curl http://localhost:5050/metrics | grep stagesync_room_connected_players
```

**Locust 로 REST 부하 테스트 (별 프로세스)**:
```bash
pip install locust
locust -f deploy/locust/locustfile.py --host http://localhost:5050 \
       --headless -u 500 -r 10 -t 1m --html=locust_report.html
```
상세: [`deploy/locust/README.md`](./deploy/locust/README.md) · 결과 템플릿: [`docs/BENCHMARKS.md`](./docs/BENCHMARKS.md).

**Kubernetes 배포 (manifest dry-run)**:
```bash
kubectl apply --dry-run=client -f deploy/k8s/
```
readiness gate · HPA · graceful drain 설정은 [`deploy/k8s/README.md`](./deploy/k8s/README.md).

Makefile 단축: `make compose-up` / `make compose-inmem` / `make compose-load` / `make compose-down`.
환경변수는 [`.env.example`](./.env.example) 참조 (`LISTEN_ADDR` · `LOG_LEVEL` · `SHUTDOWN_TIMEOUT` · `REQUEST_TIMEOUT` · `MYSQL_DSN` · `REDIS_ADDR`).

### 네이티브 실행 (소스 수정 · 개발자용)

툴체인 일괄 설치 (macOS 는 bash, Windows 는 PowerShell):

```bash
./scripts/setup.sh     # macOS: Homebrew 로 Go · protoc · sqlc · goose · golangci-lint
./scripts/setup.ps1    # Windows: Chocolatey 로 동등한 셋
```

그 후:

```bash
make run               # inmem 모드
make run-mysql         # MySQL 접속 (make dev-up 으로 사전에 MySQL 기동)
```

### 보너스축 — WebSocket 부하 시뮬

```bash
make run                                    # 다른 터미널
go run ./cmd/bots -n=50 -scenario=herd      # 50 bots 가 원점 근처로 군집
go run ./cmd/bots -n=100 -scenario=even     # 100 bots 가 맵 전체에 균등 분산
curl -X POST http://localhost:5050/api/optimize \
     -H "Content-Type: application/json" -d '{"on":true}'
curl http://localhost:5050/metrics | grep stagesync_
```

### 테스트 · 정적 분석

```bash
make test              # go test -race ./...
make bench             # AOI 벤치마크
```

API 상세는 [`docs/API.md`](./docs/API.md) 참조.

---

## 디렉토리 구조

```
StageSync/
├── cmd/
│   ├── server/          REST + WebSocket 서버
│   └── bots/            WebSocket 부하 시뮬 (even/herd/cluster)
├── api/proto/roompb/    protobuf 스키마 + 생성 코드
├── internal/
│   ├── config/          환경변수 기반 설정 + 유효성 검증
│   ├── domain/          순수 도메인 오브젝트 (profile, gacha, event, ranking)
│   ├── service/         비즈니스 로직 (profile, gacha, event, ranking, aoi)
│   ├── persistence/
│   │   ├── inmem/       메모리 구현 (개발 · 테스트 · Redis fallback)
│   │   ├── mysql/       sqlc + goose + schema + queries
│   │   └── redis/       랭킹 ZSET (miniredis 테스트)
│   ├── endpoint/        HTTP 핸들러 + 미들웨어 (Mount 패턴 · Prometheus Histogram · pprof)
│   ├── apperror/        에러 타입 계층 + HTTP 매핑
│   ├── room/            WebSocket Room 상태 (보너스축)
│   └── lifecycle/       런타임 플래그 (최적화 토글 · readiness gate)
├── docs/
│   ├── MISSION.md · PLAN.md · STATUS.md
│   ├── API.md · BENCHMARKS.md · CHANGELOG 는 루트
│   ├── adr/             Architecture Decision Records
│   └── demo/            데모 GIF · 스크린샷
├── scripts/setup.sh     툴체인 일괄 설치
├── .github/workflows/   CI 파이프라인 (test + lint + docker build + bench)
├── deploy/
│   ├── k8s/             namespace · deployment · service · hpa · configmap
│   └── locust/          cluster 시나리오 + README
├── Dockerfile           multi-target (server + bots, distroless/static)
├── docker-compose.yml   server + MySQL + Redis + bots (profile 별)
├── .env.example
├── CHANGELOG.md
├── Makefile · sqlc.yaml · .golangci.yml · go.mod
```

---

## 관련 문서

- [**docs/MISSION.md**](./docs/MISSION.md) — 프로젝트 미션 · 공고 매핑 · 5 축 프레임
- [**docs/PLAN.md**](./docs/PLAN.md) — Phase 로드맵 (v3) · 제외 Phase 이유 · 학습 트래커
- [**docs/STATUS.md**](./docs/STATUS.md) — 현재 스냅샷 (PLAN ↔ 리포지토리 대조)
- [**docs/API.md**](./docs/API.md) — 엔드포인트 사양 (SSOT)
- [**docs/BENCHMARKS.md**](./docs/BENCHMARKS.md) — AOI + Locust 실측
- [**docs/adr/**](./docs/adr/) — 주요 기술 선택 기록 (chi · sqlc · h2c)
- [**docs/TRADEOFFS.md**](./docs/TRADEOFFS.md) — 의도적 scope-out 5 지점 + 면접 대응 논리
- [**CHANGELOG.md**](./CHANGELOG.md) — 완료된 변경 이력
- [**deploy/k8s/README.md**](./deploy/k8s/README.md) — K8s 배포 절차 · readiness drain 동작
- [**deploy/locust/README.md**](./deploy/locust/README.md) — 부하 테스트 실행 방법
- [**README.md**](./README.md) — 日本語 版

---

## 라이선스

개인 포트폴리오용 · 상업적 재사용 제한.
