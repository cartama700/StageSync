# StageSync Benchmarks

> 보너스축 (Phase B) AOI 최적화의 정량 근거.
> 재현: `make bench` (또는 `go test -bench=. -benchmem -run=^$ -benchtime=2s -count=3 ./internal/service/aoi/`)

## AOI 필터링 — Naive vs Pooled

한 플레이어의 시야 반경(30.0) 안에 있는 다른 플레이어 1000명을 필터링하는 핫패스.

| 구현 | ns/op | B/op | allocs/op | 비고 |
|---|---|---|---|---|
| `Naive` | ~1580 | 512 | 1 | 매 호출마다 `[]int` 새로 할당 |
| `Pooled` | ~636 | 0 | 0 | `sync.Pool` 로 슬라이스 재사용 + callback 패턴 |

**속도**: 약 **2.48×** (1580 / 636)
**할당**: **512 B → 0 B** (GC 압력 완전 제거)

### 실측 원본 (2026-04-19)

```
goos: windows
goarch: amd64
pkg: github.com/kimsehoon/stagesync/internal/service/aoi
cpu: AMD Ryzen 7 9800X3D 8-Core Processor
BenchmarkNaive-16     1535606    1561 ns/op    512 B/op    1 allocs/op
BenchmarkNaive-16     1495078    1602 ns/op    512 B/op    1 allocs/op
BenchmarkNaive-16     1491817    1604 ns/op    512 B/op    1 allocs/op
BenchmarkPooled-16    3839725     628.5 ns/op    0 B/op    0 allocs/op
BenchmarkPooled-16    3783153     626.1 ns/op    0 B/op    0 allocs/op
BenchmarkPooled-16    3718339     654.6 ns/op    0 B/op    0 allocs/op
```

### 설계 포인트

- `Naive` 의 반환 슬라이스를 패키지 변수에 대입 → escape analysis 가 heap 에 올림을 강제.
  실제 브로드캐스트 / 네트워크 송신 경로와 동등 조건.
- `Pooled` 는 callback 내에서만 슬라이스 참조 → escape 없음 + pool 복귀 가능.
- `sync.Pool` 의 재사용 슬라이스가 이전 호출의 길이만큼 공간을 이미 보유 →
  `append` 가 grow 하지 않아 실제 쓰기 비용까지 절감.

### 적용 정책

런타임에 `POST /api/optimize` 로 경로 전환 가능. 기본은 `off` (Naive) →
관측 대시보드에서 `stagesync_optimize_on` 지표로 on/off 상태 확인.

---

## Locust — Event open spike (cluster 시나리오)

> 2026 프로덕션 트래픽 중 가장 큰 스파이크 타입 (이벤트 개시 ±1 분) 모사.
> 재현 가이드: [`deploy/locust/README.md`](../deploy/locust/README.md)

**시나리오**: 50 → 500 유저 10 초 램프업 → 1 분 유지. 각 유저가 병렬로 3 엔드포인트 호출 (3:2:1 가중치).
**타깃**: `docker compose --profile load` 기준 — server + MySQL + Redis + WebSocket bots 2종 동시 기동.
**측정 도구**: Locust `--headless -t 1m` + Prometheus `/metrics` scrape.

### 결과 (2026-MM-DD 측정 예정)

| 지표 | 값 |
|---|---|
| Total RPS (평균) | ______ |
| Total RPS (peak) | ______ |
| p50 latency | ______ ms |
| p95 latency | ______ ms |
| p99 latency | ______ ms |
| 에러율 | ______ % |
| Prometheus `http_request_duration_seconds` p99 (path="/api/event/{id}/score") | ______ ms |
| Prometheus `http_request_duration_seconds` p99 (path="/api/gacha/roll") | ______ ms |
| `stagesync_room_connected_players` (peak) | ______ |

### 관찰 포인트

- `POST /api/gacha/roll` count=10 이 MySQL 단일 트랜잭션 11 INSERT + UPSERT — Histogram 꼬리 주범일 것.
- `POST /api/event/{id}/score` 가 MySQL UPSERT + Redis ZINCRBY 이중 쓰기 — 두 backend 중 느린 쪽이 병목.
  Redis 미연결 (`REDIS_ADDR` unset) 시 graceful fallback 되는지 비교.
- `GET /api/ranking/{id}/top` 은 ZREVRANGE 0 N-1 — O(log N + N) 이라 p50 < 5 ms 예상.
- `go_goroutines` 증가 기울기 — WebSocket bots + Locust 유저 합산이 CPU bound 되는 지점.

### TODO (측정 이후 채움)

- [ ] 측정 실행 + 위 표 채우기
- [ ] Locust HTML 리포트 스크린샷 → `docs/assets/locust-cluster.png`
- [ ] Grafana 대시보드 p99 그래프 캡처 (Phase 18 demo GIF 용)
- [ ] `REDIS_ADDR` 유/무 비교 (graceful degrade 증명)

---

## Phase 19 — HP 同時減算デッドロック ラボ

> 실 운영에서 겪었던 "한 유저 row 에 트래픽이 쏠리면 락 경합" 장애 패턴을 재현 · 해결 · 벤치.
> 재현 도구: [`cmd/battlebench/main.go`](../cmd/battlebench/main.go) (MySQL 필수)
> 배경 · 설계 논리: [`PORTFOLIO_SCENARIOS.md`](./PORTFOLIO_SCENARIOS.md) #0

**시나리오**: 한 타깃 playerID (예: `boss-1`) 에 100 고루틴이 동시에 `ApplyDamage(10)` 를 호출.
**재현 조건 (권장)**: `SET GLOBAL innodb_lock_wait_timeout = 2` 로 설정하면 v1-naive 의 lock wait 에러가 선명히 재현됨.

### 실행 방법

```bash
# MySQL 기동 (docker-compose 의 mysql 서비스).
docker compose up mysql -d
# 서버 최초 1 회 실행 → goose 가 자동 마이그레이션 (player_hp 테이블 생성).
make run-mysql

# 벤치 실행 — v1-naive
MYSQL_DSN='root:root@tcp(127.0.0.1:3306)/stagesync?parseTime=true' \
  go run ./cmd/battlebench -impl=naive -n=100 -target=boss-1

# 벤치 실행 — v2-queue
MYSQL_DSN='...' go run ./cmd/battlebench -impl=queue -n=100 -target=boss-1
```

Makefile 단축:
```
make battle-bench-naive
make battle-bench-queue
```

### 결과 (2026-MM-DD 측정 예정)

| 지표 | v1-naive (FOR UPDATE) | v2-queue (playerID 단일 워커) |
|---|---|---|
| 처리량 (req/s) | ______ | ______ |
| 성공률 | ______% | ______% (100 기대) |
| lock wait 에러 수 | ______ | 0 (기대) |
| p50 latency | ______ ms | ______ ms |
| p95 latency | ______ ms | ______ ms |
| p99 latency | ______ ms | ______ ms |
| 최종 HP 일치 여부 | 실패 건은 반영 안 됨 | `hp_init - damage × n` 과 정확히 일치 |

### 설계 논리

**v1-naive** — 모든 요청이 MySQL 로 직접 흘러가 `SELECT ... FOR UPDATE` 로 행 락 대기.
- 장점: 구현 단순, 정합성은 MySQL 이 보장.
- 단점: 같은 playerID 에 폭증 시 락 큐 + timeout 에러, p99 급증.

**v2-queue** — `playerID → chan queueReq` 맵 + 전용 워커 고루틴으로 **Go 레벨 직렬화**.
- 장점: DB 락 경합 0, p99 안정, 에러율 0.
- 단점: 단일 프로세스 기준 — 다중 Pod 에선 Pod 별 워커 × DB 락 경합 부활.
  분산 환경은 Redis Stream · NATS 같은 **외부 분산 큐** 로 승격 필요 (향후 v3).

### 서사 (면접용)

> 「실 운영에서 '평균 부하 테스트는 통과했는데 이벤트 개시 때만 특정 유저 row 에 트래픽이 쏠려 lock wait 에러가 쌓이는' 장애를 겪었습니다. 본 랩은 그 상황을 MySQL `FOR UPDATE` + 100 동시 요청으로 재현하고, **경합 지점을 DB 레벨 → Go 레벨로 옮기는** 해결책을 3 단계로 비교합니다.
>
> **배운 것**:
> 1. 부하 테스트는 '평균' 이 아니라 '최악 동시성 시나리오' 로 설계해야 한다.
> 2. 정합성 vs 성능은 트레이드오프가 아니라 **경합 지점을 어디로 옮길지**의 문제.」

### TODO

- [ ] `-dsn` 플래그로 `innodb_lock_wait_timeout=2` 설정해서 v1 에러율 선명히 재현
- [ ] v3-redis-wb (Redis 1차 저장 + bounded channel Write-Behind) 구현 후 재측정
- [ ] Grafana 대시보드 스크린샷을 `docs/assets/phase-19-*.png` 에 저장

---

## TODO — 추가 벤치마크 (Phase 9·16)

- Gacha 10-roll p99 latency (in-memory vs MySQL 트랜잭션)
- Goose migration 적용 시간 (cold start)
- Prometheus `/metrics` scrape 크기 · 응답시간
- Locust 1k 동시 사용자 시나리오 (Phase 16)
