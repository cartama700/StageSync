# StageSync 面接ピッチスクリプト

> カジュアル面談 / 技術面接で「StageSync とは?」と聞かれた時の応答ドラフト。
> **30 秒** (つかみ) · **2 分** (技術深掘り) · **5 分** (プロセス・学び) の 3 レベル。
> 提出書類に入れるものではなく、本人練習用。

---

## 🎯 30 秒ピッチ (日本語 · メイン)

> 「StageSync は、Colorful Palette の BA-09-04a サーバサイド求人に応募するために作ったポートフォリオです。
> 求人に明記されていた Aurora MySQL · Redis · Docker · Kubernetes · Locust を全部使って、
> プロフィール · ガチャ · イベント · ランキングの 4 つの REST API を 3 日で実装しました。
> ガチャは 10-roll 原子トランザクション · 天井 80 回 · 確率分布 10,000 サンプルで検証してあります。
> `docker compose up` 一発で MySQL · Redis · 負荷 bot まで全部立ち上がります。
> ランキングは Redis ZSET、`REDIS_ADDR` が無ければ inmem に graceful degrade します。
> Go は今回が初めての実務挑戦ですが、AI ツールを活用して学習速度を証明するのが本プロジェクトの狙いです。」

**Key phrase 候補** (必要に応じて差し替え):
- `原子トランザクション` (`げんしとらんざくしょん`)
- `天井システム` / `確率分布テスト`
- `graceful degrade`
- `readiness gate` / `preStop drain`
- `consumer-defined interface`

---

## 🎯 30 초 피치 (한국어 · 예비)

> "StageSync 는 Colorful Palette 의 BA-09-04a 서버사이드 공고에 지원하기 위해 만든 포트폴리오입니다.
> 공고에 명시된 Aurora MySQL · Redis · Docker · Kubernetes · Locust 를 전부 써서
> 프로필 · 가챠 · 이벤트 · 랭킹 4개 REST API 를 3일 만에 구현했어요.
> 가챠는 10-roll 원자 트랜잭션 · 천장 80회 · 10,000 샘플 분포 테스트로 검증했고,
> `docker compose up` 한 번으로 MySQL · Redis · 부하 봇까지 다 올라옵니다.
> 랭킹은 Redis ZSET 인데 `REDIS_ADDR` 없으면 inmem 으로 graceful degrade 하고요.
> Go 는 이번이 첫 실무 도전이지만 AI 툴 적극 활용으로 학습 속도를 증명하는 게 목표입니다."

---

## 🔧 2 分バージョン (技術深掘り)

> 「StageSync の構造は 3 層アーキテクチャです。handler → service → repository の consumer-defined interface パターンで、
> 例えば handler は自分が必要とするサービス I/F を自分のファイルに宣言するので、
> テスト時に簡単にモック注入できます。
>
> ポイントを 3 つ挙げると:
>
> **1 つ目、ガチャの原子トランザクション。** 10-roll は MySQL の単一トランザクションで
> INSERT 10 件 + UPSERT pity を実行します。`sqlc` が `DBTX` インターフェイスを生成するので
> `*sql.DB` と `*sql.Tx` の両方で同じクエリ関数が動きます。テストは `go-sqlmock` で
> rollback のシーケンスまで検証してあります。
>
> **2 つ目、ランキングの graceful degrade。** Redis ZSET で `ZINCRBY` / `ZREVRANGE` を使ってますが、
> `REDIS_ADDR` が空だと `main.go` で inmem leaderboard に fallback します。
> Event サービスは `LeaderboardWriter` インターフェイスだけを要求する consumer-defined パターンで、
> Redis が落ちても MySQL=truth として機能し続けます。
>
> **3 つ目、観測性と graceful shutdown。** Prometheus `/metrics` に
> `http_request_duration_seconds` を method × chi RoutePattern × status で出していて、
> path は `/api/event/{id}` の形で維持するので high cardinality になりません。
> SIGTERM を受けると readiness が `false` に切り替わって `/health/ready` が 503 を返し、
> 5 秒待ってから `http.Server.Shutdown()` を実行します。
> K8s manifest の `terminationGracePeriodSeconds: 60` と組み合わせて
> in-flight リクエストを保護する設計です。」

---

## 📚 5 分バージョン (プロセス · 学び)

**Q: なぜ Go を選んだ?**
> 「公고が Go · Java 両方対応だったのと、Project Sekai の実プロダクションが Go + Java の
> ハイブリッドだからです。私は C#/.NET の経験が数年あるので、文法的には Java より Go の方が
> 近いと感じました。ゴルーチン · channel · context propagation のようなパターンは
> 初めてでしたが、実装しながら体で覚えるのが一番早いと判断して Go 一本にしました。」

**Q: 3 日で 14 Phase は無理があるのでは?**
> 「最初は 7-9 週の計画でした。ただ 3 日目の時点で、時間で完成させるべき Phase を全部やると
> 『半分だけできた Phase』が増えてポートフォリオの完成度を下げると気づきました。
> それで PLAN を v2 → v3 に再編して、公告の核心スタック (MySQL · Redis · Docker · K8s · Locust) を
> 確実にカバーする 15 Phase に絞り、Spanner や LLM のような『あると豪華だけど核じゃない』ものは
> 除外しました。除外理由も PLAN.md に書いてあります — 面接で突っ込まれた時の根拠です。」

**Q: AI ツールはどう使った?**
> 「設計の壁打ち · 定型コードの生成 · テストケース列挙 · 文書化に使いました。
> ただ『AI が書いたコードをそのままコミット』はしてません。全ての diff を自分で読んで、
> 理解できないパターンは一度消して自力で書き直すルールにしました。
> 結果として 3 日で 3,000 LoC + 181 テスト + コメント付きの状態まで到達できました。
> AI は学習加速器として使い、コードの理解責任は自分が持つ — これが自分のスタイルです。」

**Q: 一番苦労したのは?**
> 「Redis ZSET の動点処理が inmem 実装と違っていたことです。
> ZREVRANGE のデフォルトが lex DESC で、私の inmem 実装は playerID ASC で実装していたので
> テストが落ちました。`go-sqlmock` のモック実装を頼りにするのではなく、実行結果から
> Redis の実際の挙動を確かめて、inmem を一致させる方向で修正しました。
> 『二つの実装間で挙動が微妙に違う』 こと自体がバグだと学びました。」

**Q: 提出後に何をする?**
> 「Phase 19 の HP 同時減算デッドロックラボを追加します。
> `SELECT ... FOR UPDATE` で一人のユーザーに集中するトラフィックでデッドロックを再現し、
> ユーザー別のパーティションキューで直列化して解消 · ベンチ比較を 3 段構造 (v1 naive → v2 queue → v3 write-behind)
> で残す計画です。これは実務で経験した事件のシミュレーションでもあります。
> 面接の間に追加することで、『最近の更新』として会話のきっかけにする狙いです。」

---

## 🇯🇵 日本語メモ (発音注意)

- **原子 (げんし) トランザクション** — 「原子」は「原子力」の「原子」。
- **天井 (てんじょう) システム** — 天井の上限が SSR 確定。
- **冪等性 (べきとうせい) / idempotency** — ガチャが提出スコープ外の理由 (Phase 11).
- **観測性 (かんそくせい) / observability** — Prometheus の話題で使う。
- **ランキング** 외래어 그대로. 英語: leaderboard.
- **イベント** 같은 단어는 피해야 하면: `キャンペーン` 로 바꿔도 통함.

---

## 🛡️ 한계 · 트레이드오프 브리핑

면접관이 "이거 왜 안 했나?" 라고 묻기 **전에 먼저 선제**.

상세 대응 논리 5 가지 (인증 · 분산 Room · 마스터 데이터 캐시 · 재화 도메인 · Idempotency / Rate Limit) 는 [`TRADEOFFS.md`](./TRADEOFFS.md) 에 일본어 대응 문구까지 전부 정리해둠. 면접 전에 한 번 통독 → 자주 나올 각 항목의 **日本語 대응 문구** 를 암기.

---

## 🚫 避けたい表現

- ❌ 「**全部できました**」 — 실제는 v3 MVP 스코프. `MVP スコープは完了しました` 라고 정직하게.
- ❌ 「**簡単です**」— 어느 레벨에서든 겸손. `実装しながら学びました` 가 안전.
- ❌ 「**AI が全部やりました**」 — AI 는 보조. `AI を学習加速器として使いました` 가 실제이자 어필 포인트.
- ❌ 「**Go 経験は今回が初です**」 를 맨 앞에 두지 마라 — 성과 먼저, 실무 경력 맥락은 뒤에.

---

## 📝 연습 체크리스트

- [ ] 30 초 버전을 시간 안에 읽을 수 있음 (JP · ko 양쪽)
- [ ] 2 분 버전에서 "3 つのポイント" 를 암기
- [ ] 기술 용어 발음 확인 (`げんし` · `てんじょう` · `べきとうせい` · `かんそくせい`)
- [ ] Q&A 5 개에 대해 60 초 내 답변 준비
- [ ] 어떤 파일을 요청 받으면 즉시 열 수 있는지 (`internal/service/gacha/service.go` 등)
