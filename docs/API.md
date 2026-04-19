# StageSync API Reference

> 본 문서는 현재 구현된 엔드포인트의 계약(contract) 을 요약. 소스 변경 시 이 문서도 함께 갱신.
> OpenAPI YAML 은 제출 후 Phase 19 작업 중 생성 검토. 그 전까지는 이 문서가 SSOT.

**베이스 URL**: `http://localhost:5050` (기본값, `LISTEN_ADDR` 로 변경)
**엔드포인트 수**: 20 개 (Auth 1 · Health/Observability 5 · Profile 2 · Gacha 3 · Event 6 · Ranking 2 · Optimize 1 · WebSocket 1 · pprof)

모든 응답은 `Content-Type: application/json` (바이너리 WebSocket 제외).
공통 에러 응답 포맷 ([`internal/apperror`](../internal/apperror/) 참조):

```json
{
  "code":    "VALIDATION_FAILED | NOT_FOUND | CONFLICT | INTERNAL",
  "message": "human readable",
  "fields":  [
    { "field": "name", "tag": "required", "message": "name is required" }
  ]
}
```

| code | HTTP status |
|---|---|
| `VALIDATION_FAILED` | 400 |
| `NOT_FOUND` | 404 |
| `CONFLICT` | 409 |
| `INTERNAL` | 500 |

---

## 認証 (Auth)

JWT (HS256) 기반. `AUTH_SECRET` 환경변수 설정 시 활성화, 빈 문자열이면 **dev-only pass-through** 로 동작 (기존 테스트 호환).

### `POST /api/auth/login`

⚠️ **개발 · 데모 전용**: 자격증명 검증 없이 `player_id` 만 받아 JWT 발급. 프로덕션은 외부 IdP (Auth0 · Cognito · 자체 SSO) 로 대체 또는 패스워드 검증 로직 추가.

**Body**
```json
{ "player": "p1" }
```

**응답 `200 OK`**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "expires_at": "2026-04-20T12:15:00Z",
  "player_id": "p1"
}
```
- `400 VALIDATION_FAILED` — `player` 누락 또는 너무 긺.
- `500 INTERNAL` — `AUTH_SECRET` 미설정 시 서버가 이 엔드포인트를 비활성화.

### 보호 라우트 — `Authorization: Bearer <token>`

현재 보호되는 엔드포인트 (`RequireAuth` 미들웨어 적용):

- `POST /api/gacha/roll`
- `GET /api/gacha/history/{player}`
- `GET /api/gacha/pity/{player}/{pool}`

요청 예:
```bash
curl -X POST http://localhost:5050/api/gacha/roll \
     -H "Authorization: Bearer $JWT" \
     -H "Content-Type: application/json" \
     -d '{"player":"p1","pool":"demo","count":10}'
```

**인증 에러**
- `401 Unauthorized` + `WWW-Authenticate: Bearer realm="stagesync"`
  ```json
  { "code": "UNAUTHORIZED", "message": "missing or malformed Authorization header" }
  ```
  원인: Authorization 헤더 없음 · Bearer 형식 아님 · 토큰 만료 · 서명 불일치.

**미보호 상태로 두는 이유 (지금은)**
- `/api/profile` (POST/GET): 프로필 생성은 signup 성격이라 공개. 실프로덕션은 여기도 SSO 콜백으로 보호.
- `/api/event/*`: 조회는 공개가 자연스러움. `POST /api/event/{id}/score` 만 분리 보호는 후속 작업 (핸들러 구조 리팩터 필요).
- `/api/ranking/*`: 랭킹 조회는 공개가 게임 UX 표준.
- `/ws/room`: WebSocket 인증은 쿼리 파라미터 토큰 방식으로 별도 설계 필요 (후속).

상세 배경 + 한계 인지: [`TRADEOFFS.md`](./TRADEOFFS.md) 1 번 섹션.

---

## Health & Observability

### `GET /health/live`
프로세스 liveness. 항상 `200 OK` — 프로세스가 살아 있으면 응답.

### `GET /health/ready`
Readiness. 평시에는 `200 OK`. SIGTERM / SIGINT 수신 시 내부 `lifecycle.Readiness` 가 draining 으로 전환되어 `503 Service Unavailable` + `{"ready":false}` 응답. K8s load balancer 가 이 응답을 보고 pod 를 endpoint 에서 제외 → 5 초 후 `srv.Shutdown()` 으로 in-flight 요청 정리.

### `GET /api/metrics`
간이 JSON 메트릭 (대시보드 확인용).
```json
{ "tps": 0, "connectedPlayers": 12, "optimized": true }
```

### `GET /metrics`
**Prometheus scrape endpoint**. 텍스트 포맷 0.0.4.

주요 커스텀 지표:
| 지표 | 타입 | 설명 |
|---|---|---|
| `stagesync_room_connected_players` | Gauge | WebSocket Room 접속자 수 |
| `stagesync_optimize_on` | Gauge | 최적화 경로 활성(1) / 비활성(0) |
| `http_request_duration_seconds` | Histogram | HTTP 요청 지연. 레이블 `method` × `path` (chi RoutePattern) × `status`. 기본 버킷 (5ms..10s). |

+ Go 런타임 (goroutines, GC, heap) · process (cpu, memory rss) collector 기본 장착.

**Grafana PromQL 예**:
```promql
# p99 latency per route
histogram_quantile(0.99, sum by (le, path) (rate(http_request_duration_seconds_bucket[5m])))

# RPS per route
sum by (path) (rate(http_request_duration_seconds_count[1m]))

# 5xx 비율
sum(rate(http_request_duration_seconds_count{status=~"5.."}[5m]))
  / sum(rate(http_request_duration_seconds_count[5m]))
```

### `GET /debug/pprof/*`
Go 표준 `net/http/pprof` — runtime 진단용.

주요 경로:
| 경로 | 용도 |
|---|---|
| `/debug/pprof/` | 인덱스 (사용 가능한 profile 목록) |
| `/debug/pprof/goroutine?debug=1` | 현재 고루틴 스택 |
| `/debug/pprof/heap` | 힙 sampling profile |
| `/debug/pprof/profile?seconds=30` | 30초 CPU profile |
| `/debug/pprof/block` | 블로킹 프로파일 (필요 시 `runtime.SetBlockProfileRate`) |

**주의**
- pprof 라우트는 `REQUEST_TIMEOUT` 미들웨어에서 **제외** — `profile?seconds=30` 같은 장시간 수집이 잘리지 않도록.
- 프로덕션 배포 시 ingress 레벨에서 `/debug/*` 를 내부망 전용으로 제한할 것.

**사용 예**:
```bash
# 30초 CPU profile → go tool pprof 로 분석
curl -o cpu.prof http://localhost:5050/debug/pprof/profile?seconds=30
go tool pprof cpu.prof
```

---

## Profile (プロフィール)

### `POST /api/profile`
**Body**
```json
{ "id": "p1", "name": "alice" }
```
`id` · `name` 필수 (`validator` 검증).

**응답**
- `201 Created` — 신규 생성.
- `409 Conflict` + `code: CONFLICT` — 이미 존재하는 `id`.
- `400 Bad Request` + `code: VALIDATION_FAILED` — 필드 누락/규칙 위반.

### `GET /api/profile/{id}`
**응답**
- `200 OK` — `{"id", "name", "created_at"}` (RFC3339 UTC).
- `404 Not Found` + `code: NOT_FOUND`.

---

## Gacha (ガチャ)

### `POST /api/gacha/roll`
**Body**
```json
{ "player": "p1", "pool": "demo", "count": 10 }
```
- `count` : 1 ~ 10 (범위 밖은 `400 VALIDATION_FAILED`).
- `pool` : 현재 `"demo"` 하나만 유효 (풀 로더는 Phase 5b 에서 YAML 전환 예정).

**응답 `201 Created`** — 생성된 `Roll` 배열:
```json
[
  {
    "id": "0195...",
    "player_id": "p1",
    "pool_id": "demo",
    "card_id": "ssr_01",
    "rarity": "SSR",
    "is_pity": false,
    "pulled_at": "2026-04-19T12:34:56Z"
  }
]
```

**규칙**
- 10-roll 전체는 **원자 트랜잭션**. 실패 시 부분 저장 없음.
- 천장 (pity) 80회 연속 미-SSR 이면 다음 roll 에서 SSR 확정 (`is_pity: true`).
- 자연 SSR 및 천장 발동 시 카운터 리셋.

### `GET /api/gacha/history/{player}?limit=10`
최신 순 뽑기 이력. `limit` 기본 10, 최대 100.

### `GET /api/gacha/pity/{player}/{pool}`
현재 천장 카운터.
```json
{ "player": "p1", "pool": "demo", "counter": 37 }
```

---

## Event (イベント)

이벤트 라이프사이클 (`UPCOMING` → `ONGOING` → `ENDED`) 은 `start_at` / `end_at` 와 현재 시각으로 **derived** (DB 에 저장하지 않음). 점수 누적은 `ONGOING` 상태에서만 허용.

### `POST /api/event`
이벤트 + 보상 티어 한 번에 등록.

**Body**
```json
{
  "id": "ev-2026-spring",
  "name": "Spring Festival",
  "start_at": "2026-04-20T00:00:00Z",
  "end_at":   "2026-05-05T23:59:59Z",
  "rewards": [
    { "tier": 1, "points_required": 10000, "reward_id": "ssr_card_01" },
    { "tier": 2, "points_required": 5000,  "reward_id": "sr_card_01"  }
  ]
}
```
- `start_at < end_at` 필수 (`ErrInvalidWindow`).
- `rewards` 비워둘 수 있음.

**응답**
- `201 Created` — 생성된 Event (상태 포함).
- `409 Conflict` — 중복 `id`.
- `400 Bad Request` — 필드 누락 · 윈도우 역순.

### `GET /api/event/current`
현재 `ONGOING` 인 이벤트 목록.

**응답 `200 OK`**
```json
{
  "count": 1,
  "events": [
    { "id": "ev-...", "name": "...", "status": "ONGOING", "start_at": "...", "end_at": "..." }
  ]
}
```

### `GET /api/event/{id}`
단건 조회 + 현재 시각 기준 상태.

**응답**
- `200 OK` — `{"id", "name", "start_at", "end_at", "status": "UPCOMING|ONGOING|ENDED"}`
- `404 Not Found` — 미존재.

### `POST /api/event/{id}/score`
진행 중 이벤트에 점수 누적 (MySQL UPSERT + Redis ZSET `ZINCRBY` best-effort).

**Body**
```json
{ "player": "p1", "delta": 100 }
```
- `delta` : `1..1_000_000` (과대 악의 악용 방지).

**응답**
- `200 OK` — 누적 후 총점 `{"player_id", "event_id", "points"}`
- `409 Conflict` + `code: CONFLICT` — 이벤트가 `ONGOING` 상태 아님 (`ErrNotOngoing`).
- `400 Bad Request` — delta 범위 밖.
- `404 Not Found` — 이벤트 미존재.

### `GET /api/event/{id}/score/{playerId}`
플레이어 누적 점수 스냅샷. 미반영 플레이어는 `points: 0` 으로 반환 (에러 아님).

### `GET /api/event/{id}/rewards/{playerId}`
현재 점수 기준 획득 가능 보상 + 전체 티어.

**응답 `200 OK`**
```json
{
  "status": "ONGOING",
  "points": 7500,
  "tiers": [ {...}, {...} ],
  "eligible": [ { "tier": 2, ... } ],
  "claimable": false
}
```
- `claimable` 는 이벤트 종료 후에만 `true` (Phase 8 メール 과 연동 시 실제 지급, 현재 스코프 밖).

---

## Ranking (ランキング)

Redis Sorted Set (ZSET) 기반 실시간 랭킹.
`REDIS_ADDR` env 미설정 시 inmem leaderboard 로 graceful degrade (API 동작 동일).
이벤트 점수 반영 (`POST /api/event/{id}/score`) 이 성공하면 자동으로 ZSET 에도 누적.

### `GET /api/ranking/{eventId}/top?n=10`
Top-N 조회. `n` 기본 10, 최대 100.

**응답 `200 OK`**
```json
{
  "event_id": "ev1",
  "count": 3,
  "entries": [
    { "player_id": "alice", "score": 500, "rank": 1 },
    { "player_id": "bob",   "score": 300, "rank": 2 },
    { "player_id": "carol", "score": 100, "rank": 3 }
  ]
}
```

- `400 VALIDATION_FAILED` — `n` 이 1..100 범위 밖.

### `GET /api/ranking/{eventId}/me/{playerId}?radius=5`
해당 플레이어의 순위 + 본인 ±radius 엔트리 (본인 포함).
`radius` 기본 5, 최대 25. 상·하단 경계에서 clamp.

**응답 `200 OK`**
```json
{
  "event_id": "ev1",
  "player_id": "bob",
  "rank": 2,
  "score": 300,
  "radius": 1,
  "entries": [
    { "player_id": "alice", "score": 500, "rank": 1 },
    { "player_id": "bob",   "score": 300, "rank": 2 },
    { "player_id": "carol", "score": 100, "rank": 3 }
  ]
}
```

- `404 NOT_FOUND` — 해당 플레이어가 랭킹에 등재되지 않음 (점수 반영 전).
- `400 VALIDATION_FAILED` — `radius` 가 0..25 범위 밖.

---

## Optimize toggle (ボーナス축)

### `POST /api/optimize`
AOI 경로 전환 (Naive ↔ Pooled).
```json
{ "on": true }
```
`204 No Content` on success.

---

## WebSocket (ボーナス축)

### `GET /ws/room` (Upgrade)
HTTP → WebSocket 업그레이드.

**프레임 포맷**: `MessageBinary` 만 허용. 페이로드는 `roompb.ClientMessage` protobuf.

현재 지원 payload:
- `Move { player_id, x, y }` — 위치 업서트.

브로드캐스트는 아직 미구현 (Phase 이후 확장). 현 단계는 Room 레지스트리 상태 업데이트만.
