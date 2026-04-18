# StageSync Phase 계획 (v2 — REST-first)

> **본 문서의 역할**: 실행 로드맵. 진행 추적 · 학습 트래커 · 의존성 맵.
> 미션·배경·공고 분석은 [MISSION.md](./MISSION.md) 참조.
> **v2 시작**: 2026-04-18 (v1 이식 관점에서 REST-first 로 재편)

---

## 전체 진행 현황

```
보너스축    [━━━━━━] 3/3 ✓ 완료
v0.1 기반   [━━━━━━] 4/4 ✓ 완료
v0.2 도메인 [▓·······] 1/4 (Phase 5 진행 중)
v0.3 운영   [·····]   0/3
v0.4 데이터 [·]       0/1
v0.5 배포   [·····]   0/3
v0.6 마감   [·····]   0/3
            ─────────
총           7/21 = 33%
```

**현 위치**: **Phase 5 (ガチャ API) 진행 중** — 브랜치 `feat/phase-5-gacha`
**재편 기록**: 2026-04-18 대전제 재정의 (실시간 중심 → REST 중심). 기존 Phase 1·2 는 보너스축 (A·B) 로 이관.

---

## 마일스톤

| 버전 | Phase | 내러티브 | 진행 |
|---|---|---|---|
| **보너스** | 0, A, B | "기반 + 실시간 프로토콜 + 핫패스 최적화 쇼케이스" | **3/3 ✓** |
| **v0.1 기반** | 1-4 | "clean architecture + MySQL + 테스트 CI 확립" | **4/4 ✓** |
| **v0.2 도메인** | 5-8 | "ガチャ·イベント·ランキング·メール 게임 API" | 1/4 (진행) |
| **v0.3 운영** | 9-11 | "Prometheus·pprof·비동기 배치·Write-Behind" | 0/3 |
| **v0.4 데이터** | 12 | "Spanner 듀얼 + hotspot 회피" | 0/1 |
| **v0.5 배포** | 13-15 | "Docker + K8s + Terraform GKE" | 0/3 |
| **v0.6 마감** | 16-18 | "Locust 부하 + AI Ops + 문서 마감" | 0/3 |

---

## Phase 의존성 그래프

```
[완료 — 보너스축]
Phase 0 뼈대 (chi + h2c) ✓
   ├─ Phase A WebSocket Room ✓
   └─ Phase B AOI + sync.Pool ✓

[v0.1 REST 기반]
Phase 0 ─→ Phase 1 (clean arch + inmem repo)
              └─→ Phase 2 (MySQL + goose + 트랜잭션)
                     └─→ Phase 3 (Validation + 에러 타입 + %w 래핑)
                            └─→ Phase 4 (테스트 + golangci-lint CI)

[v0.2 도메인]       Phase 5 (ガチャ)     ← Phase 2
                    Phase 6 (イベント)    ← Phase 2
                    Phase 7 (ランキング)  ← Phase 2 + Redis
                    Phase 8 (メール)     ← Phase 2

[v0.3 운영]         Phase 9 (Prometheus + pprof)     ← 전역
                    Phase 10 (비동기 배치 잡)         ← Phase 5-8
                    Phase 11 (Write-Behind)          ← Phase 10

[v0.4 데이터]       Phase 12 (Spanner 듀얼)          ← Phase 2, 11

[v0.5 배포]         Phase 13 (Docker compose)        ← 전역
                    Phase 14 (K8s + Graceful)        ← Phase 13
                    Phase 15 (Terraform GKE)         ← Phase 14

[v0.6 마감]         Phase 16 (Locust + k6)           ← Phase 15
                    Phase 17 (AI Ops LLM + SSE)      ← Phase 9
                    Phase 18 (README/GIF/JP 마감)    ← 모든 Phase
```

**병렬 가능**: Phase 5-8 (게임 도메인) 은 서로 독립 — 순서 자유.
**반드시 마지막**: Phase 18.

---

## 보너스축 Phase (완료)

### Phase 0 — 뼈대 ✓ 완료 (2026-04-18)

**산출물**: chi 라우터 + HTTP/2 cleartext (h2c) + `/api/metrics` + `/health/{live,ready}`

**배운 Go 개념**:
- 모듈 시스템 (`go.mod`, `go.sum`), `go mod init`/`tidy`
- `package main` + `func main()`, `cmd/<name>/main.go` 관습
- `:=` 짧은 선언, `_` discard, `map[string]any`
- `if err := f(); err != nil { }` 에러 반환 패턴
- chi 미들웨어 체인, `http.HandlerFunc` 시그니처
- `h2c.NewHandler` 로 HTTP/1.1 + HTTP/2 공존

**파일**: [`cmd/server/main.go`](../cmd/server/main.go) (일부), [`go.mod`](../go.mod), [`.golangci.yml`](../.golangci.yml)

### Phase A — WebSocket Room ✓ 완료 (2026-04-18) — 구 Phase 1

**산출물**: `coder/websocket` + protobuf binary frame + thread-safe Room + cmd/bots E2E

**배운 Go 개념**:
- protobuf 스키마·코드 생성 (`.proto` → `.pb.go`)
- `oneof` 타입 스위치 + 인터페이스 만족
- `sync.RWMutex` + `defer`, goroutine per connection
- `flag`, `signal.NotifyContext`, `time.NewTicker`, `select`
- 클로저 팩토리 DI, 구조체 메서드 핸들러

**파일**: [`api/proto/roompb/`](../api/proto/roompb/), [`internal/room/`](../internal/room/), [`internal/endpoint/ws.go`](../internal/endpoint/ws.go), [`cmd/bots/main.go`](../cmd/bots/main.go)

**재활용**: 유지. Phase 16 에서 부하 시나리오 확장 기반.

### Phase B — AOI + 최적화 토글 ✓ 완료 (2026-04-18) — 구 Phase 2

**산출물**: Naive vs Pooled 필터 + `sync.Pool` + `atomic.Bool` 토글 + 벤치 (1.5× · 0 allocs)

**배운 Go 개념**:
- `sync.Pool` 패턴 (Get/Put/New, reset 책임)
- `sync/atomic` (`atomic.Bool`)
- `testing.B` + `b.Loop()` (Go 1.24+)
- `math/rand/v2` + 고정 seed
- callback 패턴 DI (`func Pooled(..., fn func([]int))`)
- escape analysis 함정 (패키지 변수로 escape 강제해야 현실적 벤치)

**파일**: [`internal/service/aoi/`](../internal/service/aoi/), [`internal/lifecycle/optimize.go`](../internal/lifecycle/optimize.go), [`internal/endpoint/optimize.go`](../internal/endpoint/optimize.go)

**재활용**: `sync.Pool` 패턴은 Phase 9 (로그 버퍼), Phase 11 (Write-Behind) 에서 반복 사용.

---

## v0.1 기반 — REST + MySQL + 테스트 ✓ 완료

### Phase 1 — REST API 기반 + clean architecture ✓ 완료 (2026-04-18)

**산출물**:
- handler → service → repository 3-레이어 + Mount 패턴
- Consumer-defined interface (`endpoint.ProfileService`)
- Profile 도메인 + DTO 매퍼
- inmem `ProfileRepo` (테스트·개발)
- `POST /api/profile` · `GET /api/profile/{id}`

**배운 Go 개념**: `context.Context` 첫 파라미터 · consumer 측 인터페이스 선언 · DTO ↔ Model 매핑 · 구조체 메서드 핸들러

**주요 파일**: [`internal/domain/profile/`](../internal/domain/profile/) · [`internal/service/profile/`](../internal/service/profile/) · [`internal/persistence/inmem/profile_repo.go`](../internal/persistence/inmem/profile_repo.go) · [`internal/endpoint/profile.go`](../internal/endpoint/profile.go)

---

### Phase 2 — MySQL + sqlc + goose + inmem↔mysql swap ✓ 완료 (2026-04-18)

**산출물**:
- `sqlc` 타입 안전 쿼리 (schema.sql + queries/*.sql → gen/*.go)
- `goose` 프로그래머블 마이그레이션 + `embed.FS`
- `MYSQL_DSN` graceful degrade (inmem ↔ mysql swap)
- MySQL 1062 duplicate key → `domain.ErrAlreadyExists` 매핑
- Colima + Docker 수동 제어 (`make dev-up/down`)
- **영속성 검증 E2E** — 서버 재시작 후에도 데이터 유지

**배운 Go 개념**: `database/sql` + blank import driver · `sql.DB` pool · `//go:embed` · `errors.As(err, &mysqlErr)` · sqlc + goose 공존 패턴

**주요 파일**: [`internal/persistence/mysql/`](../internal/persistence/mysql/) · [`sqlc.yaml`](../sqlc.yaml)

---

### Phase 3 — Validation + 에러 타입 체계 ✓ 완료 (2026-04-18)

**산출물**:
- `go-playground/validator/v10` struct tag 검증
- `internal/apperror` 패키지 — 타입 계층 + HTTP 매핑
- 필드별 에러 응답 `{"error":{"code":"...","fields":[{field,tag,message}...]}}`
- 도메인 sentinel ↔ HTTP error 경계 분리 (service 는 HTTP 몰라도 됨)
- E2E 8 시나리오 검증 (400 VALIDATION / 404 / 409 / 201 / 200)

**배운 Go 개념**: struct tag validation · `RegisterTagNameFunc` (JSON 필드명 매칭) · 커스텀 error type + `Unwrap()` · `errors.As` 타입 매칭

**주요 파일**: [`internal/apperror/`](../internal/apperror/) · [`internal/endpoint/profile.go`](../internal/endpoint/profile.go) (validator 통합)

---

### Phase 4 — 테스트 + golangci-lint CI ✓ 완료 (2026-04-18)

**산출물**:
- `testify/require` + table-driven + `t.Parallel()` (31 서브케이스)
- `httptest.NewServer` in-memory HTTP E2E
- `.golangci.yml` v2 포맷 + 13 린터 (errcheck · staticcheck · revive · gocritic · bodyclose 등)
- `.github/workflows/ci.yml` — test + lint + benchmark (golangci-lint v2.11.4)
- `run() error` 패턴 리팩터 (os.Exit + defer 충돌 해결)
- Mount 패턴 전 핸들러 통일
- **race detector 통과 · lint 0 issues · PR #1 CI 녹색 머지**

**배운 Go 개념**: `testing.T` / `testing.B` · table-driven + `t.Parallel()` · `httptest.NewServer` · `t.Cleanup` · `run() error` 패턴 · golangci-lint v1→v2 마이그레이션

**주요 파일**: 각 패키지의 `*_test.go` · [`.github/workflows/ci.yml`](../.github/workflows/ci.yml) · [`.golangci.yml`](../.golangci.yml)

---

## v0.2 도메인 — 게임 API

### Phase 5 — ガチャ (Gacha) API — 진행 중 (브랜치: `feat/phase-5-gacha`)

**목표**: 가중치 RNG + 천장 시스템 + 이력 기록. 10연 뽑기 트랜잭션 원자성 증명.

**MVP 범위** (이번 PR 포함):
- 도메인: `Card` · `Pool` · `Roll` · `Rarity` (R · SR · SSR) · `PityState`
- 1 개 하드코딩 예시 풀 (embed 데이터, YAML 은 후속)
- 가중치 기반 RNG (`math/rand/v2` + 누적 확률 이분 탐색)
- **천장 시스템**: N 연속 SSR 없으면 다음 roll 확정
- **트랜잭션 원자성**: 10-roll → `gacha_rolls` N행 INSERT + `gacha_pity` UPSERT 단일 트랜잭션
- inmem + MySQL 듀얼 저장소
- sqlc 쿼리 + goose 마이그레이션 (`00002_gacha.sql`)
- 3 REST 엔드포인트 (roll / history / pity)
- 테스트: 확률 분포 (10만회 ±5%) · 천장 트리거 · 트랜잭션 롤백

**스코프 제외** (Phase 5b 또는 후속):
- YAML 풀 설정 파일 로드
- 재화 (jewel/gem) 차감 로직
- 멀티 풀 · 픽업 기간 관리
- 중복 요청 멱등성 (request_id) — Phase 11 Write-Behind 에서

**기술 요소**:
- `math/rand/v2` + 고정 seed (테스트 재현성)
- `github.com/google/uuid` v7 (time-ordered roll ID)
- `sqlc` `Queries` 가 `*sql.Tx` 수용 (`DBTX` 인터페이스)
- 서비스가 `Repository.WithinTx(fn)` 으로 원자 블록 지정

**Go 개념 (첫 등장)**:
- `math/rand/v2` 가중치 뽑기 + 이분 탐색
- UUID v7 (`uuid.NewV7()`)
- `sql.Tx` 트랜잭션 스코프 + sqlc 연동
- Service-level 트랜잭션 추상화 (Repository.WithinTx)
- 통계 테스트 (확률 분포 허용 오차 범위 검증)

**완료 기준 (시연)**:
- `POST /api/gacha/roll {"player":"p1","pool":"demo","count":10}` → 10 카드 + `is_pity` 필드
- `GET /api/gacha/history/p1` → 최근 뽑기 이력 N건
- `GET /api/gacha/pity/p1` → 풀별 천장 카운터
- 트랜잭션 도중 에러 시뮬 → 롤백 후 DB 변화 없음 (테스트)
- 10만회 roll 통계 테스트: R/SR/SSR 비율이 선언 가중치 ±5%

**예상 파일 구조**:
```
internal/domain/gacha/
  ├── gacha.go         Card · Pool · Roll · Rarity · PityState
  ├── errors.go        ErrPoolNotFound · ErrInvalidCount 등
  └── rng.go           WeightedPick 유틸 (누적 가중치)
internal/service/gacha/
  ├── service.go       Service + Repository interface + Roll 메서드
  ├── pool_data.go     하드코딩 데모 풀 1개
  └── service_test.go  확률 분포 + 천장 + 표-주도
internal/persistence/
  ├── inmem/gacha_repo.go
  └── mysql/
      ├── migrations/00002_gacha.sql
      ├── queries/gacha.sql
      └── gacha_repo.go
internal/endpoint/gacha.go + gacha_test.go
```

**의존성**: Phase 1 (REST 기반) · Phase 2 (sqlc + goose)

**추정 규모**: 코드 약 800 줄 + 테스트 약 300 줄 + SQL 약 80 줄

---

### Phase 6 — イベント (Event) API — 대기

**목표**: 이벤트 라이프사이클 (Announcement → Prologue → Live → End) + 포인트 누적.

**기술 요소**:
- Event 모델 (`start_at`, `end_at`, `status`)
- 상태 전이 로직 (시간 기반)
- 플레이어 점수 누적 (`event_scores` 테이블)
- 보상 수령 엔드포인트

**완료 기준 (시연)**:
- `GET /api/event/current` → 진행 중 이벤트 리스트
- `POST /api/event/:id/score` `{"delta":100}` → 누적 반영
- 이벤트 종료 후 수령 가능 보상 조회

**의존성**: Phase 2

**추정 규모**: 약 400 줄

---

### Phase 7 — ランキング (Ranking) API — 대기

**목표**: Redis Sorted Set (ZSET) 기반 실시간 랭킹 + 15s 주기 DB 스냅샷.

**기술 요소**:
- `redis/go-redis/v9` — ZADD, ZRANGE, ZREVRANGE
- `internal/service/leaderboard/redis.go` + `inmem.go` (graceful degrade)
- `internal/job/rankingsnapshot/` — 15s ticker goroutine 으로 DB 백업
- 내 주변 랭킹 조회 (ZRANK + ZRANGE 조합)

**Go 개념 (첫 등장)**:
- Redis 클라이언트 API
- Ticker-based background job
- Graceful degrade 패턴 (REDIS_ADDR 빈 값 → inmem)

**완료 기준 (시연)**:
- `GET /api/ranking/top?n=10` → Redis Top 10
- `GET /api/ranking/me?id=p1` → 내 랭킹 + 주변 ±5
- Redis 컨테이너 없을 때 inmem 으로 동작

**의존성**: Phase 2 + Phase 6 (score 공급)

**추정 규모**: 약 400 줄

---

### Phase 8 — メール (Mail) API — 대기

**목표**: 플레이어 우편함 — 수신 · 수령 · 만료.

**기술 요소**:
- Mail 모델 (`player_id`, `subject`, `body`, `attachments`, `expires_at`, `read_at`)
- 대량 발송 (운영자 API) — 전체 플레이어 bulk INSERT
- 수령 시 첨부 아이템 인벤토리로 이동 (트랜잭션)
- 만료 메일 정리 배치 (Phase 10 에서 잡으로)

**완료 기준 (시연)**:
- `GET /api/mail` → 미수령 메일 리스트
- `POST /api/mail/:id/claim` → 보상 수령 + 읽음 처리
- 만료된 메일은 자동 숨김

**의존성**: Phase 2

**추정 규모**: 약 400 줄

---

## v0.3 운영 — 관측성·비동기

### Phase 9 — Prometheus + pprof + 로그 — 대기

**목표**: 표준 관측 툴체인 정비 + KPI 롤업 잡.

**기술 요소**:
- `prometheus/client_golang` (Counter, Histogram, Gauge)
- HTTP 요청별 Histogram (p50/p95/p99)
- `net/http/pprof` 마운트 (`/debug/pprof/*`)
- 로그 correlation (request_id → slog.With)
- KPI rollup goroutine (1s ticker → `/api/kpi` 엔드포인트)

**완료 기준 (시연)**:
- `GET /metrics` → Prometheus scrape 가능
- `curl /debug/pprof/heap` → 프로필 획득
- 로그에 request_id 전파 확인

**의존성**: Phase 1+

**추정 규모**: 약 300 줄

---

### Phase 10 — 비동기 배치 잡 (errgroup + chan) — 대기

**목표**: 이벤트 집계·만료 메일 정리 등 주기적 백그라운드 잡 틀 확립.

**기술 요소**:
- `golang.org/x/sync/errgroup` — 잡 묶음 관리
- `time.NewTicker` 기반 주기 실행
- 잡 스코프: `JobRunner` 구조체 + `Register(name, interval, fn)`
- 단일 잡 실패 시 전체 잡 취소 로직
- Phase 6 이벤트 상태 전이 잡 · Phase 8 메일 만료 잡 실 적용

**Go 개념 (첫 등장)**:
- `errgroup.Group` 동시 에러 전파
- 잡 생명주기 + ctx 취소 전파

**완료 기준 (시연)**:
- 잡 3개 동시 실행 (이벤트 전이·메일 정리·랭킹 스냅샷)
- 한 잡이 panic 시 다른 잡도 정상 종료
- /api/kpi 에 각 잡 최종 실행 시각 표시

**의존성**: Phase 5-8

**추정 규모**: 약 250 줄

---

### Phase 11 — Write-Behind 파이프라인 — 대기

**목표**: 핫패스에서 DB I/O 제거. 이벤트 점수·가챠 이력처럼 대량 이벤트를 배치로 DB flush.

**기술 요소**:
- `github.com/google/uuid` v7 (time-ordered)
- `chan Record` cap=65536 bounded
- `select default` 로 non-blocking drop-on-full
- Flush worker goroutine — 배치 모아서 `INSERT INTO ... VALUES (),(),(...)`
- 100ms / 1000건 중 먼저 오는 것으로 flush 트리거

**Go 개념 (첫 등장)**:
- Bounded channel + non-blocking send
- 백그라운드 worker 패턴
- UUID v7 time-ordered의 shard key 특성

**완료 기준 (시연)**:
- 부하 테스트 중 DB write lat p99 안정
- 채널 가득 시 drop 카운터 로그 출력

**의존성**: Phase 10

**추정 규모**: 약 300 줄

---

## v0.4 데이터 — Spanner

### Phase 12 — Spanner 듀얼 + hotspot 회피 — 대기

**목표**: Aurora MySQL 에 쓰던 레포지토리를 Spanner 로도 전환 가능. `STORE=mysql|spanner` env 로 선택. **hotspot 회피 shard key 설계**가 차별 포인트.

**기술 요소**:
- `cloud.google.com/go/spanner` + Spanner emulator (`gcr.io/cloud-spanner-emulator/emulator`)
- `internal/persistence/spanner/` 레포지토리 구현
- **Hotspot 회피**: UUID v7 + shard prefix 로 순차 쓰기 분산
- 벤치 표 — MySQL vs Spanner 쓰기 latency 비교

**Go 개념 (첫 등장)**:
- `spanner.Client` · `ReadOnlyTransaction` · `ReadWriteTransaction`
- Spanner 스키마 + `INTERLEAVE IN PARENT`
- Hotspot 이론 + shard key 설계

**완료 기준 (시연)**:
- 동일 REST API 가 MySQL · Spanner 양쪽에서 동작
- README 에 latency 비교 표

**의존성**: Phase 2, 11

**추정 규모**: 약 400 줄

---

## v0.5 배포 — Docker · K8s · Terraform

### Phase 13 — Docker + compose — 대기

**목표**: `docker compose up` 한 방에 전 스택 기동.

**기술 요소**:
- `deploy/docker/server.Dockerfile` — multi-stage + **distroless**
- `deploy/docker/bots.Dockerfile`
- `deploy/compose/docker-compose.yml` — profiles: `default / load / scale`

**완료 기준 (시연)**: `docker compose --profile load up` → 서버 + 봇 + MySQL + Redis 동시 기동

**추정 규모**: Dockerfile 50 줄 + compose 150 줄

---

### Phase 14 — K8s + HPA + Graceful Shutdown — 대기

**목표**: Kubernetes 매니페스트 + SIGTERM drain + preStop hook.

**기술 요소**:
- `deploy/k8s/` — namespace, deployment, service, hpa, secret
- `preStop: sleep 10` + `terminationGracePeriodSeconds: 60`
- `signal.NotifyContext` + `http.Server.Shutdown(drainCtx)` 전면 적용
- Readiness gate (`atomic.Bool` → `/health/ready` 503)
- HPA CPU 70% 기준

**Go 개념 (첫 등장)**:
- `errgroup.WithContext` 전면 (HTTP 서버 + 배치 잡 동시 관리)
- SIGTERM drain 시퀀스

**완료 기준 (시연)**: `kubectl delete pod` 후 진행 중인 요청 drain 완료 영상

**추정 규모**: YAML 300 줄 + Go 100 줄

---

### Phase 15 — Terraform GKE — 대기

**목표**: `terraform apply` 로 GKE 클러스터 + Artifact Registry + VPC + Workload Identity 프로비저닝.

**기술 요소**:
- `deploy/terraform/main.tf` + `gke.tf` + `variables.tf`
- Workload Identity 바인딩
- Artifact Registry 리포 자동 생성
- `tfvars` 로 환경 분리

**완료 기준 (시연)**: 실제 GKE 클러스터 스크린샷 + Pod 동작 확인

**의존성**: Phase 14

**추정 규모**: HCL 250 줄

---

## v0.6 마감 — 부하·AI·문서

### Phase 16 — Locust + k6 부하 시나리오 — 대기

**목표**: 공고 명시 Locust 로 event spike 시뮬 + k6 대칭 제공.

**기술 요소**:
- `deploy/locust/locustfile.py` — 공고 명시 매칭 시그널
- `deploy/k6/scenario.js` — 보조
- `cmd/bots -scenario=even|herd|cluster` CLI 확장
- README 에 부하 결과 표 + GIF

**시나리오 의미**:
- **even**: 균등 분산 접속 (기준선)
- **herd**: 동시 접속 폭주 (thundering herd — 로그인 이벤트)
- **cluster**: 시간대 집중 (이벤트 개시 스파이크 — Colorful Palette 실무 시나리오)

**완료 기준 (시연)**: GIF — cluster 모드 → P99 스파이크 → 핫패스 최적화 효과 비교

**의존성**: Phase 15

**추정 규모**: Python 150 줄 + JS 100 줄 + Go 100 줄

---

### Phase 17 — AI Ops Assistant (LLM + SSE) — 대기

**목표**: 대시보드 "Analyze Spike" 버튼 → SSE 로 자연어 진단 스트리밍.

**기술 요소**:
- `internal/service/llm/` — `Provider` 인터페이스 + `MockLlmProvider` + `OpenAiLlmProvider`
- `internal/service/ops/spike.go` — 텔레메트리 → 프롬프트 빌더 (순수 함수, 테스트 가능)
- `/api/ops/analyze/spike` SSE 엔드포인트
- `text/event-stream` + `http.Flusher`
- 대시보드 버튼 + EventSource

**Go 개념 (첫 등장)**:
- SSE + `http.Flusher` 캐스팅
- Provider 인터페이스 추상화 (Mock/Real 전환)
- 스트리밍 토큰 `<-chan string`

**완료 기준 (시연)**: 대시보드 버튼 → 타이핑 애니메이션처럼 자연어 진단 스트리밍

**의존성**: Phase 9 (메트릭 공급)

**추정 규모**: 약 400 줄 + HTML 수정

---

### Phase 18 — README + Demo GIF + JP 번역 마감 — 대기

**목표**: 제출 가능한 상태.

**체크리스트**:
- [ ] `README.md` 최종 정돈 (GIF, 30초 스크립트)
- [ ] `README.md` (日本語 기본) 최종 정돈 + `README.ko.md` 동기화 (Phase 0-17 전 내용)
- [ ] `docs/MISSION.ja.md` 핵심 섹션 JP 번역
- [ ] 각 Phase demo GIF 녹화 (`demo/*.gif`)
- [ ] 면접 30초 스크립트 다듬기 (한·일 양쪽)
- [ ] GitHub Actions 초록 뱃지 확인
- [ ] 최종 린터 pass 확인

**의존성**: 모든 Phase

**추정 규모**: 번역 + 녹화 (코드 거의 없음)

---

## 학습 추적 — Phase 별 Go 개념 누적

| Phase | 새로 배우는 것 |
|---|---|
| 0 | 모듈 · 패키지 · `:=` · `_` · 에러 반환 · chi · h2c |
| A | protobuf 생성 · oneof 타입 스위치 · `sync.RWMutex` · `defer` · goroutine per conn · `flag` · `signal.NotifyContext` · `select` · 클로저 팩토리 DI |
| B | `sync.Pool` · `atomic.Bool` · `testing.B` + `b.Loop()` · escape analysis |
| **1** | **context 전파 전면 · consumer 측 인터페이스 · DTO 매퍼 · 구조체 메서드 핸들러** |
| **2** | **`sqlx` · `goose` · Transaction scope · `%w` 래핑 · `errors.Is/As` · graceful degrade** |
| **3** | **validator · sentinel errors · 커스텀 에러 타입 · 공통 에러 미들웨어** |
| **4** | **`testing.T` · `httptest` · table-driven + `t.Parallel()` · mock repo** |
| **5** | **`math/rand/v2` · 복잡한 트랜잭션 · 확률 엔진** |
| **7** | **`redis/go-redis/v9` · Ticker-based job · graceful degrade** |
| **9** | **`prometheus/client_golang` · `pprof` · slog.With correlation** |
| **10** | **`errgroup.Group` · 잡 라이프사이클 + ctx 취소** |
| **11** | **Bounded channel + non-blocking send · UUID v7 · Write-Behind 패턴** |
| **12** | **`spanner.Client` · `ReadWriteTransaction` · hotspot 회피 shard key** |
| **14** | **SIGTERM drain · `http.Server.Shutdown` · Readiness gate** |
| **17** | **SSE + `http.Flusher` · 스트리밍 채널 · Provider 추상화** |

---

## 추정 일정

| 마일스톤 | Phase 수 | 누적 일수 (추정) |
|---|---|---|
| 보너스축 ✓ | 3 | 0 (완료) |
| v0.1 기반 | 4 | +7-10 일 |
| v0.2 도메인 | 4 | +10-14 일 |
| v0.3 운영 | 3 | +7-10 일 |
| v0.4 데이터 | 1 | +3-5 일 |
| v0.5 배포 | 3 | +7-10 일 |
| v0.6 마감 | 3 | +5-7 일 |
| **총 완성 예상** | 18 신규 | **약 5-7 주** |

일본어 문서 작성·리뷰 대기·Go 학습 곡선에 따라 변동.

---

## 참조

- [`MISSION.md`](./MISSION.md) — 프로젝트 미션 · 공고 매핑 · 5축 프레임 (SSOT)
- [`archive/PORTING_GUIDE_v1_legacy.md`](./archive/PORTING_GUIDE_v1_legacy.md) — 이전 관점 (아카이브)
- [`../README.md`](../README.md) — 채용 담당자용 메인 (日本語 기본)
- [`../README.ko.md`](../README.ko.md) — 한국어 버전
