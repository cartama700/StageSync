# StageSync

> **リズムゲーム バックエンド 서버 엔지니어 일상 업무 시뮬레이션** — 株式会社Colorful Palette サーバサイドエンジニア 공고 저격 포트폴리오.

**言語 / 언어**: [한국어 (현재 파일)](./README.md) · [日本語](./README.ja.md) (準備中)
**SSOT**: [`docs/MISSION.md`](./docs/MISSION.md) · [`docs/PLAN.md`](./docs/PLAN.md)

---

## 한 줄 미션

**株式会社Colorful Palette サーバサイドエンジニア [BA-09-04a](https://hrmos.co/pages/colorfulpalette/jobs/BA-09-04a) 의 일상 업무를 Go 로 시뮬레이션한 포트폴리오.**

리듬게임 백엔드 — **プロフィール · ガチャ · イベント · ランキング · メール** — 를 Spring Boot 대응 Go 구조 (handler-service-repository 3-레이어) 로 구현하고, 공고 명시 스택 (Aurora MySQL · Spanner · Redis · GKE · Terraform · Locust) 을 그대로 정렬.

---

## 포폴이 증명하는 이야기

1. **REST API 설계·구현** — clean architecture · DTO · validation · 에러 타입 체계
2. **관계형 DB (MySQL) + NewSQL (Spanner) 운영 감각** — 트랜잭션 · 마이그레이션 · hotspot 회피
3. **비동기 배치·큐 패턴** — 이벤트 집계 · 랭킹 스냅샷 · Write-Behind
4. **게임 도메인 로직** — 가챠 확률 엔진 · 이벤트 라이프사이클 · 랭킹 계산
5. **관측성·테스트·운영 자동화** — Prometheus · pprof · Locust · K8s · Terraform
6. **일본 Go 업계 코딩 관행 숙지** — Mercari/CyberAgent 스타일 + 린터 + KR/JP 이중 언어 주석
7. **AI 시대 생산성** — Claude Code 협업 + AI Ops Assistant 구축

---

## 작성자 맥락 (정직하게)

이 프로젝트는 **"장인의 대표작"** 이 아니라 **"성장 포트폴리오"** 입니다.

- **Go · Java 이번이 첫 실무 경험** (C#/.NET 여러 년 배경).
- **일본어 N2~N3** 수준 (공고는 N1 필수 — 에이전시 컨택 성사 후 포폴이 결정타).
- **AI 도구 (Claude Code) 적극 협업** — 2026 년 기준 개발 생산성 · 학습 가속 증명.
- 증명하려는 것: **학습 속도** · **기술적 깊이** · **일본 업계 관행 정렬 의지**.

---

## 기술 스택

### 공고 스택 그대로 정렬
- **언어**: Go 1.26 (+ Java/Spring Boot 대응 구조)
- **REST**: chi + HTTP/2 cleartext (h2c)
- **DB**: Aurora MySQL (`sqlx` + `goose`) + Cloud Spanner (`spanner-go`) 듀얼
- **캐시·Pub/Sub**: Redis (`redis/go-redis/v9`)
- **관측**: Prometheus + pprof + `log/slog` JSON
- **인프라**: Docker (distroless) + GKE + HPA + Terraform
- **부하 테스트**: Locust `.py` + k6 `.js` + 자체 `cmd/bots`
- **AI Ops**: OpenAI 호환 LLM + SSE 스트리밍

### 일본 Go 업계 관행 반영
- `.golangci.yml` (errcheck · staticcheck · revive · gocritic · bodyclose 등)
- `fmt.Errorf("...: %w", err)` 에러 래핑 전면
- Table-driven test + `t.Parallel()`
- 수동 DI 또는 `google/wire` (uber/fx 비주류)
- 주석: 한국어 주 + 짧은 JP 도메인 용어 (`ルーム`, `協奏`, `バーチャルライブ`)

---

## Phase 로드맵 요약

| 마일스톤 | Phase | 내용 | 상태 |
|---|---|---|---|
| **보너스축** | 0, A, B | 뼈대 · WebSocket Room · AOI 토글 | ✓ 완료 |
| **v0.1 기반** | 1-4 | REST + clean arch · MySQL · Validation · 테스트 CI | 대기 |
| **v0.2 도메인** | 5-8 | ガチャ · イベント · ランキング · メール | 대기 |
| **v0.3 운영** | 9-11 | Prometheus · 비동기 배치 · Write-Behind | 대기 |
| **v0.4 데이터** | 12 | Spanner 듀얼 + hotspot 회피 | 대기 |
| **v0.5 배포** | 13-15 | Docker · K8s · Terraform GKE | 대기 |
| **v0.6 마감** | 16-18 | Locust · AI Ops · 문서 마감 | 대기 |

상세: [`docs/PLAN.md`](./docs/PLAN.md)

---

## 보너스축 (완료) — 실시간 기능

공고의 메인 축은 아니지만, **"Diarkis 운영 맥락 이해 + 실시간 프로토콜도 구현 가능"** 시그널로 선행 완성:

| Phase | 내용 | 주요 기술 |
|---|---|---|
| **0** ✓ | 서버 뼈대 · REST 기반 · 관측 엔드포인트 | chi · HTTP/2 h2c · log/slog |
| **A** ✓ | WebSocket Room 시연 (봇이 좌표 쏘고 서버 로그 수신) | `coder/websocket` · protobuf · `sync.RWMutex` |
| **B** ✓ | AOI 필터 + `sync.Pool` 토글 (1.5× · 0 allocs) | `sync.Pool` · `atomic.Bool` · `testing.B` |

*이 자산은 유지되며 Phase 16 부하 시나리오 등에서 계속 활용.*

---

## 빠른 시작

### 일괄 환경 설치 (macOS · 재실행 안전)
```bash
./scripts/setup.sh
```
Homebrew 만 있으면 Go · protoc · sqlc · goose · golangci-lint · Colima · Docker 까지 일괄 설치 + Colima VM 기동. **idempotent** 이므로 여러 번 실행해도 안전하게 스킵.

### 개별 명령
```bash
make tidy        # Go 의존성 다운로드
make proto       # room.proto → room.pb.go (Phase A 자산)
make sqlc        # SQL → Go 타입 안전 코드 생성
make build       # bin/server, bin/bots
make test        # 전 단위 테스트
make bench       # AOI 벤치마크 (보너스축)
```

### 현재 시연 가능한 것

#### v0.1 기반 — REST API (Phase 1)
```bash
make run                                             # 서버 기동 (inmem 모드)

# 프로필 생성 → 조회 → 중복 거부
curl -X POST http://localhost:5050/api/profile \
     -H "Content-Type: application/json" -d '{"id":"p1","name":"sekai"}'
curl http://localhost:5050/api/profile/p1
```

#### Phase 2 — MySQL 실 연결 (Docker 필요)
```bash
make dev-up          # Colima + MySQL 동시 기동 (필요할 때만)
make run-mysql       # 서버 (MYSQL_DSN 자동 + goose 자동 마이그레이션)

curl -X POST http://localhost:5050/api/profile \
     -H "Content-Type: application/json" -d '{"id":"p1","name":"sekai"}'
# 서버 재시작해도 프로필 유지됨 (DB 영속 확인)

make dev-down        # 끝났으면 MySQL + Colima 둘 다 정리 (배터리 절약)
```

**개별 제어**: `make docker-up / docker-down / docker-status / mysql-dev / mysql-stop`

#### 보너스축 — WebSocket 실시간 (Phase A·B)
```bash
# 터미널 1
make run

# 터미널 2 — WebSocket 부하 봇 1개
go run ./cmd/bots -player=p1 -tick=200

# 터미널 3 — 최적화 토글
curl http://localhost:5050/api/metrics
curl -X POST http://localhost:5050/api/optimize \
     -H "Content-Type: application/json" -d '{"on":true}'
curl http://localhost:5050/api/metrics  # optimized=true 반영
```

### AOI 벤치마크 (Phase B)
```bash
go test -bench=. -benchmem -benchtime=3s -count=3 ./internal/service/aoi/
# Naive  : ~445 ns/op, 512 B/op, 1 allocs/op
# Pooled : ~305 ns/op,   0 B/op, 0 allocs/op
```

---

## 디렉토리 구조

```
StageSync/
├── cmd/
│   ├── server/main.go             chi + HTTP/2 + REST + WebSocket
│   └── bots/main.go               WebSocket 부하 시뮬
├── api/proto/roompb/              [보너스축] 실시간 메시지 스키마
├── internal/
│   ├── endpoint/                  HTTP handler 레이어
│   ├── room/                      [보너스축] thread-safe Room
│   ├── service/aoi/               [보너스축] AOI 필터
│   └── lifecycle/                 optimize 토글 · readiness gate
├── docs/
│   ├── MISSION.md                 프로젝트 미션 (현 SSOT)
│   ├── PLAN.md                    Phase 0-18 로드맵
│   └── archive/
│       └── PORTING_GUIDE_v1_legacy.md   이전 관점 (폐기)
├── Makefile                       proto / tidy / run / build / clean
├── .golangci.yml                  일본 Go 업계 린터 룰셋
├── go.mod / go.sum
└── .gitignore
```

---

## 관련 문서

- [**docs/MISSION.md**](./docs/MISSION.md) — 프로젝트 미션 · 공고 bullet 매핑 · 5축 프레임 (SSOT)
- [**docs/PLAN.md**](./docs/PLAN.md) — Phase 0-18 로드맵 · 의존성 · 학습 트래커
- [**README.ja.md**](./README.ja.md) — 日本語版 (Phase 18 에서 완성 예정)
- [**docs/archive/PORTING_GUIDE_v1_legacy.md**](./docs/archive/PORTING_GUIDE_v1_legacy.md) — 이전 관점 문서 (아카이브, 현 결정 근거 아님)

---

## 라이선스

개인 포트폴리오용 비공개 프로젝트. 상업적 재사용 제한.
