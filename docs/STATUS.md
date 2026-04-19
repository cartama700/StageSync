# StageSync 현황 보고서

> **문서 역할**: `docs/PLAN.md` 의 로드맵과 **2026-04-19 시점 실제 리포지토리 상태** 를 대조.
> 리뷰어·면접관이 README 와 코드를 왕복하지 않고 한 장으로 "뭐가 됐고 다음은 뭐" 를 파악할 수 있도록.
> 출처: [`docs/PLAN.md`](./PLAN.md) + 리포지토리 스캔.

---

## 1. 한눈에

| 항목 | 값 |
|---|---|
| 시작 | 2026-04-17 |
| 현 시점 | 2026-04-20 (4 일차 · 문서 마감 단계) |
| 현 위치 | **MVP 15/15 완료 (100%)** — 문서 마감 · GIF 녹화 · 제출 준비 |
| 다음 예정 | **제출** → 면접 기간 중 **Phase 19** (HP 데드락 랩) 추가 |
| 제출 후 | **Phase 19** (HP 데드락 랩 · 면접 기간 중 추가) |
| Go 소스 (생성·테스트 제외) | ~3,000 LoC |
| 테스트 | **181 PASS** · 주요 모듈 커버리지 80~100% |
| CI | GitHub Actions — test(-race) + lint + docker build + bench (전 green) |
| 엔드포인트 | 19 개 (health 2 · metrics 2 · pprof · profile 2 + gacha 3 + event 6 + ranking 2 + optimize 1 + ws 1) |

---

## 2. PLAN v3 대비 스코어보드

```
보너스축      [████████] 3/3 ✓   (Phase 0, A, B)
v0.1 기반     [████████] 4/4 ✓   (Phase 1, 2, 3, 4)
v0.2 도메인   [████████] 3/3 ✓   (Phase 5, 6, 7)
v0.3 운영     [████████] 1/1 ✓   (Phase 9 lite — Histogram + pprof)
v0.5 배포     [████████] 2/2 ✓   (Phase 13, 14 lite)
v0.6 마감     [████████] 2/2 ✓   (Phase 16 ✓, 18 ✓ — 코드·문서 완료)
v0.7 장애랩   [▓▓▓▓▓▓▓▓] 0/1     (Phase 19 — 제출 후 면접 기간 중)

총            15/15 MVP ✅ + Phase 19 옵션
```

> **v2 → v3 재편 (2026-04-19)**: 25 → 15 Phase 로 축소. 제외된 11 Phase 의 이유는
> [`PLAN.md`](./PLAN.md) "스코프 재편 기록" 참조.
> **제출 준비**: 코드·문서는 완료. 남은 것은 사용자 작업 (데모 GIF 녹화 · CI green 확인 · git tag v0.1 push) — [`SUBMISSION_CHECKLIST.md`](./SUBMISSION_CHECKLIST.md) 참조.

---

## 3. 완료 Phase 상세

### ✅ Phase 0 — 뼈대 (chi + h2c)
- 핸들러: `/api/metrics` (JSON), `/health/{live,ready}`
- 핵심 파일: [`cmd/server/main.go`](../cmd/server/main.go) · [`go.mod`](../go.mod) · [`.golangci.yml`](../.golangci.yml)

### ✅ Phase A — WebSocket Room (보너스축)
- `coder/websocket` + protobuf ClientMessage · thread-safe Room (`sync.RWMutex`)
- `cmd/bots` WebSocket 부하 봇

### ✅ Phase B — AOI + sync.Pool 최적화
- Naive ↔ Pooled 런타임 토글 (`POST /api/optimize`)
- 실측 **2.48× 속도 · 0 alloc/op**

### ✅ Phase 1 — clean architecture + inmem ProfileRepo
- handler → service → repository 3층 + `Mount(r)` 패턴

### ✅ Phase 2 — MySQL + sqlc + goose
- 타입 안전 쿼리 · 임베드 마이그레이션 · `MYSQL_DSN` graceful swap

### ✅ Phase 3 — Validation + 에러 타입 체계
- `go-playground/validator/v10` + 커스텀 `apperror`

### ✅ Phase 4 — 테스트 + golangci-lint CI
- `testify/require` + table-driven + `t.Parallel()` + `httptest`

### ✅ Phase 5 — ガチャ API (PR #2 merged 2026-04-18)
- 원자 트랜잭션 10-roll + 천장 80회 + 10,000 분포 테스트

### ✅ Phase 6 — イベント API (2026-04-19)
- 시간 기반 derived 상태 · `ON DUPLICATE KEY UPDATE ... + VALUES()` 누적 UPSERT
- 6 엔드포인트 · clock DI 테스트

### ✅ Phase 7 — ランキング API (2026-04-19)
- Redis ZSET (`ZADD`/`ZREVRANGE`/`ZREVRANK`/`ZINCRBY`) · `REDIS_ADDR` graceful fallback (inmem)
- Event 서비스 `AddScore` 와 연동 (best-effort `ZINCRBY`) — MySQL=truth, Redis=cache 패턴
- miniredis 로 단위 테스트 (실 Redis 불필요)
- 2 엔드포인트: `GET /api/ranking/{eventId}/top` · `GET /api/ranking/{eventId}/me/{playerId}`

### ✅ Phase 9 lite — Histogram + pprof (2026-04-19)
- `http_request_duration_seconds` HistogramVec (method × chi RoutePattern × status)
- `RequestMetrics` 미들웨어 (route pattern 정확 집계)
- `/debug/pprof/*` 마운트 (timeout 제외 Group)

### ✅ Phase 13 — Docker profiles (2026-04-19)
- Multi-target Dockerfile (`builder` + `bots` + `server`)
- docker-compose 3 profile: default / inmem / load
- `load` profile 에서 `bots-cluster` + `bots-herd` 2종 자동 실행

### ✅ Phase 14 lite — K8s manifest + readiness gate (2026-04-19)
- `atomic.Bool` 기반 readiness → SIGTERM 시 drain → 5s sleep → `srv.Shutdown()`
- `deploy/k8s/`: namespace · configmap · deployment · service · hpa (autoscaling/v2)
- distroless 환경의 preStop 제약 → 앱 내부 drain 흐름으로 대체

### ✅ Phase 16 lite — Locust 시나리오 (2026-04-19)
- `deploy/locust/locustfile.py` (FastHttpUser) — 3 task 3:2:1 (score/gacha/ranking)
- `docs/BENCHMARKS.md` 결과 템플릿 (측정은 Phase 18 에서 실행)

### ✅ Phase 18 — README + 문서 마감 (2026-04-20)
- ✅ README.md 최신화 (기술 스택 · Phase 진행 · 디렉토리)
- ✅ README.ko.md 완전 재작성 · README.md 와 동기화
- ✅ STATUS.md (본 문서) 최신화
- ✅ CHANGELOG v0.1 태그 섹션 완성
- ✅ MISSION.md v3 스코프 반영
- ✅ API.md Event 6 엔드포인트 추가 + 에러 포맷 실제와 일치
- ✅ PORTFOLIO_SCENARIOS.md v3 (Phase 19 만 유지, 나머지 기획만 보존)
- ✅ PITCH.md (30초/2분/5분 JP + ko) · SUBMISSION_CHECKLIST.md · docs/demo/README.md 신규
- ⏳ 데모 GIF 녹화 (사용자 작업 — `docs/demo/README.md` 가이드 참조)
- ⏳ 최종 CI green 확인 + git tag v0.1 push (사용자 작업)

---

## 4. 디렉토리 ↔ Phase 맵

```
StageSync/
├─ cmd/
│  ├─ server/main.go .............. Phase 0 + 운영 기반 + 14 lite drain
│  └─ bots/main.go ................ Phase A + 13 (cluster/herd/even 시나리오)
├─ api/proto/roompb/ .............. Phase A
├─ internal/
│  ├─ config/ ..................... 운영 기반 (2026-04-19)
│  ├─ domain/
│  │   ├─ profile/ ................ Phase 1
│  │   ├─ gacha/ .................. Phase 5
│  │   ├─ event/ .................. Phase 6
│  │   └─ ranking/ ................ Phase 7
│  ├─ service/
│  │   ├─ profile/ ................ Phase 1
│  │   ├─ gacha/ .................. Phase 5
│  │   ├─ event/ .................. Phase 6
│  │   ├─ ranking/ ................ Phase 7
│  │   └─ aoi/ .................... Phase B
│  ├─ persistence/
│  │   ├─ inmem/ .................. Phase 1, 5, 6, 7 (leaderboard fallback)
│  │   ├─ mysql/ .................. Phase 2, 5, 6 (sqlc + goose)
│  │   └─ redis/ .................. Phase 7 (ZSET)
│  ├─ endpoint/
│  │   ├─ profile.go .............. Phase 1
│  │   ├─ gacha.go ................ Phase 5
│  │   ├─ event.go ................ Phase 6
│  │   ├─ ranking.go .............. Phase 7
│  │   ├─ ws.go ................... Phase A
│  │   ├─ optimize.go ............. Phase B
│  │   ├─ metrics.go (JSON) ....... Phase 0
│  │   ├─ health.go ............... Phase 0 + 14 lite (drain)
│  │   ├─ middleware.go ........... 운영 기반 + 9 lite (Histogram)
│  │   └─ prometheus.go ........... 운영 기반 + 9 lite (HistogramVec)
│  ├─ apperror/ ................... Phase 3
│  ├─ room/ ....................... Phase A
│  └─ lifecycle/
│      ├─ optimize.go ............. Phase B
│      └─ readiness.go ............ Phase 14 lite
├─ docs/
│  ├─ MISSION.md · PLAN.md · STATUS.md
│  ├─ API.md · BENCHMARKS.md · adr/
│  └─ demo/ ....................... Phase 18 (GIF 배치 예정)
├─ deploy/
│  ├─ k8s/ ........................ Phase 14 lite
│  └─ locust/ ..................... Phase 16 lite
├─ .github/workflows/ci.yml ....... Phase 4 + 2026-04-19 강화 (docker build job 추가)
├─ Dockerfile (multi-target) ...... Phase 13
├─ docker-compose.yml ............. Phase 13 (3 profile)
├─ .env.example · CHANGELOG.md
└─ Makefile · sqlc.yaml · .golangci.yml
```

---

## 5. API 엔드포인트 현황

| 경로 | 메서드 | 도메인 | Phase |
|---|---|---|---|
| `/health/live` · `/health/ready` | GET | 운영 | 0 + 14 lite |
| `/api/metrics` | GET | 운영 (JSON) | 0 |
| `/metrics` | GET | Prometheus (Histogram + Gauge) | 운영 기반 + 9 lite |
| `/debug/pprof/*` | GET | 런타임 진단 | 9 lite |
| `/api/optimize` | POST | AOI 토글 | B |
| `/api/profile` | POST | Profile | 1 |
| `/api/profile/{id}` | GET | Profile | 1 |
| `/api/gacha/roll` | POST | Gacha | 5 |
| `/api/gacha/history/{player}` | GET | Gacha | 5 |
| `/api/gacha/pity/{player}/{pool}` | GET | Gacha | 5 |
| `/api/event` | POST | Event | 6 |
| `/api/event/current` | GET | Event | 6 |
| `/api/event/{id}` | GET | Event | 6 |
| `/api/event/{id}/score` | POST | Event | 6 |
| `/api/event/{id}/score/{playerId}` | GET | Event | 6 |
| `/api/event/{id}/rewards/{playerId}` | GET | Event | 6 |
| `/api/ranking/{eventId}/top` | GET | Ranking | 7 |
| `/api/ranking/{eventId}/me/{playerId}` | GET | Ranking | 7 |
| `/ws/room` | GET (Upgrade) | WebSocket | A |

상세 계약: [`docs/API.md`](./API.md).

---

## 6. 테스트 · 품질 현황 (2026-04-20 기준)

**전체**: **181 PASS** · `go vet` clean · `go test ./...` 0.4-0.7s per package

| 패키지 | 주요 테스트 |
|---|---|
| `internal/apperror` | 타입 계층 + Unwrap 체인 |
| `internal/config` | 디폴트 · override · invalid level |
| `internal/domain/gacha` | WeightedPick 경계값 table-driven |
| `internal/endpoint` | httptest + 미들웨어 + /metrics + pprof smoke + RequestMetrics pattern 집계 |
| `internal/lifecycle` | atomic 토글 + readiness + 동시성 |
| `internal/persistence/mysql` | go-sqlmock (tx rollback · 1062 매핑) |
| `internal/persistence/redis` | miniredis (ZSET 호환 검증) |
| `internal/room` | 1000 고루틴 upsert |
| `internal/service/aoi` | Naive vs Pooled (+ 벤치 2 건) |
| `internal/service/gacha` | 10,000 분포 검증 + 천장 시나리오 |
| `internal/service/event` | 상태 전이 + 누적 + rewards |
| `internal/service/ranking` | 경계값 + 미등재 에러 전파 |
| `cmd/bots` | scenario 좌표 범위·결정성 |

---

## 7. 남은 작업 (제출 전 사용자 체크리스트)

**문서·코드 마감 ✅**
- [x] README.md · README.ko.md 최신화 + 동기화
- [x] STATUS.md (본 문서) 최신화
- [x] MISSION.md · PLAN.md · API.md · PORTFOLIO_SCENARIOS.md v3 정합
- [x] CHANGELOG.md v0.1 릴리즈 태그 섹션
- [x] PITCH.md · SUBMISSION_CHECKLIST.md · docs/demo/README.md 신규

**사용자 남은 작업 ⏳**
- [ ] 데모 GIF 녹화 2 종 (quickstart + loadtest) — 가이드: [`docs/demo/README.md`](./demo/README.md)
- [ ] (선택) Locust 실측 후 `docs/BENCHMARKS.md` 표 채우기
- [ ] CI green 최종 확인 (GitHub Actions 탭)
- [ ] `git tag v0.1 && git push --tags`
- [ ] 에이전시 · 리쿠르터에게 URL 공유 — 템플릿: [`SUBMISSION_CHECKLIST.md`](./SUBMISSION_CHECKLIST.md) G 섹션
- [ ] 면접 피치 연습 — 대본: [`PITCH.md`](./PITCH.md)

---

## 8. 포트폴리오 리뷰 순회 경로 (30초 → 5분 → 30분)

**30초**
1. README 상단 뱃지 3 개 + 기술 스택 표
2. `docker compose up --build` → `curl localhost:5050/metrics`

**5분**
1. [`docs/API.md`](./API.md) 한 번 훑기 (19 엔드포인트 일람)
2. [`internal/service/gacha/service.go`](../internal/service/gacha/service.go) — 천장 로직
3. [`internal/persistence/mysql/gacha_repo.go`](../internal/persistence/mysql/gacha_repo.go) — 트랜잭션 관용구
4. [`internal/persistence/redis/leaderboard.go`](../internal/persistence/redis/leaderboard.go) — ZSET 사용법
5. [`docs/BENCHMARKS.md`](./BENCHMARKS.md) — AOI 수치 + Locust 템플릿
6. [`.github/workflows/ci.yml`](../.github/workflows/ci.yml) — 파이프라인

**30분**
1. [`docs/PLAN.md`](./PLAN.md) — 전체 로드맵 + v3 스코프 재편 사유
2. [`docs/adr/`](./adr/) — chi · sqlc · h2c 3 건
3. [`internal/service/event/service.go`](../internal/service/event/service.go) — clock DI + best-effort Redis 연동
4. [`internal/service/gacha/service_test.go`](../internal/service/gacha/service_test.go) — 10k 분포 테스트
5. [`cmd/server/main.go`](../cmd/server/main.go) — graceful drain + middleware stack + 조립
6. [`deploy/k8s/deployment.yaml`](../deploy/k8s/deployment.yaml) — readiness probe · HPA · resource limits
7. [`CHANGELOG.md`](../CHANGELOG.md) — 3 일간 작업 타임라인

---

## 9. 참조

- [`PLAN.md`](./PLAN.md) — 로드맵 (SSOT)
- [`MISSION.md`](./MISSION.md) — 프로젝트 미션 · 공고 매핑
- [`API.md`](./API.md) · [`BENCHMARKS.md`](./BENCHMARKS.md) · [`adr/`](./adr/)
- [`../CHANGELOG.md`](../CHANGELOG.md) — 완료된 변경 이력
- [`../README.md`](../README.md) (日本語) · [`../README.ko.md`](../README.ko.md)
- [`../deploy/k8s/README.md`](../deploy/k8s/README.md) · [`../deploy/locust/README.md`](../deploy/locust/README.md)
