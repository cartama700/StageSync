GO_BIN := $(shell go env GOPATH)/bin
export PATH := $(GO_BIN):$(PATH)

MYSQL_DSN ?= root:root@tcp(127.0.0.1:3306)/stagesync?parseTime=true&loc=Local

.PHONY: help proto sqlc run run-mysql build tidy clean test bench \
        migrate-up migrate-down \
        docker-up docker-down docker-status \
        mysql-dev mysql-stop dev-up dev-down

.DEFAULT_GOAL := help

## help: 이 Makefile 의 주요 명령 (기본값)
help:
	@echo "StageSync Makefile"
	@echo ""
	@echo "[개발 환경 — 포폴용 수동 제어]"
	@echo "  make dev-up           Colima + MySQL 동시 기동 (한 번에)"
	@echo "  make dev-down         전체 정리 (MySQL + Colima, 배터리 절약)"
	@echo "  make docker-up        Colima VM 만 기동"
	@echo "  make docker-down      Colima VM 정지"
	@echo "  make docker-status    Colima + 실행 중 컨테이너 상태"
	@echo "  make mysql-dev        MySQL 컨테이너만 기동 (Colima 필요)"
	@echo "  make mysql-stop       MySQL 컨테이너 정지"
	@echo ""
	@echo "[빌드·테스트]"
	@echo "  make tidy             go mod tidy"
	@echo "  make proto            .proto  → .pb.go"
	@echo "  make sqlc             .sql    → gen/*.go"
	@echo "  make build            bin/server + bin/bots"
	@echo "  make test             go test ./..."
	@echo "  make bench            AOI 벤치마크 (보너스축)"
	@echo ""
	@echo "[실행]"
	@echo "  make run              서버 (inmem fallback)"
	@echo "  make run-mysql        서버 (MYSQL_DSN 자동 세팅)"
	@echo ""
	@echo "[DB 마이그레이션 수동]"
	@echo "  make migrate-up       goose up"
	@echo "  make migrate-down     직전 goose down"
	@echo ""
	@echo "[정리]"
	@echo "  make clean            bin/ 와 생성물 삭제"

## proto: .proto 파일들을 Go 코드로 생성
proto:
	protoc --go_out=. --go_opt=paths=source_relative api/proto/roompb/room.proto

## sqlc: .sql 쿼리 → Go 타입 안전 코드 생성
sqlc:
	sqlc generate

## tidy: go.mod 정리 + 의존성 다운로드
tidy:
	go mod tidy

## run: 서버 실행 (MYSQL_DSN 미설정 시 inmem fallback)
run:
	go run ./cmd/server

## run-mysql: MySQL 연결 상태로 서버 실행
run-mysql:
	MYSQL_DSN="$(MYSQL_DSN)" go run ./cmd/server

## build: 서버 + 봇 바이너리 빌드
build:
	go build -o bin/server ./cmd/server
	go build -o bin/bots ./cmd/bots

## test: 모든 단위 테스트
test:
	go test ./...

## bench: AOI 벤치마크 (보너스축 증명)
bench:
	go test -bench=. -benchmem -benchtime=3s -count=3 ./internal/service/aoi/

## migrate-up: goose up CLI 실행 (서버 기동과 별도로 수동 마이그레이션)
migrate-up:
	cd internal/persistence/mysql && goose -dir migrations mysql "$(MYSQL_DSN)" up

## migrate-down: 직전 마이그레이션 rollback
migrate-down:
	cd internal/persistence/mysql && goose -dir migrations mysql "$(MYSQL_DSN)" down

## docker-up: Colima VM 기동 (Docker 사용 시작 전 1회)
docker-up:
	@if colima status >/dev/null 2>&1; then \
	  echo "Colima 이미 실행 중"; \
	else \
	  colima start && docker context use colima >/dev/null; \
	fi

## docker-down: Colima VM 정지 (리소스 반환 · 배터리 절약)
docker-down:
	@if colima status >/dev/null 2>&1; then \
	  colima stop; \
	else \
	  echo "Colima 이미 정지됨"; \
	fi

## docker-status: Colima · Docker 상태 확인
docker-status:
	@colima status 2>&1 || true
	@echo "---"
	@docker ps 2>&1 || echo "(docker daemon 미동작)"

## mysql-dev: 로컬 MySQL 컨테이너 기동 (Colima 먼저 필요)
mysql-dev:
	@colima status >/dev/null 2>&1 || { echo "❌ Colima 미기동. 'make docker-up' 먼저 실행."; exit 1; }
	docker run --name stagesync-mysql --rm -d \
	  -e MYSQL_ROOT_PASSWORD=root \
	  -e MYSQL_DATABASE=stagesync \
	  -p 3306:3306 \
	  mysql:8

## mysql-stop: 로컬 MySQL 컨테이너 종료
mysql-stop:
	@if docker ps --filter name=stagesync-mysql -q 2>/dev/null | grep -q .; then \
	  docker stop stagesync-mysql; \
	else \
	  echo "MySQL 컨테이너 이미 정지됨"; \
	fi

## dev-up: 개발 환경 전체 기동 (Colima + MySQL)
dev-up: docker-up mysql-dev
	@echo ""
	@echo "✓ 개발 환경 준비 완료."
	@echo "  → make run-mysql    # 서버 기동 (MYSQL_DSN 자동 세팅)"
	@echo "  → make dev-down     # 종료 시 (Colima · MySQL 둘 다 정리)"

## dev-down: 개발 환경 정리 (MySQL + Colima)
dev-down: mysql-stop docker-down
	@echo ""
	@echo "✓ 개발 환경 정리 완료. 배터리 안전."

## clean: 빌드 생성물 정리
clean:
	rm -rf bin/
	find . -name '*.pb.go' -delete
