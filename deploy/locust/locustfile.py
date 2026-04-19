"""
StageSync — event open spike (cluster) 부하 시나리오.

실행 예:
  # headless, 500 유저까지 10 초 램프업, 1 분 유지, HTML 리포트 생성
  locust -f deploy/locust/locustfile.py --host http://localhost:5050 \
         --headless -u 500 -r 10 -t 1m --html=locust_report.html

  # GUI 모드 (http://localhost:8089)
  locust -f deploy/locust/locustfile.py --host http://localhost:5050

  # Docker compose load profile 과 함께
  (터미널 1) docker compose --profile load up --build
  (터미널 2) locust ... --host http://localhost:5050

시나리오 요약:
  - 모든 유저가 동일한 event_id 에 점수를 몰아 랭킹 hot-spot 재현.
  - 3 엔드포인트 3:2:1 가중치 — post_score / gacha_roll / ranking_top.
"""

import logging
import random
import threading
import uuid
from datetime import datetime, timedelta, timezone

from locust import FastHttpUser, between, task

# ---------------------------------------------------------------------------
# 공유 상태 — worker 당 1 회만 이벤트를 만들고, 이후 모든 유저가 같은 id 사용.
# ---------------------------------------------------------------------------
_EVENT_LOCK = threading.Lock()
_EVENT_ID: str | None = None  # 모든 유저가 같은 event_id 에 점수 몰아 → 랭킹 hot-spot 시뮬

log = logging.getLogger(__name__)


def _now_iso() -> str:
    """RFC3339 UTC — Go time.Parse(time.RFC3339) 와 호환."""
    return datetime.now(timezone.utc).replace(microsecond=0).isoformat().replace("+00:00", "Z")


def _later_iso(delta: timedelta) -> str:
    return (
        (datetime.now(timezone.utc) + delta)
        .replace(microsecond=0)
        .isoformat()
        .replace("+00:00", "Z")
    )


class ClusterUser(FastHttpUser):
    """이벤트 개시 스파이크 (T-60s ~ T+30s) 유저."""

    wait_time = between(0.1, 1.0)  # 게임 틱 모사

    def on_start(self) -> None:
        global _EVENT_ID

        self.user_id = f"load-{uuid.uuid4().hex[:8]}"

        with _EVENT_LOCK:
            if _EVENT_ID is None:
                candidate = f"load-event-{uuid.uuid4().hex[:12]}"
                body = {
                    "id": candidate,
                    "name": "load test event",
                    "start_at": _now_iso(),
                    "end_at": _later_iso(timedelta(hours=1)),
                    "rewards": [],
                }
                with self.client.post(
                    "/api/event",
                    json=body,
                    name="/api/event (bootstrap)",
                    catch_response=True,
                ) as resp:
                    if resp.status_code == 201:
                        _EVENT_ID = candidate
                        resp.success()
                        log.info("bootstrap event_id=%s", _EVENT_ID)
                    elif resp.status_code == 409:
                        # uuid 충돌은 사실상 발생 안 함. 경고만 남기고 다음 유저가 재시도.
                        log.warning("event bootstrap 409 conflict — retry on next user")
                        resp.success()  # locust 에러 카운트에 잡히지 않도록
                    else:
                        resp.failure(
                            f"event bootstrap failed: {resp.status_code} {resp.text[:200]}"
                        )
                        return

    # ----- tasks ----------------------------------------------------------

    @task(3)
    def post_score(self) -> None:
        if _EVENT_ID is None:
            return
        self.client.post(
            f"/api/event/{_EVENT_ID}/score",
            json={"player": self.user_id, "delta": random.randint(10, 1000)},
            name="/api/event/{id}/score",
        )

    @task(2)
    def gacha_roll(self) -> None:
        self.client.post(
            "/api/gacha/roll",
            json={
                "player": self.user_id,
                "pool": "demo",
                "count": random.choice([1, 10]),
            },
            name="/api/gacha/roll",
        )

    @task(1)
    def ranking_top(self) -> None:
        if _EVENT_ID is None:
            return
        self.client.get(
            f"/api/ranking/{_EVENT_ID}/top?n=10",
            name="/api/ranking/{id}/top",
        )
