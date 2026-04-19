# Locust — Event open spike (cluster)

StageSync 의 **이벤트 개시 스파이크** (T-60s ~ T+30s) 시나리오를 재현하는 부하 스크립트.
PLAN Phase 16 lite.

## 목적

- 동일한 event_id 에 수백 유저가 점수를 몰아 붓는 **랭킹 hot-spot** 재현.
- 3 개 핵심 엔드포인트를 3:2:1 가중치로 병렬 호출 → 복합 부하.
  - `POST /api/event/{id}/score` (×3) — MySQL UPSERT + Redis ZINCRBY 이중 쓰기
  - `POST /api/gacha/roll` (×2) — 단일 트랜잭션 최대 11 INSERT + UPSERT
  - `GET  /api/ranking/{id}/top` (×1) — Redis ZREVRANGE

## 전제

- StageSync 서버 실행 — `docker compose --profile load up --build` 권장.
  `load` 프로파일은 WebSocket bots 까지 동시에 띄워 Room / AOI 경로도 같이 달군다.
- Python 3.10+ 및 `pip install locust` (혹은 `pip install "locust>=2.30"`).

## 실행

### headless (CI / 재현 측정용)

```bash
# 500 유저까지 10 초 램프업, 1 분 유지, HTML 리포트 저장
locust -f deploy/locust/locustfile.py --host http://localhost:5050 \
       --headless -u 500 -r 10 -t 1m --html=locust_report.html
```

### GUI

```bash
locust -f deploy/locust/locustfile.py --host http://localhost:5050
# 브라우저 http://localhost:8089 에서 u/r 지정 후 start
```

### docker compose load profile 과 동시 실행

```bash
# 터미널 1
docker compose --profile load up --build

# 터미널 2
locust -f deploy/locust/locustfile.py --host http://localhost:5050 \
       --headless -u 500 -r 10 -t 1m --html=locust_report.html
```

## 시나리오 의미

| 플래그 | 의미 |
|---|---|
| `-u 500` | 최대 동시 유저 500 |
| `-r 10`  | 초당 10 유저씩 램프업 → 50 초 후 500 유저 도달 |
| `-t 1m`  | 1 분 유지 (또는 램프업 중 마감) |

각 유저는 `wait_time = between(0.1, 1.0)` 의 게임 틱 리듬으로
3 엔드포인트를 3:2:1 가중치로 호출한다.

## 동시 관측 지표

### Locust 자체
- **Total RPS**
- **Latency 분포** (p50 / p95 / p99, max)
- **Failure rate**

### 서버 `/metrics` (Prometheus scrape)
- `http_request_duration_seconds` — `method × path × status` 레이블. PromQL 예:
  ```promql
  histogram_quantile(0.99,
    sum by (le, path) (rate(http_request_duration_seconds_bucket[1m])))
  ```
- `stagesync_room_connected_players` — compose `--profile load` 일 때 bots 가 움직이므로
  Locust 와 상관없이 꾸준히 올라감. CPU bound 판정용 보조 지표.
- `go_goroutines`, `process_cpu_seconds_total` — 포화점 탐지.

## 결과 반영

측정 완료 후 `docs/BENCHMARKS.md` 의 **Locust — Event open spike (cluster 시나리오)**
섹션 표를 채울 것. 템플릿 자리 (`______`) 그대로 두면 됨.
