# StageSync Phase 계획 (v2 — REST-first)

> **본 문서의 역할**: 실행 로드맵. 진행 추적 · 학습 트래커 · 의존성 맵.
> 미션·배경·공고 분석은 [MISSION.md](./MISSION.md) 참조.
> **v2 시작**: 2026-04-18 (v1 이식 관점에서 REST-first 로 재편)

---

## 전체 진행 현황

```
보너스축    [━━━━━━] 3/3 ✓ 완료
v0.1 기반   [·······] 0/4
v0.2 도메인 [·······] 0/4
v0.3 운영   [·····]   0/3
v0.4 데이터 [·]       0/1
v0.5 배포   [·····]   0/3
v0.6 마감   [·····]   0/3
            ─────────
총           3/21 = 14%
```

**현 위치**: 보너스축 완료 → **Phase 1 (REST + clean architecture)** 착수 준비
**재편 기록**: 2026-04-18 대전제 재정의 (실시간 중심 → REST 중심). 기존 Phase 1·2 는 보너스축 (A·B) 로 이관.

---

## 마일스톤

| 버전 | Phase | 내러티브 | 진행 |
|---|---|---|---|
| **보너스** | 0, A, B | "기반 + 실시간 프로토콜 + 핫패스 최적화 쇼케이스" | 3/3 ✓ |
| **v0.1 기반** | 1-4 | "clean architecture + MySQL + 테스트 CI 확립" | 0/4 |
| **v0.2 도메인** | 5-8 | "ガチャ·イベント·ランキング·メール 게임 API" | 0/4 |
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

## v0.1 기반 — REST + MySQL + 테스트

### Phase 1 — REST API 기반 + clean architecture — 대기 (다음)

**목표**: handler → service → repository 3-레이어 구조 확립. 첫 실 엔드포인트 `/api/profile/:id` 동작 (inmem repo).

**기술 요소**:
- 디렉토리: `internal/endpoint/`, `internal/service/`, `internal/persistence/inmem/`
- `Server` 구조체 + 메서드 핸들러 (closure factory → 구조체 메서드로 업그레이드)
- consumer-defined interface: `endpoint` 가 `ProfileService interface` 선언
- Profile 도메인 모델 + DTO 매퍼

**Go 개념 (첫 등장)**:
- `context.Context` 첫 파라미터 전면 적용 (R7)
- 레이어 간 consumer 측 인터페이스 선언
- DTO ↔ Model 매핑 패턴
- 구조체 메서드 핸들러 (기존 closure factory 보완)

**완료 기준 (시연)**:
- `GET /api/profile/p1` → 404 (없음)
- `POST /api/profile` `{"id":"p1","name":"sekai"}` → 200
- `GET /api/profile/p1` → 200 `{...}`
- 각 호출마다 handler → service → repo 로그 체인 확인

**의존성**: Phase 0

**추정 규모**: 약 400 줄

---

### Phase 2 — MySQL + goose + 트랜잭션 — 대기

**목표**: MySQL 8 컨테이너에서 플레이어 프로필 실제 영속화. Phase 1 의 inmem repo 를 MySQL repo 로 교체. `ENV MYSQL_DSN` 빈 값이면 inmem fallback (graceful degrade).

**기술 요소**:
- `jmoiron/sqlx` + `go-sql-driver/mysql`
- `pressly/goose` 마이그레이션 — `V001_create_players.sql`
- `internal/persistence/mysql/player_repo.go`
- `internal/persistence/inmem/player_repo.go` (Phase 1 에서 만든 것 유지)
- 트랜잭션 패턴: `*sqlx.Tx` + `defer tx.Rollback()` + `tx.Commit()`
- `docker-compose.override.yml` 로 MySQL 컨테이너 임시 기동 (Phase 13 이전 임시)

**Go 개념 (첫 등장)**:
- `sqlx.DB` · `QueryRowxContext` · `NamedExecContext`
- Transaction scope 패턴
- `fmt.Errorf("insert player: %w", err)` 래핑 (R2) 전면 적용
- `errors.Is` / `errors.As` 에러 식별 (R13)
- env var 기반 구현 스위치 (graceful degrade)

**완료 기준 (시연)**:
- `docker run mysql:8` → `goose up` 적용 → REST API 가 실 DB 에 저장/조회
- 트랜잭션 롤백 케이스 단위 테스트 통과

**의존성**: Phase 1

**추정 규모**: 약 400 줄 + SQL 100 줄

---

### Phase 3 — Validation + 에러 타입 체계 — 대기

**목표**: 입력 검증 · 에러 타입 계층 · HTTP status 매핑 정착. API 가 진짜 "프로덕션스러운" 경계 검증 갖춤.

**기술 요소**:
- `go-playground/validator/v10` 또는 자작 validation
- Sentinel errors (`var ErrPlayerNotFound = errors.New("...")`)
- 에러 타입 계층 — `ValidationError`, `NotFoundError`, `ConflictError` 등
- 공통 에러 미들웨어 — 에러 타입 → HTTP status 코드 매핑
- 응답 포맷 `{"error":{"code":"...","message":"..."}}`

**Go 개념 (첫 등장)**:
- struct tag 기반 validation
- `errors.Is` / `errors.As` 심화 (R13)
- 커스텀 에러 타입 (구조체 + `Error() string` 메서드)
- 공통 에러 미들웨어 패턴

**완료 기준 (시연)**:
- `POST /api/profile {}` → 400 + 필드별 에러 메시지
- `GET /api/profile/nonexistent` → 404 + `{"error":{"code":"PROFILE_NOT_FOUND", ...}}`
- 중복 생성 → 409
- 내부 에러 → 500, 스택 노출 안 함

**의존성**: Phase 2

**추정 규모**: 약 300 줄

---

### Phase 4 — 테스트 + golangci-lint CI — 대기

**목표**: 전체 코드베이스 테스트 커버리지 확보 + GitHub Actions 로 자동화.

**기술 요소**:
- stdlib `testing` + `stretchr/testify/require`
- **Table-driven test + `t.Run + t.Parallel()`** (R4) 전면 적용
- `httptest.NewServer` / `httptest.NewRecorder` in-memory E2E
- Repo mock (inmem repo 가 자연스럽게 mock 역할)
- `.github/workflows/ci.yml` — `go test`, `go test -bench`, `golangci-lint run`
- `golangci-lint` 로컬 실행 + CI 통합 (R3)

**Go 개념 (첫 등장)**:
- `testing.T`, `testing.B`, `t.Run`, `t.Parallel()`
- `httptest` API
- Table-driven 패턴 관용구

**완료 기준 (시연)**:
- `go test ./...` 모두 통과
- GitHub Actions 초록 뱃지
- `golangci-lint run` 0 issues

**의존성**: Phase 1-3

**추정 규모**: 테스트 약 600 줄 + CI yaml 80 줄

---

## v0.2 도메인 — 게임 API

### Phase 5 — ガチャ (Gacha) API — 대기

**목표**: 가중치 RNG + 천장 시스템 + 이력 기록. "10연 뽑기" 시나리오 완주.

**기술 요소**:
- 가챠 풀 정의 (`config/gacha.yml` 또는 DB)
- 가중치 기반 RNG (`math/rand/v2` + `WeightedRand` 유틸)
- **천장 (pity) 시스템** — 일정 횟수 무픽업 시 SSR 확정
- **이력** — `gacha_rolls` 테이블에 기록
- **픽업 캐릭터** — 특정 기간 확률 증폭
- 트랜잭션: 가챠 roll + 플레이어 재화 차감 + 인벤토리 추가가 원자적

**Go 개념 (첫 등장)**:
- `math/rand/v2` 의 `Rand.IntN`, `Float64`
- 복잡한 트랜잭션 스코프 (여러 테이블 동시 수정)

**완료 기준 (시연)**:
- `POST /api/gacha/roll` `{"count":10}` → 10개 아이템 + 천장 카운터 업데이트
- 단위 테스트: 10만회 roll → 확률 분포가 예상값의 ±5% 내
- 중복 roll 방지 (같은 요청 ID 재사용 시 멱등)

**의존성**: Phase 2

**추정 규모**: 약 500 줄

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
