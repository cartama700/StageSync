# ADR-0001: Router 로 chi 선택

- 상태: Accepted
- 일자: 2026-04-17

## 맥락

HTTP 라우팅 라이브러리 후보:

1. **`net/http` 표준** — 의존성 0. 하지만 URL 파라미터 (`/profile/{id}`) 파싱을 직접 해야 하고, 미들웨어 체인도 손수 감쌈.
2. **Gin** — 일본·한국 Go 커뮤니티에서 가장 대중적. 빠름. 단 독자 `Context` 타입이라 `net/http.Handler` 호환성이 낮고, 표준 미들웨어 재사용 불가.
3. **Echo** — Gin 과 비슷한 위치. 문법 취향 차이.
4. **chi** — `net/http.Handler` 시그니처를 그대로 유지. 미들웨어가 `func(http.Handler) http.Handler` 표준 형태. URL 파라미터 + 서브라우터 지원.

공고 (BA-09-04a) 가 명시한 대형 게임 백엔드 운영 환경을 고려하면:
- 프로덕션 스택은 수명이 김 → **표준 호환성** 이 라이브러리 수명 종속성을 줄임.
- Prometheus / pprof / `http.ServeMux` 기반 도구들이 전부 `http.Handler` 를 기대.
- 팀 합류 시 "이건 chi 특수 문법이야" 설명이 불필요 (Gin 은 필요).

## 결정

**chi 를 선택한다.**

- `net/http.Handler` 호환 → `httptest` 통합 테스트 · 표준 미들웨어 그대로 장착.
- `middleware.RequestID` · `middleware.Recoverer` · `middleware.Timeout` 등 기성 미들웨어.
- 서브라우터 / 미들웨어 스코프 분리 — 향후 `/api/admin/*` 같은 구획에 별도 미들웨어 걸기 쉬움.

## 결과

**좋은 점**
- 테스트가 표준 `httptest.Server` 로 그대로 가능 ([internal/endpoint/gacha_test.go](../../internal/endpoint/gacha_test.go) 참조).
- 커스텀 미들웨어 ([middleware.go](../../internal/endpoint/middleware.go)) 작성 비용 0.
- h2c / WebSocket 업그레이드가 표준 `http.Server` 위에서 그대로 동작.

**나쁜 점**
- Gin 대비 "배터리 포함" 수준은 낮음 — JSON 바인딩·validation 은 `encoding/json` + `go-playground/validator` 로 직접 조합.
- 대중성 지표 (star 수) 로는 Gin 우세.

**후속 작업**
- 없음 — 전 Phase 에서 chi 지속 사용.
