# Changelog

모든 주요 변경 사항 기록. [Keep a Changelog](https://keepachangelog.com/ko/1.1.0/) 포맷 준수.
세부 Phase 로드맵: [`docs/PLAN.md`](./docs/PLAN.md).

## [Unreleased]

포트폴리오 제출 이후 면접 기간 중 추가 예정 Phase:
- **Phase 19** — HP 동시 차감 데드락 랩 (`SELECT ... FOR UPDATE` 데드락 재현 → 유저별 파티션 큐 직렬화 → v1 vs v2 벤치 비교).

---

## [v0.1] — 2026-04-20 — MVP 제출 버전

> Colorful Palette [BA-09-04a](https://hrmos.co/pages/colorfulpalette/jobs/BA-09-04a) 공고 제출용 MVP.
> 2026-04-17 ~ 2026-04-20 집중 개발 (코드 3 일 + 문서 마감 1 일).
> Phase 0-16 + Phase 18 완료. v3 재편으로 제외된 Phase 는 [`PLAN.md`](./docs/PLAN.md) 참조.

### Added — Phase 18 (README / STATUS / CHANGELOG 마감)
- `README.md` 기술 스택 · Phase 진행 표를 현 상태 (Phase 0-16 완료) 로 갱신.
- `README.ko.md` 를 `README.md` (日本語) 와 완전 동기화.
- `docs/STATUS.md` 최신화 — 19 엔드포인트 · 14/15 MVP · 디렉토리 ↔ Phase 맵.
- `docs/PITCH.md` · `docs/SUBMISSION_CHECKLIST.md` · `docs/demo/README.md` 추가.

### Added — Phase 14 lite (K8s readiness gate + manifests, 2026-04-19)
- **Readiness gate** (`internal/lifecycle/readiness.go`) — `atomic.Bool` 래퍼.
  기본값 `true`, `SetDraining()` 로 `false` 전환. SIGTERM 수신 시 drain → 5s sleep → `srv.Shutdown()`.
- **`/health/ready`** — `State.Ready()==false` 면 503 `{"ready":false}`, 아니면 200.
  `State` 가 nil 이면 항상 200 (테스트 편의).
- **K8s manifests** (`deploy/k8s/`): namespace · configmap · deployment · service · hpa (autoscaling/v2) +
  배포 가이드 README. 시크릿은 별도 생성 (`kubectl create secret generic stagesync-secrets ...`).
- **distroless preStop 노트**: `/bin/sh` · `sleep` 없어 `exec preStop` 불가 → 애플리케이션 내부
  `SetDraining → 5s sleep → Shutdown` 이 대체. `terminationGracePeriodSeconds: 60` 으로 drain 여유 확보.

### Added — Phase 16 lite (Locust event-open-spike 시나리오, 2026-04-19)
- **`deploy/locust/locustfile.py`** — `FastHttpUser` 기반 cluster 시나리오.
  `on_start` 에서 공유 event 하나 생성 (`threading.Lock` 으로 worker 간 1회만) →
  3 task 가중치 `post_score : gacha_roll : ranking_top = 3 : 2 : 1`.
- **`deploy/locust/README.md`** — headless / GUI / `docker compose --profile load` 연동 실행 가이드.
- **`docs/BENCHMARKS.md`**: "Event open spike (cluster 시나리오)" 섹션 추가 (결과 표 · 관찰 포인트 · TODO).
  실제 측정값은 이후 채움.

### Added — Phase 13 (Docker profiles, 2026-04-19)
- **Dockerfile multi-target**: `AS builder` / `AS bots` / `AS server` 3 stage.
  기본 빌드 (`docker build .`) 는 `server` 이미지. `--target bots` 로 봇 이미지.
- **docker-compose `load` 프로파일**: 기본 스택 + `bots-cluster` (50 봇, 5 중심점 군집) +
  `bots-herd` (50 봇, 원점 핫스팟). WebSocket 으로 `server:5050` 에 자동 연결.
- **Makefile 타겟**: `compose-up` / `compose-inmem` / `compose-load` / `compose-down`.

### Added — Phase 9 lite (Histogram + pprof, 2026-04-19)
- **`http_request_duration_seconds`** HistogramVec — `method` × `path` (chi RoutePattern) ×
  `status` 레이블. High cardinality 회피 위해 원시 URL 대신 route pattern 사용.
- **`RequestMetrics` 미들웨어** — `chi.RouteContext(r).RoutePattern()` 을 `next.ServeHTTP`
  이후에 읽어 정확한 pattern 집계.
- **`/debug/pprof/*`** — `chi/middleware.Profiler()` 마운트. `middleware.Timeout` 은
  `r.Group` 내부에만 적용해 `profile?seconds=30` 같은 장시간 수집이 잘리지 않도록 분리.

### Added — Phase 7 (Redis Ranking, 2026-04-19)
- **`GET /api/ranking/{eventId}/top?n=10`** — ZSET Top-N.
- **`GET /api/ranking/{eventId}/me/{playerId}?radius=5`** — 본인 ±radius.
- **`internal/persistence/redis/leaderboard.go`** — ZADD · ZREVRANGE · ZREVRANK · ZINCRBY.
- **`internal/persistence/inmem/leaderboard.go`** — graceful degrade fallback.
  Redis `ZREVRANGE` 의 lex DESC 동점 처리를 inmem 도 동일하게 구현 → 두 구현 동작 일치.
- **Event 서비스 연동**: `WithLeaderboard` 옵션 — `AddScore` 성공 후 ZINCRBY (best-effort).
  Redis 실패 시 warn 로그만 찍고 MySQL(=truth) 은 이미 반영됐으므로 응답은 성공.
- **테스트**: `miniredis` 로 Redis 명령 호환 검증 (실 Redis 불필요).
- **`REDIS_ADDR` env** · docker-compose 에 Redis 서비스 추가.

### Added — Phase 6 (Event API, 2026-04-19)
- **6 REST 엔드포인트**: create · current · get · add-score · get-score · rewards.
- **시간 기반 derived 상태**: UPCOMING/ONGOING/ENDED 를 DB 에 저장하지 않고
  `StatusAt(now)` 로 계산. `WithNow` 옵션 주입으로 상태 전이 테스트.
- **MySQL `ON DUPLICATE KEY UPDATE points = points + VALUES(points)`** — 원자적 누적 UPSERT.
- sqlc + goose `00003_event.sql`.

### Added — 운영 기반 선행 투입 (2026-04-19)
- **Config**: `internal/config` 패키지 — 환경변수 기반 설정 + 유효성 검증.
  `LISTEN_ADDR` · `LOG_LEVEL` · `SHUTDOWN_TIMEOUT` · `REQUEST_TIMEOUT` · `MYSQL_DSN`.
- **Graceful shutdown**: `SIGTERM` / `SIGINT` 수신 시 `http.Server.Shutdown(ctx)` 로
  in-flight 요청 보호 (`SHUTDOWN_TIMEOUT` 기본 15s).
- **Request-scoped structured logging**: `endpoint.RequestLogger` 미들웨어 —
  `chi middleware.RequestID` → slog logger 를 ctx 에 주입하여 핸들러·서비스 레이어에서
  `endpoint.LoggerFrom(ctx)` 로 `request_id` 가 포함된 로거 사용 가능.
  요청 종료 시 access log 1 줄 자동 출력 (status · bytes · duration_ms).
- **Request timeout 미들웨어**: `middleware.Timeout(REQUEST_TIMEOUT)` — 느린 핸들러의
  ctx 에 deadline 주입. 기본 30s.
- **Prometheus `/metrics` 엔드포인트**: `prometheus/client_golang` 기반.
  커스텀 지표 `stagesync_room_connected_players` · `stagesync_optimize_on` +
  Go 런타임 / process collector 기본 탑재.
- **Dockerfile + docker-compose.yml**: 멀티스테이지 (golang:1.26-alpine → distroless/static)
  정적 바이너리 이미지. `docker compose up --build` 한 번으로 server + MySQL 스택 기동.
- **`.env.example`**: Docker / 로컬 실행에 필요한 환경변수 샘플.
- **`cmd/bots` 시나리오 확장**: `-n` (봇 수) / `-scenario=even|herd|cluster` / `-seed` 플래그.
  여러 봇이 동시에 WebSocket 에 연결되어 패턴별 부하 송신.
- **Docs**: `docs/API.md` (엔드포인트 계약) · `docs/BENCHMARKS.md` (AOI 벤치 결과표) ·
  `docs/adr/` (chi · sqlc · h2c 선택 ADR 3 건).
- **테스트 보강**:
  - `internal/domain/gacha/rng_test.go` — `WeightedPick` 경계값 table-driven.
  - `internal/room/room_test.go` — Room 동시성 (1000 고루틴 upsert).
  - `internal/lifecycle/optimize_test.go` — atomic 토글.
  - `internal/persistence/mysql/*_test.go` — `go-sqlmock` 으로 tx rollback · 1062 중복키 매핑 검증.
  - `internal/config/config_test.go`, `internal/endpoint/middleware_test.go`, `prometheus_test.go`.
  - `cmd/bots/main_test.go` — 시나리오 좌표 범위·결정성 검증.

### Changed
- **`cmd/server/main.go`**: 환경변수 직접 읽기 → `config.Load()` 일원화.
  `ListenAndServe` 를 고루틴 분리 + signal-aware 종료 흐름으로 재구성.

### Dependencies
- `+ github.com/prometheus/client_golang` — Prometheus exposition.
- `+ github.com/DATA-DOG/go-sqlmock` (테스트 전용) — MySQL repo 단위 테스트.

---

## [Phase 5] — 2026-04-18 — ガチャ API

### Added
- **`POST /api/gacha/roll`**: 1-10 회 뽑기. 10-roll 은 단일 트랜잭션 원자 저장.
- **천장 시스템**: 80회 연속 미-SSR 시 다음 roll 에서 SSR 확정 (`is_pity: true`).
  자연 SSR / 천장 발동 시 카운터 리셋.
- **`GET /api/gacha/history/{player}`**: 최신순 이력 조회 (limit 1-100).
- **`GET /api/gacha/pity/{player}/{pool}`**: 현재 천장 카운터.
- **풀 레지스트리**: `StaticPoolRegistry` — demo 풀 하드코딩 (Phase 5b 에 YAML 전환 예정).
- **가중치 RNG 추상화**: `RandIntN` 함수 타입 → 결정적 seed 주입 가능 (테스트).

### Tests
- 10,000-샘플 분포 테스트 (SSR 3% / SR 17% / R 80% ±5%).
- 천장 발동 / 미발동 분기 커버.

### Migrations
- `00002_gacha.sql` — `gacha_rolls`, `gacha_pity` 테이블.

**PR**: [#2](https://github.com/cartama700/StageSync/pull/2)

---

## [Phase 4] — 2026-04-18 — 테스트 + 린트 CI

### Added
- `.golangci.yml` v2 스펙 — errcheck · staticcheck · revive · gocritic · bodyclose 등.
- GitHub Actions `ci.yml` — test (race + coverage) · lint · AOI 벤치 (non-blocking).
- `endpoint.Mount(r)` 패턴 통일 — 모든 핸들러가 자기 라우트를 직접 등록.
- `internal/endpoint/*_test.go` — httptest 기반 E2E 테스트.

---

## [Phase 0-3] — 2026-04-17~18 — REST 기반

### Added
- **Phase 0**: chi 라우터 + h2c + `/api/metrics` + `/health/{live,ready}`.
- **Phase 1 (보너스 A)**: `coder/websocket` + protobuf + thread-safe Room.
- **Phase 2**: MySQL + `sqlc` + `goose` 마이그레이션 + `POST/GET /api/profile`.
- **Phase 3**: `go-playground/validator` + `apperror` (code/http 매핑) + `fmt.Errorf("%w")`.
- **보너스 B**: AOI Naive vs Pooled + `sync.Pool` + `POST /api/optimize` 런타임 토글.
