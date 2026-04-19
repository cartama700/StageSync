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

## 5. 어뷰징 방어 — Rate Limit / Idempotency 未実装

### 현상
- `internal/endpoint/middleware.go` 에 **Request timeout** 만 있음.
- **Rate Limit · 멱등성 키 검증 · CAPTCHA · 리플레이 방어** 모두 없음.

### 리스크
- 클라가 10 연 가챠 버튼을 네트워크 지연 악용해 "따닥" (밀리초 단위 중복 클릭) → 같은 요청 N 번 처리.
- MySQL UPSERT 로 pity 카운터 무결성 자체는 보장되지만 (`ON DUPLICATE KEY UPDATE`), **중복 처리의 DB 부하 자체는 방어 못 함**.

### 本来의 正解
1. **Idempotency Key**:
   - 클라가 `Idempotency-Key: <uuid>` 헤더 포함.
   - 서버 `AuthMiddleware` 뒤에 `IdempotencyMiddleware` 추가.
   - Redis `SET NX EX 60s key:<player>:<uuid> = <response-snapshot>`.
   - 같은 키 재요청 시 저장된 응답 리플레이 → 즉시 응답, DB 까지 내려가지 않음.
2. **Rate Limit**:
   - `go.uber.org/ratelimit` 또는 `x/time/rate` Token Bucket.
   - 유저별 (예: 가챠 1 초당 1 회) + IP 별 (예: 100 rps) 이중.
   - 초과 시 `429 Too Many Requests` + `Retry-After` 헤더.
3. **분산 락** (심각한 경우):
   - Redis `SETNX` 로 짧은 분산 락 (ex. `lock:gacha:p1` TTL 2s) → 한 플레이어당 동시 1 건만 허용.

### 면접 대응
> 「DB 단의 UPSERT 로 **pity 카운터의 정합성은 보장** 되지만, 중복 요청이 DB 까지 내려가는 **낭비와 레이스** 자체는 현 구현이 막지 못합니다.
> 프로덕션이라면 **Idempotency-Key 헤더 + Redis `SET NX`** 로 네트워크 계층에서 중복 제거하고, **Token Bucket Rate Limit** 미들웨어를 `internal/endpoint/middleware.go` 에 추가하는 게 정석입니다. 둘 다 `chi` 미들웨어 패턴에 자연스럽게 녹아들어갑니다.」

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
