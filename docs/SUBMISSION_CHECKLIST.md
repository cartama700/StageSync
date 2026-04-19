# StageSync 제출 체크리스트 (Phase 18)

> 공고 [BA-09-04a](https://hrmos.co/pages/colorfulpalette/jobs/BA-09-04a) 제출 직전 최종 확인 목록.
> 체크 후 `git tag v0.1` → push → 리쿠르터에게 URL 전송.

---

## A. 코드 품질 (CI 가 확인해주지만 로컬에서도 한 번 더)

- [ ] `go build ./...` 성공
- [ ] `go vet ./...` 0 건
- [ ] `go test ./... -count=1` 전 패키지 PASS (181 이상)
- [ ] `go test -race ./...` (Linux CI 에서만 확인 가능 — Windows 로컬은 gcc 필요)
- [ ] `golangci-lint run` 0 issue
  ```bash
  # 로컬 설치 안 돼 있으면 Docker 로:
  docker run --rm -v $(pwd):/app -w /app golangci/golangci-lint:v2.11.4 golangci-lint run
  ```
- [ ] `go mod tidy` 후 git diff 없음 (CI 에서도 체크하지만 미리)

## B. 문서 (모두 최신 상태인지)

- [ ] `README.md` (JP 기본) — 기술 스택 / Phase 진행 / Quickstart 최신
- [ ] `README.ko.md` — README.md 와 내용 동기화
- [ ] `docs/STATUS.md` — 완료된 Phase 반영
- [ ] `docs/PLAN.md` — v3 스코프 확정 (제외 Phase 근거 명시)
- [ ] `docs/API.md` — 19 엔드포인트 모두 문서화
- [ ] `docs/BENCHMARKS.md` — AOI 실측 + Locust 섹션 (결과 공란도 OK)
- [ ] `docs/adr/` — 3 건 (chi · sqlc · h2c)
- [ ] `CHANGELOG.md` — `v0.1` 태그 섹션 완성
- [ ] `docs/PITCH.md` — 면접용 피치 (본인 연습)

## C. 데모 자산 (GIF)

- [ ] `docs/demo/quickstart.gif` — `docker compose up` + curl 연속 (필수)
- [ ] `docs/demo/loadtest.gif` — Locust + Prometheus (권장)
- [ ] (선택) `docs/demo/drain.gif` — SIGTERM → 503 → clean shutdown
- [ ] README.md / README.ko.md 의 "概要" 섹션 아래에 GIF embed

## D. Docker / K8s 실제 검증

- [ ] `docker compose up --build` → 3 컨테이너 모두 healthy
- [ ] `docker compose --profile load up --build` → 5 컨테이너 (server + mysql + redis + 2 bots) 동작
- [ ] `curl localhost:5050/metrics` 에서 `stagesync_*` + `http_request_duration_seconds` 확인
- [ ] `kubectl apply --dry-run=client -f deploy/k8s/` 통과 (필요 시 `kubectl` 설치)

## E. 부하 테스트 (선택 — Phase 16 lite 결과 채우기)

- [ ] `docker compose --profile load up --build` 실행 중
- [ ] `locust -f deploy/locust/locustfile.py --host http://localhost:5050 --headless -u 500 -r 10 -t 1m`
- [ ] 결과 수치를 `docs/BENCHMARKS.md` 표에 기록
- [ ] Locust HTML 리포트 스크린샷을 `docs/assets/locust-cluster.png` 에 저장 (선택)

## F. Git / GitHub

- [ ] `git status` clean (커밋되지 않은 변경 없음)
- [ ] `main` 브랜치 기준 `git log` 가 읽을 만함 (squash/rebase 로 정리됐으면 OK)
- [ ] GitHub Actions CI **green** (main 브랜치 최신 커밋)
- [ ] 뱃지 렌더링 확인 (README 최상단 CI/Go/License 3 개)
- [ ] `git tag v0.1` 후 `git push --tags`
- [ ] (선택) GitHub Release 작성 — CHANGELOG 의 `[v0.1]` 섹션 그대로 복사

## G. 제출 메시지 (JP)

에이전트/리쿠르터에게 보낼 링크 + 한 줄 설명:

```
ポートフォリオを ​更新しました — https://github.com/cartama700/StageSync

- 3 日間で Phase 0-16 (REST + MySQL + Redis + Docker + K8s + Locust) を実装
- `docker compose up --build` で 30 秒で動作確認可能
- README (日本語) を主、README.ko.md も同期済み
- 詳細: docs/PLAN.md (ロードマップ) · docs/STATUS.md (現状)
```

한국어 版 (필요 시):
```
포트폴리오 업데이트 — https://github.com/cartama700/StageSync

- 3 일 만에 Phase 0-16 (REST + MySQL + Redis + Docker + K8s + Locust) 구현
- `docker compose up --build` 로 30 초 내 동작 확인 가능
- README 는 일본어 기본, 한국어 미러도 제공
- 상세: docs/PLAN.md · docs/STATUS.md
```

## H. 면접 준비 (제출 후)

- [ ] `docs/PITCH.md` 의 30 초 / 2 분 / 5 분 버전 연습
- [ ] 키 파일 경로 암기: `internal/service/gacha/service.go` · `persistence/mysql/gacha_repo.go` · `persistence/redis/leaderboard.go` · `cmd/server/main.go`
- [ ] Q&A 5 개 (`docs/PITCH.md` 참조)
- [ ] Phase 19 (HP 데드락 랩) 를 면접 기간 중 추가 → "최근 업데이트" 로 어필

---

## 🎯 최소 제출 기준 (이것만 돼도 OK)

다음이 모두 ✅ 라면 제출 가능:
- A 모두 통과
- B 에서 README.md · README.ko.md · STATUS.md · CHANGELOG.md 4 개 최신
- C 에서 최소 `quickstart.gif` 1 개
- F 에서 CI green + tag v0.1 push

나머지는 제출 후 면접 기간 중 보강 가능.
