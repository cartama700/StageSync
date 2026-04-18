#!/usr/bin/env bash
# scripts/setup.sh — StageSync 개발 환경 일괄 설치
#
# 대상: macOS (Intel / Apple Silicon)
# 전제: Homebrew 설치 완료 (https://brew.sh)
# 실행: ./scripts/setup.sh
#
# idempotent — 여러 번 실행 OK, 이미 설치된 건 스킵.
#
# 설치 범위 (Phase 0 ~ 4 필요 도구 일괄):
#   - Go 1.26 + protobuf + sqlc       [brew]
#   - protoc-gen-go, goose, golangci-lint  [go install]
#   - Colima + Docker + docker-compose     [brew]
#   - Colima VM 기동
# Phase 14+ 도구 (kubectl, terraform, k6, locust) 는 해당 Phase 진입 시 추가 예정.

set -euo pipefail

# ---------- 색상 로그 ----------
C_OK='\033[0;32m'
C_INFO='\033[0;34m'
C_WARN='\033[1;33m'
C_ERR='\033[0;31m'
C_RESET='\033[0m'

info()  { echo -e "${C_INFO}[INFO]${C_RESET} $*"; }
ok()    { echo -e "${C_OK}[OK]  ${C_RESET} $*"; }
skip()  { echo -e "${C_WARN}[SKIP]${C_RESET} $*"; }
err()   { echo -e "${C_ERR}[ERR] ${C_RESET} $*" >&2; }

# ---------- 사전 조건 ----------
if ! command -v brew >/dev/null 2>&1; then
  err "Homebrew 미설치. https://brew.sh 에서 먼저 설치하세요."
  exit 1
fi

# ---------- 헬퍼 ----------
brew_pkg() {
  local pkg="$1"
  if brew list --formula "$pkg" >/dev/null 2>&1; then
    skip "$pkg (brew, 이미 설치됨)"
  else
    info "brew install $pkg"
    brew install "$pkg" && ok "$pkg"
  fi
}

go_cli() {
  local module="$1"
  local bin
  bin="$(basename "${module%@*}")"
  if command -v "$bin" >/dev/null 2>&1; then
    skip "$bin (go install, 이미 있음)"
  else
    info "go install $module"
    go install "$module" && ok "$bin"
  fi
}

# ---------- 설치 실행 ----------
echo "======================================================"
echo "  StageSync 개발 환경 설치 (Phase 0-4 일괄)"
echo "======================================================"
echo

info "[1/4] Go 언어·빌드 체인"
brew_pkg go
brew_pkg protobuf
brew_pkg sqlc

# Go 가 방금 설치된 경우를 위해 PATH 갱신
export PATH="$(go env GOPATH)/bin:$PATH"

info "[2/4] Go 기반 CLI 도구"
go_cli google.golang.org/protobuf/cmd/protoc-gen-go@latest
go_cli github.com/pressly/goose/v3/cmd/goose@latest
# golangci-lint 는 brew 로 설치 — v2 바이너리가 Go 최신 버전과 호환 (go install 은 v1 계열 가능성)
brew_pkg golangci-lint

info "[3/4] 컨테이너 스택 (Colima + Docker)"
brew_pkg colima
brew_pkg docker
brew_pkg docker-compose

info "[4/4] Colima 설정 확인 (자동 기동은 안 함 — 포폴 환경)"
if colima status >/dev/null 2>&1; then
  info "Colima 현재 실행 중. 원치 않으면 'make docker-down'"
else
  info "Colima 미기동 상태. 사용 시 'make docker-up' / 'make dev-up'"
fi

# ---------- PATH 영구 설정 안내 ----------
echo
GO_BIN="$(go env GOPATH)/bin"
shell_rc=""
case "$SHELL" in
  */zsh)  shell_rc="${HOME}/.zshrc" ;;
  */bash) shell_rc="${HOME}/.bashrc" ;;
esac

if [ -n "$shell_rc" ] && [ -f "$shell_rc" ]; then
  if ! grep -q "go env GOPATH" "$shell_rc" 2>/dev/null; then
    info "PATH 영구 설정이 필요합니다. 다음 라인을 ${shell_rc} 에 추가하세요:"
    echo ""
    echo "  export PATH=\"\$(go env GOPATH)/bin:\$PATH\""
    echo ""
    info "이 스크립트가 끝난 세션에선 자동 반영 안 됨 (새 터미널 열 때 필요)"
  fi
fi

# ---------- 설치 버전 요약 ----------
echo
info "설치된 도구 버전"
echo "------------------------------------------------------"
printf "%-18s %s\n" "go"              "$(go version 2>&1)"
printf "%-18s %s\n" "protoc"          "$(protoc --version 2>&1)"
printf "%-18s %s\n" "protoc-gen-go"   "$(protoc-gen-go --version 2>&1)"
printf "%-18s %s\n" "sqlc"            "$(sqlc version 2>&1)"
printf "%-18s %s\n" "goose"           "$(goose -version 2>&1 | head -1 || echo '(버전 표기 없음)')"
printf "%-18s %s\n" "golangci-lint"   "$(golangci-lint --version 2>&1 | head -1)"
printf "%-18s %s\n" "colima"          "$(colima version 2>&1 | grep -i 'colima version' | head -1 || colima version 2>&1 | head -1)"
printf "%-18s %s\n" "docker"          "$(docker --version 2>&1)"
printf "%-18s %s\n" "docker compose"  "$(docker compose version 2>&1 | head -1 || echo '(compose plugin 확인 실패)')"

echo
ok "모든 필수 도구 준비 완료!"
echo
echo "개발 워크플로우 (포폴 환경 — Colima 는 필요할 때만 기동):"
echo "  시작:  make dev-up      # Colima + MySQL 동시 기동"
echo "  실행:  make run-mysql   # 서버 기동 (MYSQL_DSN 자동 세팅)"
echo "  검증:  curl localhost:5050/api/profile/p1"
echo "  종료:  make dev-down    # MySQL + Colima 둘 다 정리 (배터리 안전)"
echo
echo "개별 제어:"
echo "  make docker-up / docker-down / docker-status"
echo "  make mysql-dev / mysql-stop"
echo
