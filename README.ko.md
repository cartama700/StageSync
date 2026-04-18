# StageSync

> **リズムゲーム 백엔드 서버 엔지니어 일상 업무를 Go 로 재현한 포트폴리오**

**언어 / 言語**: [한국어 (현재 파일)](./README.ko.md) · [日本語](./README.md)
**SSOT**: [`docs/MISSION.md`](./docs/MISSION.md) · [`docs/PLAN.md`](./docs/PLAN.md)

---

## 개요

株式会社Colorful Palette 서버사이드 엔지니어 공고 [BA-09-04a](https://hrmos.co/pages/colorfulpalette/jobs/BA-09-04a) 저격 포트폴리오. 실제 일상 업무 — **REST API · DB · 비동기 배치 · 운영** — 를 Go 로 구현.

공고 명시 스택 (**Aurora MySQL · Cloud Spanner · Redis · GKE · Terraform · Locust**) 에 완전 정렬, Mercari / CyberAgent 그룹 기준의 일본 Go 업계 관행 반영.

**Project Sekai 실 프로덕션 구조 모사**:
- 실시간 통신: **Diarkis** (Go / GKE / TCP·UDP·RUDP) — 외부 미들웨어
- REST 레이어: **Spring Boot (Java/Go)** — 본 포트폴리오의 **메인축**
- 데이터 스토어: Aurora MySQL + Spanner + Redis

2-티어 구조 (REST 메인축 + WebSocket 보너스축) 로 양쪽 모두 구현.

---

## 작성자 맥락 (정직하게)

본 프로젝트는 「장인의 대표작」 이 아니라 **「성장 포트폴리오」**.

- **Go · Java 이번이 첫 실무 경험** (C#/.NET 실무 여러 년)
- **일본어 N2 ~ N3** (공고 N1 필수 — 에이전시 컨택 성사, 포트폴리오가 결정타)
- **AI 도구 (Claude Code) 적극 활용** — 2026 년 현재 개발 생산성 · 학습 속도 실증
- 증명하려는 것: **학습 속도** · **기술적 깊이** · **일본 업계 관행 정렬 의지**

---

## 기술 스택

### 구현 완료 (Phase 0-4 + 보너스축)

| 영역 | 채택 기술 |
|---|---|
| 라우팅 | `chi` + HTTP/2 cleartext (h2c) |
| 아키텍처 | handler → service → repository 3-레이어 + `Mount(r)` 패턴 |
| DB | Aurora MySQL (`sqlx` + `sqlc` + `goose`) · inmem ↔ MySQL 전환 |
| 검증 | `go-playground/validator/v10` + 커스텀 에러 타입 계층 (`apperror`) |
| 에러 처리 | `fmt.Errorf("%w")` · `errors.Is` / `errors.As` · sentinel errors |
| 테스트 | `testify/require` · table-driven · `t.Parallel()` · httptest E2E · race detector |
| 정적 분석 | `.golangci.yml` (errcheck, staticcheck, revive, gocritic, bodyclose 등) |
| CI | GitHub Actions (test + lint + benchmark) |

### 보너스축 — 실시간 통신

- `coder/websocket` + protobuf binary frame
- `sync.RWMutex` + map 기반 thread-safe Room
- AOI 필터 + `sync.Pool` 최적화 토글 (**1.5× 빠름 · 0 allocs/op**)
- `cmd/bots` WebSocket 부하 시뮬레이터

### 구현 예정 (Phase 5-18)

가챠 · 이벤트 · 랭킹 · 메일 (게임 도메인 API) → Prometheus · pprof · 비동기 배치 · Write-Behind → Cloud Spanner 듀얼 스토어 → Docker Compose · Kubernetes + HPA · Terraform GKE → Locust + k6 부하 테스트 → AI Ops Assistant

---

## Phase 진행 현황

| 마일스톤 | Phase | 상태 |
|---|---|---|
| 보너스축 | 0, A, B | ✅ 3/3 |
| v0.1 기반 | 1-4 | ✅ 4/4 |
| v0.2 도메인 | 5-8 | ⏳ 0/4 |
| v0.3 운영 | 9-11 | ⏳ 0/3 |
| v0.4 데이터 | 12 | ⏳ 0/1 |
| v0.5 배포 | 13-15 | ⏳ 0/3 |
| v0.6 마감 | 16-18 | ⏳ 0/3 |

상세: [`docs/PLAN.md`](./docs/PLAN.md)

---

## 빠른 시작

### 일괄 환경 설치 (macOS · idempotent)

```bash
./scripts/setup.sh
```

Homebrew 만 있으면 Go · protoc · sqlc · goose · golangci-lint · Colima · Docker 까지 원샷.

### 기본 동작 (Docker 불필요 · inmem 모드)

```bash
make run
curl -X POST http://localhost:5050/api/profile \
     -H "Content-Type: application/json" \
     -d '{"id":"p1","name":"sekai"}'
curl http://localhost:5050/api/profile/p1
```

### MySQL 실 연결 (Docker 필요)

```bash
make dev-up          # Colima + MySQL 온디맨드 기동
make run-mysql       # 서버 기동 (goose 자동 마이그레이션)
# ... 동작 확인 ...
make dev-down        # 종료 시 둘 다 정리 (배터리 절약)
```

### 보너스축 — WebSocket 실시간

```bash
make run
go run ./cmd/bots -player=p1 -tick=200
# 다른 터미널:
curl http://localhost:5050/api/metrics
curl -X POST http://localhost:5050/api/optimize \
     -H "Content-Type: application/json" \
     -d '{"on":true}'
```

### 테스트 · 정적 분석

```bash
make test            # go test -race ./...
make bench           # AOI 벤치마크
make                 # Makefile 도움말 (기본값)
```

---

## 디렉토리 구조

```
StageSync/
├── cmd/
│   ├── server/          REST + WebSocket 서버
│   └── bots/            WebSocket 부하 시뮬레이터
├── api/proto/roompb/    protobuf 스키마 + 자동 생성 코드
├── internal/
│   ├── domain/          순수 도메인 객체
│   ├── service/         비즈니스 로직 (profile, aoi)
│   ├── persistence/
│   │   ├── inmem/       메모리 구현 (개발 · 테스트)
│   │   └── mysql/       sqlc + goose + schema + queries
│   ├── endpoint/        HTTP 핸들러 (Mount 패턴)
│   ├── apperror/        에러 타입 계층 + HTTP 매핑
│   ├── room/            WebSocket Room 상태 (보너스축)
│   └── lifecycle/       런타임 플래그
├── docs/                미션 · 플랜 · 아카이브
├── scripts/setup.sh     일괄 환경 설치
├── .github/workflows/   CI 파이프라인
├── Makefile
├── sqlc.yaml
├── .golangci.yml
└── go.mod
```

---

## 관련 문서

- [**docs/MISSION.md**](./docs/MISSION.md) — 프로젝트 미션 · 공고 매핑 · 5축 프레임
- [**docs/PLAN.md**](./docs/PLAN.md) — Phase 0-18 로드맵 · 의존성 · 학습 트래커
- [**README.md**](./README.md) — 日本語 版 (메인)

---

## 라이선스

개인 포트폴리오용 · 상업적 재사용 제한.
