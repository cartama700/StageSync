# トレードオフと既知の限界

> **문서 역할**: 4 일 MVP 라는 제약 안에서 **의도적으로 scope-out** 한 영역 + **아키텍처 한계**를 스스로 문서화.
> 면접관·리뷰어가 "이건 왜 안 한 건가?" 라고 지적할 가능성이 높은 5 가지 지점에 대한 자각 · 정답 · 대응을 미리 정리.
> **시니어 시그널**: "한계를 인지하고 정답까지 이미 알고 있음" 을 보여주기 위한 자료.

**참조**: [`PLAN.md`](./PLAN.md) (v3 제외 Phase 근거) · [`PITCH.md`](./PITCH.md) (면접 대본) · [`STATUS.md`](./STATUS.md) (현 스냅샷)

---

## 1. 認証 · 認可 (Auth / Session) — 未実装

### 현상
- `profile` · `gacha` · `event` · `ranking` 핸들러가 요청 파라미터의 `player_id` · `playerId` 를 **그대로 신뢰**.
- JWT 검증 미들웨어 · 세션 도메인 · `auth` 패키지 모두 없음.

### 리스크
- 악의적 클라이언트가 타인의 `player_id` 를 넘기면 **그 유저의 가챠 · 점수 · 프로필 조작 가능**.
- 재화 차감이 구현된 상태라면 **즉시 금전적 악용**.

### 本来の正解 (입사 후 1 순위 착수)
1. **API Gateway (Envoy · Cloud Armor) 에서 JWT 검증** → `X-Player-Id` 헤더로 하위 전달.
2. 또는 `internal/endpoint/middleware` 에 `AuthMiddleware` 추가:
   - `Authorization: Bearer <jwt>` → `jwt.Parse` → Redis 세션 조회 → `context.Context` 에 `PlayerID` 주입.
   - 핸들러는 `auth.PlayerIDFrom(ctx)` 로만 취득. URL 파라미터 · body 의 player_id 는 **무시**.
3. 보너스: `iat` · `exp` · `jti` 검증으로 토큰 재사용 공격 방어.

### 면접 대응
> 「MVP スコープ では **ガチャの原子トランザクション · ランキング ZSET · AOI 最適化** のようなバックエンド **코어 퍼포먼스** 증명에 집중하느라 認証レイヤ を **의도적으로 scope-out** 했습니다.
> 프로덕션이라면 **API Gateway JWT 검증 → `context.Context` 주입** 패턴이 1 순위 추가 항목입니다. 핸들러 레이어는 이미 `context.Context` 전파를 전제로 설계돼 있어, `auth.PlayerIDFrom(ctx)` 한 줄 교체로 마이그레이션 가능합니다.」

---

## 2. 분산 환경의 WebSocket 상태 공유 — 単一ノード前提

### 현상
- [`internal/room/room.go`](../internal/room/room.go) 가 `sync.RWMutex` + `map` 으로 **단일 프로세스 메모리**에 Room 보관.
- [`deploy/k8s/hpa.yaml`](../deploy/k8s/hpa.yaml) 은 HPA `minReplicas: 2, maxReplicas: 10` — 분산 전제.

### 리스크 (이상적 조합이 모순)
- 유저 A 가 Pod-1 · 유저 B 가 Pod-2 에 접속 → **같은 Room 이어도 서로의 Move 이벤트를 못 받음**.

### 本来の正解
- **Redis Pub/Sub** 또는 **NATS** 같은 메시지 브로커로 Pod 간 브로드캐스트.
- 또는 프로젝트 세카이가 실 프로덕션에서 쓰는 **Diarkis** (외부 분산 상태 서버) 에 리얼타임 통신 **위임**.
- PLAN 에 원래 있던 **Phase 21 — 라이브 브로드캐스트 샤딩 랩** 이 이 문제의 정면 해결책이었으나, v3 에서 제외.

### 이 프로젝트의 포지셔닝
- `internal/room/` · `internal/service/aoi/` 는 **단일 노드 내 핫패스 메모리 최적화** (`sync.Pool` 로 0 allocs/op, 2.48× 고속화) 쇼케이스가 목적.
- 분산 확장은 "플라이휠 기반" — 현재 구현이 그 기반을 어떻게 다졌는지가 본 포트폴리오의 증명 대상.

### 면접 대응
> 「現在の Room 実装は **単一ノード内の AOI 메모리 최적화 알고리즘 역량 (0 allocs/op)** を証明するショーケースです.
> 実際に多数の Pod が分散する本番環境なら, **Redis Pub/Sub · NATS** 같은 메시지 브로커로 Pod 間 broadcast 를 붙이거나, プロセカ가 실제로 사용하는 **Diarkis** 같은 외부 분산 상태 서버에 리얼타임 통신을 **위임** 하는 게 정석입니다. PLAN 原案의 Phase 21 이 정확히 그 해결책이었지만 4 일 제약 안에서 스코프 밖으로 뺐습니다.」

---

## 3. 마스터 데이터 인메모리 캐싱 — 하드코딩

### 현상
- [`internal/service/gacha/pool_data.go`](../internal/service/gacha/pool_data.go) 에 demo 풀을 **하드코딩** + `StaticPoolRegistry` 싱글톤.
- 마스터 데이터 로더 · 캐시 · Hot-Reload · 스키마 버전링 부재.

### 리스크
- 리듬게임 특성상 픽업 가챠 · 이벤트가 **주 단위** 로 변경 → 현 구조에선 **서버 코드 수정 + 재배포** 필요.
- RDBMS 직접 조회 방식으로 바꾸면 매 요청마다 쿼리 → 커넥션 풀 병목.

### 本来の正解
1. 서버 부팅 시 **S3 (CSV/JSON) 또는 DB 에서 읽어 서버 메모리 Read-Only 싱글톤에 로드** → O(1) 조회.
2. 관리자 API (`POST /admin/masterdata/reload`) 로 **무중단 Hot-Reload** + `SIGHUP` 수신 시에도 재로드.
3. 마스터 데이터에 `version` 필드 → 로드 시 기록 → Prometheus gauge `masterdata_version` 으로 관측.
4. [`PLAN.md`](./PLAN.md) Phase 5 "후속 이월" 에 **"YAML 풀 설정 파일 로드"** 로 명시되어 있음.

### 면접 대응
> 「現在는 복잡도 감소를 위해 코드 베이스에 두었지만, 실제 게임 서버 라면 마스터 데이터를 **서버 부팅 시점에 읽어 Read-Only 싱글톤 메모리 캐시 (O(1))** 에 로드하는 게 정석이라는 걸 알고 있습니다. 픽업 변경 시엔 관리자 API 로 **무중단 Hot-Reload**, Prometheus 의 `masterdata_version` gauge 로 관측. PLAN 의 Phase 5 후속 이월 항목에 이미 계획 명시되어 있습니다.」

---

## 4. 재화 (Inventory / Wallet) 도메인 결합

### 현상
- 가챠 `Roll` 이 **재화 차감 로직 없이** 카드만 뽑음.
- 독립된 `inventory` · `wallet` · `currency` 도메인 **없음**.
- 실제 프로덕션에선 "가챠 = **재화 차감** + **카드 획득 기록** + **뽑기 이력**" 의 **3 건이 단일 원자 트랜잭션** 이어야 함.

### リスク
- 현 구현은 가챠 확률 엔진 · 원자성 쇼케이스에는 완결적이나, **재화 경제 시스템** 을 다루지 않아 모바일 게임 서버로선 미완성.

### 本来의 正解
- 독립된 `internal/domain/inventory/` + `internal/service/inventory/` 패키지.
- `gacha.Service.Roll` 이 consumer-defined interface 로 `inventory.Wallet` 요구:
  ```go
  type Wallet interface {
      Deduct(ctx context.Context, playerID, currency string, amount int64) error
  }
  ```
- **단일 MySQL 트랜잭션** 안에서 `Deduct` + `InsertRollsAndUpdatePity` 실행 — `sqlc` 의 `DBTX` 인터페이스가 `*sql.Tx` 를 투명하게 수용하므로 이미 지원 가능.
- 실패 시 전체 롤백 → 재화 차감도 취소.

### 면접 대응
> 「Phase 5 에서는 **확률 엔진 + 원자 트랜잭션 자체의 정확성** 을 증명하는 데 집중해 재화 차감은 후속 이월로 분리했습니다. 하지만 현재도 `Roll` 메서드는 이미 **N 건의 INSERT + 1 건의 UPSERT** 를 단일 tx 로 묶는 패턴을 갖고 있기 때문에, **`Wallet.Deduct` 한 단을 같은 tx 에 추가하는 형태의 확장이 rigorous 하게 가능**합니다. sqlc 의 `DBTX` 인터페이스 덕에 `*sql.DB` 와 `*sql.Tx` 어느 쪽에서도 동작한다는 전제가 이미 설계에 녹아있습니다.」

---

## 5. 어뷰징 방어 — Rate Limit / Idempotency — ✅ **MVP 수준 구현 완료 (2026-04-20)**

### 구현 (현재 상태)
- [`internal/ratelimit/`](../internal/ratelimit/) — Token Bucket per identity (`golang.org/x/time/rate` 기반) · TTL sweep 고루틴 · nil 리미터 pass-through
- [`internal/idempotency/`](../internal/idempotency/) — `Store` 인터페이스 + Redis 구현 (`SET NX EX`) + inmem 구현 (lazy expiration + periodic Sweep)
- [`endpoint.RateLimit`](../internal/endpoint/ratelimit.go) 미들웨어 — identity 우선순위: authenticated player → X-Forwarded-For → X-Real-IP → RemoteAddr
- [`endpoint.Idempotency`](../internal/endpoint/idempotency.go) 미들웨어 — write 요청만 적용, GET/HEAD + 헤더 없으면 pass-through, 히트 시 `Idempotency-Replayed: true` 헤더 포함 응답 리플레이
- **Graceful degrade**: `RATE_LIMIT_RPS=0` 또는 `REDIS_ADDR=""` → 미들웨어 자동 비활성
- **적용 범위**: `/api/*` 전역 (chi Group · 공개/보호 구분 없이). auth 전에 평가되므로 현재는 **per-IP** 기반 — authenticated per-player 로 승격은 후속

### 남은 과제
1. **per-player rate limit**: RateLimit 을 RequireAuth 뒤에도 한 번 더 배치하여, 인증된 요청은 per-player 로 엄격히 제한.
2. **Distributed Rate Limit**: 현재 inmem bucket map → Pod 수 × RPS 가 실제 상한. 엄격한 글로벌 제한이 필요하면 Redis `INCR` + TTL 로 교체.
3. **Backoff hints**: 현재 `Retry-After: 1` 고정. token bucket 의 다음 토큰 생성 시각을 계산해 동적 설정 가능.
4. **Idempotency body hashing**: 현재 같은 키면 body 무시. Stripe 는 body 해시도 검증해서 "같은 키에 다른 body" 를 거절 — 추가 강화 여지.

### 면접 대응
> 「**Idempotency-Key + Redis `SET NX`** 미들웨어와 **Token Bucket Rate Limit** 을 `internal/endpoint/` 에 추가했습니다.
> 클라가 10 連ガチャ 버튼을 "따닥" 클릭해도 Idempotency-Key 가 일치하면 DB 까지 도달하지 않고 캐시된 응답이 리플레이됩니다. Rate Limit 은 평시 10 rps · burst 20 으로 identity (authenticated player → X-Forwarded-For → RemoteAddr) 별 독립 버킷.
> 남은 한계는 per-Pod rate limit (분산 환경에서 전체 상한이 Pod 수 × RPS) 과 body 해싱 부재 — 둘 다 Redis `INCR` + body hash 비교로 후속 강화 가능합니다.」

---

## 総評 — 면접 오프닝 브리핑 (한 번에 읽는 버전)

면접관이 위의 지점을 **물어보기 전에**, 본인이 **먼저 짚고 넘어가기**.

### 日本語 (面接メイン)

> 「4 日間という時間的制約の中で、ガチャ確率エンジン · アトミックなトランザクション · Redis ZSET ランキング · AOI 最適化 のような **バックエンドの코어 로직 완성** に集中したため、本来の運用ゲームサーバなら必須となる以下を意図的に スコープアウト しました。
>
> 1. **JWT 인증 · 세션 관리** (`context.Context` 주입 패턴)
> 2. **분산 WebSocket 상태 공유** (Pub/Sub 또는 Diarkis 위임)
> 3. **マスターデータ の In-Memory キャッシュ + Hot-Reload**
> 4. **재화 도메인 분리 + 트랜잭션 경계**
> 5. **Idempotency-Key + Rate Limit** (어뷰징 방어)
>
> アーキテクチャ骨格 は 확장 용이하게 설계 (consumer-defined interface · `context.Context` 전파 · sqlc `DBTX` 추상) 해 두었기 때문에, 입사 후엔 이 5 가지의 추가부터 진행하고 싶습니다.」

### 한국어 (내부 준비용)

> "4 일 제약 안에서 코어 로직 완성에 집중하느라 인증 · 분산 브로드캐스트 · 마스터 데이터 캐시 · 재화 도메인 분리 · 어뷰징 방어의 5 가지를 스코프 밖으로 뺐다. 아키텍처 뼈대는 확장이 쉽게 설계돼 있어서 입사 후 1 순위로 추가할 영역이다."

이 브리핑의 **시그널**:
- "한계 자각" → 시니어 특성
- "아키텍처 정답 이미 인지" → 깊이
- "지금 결정의 근거" → 엔지니어링 판단력
- "다음 액션 선명" → 리더급 오너십

---

## 관련 링크

- [`PLAN.md`](./PLAN.md#스코프-재편-기록-v2--v3-2026-04-19) — v3 스코프 재편 (11 Phase 제외)
- [`PITCH.md`](./PITCH.md) — 30 초 / 2 분 / 5 분 면접 대본
- [`MISSION.md`](./MISSION.md) — 공고 bullet → 포폴 매핑
- [`PORTFOLIO_SCENARIOS.md`](./PORTFOLIO_SCENARIOS.md) — 제출 후 장애 랩 (Phase 19)
- [`SUBMISSION_CHECKLIST.md`](./SUBMISSION_CHECKLIST.md) — 제출 직전 체크리스트
