# StageSync 미션 문서 (Mission Statement)

> **SSOT (Single Source of Truth)** — 본 문서는 프로젝트의 단일 진실 원천.
> **재정의 이력**:
> - 2026-04-18: Aiming PoC 이식 관점 → REST-first 공고 저격
> - 2026-04-19: v2 → v3 스코프 축소 — 25 Phase 로드맵 → 15 Phase MVP ([`PLAN.md`](./PLAN.md) "스코프 재편 기록" 참조)
> 이전 관점: [`archive/PORTING_GUIDE_v1_legacy.md`](./archive/PORTING_GUIDE_v1_legacy.md)

---

## 미션 (한 줄)

> **株式会社Colorful Palette サーバサイドエンジニア 공고 [BA-09-04a](https://hrmos.co/pages/colorfulpalette/jobs/BA-09-04a) 의 일상 업무를 Go 로 시뮬레이션하는 포트폴리오.**

리듬게임 백엔드 — **プロフィール · ガチャ · イベント · ランキング** — 를 Spring Boot 대응 Go 구조 (handler-service-repository 3-레이어) 로 구현하고, 공고 명시 스택 (Aurora MySQL · Redis · Docker/GKE · Locust) 을 그대로 정렬.

> **v3 스코프**: 메일 (Phase 8) · Spanner (Phase 12) · Terraform GKE (Phase 15) · AI Ops LLM (Phase 17) 은 제출 후 학습/추가 대상으로 분리. 제외 이유는 [`PLAN.md`](./PLAN.md#스코프-재편-기록-v2--v3-2026-04-19) 참조.

---

## 프로젝트가 증명하는 이야기

1. **REST API 설계·구현** — clean architecture · consumer-defined interface · DTO · validation · 에러 타입 체계
2. **관계형 DB (MySQL) 운영 감각** — 트랜잭션 · 마이그레이션 · UPSERT 원자성 · graceful degrade (Spanner 는 입사 후 학습)
3. **KV 캐시 (Redis) 활용** — ZSET 랭킹 · `ZINCRBY` best-effort 이중쓰기 · 연결 실패 시 inmem fallback
4. **게임 도메인 로직** — 가챠 확률 엔진 (천장 80회 + 10k 분포 검증) · 이벤트 라이프사이클 (시간 derived 상태) · 실시간 랭킹
5. **관측성·테스트 자동화** — Prometheus Histogram + Gauge · pprof · slog `request_id` 전파 · race-safe 단위 테스트
6. **컨테이너·K8s 운영** — Multi-target Dockerfile (distroless) · docker-compose profile · K8s manifest + readiness gate drain (Terraform 은 제외)
7. **부하 시뮬레이션** — Locust cluster 시나리오 + cmd/bots WebSocket (even/herd/cluster 3 패턴)
8. **일본 Go 업계 코딩 관행 숙지** — Mercari/CyberAgent 스타일 + 린터 + 한국어/JP 이중 언어 주석
9. **AI 시대 생산성** — Claude Code 협업 개발로 3 일 만에 15 Phase 중 14 구현 (LLM ops 는 제외)

---

## 타겟 공고 — BA-09-04a

| 항목 | 내용 |
|---|---|
| **회사** | 株式会社Colorful Palette (CyberAgent 자회사) |
| **직군** | サーバサイドエンジニア |
| **프로덕트** | プロジェクトセカイ カラフルステージ feat. 初音ミク + 신규 개발 |
| **필수 요건** | 서버사이드 1-2년+, Java/Golang/PHP/Ruby/Python, AWS/GCP, 日本語 N1 |
| **우대 요건** | 高負荷 경험, MySQL 설계, Docker/K8s, Locust/Gatling/JMeter, Ansible/Terraform, Spring DI |
| **명시 스택** | AWS · Google Cloud · Docker · Fargate · **GKE** · **Java** · **Golang** · **Aurora MySQL** · **Spanner** · **Redis** · GitHub |

### 일상 업무의 실체
- REST API 개발 (プロフィール·ガチャ·イベント·ランキング·メール·ショップ·ミッション)
- DB 쿼리·트랜잭션·마이그레이션 관리 (Aurora MySQL + Spanner)
- 비동기 배치 잡 (이벤트 집계·랭킹·푸시 알림)
- 비즈니스 로직·게임 밸런스
- 문의 대응 → 로그 조회·데이터 추적
- **실시간 통신은 Diarkis 미들웨어가 담당 → 이 직군은 직접 건드리지 않음**

---

## 새 5축 프레임 (REST-centric, v3 스코프)

| # | 축 | 정의 | 증명 수단 | 담당 Phase | v3 상태 |
|---|---|---|---|---|---|
| **①** | **REST + DB 기본기** | 레이어 분리 · 트랜잭션 · 마이그레이션 | handler-service-repo · sqlc · goose | 1-4 | ✅ 완료 |
| **②** | **게임 도메인 로직** | 모바일 게임 서버 핵심 기능 | ガチャ · イベント · ランキング | 5-7 | ✅ 완료 (Mail=Phase 8 제외) |
| **③** | **관측성** | 운영 중 문제 추적 | Prometheus Histogram · pprof · request_id 전파 | 9 lite | ✅ 완료 (배치잡=10/11 제외) |
| **④** | **데이터 확장** | NewSQL · hotspot 회피 | Spanner 듀얼 + shard key | 12 | ❌ v3 제외 (입사 후 학습) |
| **⑤** | **배포·부하** | 프로덕션 운영 역량 | Docker · K8s manifest · Locust | 13, 14 lite, 16 lite | ✅ 완료 (Terraform=15 / LLM=17 제외) |
| **⑥** | **서사 (제출 후)** | 실운영 장애 재현 → 해결 → 벤치 | HP 데드락 랩 (v1 naive → v2 queue) | 19 | ⏳ 제출 후

---

## 공고 bullet → 포폴 매핑 (v3)

| 공고 요건 (원문) | 포폴에서의 답 | 담당 Phase | 상태 |
|---|---|---|---|
| スマートフォンゲームの設計/開発/テスト/運用 | REST API 4 도메인 + 관측성 + 테스트 181 건 | 1-7, 9 | ✅ |
| 運用負荷·コスト削減 최적화 | `sync.Pool` · chi RoutePattern 레이블 (cardinality) · distroless 이미지 | B, 9 | ✅ |
| WEB技術 스킬업 | 일본 Go 업계 관행 + 린터 + JP/ko 이중 언어 주석 | 전역 | ✅ |
| 서버 측 데이터/비동기 통신 | REST (chi) + 보너스 WebSocket (coder/websocket + protobuf) | 1, A | ✅ |
| お客様 조사 대응 | 구조화 로그 (request_id) + pprof + Prometheus Histogram | 9 lite | ✅ |
| MySQL 설계·운용 | Aurora MySQL + sqlc + goose + 원자 트랜잭션 · UPSERT | 2-6 | ✅ |
| Docker·Kubernetes | Multi-target Dockerfile + compose 3 profile + K8s manifest + HPA + readiness drain | 13, 14 lite | ✅ |
| Locust/Gatling/JMeter | Locust cluster 시나리오 + cmd/bots (even/herd/cluster) | 16 lite | ✅ |
| Ansible/Terraform | K8s manifest 까지 (Terraform 은 v3 제외, 입사 후 학습) | ~~15~~ | ❌ |
| Spring DI 컨테이너 경험 | 수동 DI + functional options (일본 Go 관행) | 1 | ✅ |
| 高負荷サービス 릴리스/운용 | Locust cluster 시나리오 + graceful drain + readiness gate + Redis graceful fallback | 14 lite, 16 lite | ✅ |
| Redis 활용 | ZSET 랭킹 + `ZINCRBY` best-effort + inmem fallback | 7 | ✅ |

---

## 보너스 축 — 실시간 기능 (Phase 0·A·B 완료)

메인 축 **아님**. Phase 0-2 에서 선행 완성. 면접 자산으로 유지.

| Phase | 내용 | 의미 |
|---|---|---|
| **0** ✓ | chi + HTTP/2 h2c + `/api/metrics` + `/health/*` | 전체 프로젝트 기반 (계속 활용) |
| **A** ✓ (구 1) | WebSocket Room + protobuf + cmd/bots E2E | "Diarkis 운영 맥락 이해" + "실시간 프로토콜도 구현 가능" |
| **B** ✓ (구 2) | AOI + sync.Pool + 벤치마크 (1.5× · 0 allocs) | "핫패스·GC 최적화 문화" |

### 기존 자산 재포지셔닝
- `cmd/server/main.go` — 기반 (계속 사용)
- `cmd/bots/main.go` — Phase 16 부하 테스트 확장 기반
- `internal/room/`, `internal/endpoint/ws.go`, `api/proto/roompb/` — 유지, "실시간 부가 모듈"
- `internal/service/aoi/` — 핫패스 최적화 쇼케이스 (사용처는 Phase 16 부하 시에)
- `internal/lifecycle/optimize.go` + `/api/optimize` — 런타임 토글 인프라, 다른 Phase 에서도 재활용 가능

---

## 주요 기술 결정 (v3 실제)

| 결정 | 이유 | 영향 Phase | ADR |
|---|---|---|---|
| Go 1.26 + chi + `log/slog` | 공고 명시 · 일본 Go 관행 표준 | 전역 | [ADR-0001](./adr/0001-chi-over-gin.md) |
| HTTP/2 cleartext (h2c) | GKE Cloud LB 패턴 정렬 | 0 | [ADR-0003](./adr/0003-h2c-for-websocket-coexistence.md) |
| **handler-service-repository 3-레이어 + consumer-defined interface** | clean architecture · 테스트 용이성 · Spring Boot 대응 | 1 | — |
| **수동 DI + functional options** (`With...`) | 일본 Go 관행 (uber/fx 비주류) · 테스트 시 clock/RNG 주입 | 1, 5, 6 | — |
| `sqlc` + `goose` | Aurora MySQL 호환 · 쿼리 1급 시민 · DBTX 인터페이스로 트랜잭션 투명 | 2 | [ADR-0002](./adr/0002-sqlc-over-orm.md) |
| `redis/go-redis/v9` + `miniredis` (테스트) | 공고 명시 · 표준 · graceful degrade | 7 | — |
| ~~`cloud.google.com/go/spanner`~~ | v3 제외 (에뮬레이터만으론 "운영 경험" 어필 부족) | ~~12~~ | — |
| `prometheus/client_golang` + `net/http/pprof` | 업계 표준 관측 툴체인. HistogramVec 에 chi RoutePattern 레이블 → cardinality 제어 | 9 lite | — |
| Docker distroless + K8s manifest + readiness gate | 공고 명시 인프라 · Phase 14 lite 로 YAML + 앱 drain 로직까지 | 13, 14 lite | — |
| ~~Terraform GKE~~ | v3 제외 (실 GCP 청구서 없이 어필 제한) | ~~15~~ | — |
| Locust `.py` (cluster 시나리오) | 공고 명시 · event-open spike 모사 | 16 lite | — |
| ~~AI Ops LLM SSE~~ | v3 제외 (본질 업무와 거리) | ~~17~~ | — |
| **주석: 한국어 주 + JP 도메인 용어** | 작성자 N2-N3 현실 · 면접 방어 가능성 우선 | 전역 | — |
| **AI 협업 (Claude Code) 명시** | 2026 생산성 시그널 · 3 일 구현 증명 | 전역 · README | — |

---

## 리서치 기반 (움직이지 않는 사실)

### Project Sekai 실 프로덕션 구조
- **실시간 레이어**: Diarkis (Go/GKE/TCP·UDP·RUDP 독자 프로토콜) — 외부 미들웨어
- **REST 레이어**: 자체 Spring Boot + Java/Go — **포폴의 메인 축**
- **데이터스토어**: Aurora MySQL + Spanner + Redis
- **근거**: [Google Cloud 사례](https://cloud.google.com/blog/ja/topics/customers/colorful-palettegke-diarkis?hl=ja), [Diarkis 사례](https://www.diarkis.io/case-study/project-sekai-colorful-stage-feat-hatsune-miku)

### 실 규모 지표
- 피크 동접 ~**10만** · Locust 워커 **2500** · C2 머신 **600+** · Virtual Live 100명 동시
- 근거: [gamebiz 인터뷰](https://gamebiz.jp/news/294249)

### 클라이언트 스택
- **Unity + C#** (확정, 공고 BA-09-01a · BB-09-01a)
- 서버는 REST 공급 (클라 직접 접속), 실시간은 Diarkis SDK 경유

### 일본 Go 업계 관행 (메모리 `feedback_jp_go_conventions.md` 참조)
- `errcheck` · `staticcheck` · `revive` · `gocritic` 엄격
- `context.Context` 첫 파라미터, `fmt.Errorf("...: %w", err)` 필수
- table-driven test + `t.Parallel()`, stdlib `testing` 우선
- 수동 DI 또는 `google/wire` (`uber/fx` 비주류)
- 근거: Mercari Engineering, CyberAgent Developers Blog, Google Go Style 日本語版, Go Conference Japan 2024

---

## 작성자 맥락 (정직하게)

| 항목 | 현실 |
|---|---|
| 경력 배경 | C# / .NET 10 실무 여러 년 (이전 Aiming PoC 포트폴리오 존재) |
| **Go** | **이번이 첫 실무 경험** |
| **Java** | **이번이 첫 실무 경험** (향후 Spring Boot 학습 예정) |
| **일본어** | **N2 ~ N3** (공고 N1 필수) |
| **채용 경로** | **에이전시가 Colorful Palette 컨택 성사** → 포폴이 결정타 |
| **개발 방식** | **AI 도구 (Claude Code) 적극 협업** — 2026 생산성 모델 |

**포폴 서사**: 장인 대표작 X. **성장 포트폴리오** — 학습 속도 · 기술 깊이 · 일본 업계 관행 정렬 의지 증명.

---

## 참조 문서

- [`PLAN.md`](./PLAN.md) — Phase 로드맵 (v3) · 학습 트래커 · 의존성 맵 · 제외 Phase 근거
- [`STATUS.md`](./STATUS.md) — 현 스냅샷 (PLAN ↔ 리포 대조)
- [`API.md`](./API.md) — 엔드포인트 계약 (SSOT)
- [`BENCHMARKS.md`](./BENCHMARKS.md) — AOI + Locust 실측
- [`adr/`](./adr/) — 기술 결정 상세 (chi · sqlc · h2c)
- [`PITCH.md`](./PITCH.md) — 면접 피치 스크립트 (JP + ko)
- [`SUBMISSION_CHECKLIST.md`](./SUBMISSION_CHECKLIST.md) — 제출 직전 체크리스트
- [`PORTFOLIO_SCENARIOS.md`](./PORTFOLIO_SCENARIOS.md) — 제출 후 장애 랩 (Phase 19)
- [`../README.md`](../README.md) — 채용 담당자용 메인 (日本語 기본)
- [`../README.ko.md`](../README.ko.md) — 한국어 버전
- [`../CHANGELOG.md`](../CHANGELOG.md) — v0.1 릴리즈 변경 이력
- [`archive/PORTING_GUIDE_v1_legacy.md`](./archive/PORTING_GUIDE_v1_legacy.md) — 이전 관점 문서 (아카이브)
