# ADR-0003: HTTP/2 cleartext (h2c) 로 REST + WebSocket 공존

- 상태: Accepted
- 일자: 2026-04-17

## 맥락

StageSync 는 두 개의 통신축을 동시에 제공:
- **REST 주축**: `POST /api/gacha/roll`, `GET /api/profile/{id}` 등.
- **WebSocket 보너스축**: `GET /ws/room` 업그레이드.

옵션:

1. **별도 포트 + 프로세스 분리**
   예) REST 는 `:5050`, WebSocket 은 `:5051`.
   - 단점: 운영 환경(K8s) 에서 Service 두 개. 로드밸런서 설정 이중화. 메트릭 / 로그 / 트레이스 상관 끊김.

2. **HTTP/1.1 단일 포트**
   - chi + `http.Server` 기본 동작. WebSocket 업그레이드 `Upgrade: websocket` 가 표준 동작.
   - 단점: HTTP/2 multiplexing · 헤더 압축 이득 포기.

3. **HTTP/2 TLS (h2) 단일 포트**
   - 프로덕션 표준. 단 로컬 개발에 인증서 세팅 부담.
   - 일부 WebSocket 클라이언트는 여전히 HTTP/1.1 업그레이드 가정 → 혼합 클라이언트 지원 복잡.

4. **HTTP/2 cleartext (h2c) 단일 포트** — `golang.org/x/net/http2/h2c`
   - TLS 없이 HTTP/2 multiplexing + HTTP/1.1 clients 도 같은 포트로 수용.
   - WebSocket `Upgrade` 는 h2c handler 가 1.1 로 fallback 하여 통과시킴.

## 결정

**h2c (`h2c.NewHandler`) 를 선택한다.**

구현: [cmd/server/main.go](../../cmd/server/main.go)
```go
h2s := &http2.Server{}
srv := &http.Server{
    Addr:    cfg.Listen,
    Handler: h2c.NewHandler(r, h2s),
}
```

로컬·내부망(CI / K8s ClusterIP) 에서는 TLS 종료를 프록시 (Envoy / nginx / ALB) 가 담당하고
애플리케이션은 평문 HTTP/2 를 쓰는 것이 일반적 → h2c 가 합리적 중간지대.

## 결과

**좋은 점**
- 단일 포트 · 단일 프로세스 → 운영·메트릭 복잡도 감소.
- HTTP/2 의 헤더 압축 / multiplexing 이득을 REST 쪽에서 바로 확보.
- WebSocket 업그레이드는 `Upgrade: websocket` 헤더로 HTTP/1.1 경로를 자연스럽게 탐 ([ws.go](../../internal/endpoint/ws.go)).

**나쁜 점**
- 평문 — 프로덕션 노출 경로엔 반드시 앞단 TLS 프록시 필요.
- HTTP/2 prior-knowledge 클라이언트 (일부 Go 클라이언트) 와 HTTP/1.1 업그레이드 클라이언트 공존 시 미묘한 케이스 디버깅 필요할 수 있음.

**후속 작업**
- Phase 15 (GKE + Terraform) 에서 Ingress/TLS 앞단을 어떻게 둘지 확정 — Cloud Armor + HTTPS LB 전제.
