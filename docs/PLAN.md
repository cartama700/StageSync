# StageSync Phase 계획 (v3 — MVP-shipping)

> **본 문서의 역할**: 실행 로드맵. 진행 추적 · 학습 트래커 · 의존성 맵.
> 미션·배경·공고 분석은 [MISSION.md](./MISSION.md) 참조.
> 현재 스냅샷은 [STATUS.md](./STATUS.md) 참조 (PLAN ↔ 실제 리포지토리 대조표).
> **v3 재편**: 2026-04-19 — 7-9 주짜리 풀 계획을 **"제출 가능한 MVP + 포폴 서사 1개"** 로 축소.

---

## 전체 진행 현황

```
보너스축      [━━━━━━] 3/3 ✓ 완료
v0.1 기반     [━━━━━━] 4/4 ✓ 완료
v0.2 도메인   [━━━━━━] 3/3 ✓ 완료 (Phase 5·6·7)
v0.3 운영     [━━━━━━] 1/1 ✓ 완료 (Phase 9 lite)
v0.5 배포     [━━━━━━] 2/2 ✓ 완료 (Phase 13 · 14 lite)
v0.6 마감     [━━━━━━] 2/2 ✓ 완료 (Phase 16 lite · 18)
v0.7 장애랩   [·]         0/1    (Phase 19 · 제출 후)
              ─────────
총             15/15 MVP = 100% ✅
```

**현 위치**: **Phase 18 (문서 마감) ✓ 완료** (2026-04-20) — **제출 가능 상태**
**다음**: 면접 기간 중 **Phase 19** (HP 데드락 랩) 추가.
**재편 기록**: 2026-04-19 v2→v3. 11 개 Phase 를 제외하고 MVP 제출에 필수적인 것만 남김.

---

## 마일스톤

| 버전 | Phase | 내러티브 | 진행 |
|---|---|---|---|
| **보너스** | 0, A, B | "기반 + 실시간 프로토콜 + 핫패스 최적화 쇼케이스" | **3/3 ✓** |
| **v0.1 기반** | 1-4 | "clean architecture + MySQL + 테스트 CI 확립" | **4/4 ✓** |
| **v0.2 도메인** | 5-7 | "ガチャ·イベント·ランキング — 게임 API 3 핵심" | **2/3** (Phase 5·6 ✓) |
| **v0.3 운영 lite** | 9 | "Prometheus Histogram + pprof (기반 완료, 확장만)" | 0/1 |
| **v0.5 배포 lite** | 13, 14 | "Docker profiles 확장 + K8s manifest" | 0/2 |
| **v0.6 마감** | 16, 18 | "Locust 1 시나리오 + README/GIF/JP 마감" | 0/2 |
| **v0.7 장애랩 (제출 후)** | 19 | "HP 데드락 랩 — 포폴 서사 1 개" | 0/1 |

---

## Phase 의존성 그래프

```
[완료 — 보너스축]
Phase 0 뼈대 (chi + h2c) ✓
   ├─ Phase A WebSocket Room ✓
   └─ Phase B AOI + sync.Pool ✓

[완료 — v0.1]
Phase 0 ─→ Phase 1 ─→ Phase 2 ─→ Phase 3 ─→ Phase 4 ✓

[완료 — v0.2 도메인]
Phase 2 ─→ Phase 5 (ガチャ) ✓
        ─→ Phase 6 (イベント) ✓

[대기 — MVP 필수 경로]
Phase 6 ─→ Phase 7 (ランキング + Redis)    ← 공고 명시 스택
                     │
                     ├─→ Phase 9 lite (Histogram + pprof)  ← 기반 완료
                     ├─→ Phase 13 (Docker profiles)        ← 기반 완료
                     │    └─→ Phase 14 lite (K8s manifest)
                     │         └─→ Phase 16 lite (Locust 1 시나리오)
                     └─→ Phase 18 (README + GIF + JP 마감)  ← 모든 MVP 이후

[제출 후 — 포폴 서사]
Phase 19 (HP 데드락 랩)                    ← Phase 2, 5
```

**MVP 크리티컬 패스**: 7 → 9 lite → 13 → 14 lite → 16 lite → 18.
**제외된 11 Phase**: 아래 "스코프 재편" 섹션에 이유와 함께 명시.

---

## 스코프 재편 기록 (v2 → v3, 2026-04-19)

v2 의 25 Phase 는 총 **7-9 주** 짜리 설계였음. 현 시점에서 3 일차인데 이걸 다 수행하면
"반쯤 완성된 Phase" 가 쌓여 포폴 완성도가 떨어짐. 포폴은 **"많이 했다" 보다 "완결된 이야기"** 가 중요.

**제외 기준**:
1. 공고 핵심 스택 (Aurora MySQL · Redis · Docker · K8s · Locust) 증명에 **중복** 되는 것
2. 실제 클라우드 청구서 없이는 어필 부족한 것 (Spanner, Terraform GKE)
3. 기본 스택 증명과 독립적인 부가 기능 (LLM, 메일 시스템)
4. 서사 반복 (장애랩 4 개 중 가장 강력한 1 개 = Phase 19 만 유지)

**제외된 Phase (11 개)**:

| Phase | 원래 내용 | 제외 이유 |
|---|---|---|
| ~~Phase 8~~ | メール API | Phase 5·6 와 CRUD 구조 유사 — 도메인 반복 증명 불필요 |
| ~~Phase 10~~ | 비동기 배치 잡 (errgroup) | 유스케이스가 Phase 8 · 11 · 20 에 의존 — 모두 제외되므로 없어도 무방 |
| ~~Phase 11~~ | Write-Behind 파이프라인 | Phase 19 v3 에서만 쓰이는데 v3 는 post-submission 옵션 |
| ~~Phase 12~~ | Spanner 듀얼 | 에뮬레이터로 가능하나 "실제 운영" 어필 부족 → 입사 후 학습 대상 |
| ~~Phase 15~~ | Terraform GKE | 실제 GCP 없이는 어필 제한 — YAML (Phase 14) 까지로 충분 |
| ~~Phase 17~~ | AI Ops LLM + SSE | 화려하나 공고 본질 업무와 거리 — 노잼 |
| ~~Phase 20~~ | 이벤트 랭킹 Hot-Spot 랩 | Phase 7·11 의존. Phase 11 제외로 함께 제외 |
| ~~Phase 21~~ | 라이브 브로드캐스트 샤딩 랩 | 추정 600 줄 — 제출 일정에 비해 비대 |
| ~~Phase 22~~ | 서킷 브레이커 랩 | Phase 14 full + 15 + 16 전제 — 모두 축소되므로 함께 축소 |

**축소된 Phase (lite 표시)**:
- **Phase 9 → lite**: Histogram + pprof 만. Gauge · collector 는 이미 운영 기반 투입으로 완료.
- **Phase 13 → lite**: profiles (`default / load`) 확장 + bots 이미지만. Dockerfile · compose 는 이미 완료.
- **Phase 14 → lite**: K8s YAML (deployment / service / hpa / secret) 작성. 실 kubectl apply 불필요 — 리뷰어는 YAML 만 확인.
- **Phase 16 → lite**: Locust 파일 1 개 + 결과 스크린샷 1 장. GIF 녹화는 Phase 18 에서 함께.

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

**재활용**: 유지. Phase 16 lite 에서 부하 시나리오 기반.

### Phase B — AOI + 최적화 토글 ✓ 완료 (2026-04-18) — 구 Phase 2

**산출물**: Naive vs Pooled 필터 + `sync.Pool` + `atomic.Bool` 토글 + 벤치 (2.48× · 0 allocs)

**배운 Go 개념**:
- `sync.Pool` 패턴 (Get/Put/New, reset 책임)
- `sync/atomic` (`atomic.Bool`)
- `testing.B` + `b.Loop()` (Go 1.24+)
- `math/rand/v2` + 고정 seed
- callback 패턴 DI (`func Pooled(..., fn func([]int))`)
- escape analysis 함정 (패키지 변수로 escape 강제해야 현실적 벤치)

**파일**: [`internal/service/aoi/`](../internal/service/aoi/), [`internal/lifecycle/optimize.go`](../internal/lifecycle/optimize.go), [`internal/endpoint/optimize.go`](../internal/endpoint/optimize.go)

**재활용**: `sync.Pool` 패턴은 Phase 19 v3 에서 재사용 예정.

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

### Phase 5 — ガチャ (Gacha) API ✓ 완료 (2026-04-18, PR #2 merge)

**산출물**:
- 도메인: `Rarity` (R/SR/SSR) · `Card` · `Pool` · `Roll` · `PityState` + `WeightedPick` (O(N) 누적 가중치)
- 서비스: `Roll()` — pity 가산 → 천장(80회) 판정 → SSR 강제 → 원자적 저장
- `WithRand` / `WithNow` **옵션 패턴 DI** (테스트 결정성 확보)
- inmem (`sync.Mutex` + slice/map) · MySQL **단일 트랜잭션** UPSERT (`defer tx.Rollback` + `tx.Commit`)
- sqlc + goose `00002_gacha.sql` 마이그레이션 · `DBTX` 인터페이스로 `*sql.Tx` 수용
- 3 REST 엔드포인트
  - `POST /api/gacha/roll`
  - `GET  /api/gacha/history/{player_id}`
  - `GET  /api/gacha/pity/{player_id}/{pool}`
- `apperror` 통합 (404 / 400 / 500) · validate tag
- 테스트: **10 000 회 분포 ±5 %** (R 80/SR 17/SSR 3) · 천장 트리거 · 천장 carry-over · 실패 전파 · httptest 핸들러 테이블
- `cmd/server/main.go` — `openMaybeMySQL` + `buildRepos` 분리로 복수 리포지토리 배선 정리

**배운 Go 개념 (첫 등장)**:
- `math/rand/v2` 가중치 뽑기 + 누적 테이블
- `uuid.NewV7()` time-ordered ID
- `*sql.Tx` 트랜잭션 스코프 + sqlc `DBTX` 인터페이스
- 옵션 패턴 (`func(*Service)`) DI — 테스트에서 시계·RNG 주입
- 통계 테스트 (확률 분포 허용 오차 검증)
- `defer tx.Rollback()` + `tx.Commit()` 관용구 (Commit 성공 시 Rollback no-op)

**주요 파일**:
[`internal/domain/gacha/`](../internal/domain/gacha/) · [`internal/service/gacha/`](../internal/service/gacha/) · [`internal/persistence/inmem/gacha_repo.go`](../internal/persistence/inmem/gacha_repo.go) · [`internal/persistence/mysql/gacha_repo.go`](../internal/persistence/mysql/gacha_repo.go) · [`internal/persistence/mysql/migrations/00002_gacha.sql`](../internal/persistence/mysql/migrations/00002_gacha.sql) · [`internal/endpoint/gacha.go`](../internal/endpoint/gacha.go)

**후속 이월 (Phase 5b — 선택적)**:
- YAML 풀 설정 파일 로드 (현재 `pool_data.go` 하드코딩)
- 재화 (jewel/gem) 차감 로직
- 멀티 풀 · 픽업 기간 관리
- 중복 요청 멱등성 (request_id) — v3 스코프에서 제외됨

---

### Phase 6 — イベント (Event) API ✓ 완료 (2026-04-19)

**산출물**:
- 도메인: `Event` (id·name·start_at·end_at) + `Status` (UPCOMING/ONGOING/ENDED, 시간 기반 derived) + `EventScore` + `RewardTier`
- 서비스: `Create` (이벤트+보상 티어 묶음) · `Get` (상태 포함) · `ListCurrent` · `AddScore` (ONGOING 게이팅 + delta 상한) · `GetScore` · `GetRewards` (티어별 eligible + claimable)
- `WithNow` 옵션 패턴 DI — 테스트에서 시계 주입으로 상태 전이 검증
- inmem (`sync.Mutex` + map/slice) · MySQL (UPSERT `ON DUPLICATE KEY UPDATE points = points + VALUES(points)` 원자적 누적)
- sqlc + goose `00003_event.sql` 마이그레이션 + schema.sql 동기화
- 6 REST 엔드포인트
  - `POST /api/event` (생성 + 보상 티어 한 번에)
  - `GET  /api/event/current`
  - `GET  /api/event/{id}`
  - `POST /api/event/{id}/score`
  - `GET  /api/event/{id}/score/{playerId}`
  - `GET  /api/event/{id}/rewards/{playerId}`
- `apperror` 통합 (400 VALIDATION / 404 NOT_FOUND / 409 CONFLICT — 중복 + ErrNotOngoing / 500)
- 테스트: 상태 전이 5-case table · AddScore ONGOING 게이팅 · 누적 검증 · Invalid delta · rewards eligibility end-to-end (공유 repo + 서로 다른 clock) · 핸들러 httptest

**배운 Go 개념 (첫 등장)**:
- 시간 기반 derived 상태 — DB 에 저장하지 않고 `StatusAt(now)` 로 계산
- MySQL `ON DUPLICATE KEY UPDATE ... + VALUES(col)` 원자적 누적 UPSERT
- clock DI 로 시각 분기 테스트 (`WithNow(fixedClock(t))`)
- `validate:"dive"` — 중첩 slice struct 필드 검증

**주요 파일**:
[`internal/domain/event/`](../internal/domain/event/) · [`internal/service/event/`](../internal/service/event/) · [`internal/persistence/inmem/event_repo.go`](../internal/persistence/inmem/event_repo.go) · [`internal/persistence/mysql/event_repo.go`](../internal/persistence/mysql/event_repo.go) · [`internal/persistence/mysql/migrations/00003_event.sql`](../internal/persistence/mysql/migrations/00003_event.sql) · [`internal/endpoint/event.go`](../internal/endpoint/event.go)

**후속 이월**:
- 실제 claim 플로우 (인벤토리 이동) → v3 에서 제외 (Phase 8 메일 스킵)
- 상태 전이 잡 → v3 에서 제외 (Phase 10 스킵 — 시간 기반 derived 로 충분)

---

### Phase 7 — ランキング (Ranking) API — **다음 진행 (MVP 필수)**

**목표**: Redis Sorted Set (ZSET) 기반 실시간 랭킹 + graceful degrade (Redis 없으면 inmem).

**왜 MVP 필수**: 공고가 Redis 를 **명시 스택**으로 요구. 현 스택에서 빠진 가장 큰 구멍.

**기술 요소**:
- `redis/go-redis/v9` — ZADD, ZRANGE, ZREVRANGE, ZRANK
- `internal/service/leaderboard/` — `Repository` 인터페이스 · Redis 구현 · inmem 구현
- `REDIS_ADDR` env 유무로 graceful degrade (Phase 2 의 `MYSQL_DSN` 패턴 그대로)
- 내 주변 랭킹 조회 (ZRANK + ZRANGE 조합)
- Phase 6 `AddScore` 와 연동 — 이벤트 점수 반영 시 Redis ZSET 도 업데이트

**Go 개념 (첫 등장)**:
- Redis 클라이언트 API (go-redis/v9)
- Graceful degrade 패턴 (REDIS_ADDR 빈 값 → inmem)

**완료 기준 (시연)**:
- `GET /api/ranking/{eventId}/top?n=10` → Top 10
- `GET /api/ranking/{eventId}/me/{playerId}` → 내 순위 + 주변 ±5
- `docker compose` 에 Redis 서비스 추가
- Redis 컨테이너 없이 `go run ./cmd/server` 하면 inmem 으로 자동 fallback

**의존성**: Phase 2 (MySQL — 선택적 스냅샷) + Phase 6 (score 공급)

**추정 규모**: 약 400 줄 (Go) + docker-compose 갱신

---

## v0.3 운영 — 관측성 (축소됨)

### Phase 9 lite — Histogram + pprof 추가 — 대기

**왜 lite**: 기반 (`/metrics` Prometheus exposition · 커스텀 Gauge · Go runtime collector · `request_id` slog 전파) 은 **2026-04-19 운영 기반 선행 투입** 에서 이미 완료. 남은 건 HTTP request 계측과 pprof 둘.

**목표**: 요청별 latency 관측 + 런타임 진단 툴 노출.

**기술 요소**:
- `prometheus.HistogramVec` — `method` × `path` × `status` 레이블
- 기존 `RequestLogger` 미들웨어 뒤에 Histogram 관측 미들웨어 추가
- `net/http/pprof` 마운트 (`/debug/pprof/*`) — 프로덕션에선 내부망 한정
- (선택) `/debug/pprof/*` 경로를 `middleware.Timeout` 에서 제외

**Go 개념 (첫 등장)**:
- `prometheus.HistogramVec` 버킷 설계 (p50/p95/p99 가 유의미한 범위)
- `net/http/pprof` 의 side-effect import

**완료 기준 (시연)**:
- `GET /metrics` 에서 `http_request_duration_seconds_bucket{...}` 노출
- `curl http://localhost:5050/debug/pprof/heap` → pprof 프로필 획득
- README `docs/API.md` 업데이트

**의존성**: 운영 기반 (2026-04-19 완료분)

**추정 규모**: 약 150 줄

---

## v0.5 배포 — Docker · K8s (축소됨)

### Phase 13 — Docker profiles 확장 — 대기

**왜 남음**: 기본 Dockerfile + docker-compose (`server` + `mysql` + `server-inmem`) 는 운영 기반 선행 투입에서 완료. 남은 건 profile 확장 + bots 이미지.

**목표**: `docker compose --profile load up` 한 방에 서버 + bots + MySQL + Redis 동시 기동.

**기술 요소**:
- `Dockerfile.bots` — cmd/bots 용 별도 이미지 (또는 동일 멀티타겟)
- `docker-compose.yml` profile 확장:
  - `default` — server + MySQL (현재와 동일)
  - `load` — + bots (herd/cluster 시나리오) + Redis
- Redis 서비스 추가 (Phase 7 과 동반)

**완료 기준 (시연)**:
- `docker compose --profile load up --build` → 4 컨테이너 모두 Ready + bots 가 server 로 연결됨

**의존성**: Phase 7 (Redis compose 서비스 필요)

**추정 규모**: Dockerfile 20 줄 + compose 30 줄

---

### Phase 14 lite — K8s Manifest (작성만) — 대기

**왜 lite**: 실제 클러스터 없이 **YAML 파일 + kustomize 구조 + preStop 스크립트** 만 제공. 리뷰어는 YAML 검증 + 설명 문서로 K8s 이해도를 확인하면 충분.

**목표**: GKE 배포 가능한 최소 매니페스트 셋 + graceful drain 검증.

**기술 요소**:
- `deploy/k8s/` — `namespace.yaml` · `deployment.yaml` · `service.yaml` · `hpa.yaml` · `configmap.yaml`
- `preStop: sleep 10` + `terminationGracePeriodSeconds: 60`
- Readiness gate — `atomic.Bool` 을 `/health/ready` 로 연결 (drain 시작 시 503)
- HPA CPU 70%, min 2 / max 10
- `README` 에 "kubectl apply 하면 어떻게 동작하는지" 설명 (실 배포 없어도 이해 가능)

**Go 개념 (첫 등장)**:
- SIGTERM → Ready=false → 기존 요청 drain → Shutdown 시퀀스
- `atomic.Bool` 로 런타임 readiness 게이트 제어

**완료 기준**:
- `kubectl apply --dry-run=client -f deploy/k8s/` 통과
- `/health/ready` 가 drain 중 503 을 반환하는 **유닛 테스트** (실 K8s 없이)

**의존성**: 운영 기반 (graceful shutdown 완료분) · Phase 13

**추정 규모**: YAML 200 줄 + Go 50 줄

---

## v0.6 마감 — 부하·문서

### Phase 16 lite — Locust 1 시나리오 — 대기

**왜 lite**: Locust 파일 1 개 + 결과 스크린샷 1 장이면 "Locust 할 줄 안다" 증명에 충분. k6 대칭 · 3 시나리오 전체 · GIF 는 스킵.

**목표**: 공고 명시 Locust 사용 증명 + 핫패스 성능 1 그래프.

**기술 요소**:
- `deploy/locust/locustfile.py` — **cluster 시나리오 1 개** (이벤트 개시 스파이크 모사)
  - 50 → 500 유저 램프업 → 30초 유지 → 드롭
  - `/api/gacha/roll` + `/api/event/{id}/score` + `/api/ranking/{eventId}/top` 3 엔드포인트
- `docs/BENCHMARKS.md` 에 "Locust cluster" 섹션 추가 — p50/p95/p99 표 + 스크린샷

**완료 기준 (시연)**:
- `locust -f deploy/locust/locustfile.py --host http://localhost:5050 --headless -u 500 -r 10 --run-time 1m`
- `BENCHMARKS.md` 에 결과 표 + Locust 웹 UI 스크린샷 1 장

**의존성**: Phase 7 (ranking 엔드포인트 필요) · Phase 13 (docker compose up 으로 실행 기반)

**추정 규모**: Python 80 줄 + 스크린샷 + docs 갱신

---

### Phase 18 — README + Demo GIF + JP/ko 동기화 — 대기 (제출 직전 필수)

**목표**: 제출 가능한 상태.

**체크리스트**:
- [ ] `README.md` 최상단 30초 스크립트 (JP 기본) 다듬기
- [ ] `README.ko.md` 동기화 (현재 `README.md` 가 JP/ko 격차 있음)
- [ ] 데모 GIF 1-2 개 녹화
  - (1) `docker compose up` → `curl profile + gacha + event + ranking`
  - (2) `/metrics` 확인 + Locust 부하 스샷
- [ ] GIF 를 `docs/demo/` 에 배치 후 README 에 embed
- [ ] `docs/STATUS.md` 최종 갱신 (Phase 7-16 완료 반영)
- [ ] `CHANGELOG.md` unreleased → v0.1 태그
- [ ] GitHub Actions 초록 뱃지 확인
- [ ] 최종 `golangci-lint` pass 확인
- [ ] 30초 자기소개 스크립트 (JP 중심 · ko 보조)

**의존성**: 모든 MVP Phase (7, 9 lite, 13, 14 lite, 16 lite)

**추정 규모**: 번역 + 녹화 (코드 거의 없음) · 약 1 일

---

## v0.7 장애 시나리오 랩 (제출 후 서사 추가)

> v3 에서는 **Phase 19 만 유지**. 제출 후 면접 대기 기간에 작업 → "최근 업데이트" 로 어필.

### Phase 19 — HP 동시 차감 데드락 랩 — **제출 후 작업**

**왜 Phase 19 만 남겼나**:
- 공고 필수 역량 중 "高負荷 경험" 을 증명하는 유일한 Phase
- Phase 2 (MySQL) · Phase 5 (트랜잭션 관용구) 만 전제 — 독립적으로 구현 가능
- 서사 구조 (`v1 naive → v2 queue → v3 wb`) 가 면접에서 바로 3 단 이야기로 써짐
- Phase 20·21·22 는 의존 Phase (7·11·A·B·14·15·16) 가 커서 "가성비" 떨어짐

**목표 (포폴 서사)**: 실시간 대전에서 한 유저에게 집중되는 쓰기 경합 → 행 잠금 데드락을 재현하고, 유저별 큐 직렬화로 해결.

**배경 (실제 경험)**:
- 부하 테스트(평균)는 통과, 운영에서 예상 초과 동시 요청으로 데드락 발생
- 유저 데이터 파편화 + 중간 합류 프로젝트 → 아키텍처 근본 변경 불가
- 제약 하 최선책 = 메시지 큐로 유저별 쓰기 직렬화

**구현 3 단**:
- **v1-naive**: `SELECT ... FOR UPDATE` 기반 직렬 잠금 — 데드락 재현
- **v2-queue**: 유저별 파티션 채널 (in-process) 또는 Redis Stream 으로 쓰기 직렬화
- **v3-redis-wb**: Redis 1차 저장 + bounded channel Write-Behind 플러시
  (v3 는 시간 있으면만 — v1/v2 만으로도 서사 완결)

**기술 요소**:
- 전투(Battle) 도메인 최소 구현 — `player_hp` 테이블 + `ApplyDamage(playerID, dmg)`
- `internal/service/battle/` — 3 구현 스왑
- k6 또는 Locust 시나리오: 한 타깃 유저에 N 명 동시 공격
- `docs/BENCHMARKS.md` 에 3 구현 벤치 비교 표

**Go 개념 (첫 등장)**:
- `SELECT ... FOR UPDATE` · 행 락 · 데드락 재현 조건
- 유저별 파티션 채널 패턴 (`map[playerID]chan Command`)
- Bounded channel + non-blocking send (v3 에서)

**완료 기준 (시연)**:
- v1 에서 데드락 로그 재현 가능
- v2 로 데드락 0 + p99 안정
- README 에 v1 vs v2 벤치 비교 표 (v3 는 옵션)

**의존성**: Phase 2 (MySQL), Phase 5 (트랜잭션 관용구)

**추정 규모**: 약 400 줄 (v1 + v2) + 부하 스크립트

**배운 것 (서사 포인트)**:
1. 부하 테스트는 '평균' 이 아니라 '최악 동시성 시나리오' 로 설계해야 한다
2. 정합성 vs 성능은 트레이드오프가 아니라 **경합 지점을 어디로 옮길지**의 문제다

---

## 학습 추적 — Phase 별 Go 개념 누적

| Phase | 새로 배우는 것 | 상태 |
|---|---|---|
| 0 | 모듈 · 패키지 · `:=` · `_` · 에러 반환 · chi · h2c | ✓ |
| A | protobuf 생성 · oneof 타입 스위치 · `sync.RWMutex` · `defer` · goroutine per conn · `flag` · `signal.NotifyContext` · `select` · 클로저 팩토리 DI | ✓ |
| B | `sync.Pool` · `atomic.Bool` · `testing.B` + `b.Loop()` · escape analysis | ✓ |
| **1** | **context 전파 전면 · consumer 측 인터페이스 · DTO 매퍼 · 구조체 메서드 핸들러** | ✓ |
| **2** | **`database/sql` · `goose` · Transaction scope · `%w` 래핑 · `errors.Is/As` · graceful degrade** | ✓ |
| **3** | **validator · sentinel errors · 커스텀 에러 타입 · 공통 에러 미들웨어** | ✓ |
| **4** | **`testing.T` · `httptest` · table-driven + `t.Parallel()` · mock repo** | ✓ |
| **5** | **`math/rand/v2` · 복잡한 트랜잭션 · 확률 엔진 · 옵션 패턴 DI** | ✓ |
| **6** | **시간 기반 derived 상태 · `ON DUPLICATE KEY UPDATE ... + VALUES()` 누적 UPSERT · clock DI · `validate:"dive"`** | ✓ |
| **운영 기반** | **slog with context · chi middleware 조립 · signal.NotifyContext + srv.Shutdown · `prometheus.GaugeFunc`** | ✓ |
| **7** | **`redis/go-redis/v9` ZSET · graceful degrade (REDIS 유무)** | ⏳ |
| **9 lite** | **`prometheus.HistogramVec` · `net/http/pprof`** | ⏳ |
| **14 lite** | **readiness gate · SIGTERM drain 시퀀스 · K8s manifest 구조** | ⏳ |
| **19 (post)** | **`SELECT ... FOR UPDATE` · 행 락 · 유저별 파티션 채널 패턴** | ⏳ |

---

## 추정 일정 (v3 재편 후)

| 마일스톤 | Phase 수 | 누적 일수 (추정) | 상태 |
|---|---|---|---|
| 보너스축 | 3 | — | ✓ 완료 |
| v0.1 기반 | 4 | — | ✓ 완료 |
| v0.2 도메인 (5, 6) | 2 | — | ✓ 완료 |
| **운영 기반 선행** | (Phase 9/13/14 일부) | — | ✓ 완료 |
| **Phase 7 (Redis 랭킹)** | 1 | **+1-2 일** | ⏳ |
| **Phase 9 lite (Histogram + pprof)** | 1 | **+0.5 일** | ⏳ |
| **Phase 13 (Docker profiles)** | 1 | **+0.5 일** | ⏳ |
| **Phase 14 lite (K8s manifest)** | 1 | **+0.5 일** | ⏳ |
| **Phase 16 lite (Locust 1개)** | 1 | **+0.5 일** | ⏳ |
| **Phase 18 (README/GIF/JP 마감)** | 1 | **+1 일** | ⏳ |
| **MVP 제출 준비 완료** | — | **약 4-5 일** | — |
| 제출 후 추가 | Phase 19 | +2-3 일 | 선택적 |

**타임라인 (실제)**:
- **2026-04-17 (목)**: Phase 0-3 + 보너스축 (A/B) + 초기 인프라 ✓
- **2026-04-18 (금)**: Phase 4 (CI) + Phase 5 (가챠, PR #2) ✓
- **2026-04-19 (토)**: Phase 6 + 운영 기반 + Phase 7 + 9 lite + 13 + 14 lite + 16 lite + v2→v3 재편 ✓
- **2026-04-20 (일)**: Phase 18 (README/ko sync/STATUS/PITCH/CHANGELOG v0.1) ✓ — **제출 가능 상태**
- **2026-04-21 이후**: Phase 19 추가 (면접 대기 기간)

**실측**: 4 일 만에 15 Phase MVP 완료. 당초 v2 의 7-9 주 계획을 v3 재편 + AI 협업으로 단축.

---

## 참조

- [`MISSION.md`](./MISSION.md) — 프로젝트 미션 · 공고 매핑 · 5축 프레임 (SSOT)
- [`STATUS.md`](./STATUS.md) — 현 시점 스냅샷 (PLAN ↔ 리포지토리 대조)
- [`API.md`](./API.md) — 엔드포인트 계약
- [`BENCHMARKS.md`](./BENCHMARKS.md) — 실측 데이터
- [`adr/`](./adr/) — 기술 선택 기록
- [`archive/PORTING_GUIDE_v1_legacy.md`](./archive/PORTING_GUIDE_v1_legacy.md) — 이전 관점 (아카이브)
- [`../README.md`](../README.md) — 채용 담당자용 메인 (日本語 기본)
- [`../README.ko.md`](../README.ko.md) — 한국어 버전
- [`../CHANGELOG.md`](../CHANGELOG.md) — 완료된 변경 이력
