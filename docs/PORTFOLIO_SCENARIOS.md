# 포트폴리오 서사 — 장애 시나리오 랩

> **문서 역할**: 제출 후 면접 기간 중 추가할 서사적 Phase 의 구현 설계.
> **v3 방침**: 원래 4 개 시나리오 모두 계획했으나, **Phase 19 (HP 데드락 랩) 1 개만** 실제 구현.
> 나머지 3 개 (#1 라이브 브로드캐스트 · #2 이벤트 랭킹 Hot-Spot · #3 서킷 브레이커) 는
> PLAN v3 재편에서 제외 ([`PLAN.md`](./PLAN.md#스코프-재편-기록-v2--v3-2026-04-19) 참조).
> 기획 의도와 실무 경험 근거는 아래에 보존 — 면접 시 "생각은 했지만 일정상 뺐다" 근거로 활용.

---

## ✅ 구현 예정 — Phase 19: HP 동시 차감 데드락 랩 (제출 후)

### 배경 (실제 경험 기반)

실시간 대전에서 한 유저에게 여러 공격이 동시에 들어올 때, HP / 스테이터스 차감이 동일 행에 집중.
부하 테스트 (평균 시나리오) 는 통과했지만, **운영에서 예상을 넘는 동시 요청으로 행 잠금 대기 → 데드락**이 연쇄적으로 터짐.

### 제약 (왜 근본 해결 불가였나)
- 유저 상태가 여러 서버에 파편화 → 인메모리 단일 소스로 전환 불가
- 중간 합류 프로젝트 → 아키텍처 근본 변경 불가

### 해결 흐름 (StageSync 에서 재현)

**v1-naive**: `SELECT ... FOR UPDATE` 기반 직렬 잠금 — **데드락 재현**
```sql
BEGIN;
SELECT hp FROM player_hp WHERE player_id = ? FOR UPDATE;
UPDATE player_hp SET hp = hp - ? WHERE player_id = ?;
COMMIT;
```
→ 동일 `player_id` 에 병렬 요청 N 개가 들어오면 대기 큐 폭주 · 일부는 타임아웃 · 일부는 데드락 로그.

**v2-queue**: 유저별 파티션 채널 (in-process) 로 쓰기 직렬화.
```go
type Shard struct {
    cmds map[string]chan Cmd  // playerID -> commands
}
// 한 player 의 모든 damage 가 같은 고루틴에서 직렬 처리 → 동시성 제거
```
→ DB 에는 단일 라이터만 상대. 데드락 0 건. p99 안정화.

**v3-redis-wb** (시간 여유 있으면): Redis 1차 상태 + bounded channel Write-Behind.
→ 핫패스 에서 DB I/O 제거. Phase 11 Write-Behind 와 동일한 패턴 도입.

### 성공 기준 (시연)
- v1 에서 `ERROR 1213: Deadlock found` 로그 재현 가능
- v2 로 데드락 0 + p99 안정 (20ms 이하)
- `docs/BENCHMARKS.md` 에 v1 vs v2 비교 표 (RPS · p50/p95/p99 · 데드락 수)
- k6 또는 Locust 시나리오: 한 타깃 유저에 N 명 동시 공격

### 배운 것 (서사 포인트)
1. 부하 테스트는 **'평균' 이 아니라 '최악 동시성 시나리오'** 로 설계해야 한다
2. 정합성 vs 성능은 트레이드오프가 아니라 **경합 지점을 어디로 옮길지** 의 문제다

### 의존성
Phase 2 (MySQL) · Phase 5 (트랜잭션 관용구) — 이미 완료. 독립적으로 착수 가능.

**추정 규모**: v1 + v2 약 400 줄 (Go) + 부하 스크립트 100 줄. 2-3 일.

상세: [`PLAN.md`](./PLAN.md#phase-19--hp-동시-차감-데드락-랩--제출-후-작업) Phase 19 섹션.

---

## 🚫 v3 에서 제외한 시나리오 (기획 의도 보존)

> 아래 3 개 시나리오는 v2 PLAN 에서 각각 Phase 20 · 21 · 22 로 계획했으나 v3 축소 시 제외.
> 기획 수준에서는 완결되어 있으므로 면접에서 "이런 방향도 생각했다" 근거로 활용 가능.

### #1 이벤트 종료 직전 랭킹 핫스팟 (구 Phase 20)

**이슈**: 이벤트 종료 1 분 전 트래픽 ×수십 배 → 랭킹 테이블 Hot-Spot → 점수 누락.

**해법 요약**: DB 직접 UPDATE → Redis ZSET 즉시 반영 + DB 배치 flush (Write-Behind) → 종료 ±N 초 이벤트를 별도 큐 수집 후 정산.

**v3 제외 이유**: Phase 7 (ZSET 랭킹) + Phase 11 (Write-Behind) 양쪽 의존. Phase 11 을 제외하면서 의존 체인이 끊어짐.

**그러나 이미 기반은 있음**:
- Phase 7 의 Redis ZSET 이 준비됨
- Event 서비스의 `ZINCRBY` best-effort 가 Write-Behind 의 단순화 버전

---

### #2 버추얼 라이브 브로드캐스트 샤딩 (구 Phase 21)

**이슈**: 수만 명이 동일 가상 공간에 접속 · 상태 동기화 실패 (실제 プロセカ 운영 사례 참고).

**해법 요약**: AOI 기반 Pub/Sub · 룸 샤딩 · 커넥션 admission 티켓 큐.

**v3 제외 이유**: 추정 600 줄. 제출 일정 대비 비대.

**그러나 기반은 있음**:
- Phase A (WebSocket Room + protobuf)
- Phase B (AOI + sync.Pool)
- Phase 16 의 `cmd/bots` (even / herd / cluster 시나리오 → admission 시뮬 기반)

---

### #3 오토스케일 지연 + 서킷 브레이커 (구 Phase 22)

**이슈**: 가챠/이벤트 오픈 시각 → 파드 warm-up 지연 + DB 커넥션 풀 포화 → 연쇄 붕괴.

**해법 요약**: Scheduled Scaling (CronHPA) + 종속성 서킷 브레이커 (`sony/gobreaker`) + 비핵심 기능 차등 축소.

**v3 제외 이유**: Phase 14 full + 15 (Terraform) + 16 (Locust 3 시나리오) 전제. v3 에서 모두 축소 또는 제외됨.

**그러나 기반은 있음**:
- Phase 14 lite 의 readiness gate
- Prometheus `/metrics` 의 `http_request_duration_seconds` (5xx 전파 관측 가능)

---

## 서사 가치 (왜 Phase 19 를 선택했나)

면접 시 "장애 대응 경험" 질문에 답할 수 있는 **가장 강력한 1 개** 선정 기준:

| 시나리오 | 구현 난이도 | 서사 완성도 | 기존 의존 Phase | 선택 |
|---|---|---|---|---|
| **#0 Phase 19 HP 데드락** | 중 (400 줄) | **매우 높음** — 실무 경험 기반 | Phase 2, 5 (둘 다 완료) | ✅ |
| #1 랭킹 Hot-Spot | 중 | 높음 | Phase 7 완료 but 11 제외 | ❌ |
| #2 라이브 브로드캐스트 | **매우 높음** (600 줄) | 높음 | Phase A, B 완료 | ❌ |
| #3 서킷 브레이커 | 중 | 중 | Phase 14 full · 15 제외 | ❌ |

Phase 19 가 독립적으로 완결 가능하면서 가장 구체적인 실무 경험을 재현 — 다른 3 개는 기획만으로도 충분.

---

## 참조

- [`PLAN.md`](./PLAN.md#phase-19--hp-동시-차감-데드락-랩--제출-후-작업) — Phase 19 상세 설계
- [`MISSION.md`](./MISSION.md) — 전체 미션과 5 축 프레임 (축 ⑥ 제출 후 서사)
- [`PITCH.md`](./PITCH.md) — 면접에서 이 시나리오를 말하는 방법
