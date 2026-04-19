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

## TODO — 추가 벤치마크 (Phase 9·16)

- Gacha 10-roll p99 latency (in-memory vs MySQL 트랜잭션)
- Goose migration 적용 시간 (cold start)
- Prometheus `/metrics` scrape 크기 · 응답시간
- Locust 1k 동시 사용자 시나리오 (Phase 16)
