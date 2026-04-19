# StageSync — デモ GIF 録画ガイド

> Phase 18 の最終タスク。リポジトリ提出前に **2 種のデモ GIF** を録画してここに配置。
> README.md / README.ko.md に `![demo](docs/demo/NAME.gif)` 형태로 embed.

## 録画ツール推奨

- **macOS**: [Kap](https://getkap.co/) (무료 · 드래그 선택) 또는 **QuickTime → ffmpeg 로 gif 변환**.
- **Windows**: [ScreenToGif](https://www.screentogif.com/) (무료 · 한국어 지원 좋음).
- **Linux**: [Peek](https://github.com/phw/peek) 또는 `ffmpeg` 직접.

GIF 최적화: 완료 후 [gifski](https://gif.ski/) 로 프레임레이트 10-12fps · 팔레트 128 색 정도로 압축. 각 GIF **3 MB 이하** 권장 (GitHub 렌더링 속도).

---

## GIF 1 — "30 秒で REST が動く" (필수)

**파일명**: `docs/demo/quickstart.gif`

**구성 (약 25-30 초)**

1. 터미널에서 `docker compose up --build` 실행 → server + mysql + redis 기동 로그 (5 초 가속 OK)
2. 다른 터미널:
   ```bash
   curl -X POST http://localhost:5050/api/profile \
        -H "Content-Type: application/json" \
        -d '{"id":"p1","name":"sekai"}'
   # → 201 Created

   curl http://localhost:5050/api/profile/p1
   # → 200 {"id":"p1","name":"sekai","created_at":"..."}

   curl -X POST http://localhost:5050/api/gacha/roll \
        -H "Content-Type: application/json" \
        -d '{"player":"p1","pool":"demo","count":10}'
   # → 201 [10 개 roll, SSR 가끔 섞임]

   curl http://localhost:5050/api/gacha/pity/p1/demo
   # → {"player":"p1","pool":"demo","counter":...}
   ```
3. 마지막에 `curl -s localhost:5050/metrics | grep stagesync_` 로 관측성 한 줄 증명.

**말하고 싶은 것 (설명 없이 화면으로)**:
- clone → `docker compose up` → 30 초 내 동작
- REST 엔드포인트 정상
- Prometheus scrape 준비됨

---

## GIF 2 — "負荷 + 観測性" (권장)

**파일명**: `docs/demo/loadtest.gif`

**구성 (약 40 초)**

1. 터미널 A: `docker compose --profile load up --build` — server + MySQL + Redis + 2종 bots
2. 터미널 B: Locust headless 실행
   ```bash
   locust -f deploy/locust/locustfile.py --host http://localhost:5050 \
          --headless -u 500 -r 10 -t 30s
   ```
3. 터미널 C: `watch -n 1 'curl -s localhost:5050/metrics | grep -E "stagesync_|http_request_duration"'`
   또는 prometheus UI 스샷 (있으면).
4. 결과 확인:
   - `stagesync_room_connected_players` = 100 (bots 2종 × 50)
   - `http_request_duration_seconds_bucket` 누적
   - Locust p95/p99 수치

**말하고 싶은 것**:
- 4 컨테이너 자동 구성
- Prometheus 지표가 실시간 움직임
- p99 수치가 리포트됨 → `docs/BENCHMARKS.md` 표에 기록하는 소재

---

## (선택) GIF 3 — "Graceful drain"

**파일명**: `docs/demo/drain.gif`

1. `docker compose up` 실행 중
2. 다른 터미널: `curl localhost:5050/health/ready` → `200`
3. `Ctrl+C` 로 SIGTERM 전송
4. 바로 `curl localhost:5050/health/ready` → `503 {"ready":false}` (5 초 동안)
5. 서버 로그에 `readiness set to draining` → `server stopped cleanly` 흐름

**목적**: Phase 14 lite 의 readiness drain 을 눈에 보이게 증명. K8s 배포 이해도 어필.

---

## 파일 정리 규칙

- GIF 배치: 이 디렉토리 (`docs/demo/`) 직하.
- 파일 크기: 개당 ~3 MB 이하 (README preview 속도 유지).
- 이름은 케밥케이스 (`quickstart.gif` · `loadtest.gif` · `drain.gif`).
- README 에 embed 후, 대체 텍스트로 무엇을 보여주는 GIF 인지 명시.
  예: `![30 초 만에 REST 기동 + curl 연속](docs/demo/quickstart.gif)`

## README 에 embed 할 위치

1. README.md (JP): **概要** 섹션 바로 아래, **作者について** 위에 삽입.
2. README.ko.md: 동일 위치 (개요 바로 아래).

예:
```markdown
## 概要
...

![30 秒で REST + Prometheus scrape](docs/demo/quickstart.gif)
![Locust 500 ユーザー + Prometheus histogram](docs/demo/loadtest.gif)

## 作者について
...
```
