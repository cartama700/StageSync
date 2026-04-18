# StageSync 미션 문서 (Mission Statement)

> **SSOT (Single Source of Truth)** — 본 문서는 프로젝트의 단일 진실 원천.
> **재정의 일자**: 2026-04-18 (Aiming PoC 이식 관점 → REST-first 공고 저격)
> 이전 관점: [`archive/PORTING_GUIDE_v1_legacy.md`](./archive/PORTING_GUIDE_v1_legacy.md)

---

## 미션 (한 줄)

> **株式会社Colorful Palette サーバサイドエンジニア 공고 [BA-09-04a](https://hrmos.co/pages/colorfulpalette/jobs/BA-09-04a) 의 일상 업무를 Go 로 시뮬레이션하는 포트폴리오.**

리듬게임 백엔드 — **プロフィール · ガチャ · イベント · ランキング · メール** — 를 Spring Boot 대응 Go 구조 (handler-service-repository 3-레이어) 로 구현하고, 공고 명시 스택 (Aurora MySQL · Spanner · Redis · GKE · Terraform · Locust) 을 그대로 정렬.

---

## 프로젝트가 증명하는 이야기

1. **REST API 설계·구현** — clean architecture · DTO · validation · 에러 타입 체계
2. **관계형 DB (MySQL) + NewSQL (Spanner) 운영 감각** — 트랜잭션 · 마이그레이션 · hotspot 회피
3. **비동기 배치·큐 패턴** — 이벤트 집계 · 랭킹 스냅샷 · Write-Behind
4. **게임 도메인 로직** — 가챠 확률 엔진 · 이벤트 라이프사이클 · 랭킹 계산
5. **관측성·테스트·운영 자동화** — Prometheus · pprof · Locust · K8s · Terraform
6. **일본 Go 업계 코딩 관행 숙지** — Mercari/CyberAgent 스타일 + 린터 + 한국어/JP 이중 언어 주석
7. **AI 시대 생산성** — Claude Code 협업 개발 + AI Ops Assistant 구축

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

## 새 5축 프레임 (REST-centric)

| # | 축 | 정의 | 증명 수단 | 담당 Phase |
|---|---|---|---|---|
| **①** | **REST + DB 기본기** | 레이어 분리 · 트랜잭션 · 마이그레이션 | handler-service-repo · sqlx · goose | 1-4 |
| **②** | **게임 도메인 로직** | 모바일 게임 서버 핵심 기능 | ガチャ·イベント·ランキング·メール | 5-8 |
| **③** | **관측성·비동기** | 운영 중 문제 추적·배치 처리 | Prometheus · pprof · errgroup | 9-11 |
| **④** | **데이터 확장** | NewSQL · 고부하 친화 설계 | Spanner 듀얼 + hotspot-safe shard key | 12 |
| **⑤** | **배포·부하·AI** | 프로덕션 운영 역량 | Docker · K8s · Terraform · Locust · LLM | 13-17 |

---

## 공고 bullet → 포폴 매핑

| 공고 요건 (원문) | 포폴에서의 답 | 담당 Phase |
|---|---|---|
| スマートフォンゲームの設計/開発/テスト/運用 | REST API + 비동기 배치 + 테스트 + 관측성 | 1-11 |
| 運用負荷·コスト削減 최적화 | 쿼리 튜닝 · 캐시 계층 · sync.Pool | 2, B (완) |
| WEB技術 스킬업 | 일본 Go 업계 관행 + 린터 + 이중 언어 주석 | 전역 |
| 서버 측 데이터/비동기 통신 | REST (chi) + 비동기 잡 (errgroup) + 부가 WebSocket | 1, 10, A (완) |
| お客様 조사 대응 | 구조화 로그 + pprof + Prometheus | 9 |
| MySQL 설계·운용 | Aurora MySQL + sqlx + goose + 트랜잭션 | 2-3 |
| Docker·Kubernetes | Docker compose + K8s + HPA + preStop | 13-14 |
| Locust/Gatling/JMeter | Locust `.py` + k6 `.js` + cmd/bots | 16 |
| Ansible/Terraform | Terraform GKE 클러스터 + Artifact Registry | 15 |
| Spring DI 컨테이너 경험 | google/wire 또는 수동 DI (일본 Go 관행) | 1 |
| 高負荷サービス 릴리스/운용 | event spike 시뮬 + Spanner hotspot 회피 + Graceful shutdown | 12, 14, 16 |

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

## 주요 기술 결정

| 결정 | 이유 | 영향 Phase |
|---|---|---|
| Go 1.26 + chi + `log/slog` | 공고 명시 · 일본 Go 관행 표준 | 전역 |
| HTTP/2 cleartext (h2c) | GKE Cloud LB 패턴 정렬 | 0 (완) |
| **handler-service-repository 3-레이어** | clean architecture · 테스트 용이성 · Spring Boot 대응 | 1 |
| **수동 DI 또는 `google/wire`** | 일본 Go 관행 (uber/fx 비주류) | 1 |
| `sqlx` + `goose` | Aurora MySQL 호환 · 쿼리 1급 시민 | 2 |
| `redis/go-redis/v9` | 공고 명시 · 일본 업계 표준 | 7, 10 |
| `cloud.google.com/go/spanner` | 공고 명시 · hotspot 회피 설계 포인트 | 12 |
| `prometheus/client_golang` + `net/http/pprof` | 업계 표준 관측 툴체인 | 9 |
| Docker distroless + GKE + Terraform | 공고 명시 인프라 스택 | 13-15 |
| Locust `.py` + k6 `.js` | 공고 명시 + 보조 | 16 |
| **주석: 한국어 주 + JP 도메인 용어** | 작성자 N2-N3 현실 · 면접 방어 가능성 우선 | 전역 |
| **AI 협업 (Claude Code) 명시** | 2026 생산성 시그널 | 전역 · README |

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

- [`PLAN.md`](./PLAN.md) — Phase 0~18 로드맵, 학습 트래커, 의존성 맵
- [`../README.md`](../README.md) — 채용 담당자용 메인 (한국어)
- [`../README.ja.md`](../README.ja.md) — 日本語版 (Phase 18 에서 완성)
- [`archive/PORTING_GUIDE_v1_legacy.md`](./archive/PORTING_GUIDE_v1_legacy.md) — 이전 관점 문서 (아카이브)
