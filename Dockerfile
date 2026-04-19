# syntax=docker/dockerfile:1.7
#
# StageSync — multi-target Dockerfile.
#
# Targets:
#   server   (default, last stage) — REST + WebSocket 서버. 노출 포트 5050.
#   bots                           — cmd/bots 부하 시뮬레이터. WebSocket 클라이언트.
#
# 빌드 예:
#   docker build -t stagesync:server .                # default (server)
#   docker build -t stagesync:bots --target bots .    # bots 이미지
#
# docker compose 는 서비스별로 target 을 명시 (docker-compose.yml 참조).

# ============================================================================
# builder (공용) — 두 바이너리 모두 같은 builder 에서 생성. mod cache 재사용.
# ============================================================================
FROM golang:1.26-alpine AS builder

WORKDIR /src

# 의존성 먼저 복사 → 레이어 캐시 적중 극대화.
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .

# 두 바이너리를 한 번에 빌드 (-trimpath + ldflags 로 경량화).
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 GOOS=linux \
    go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server && \
    CGO_ENABLED=0 GOOS=linux \
    go build -trimpath -ldflags="-s -w" -o /out/bots   ./cmd/bots

# ============================================================================
# bots — 부하 시뮬 이미지. `--profile load` 아래에서 사용.
# ============================================================================
FROM gcr.io/distroless/static-debian12:nonroot AS bots

WORKDIR /app
COPY --from=builder /out/bots /app/bots

USER nonroot:nonroot
# 기본 파라미터는 docker-compose 쪽에서 `command:` 로 오버라이드.
ENTRYPOINT ["/app/bots"]

# ============================================================================
# server — 기본 타겟 (마지막 stage). `docker build .` 하면 이 이미지가 생성.
# ============================================================================
FROM gcr.io/distroless/static-debian12:nonroot AS server

WORKDIR /app
COPY --from=builder /out/server /app/server

USER nonroot:nonroot
EXPOSE 5050

ENTRYPOINT ["/app/server"]
