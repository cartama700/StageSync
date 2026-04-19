# ADR-0002: DB 접근 레이어로 sqlc 선택

- 상태: Accepted
- 일자: 2026-04-18

## 맥락

Go 의 DB 접근 레이어 후보:

1. **`database/sql` 직접** — 의존성 없음. 쿼리·스캔을 매번 직접. 빠르게 쌓이면 오타·타입 불일치가 런타임까지 살아남음.
2. **`sqlx`** — `database/sql` 확장. `StructScan` 편의. 쿼리 자체는 여전히 문자열.
3. **GORM** — Ruby on Rails 계열 ORM. 메서드 체이닝으로 쿼리 조립. 생성 SQL 이 불투명. 조인·트랜잭션 복잡해지면 제어성 저하.
4. **ent** — Facebook 계열. 스키마를 Go 코드로 선언 → 클라이언트 코드 생성. 강력하지만 러닝커브 + DB 스키마가 "코드에서 생성됨" 모델.
5. **sqlc** — `.sql` 쿼리 파일을 작성 → Go 타입 안전 함수로 컴파일. **"SQL 이 원천, Go 가 소비자"** 모델.

공고 도메인 (Aurora MySQL + Cloud Spanner) 특징:
- 쿼리 최적화 감각이 채용 평가 포인트. SQL 자체를 읽고 튜닝하는 현업.
- Spanner 는 방언이 다르므로 **쿼리가 명시적으로 보여야** 이식·대조 가능.
- 트랜잭션 / `SELECT ... FOR UPDATE` / 윈도우 함수 같은 고급 SQL 을 ORM 추상에 가두면 디버깅 비용 증가.

## 결정

**sqlc 를 선택한다.** (`internal/persistence/mysql/queries/*.sql` → `gen/*.go`)

- `.sql` 파일을 SSOT (single source of truth) 로 삼음.
- `sqlc generate` 가 타입 안전 Go 함수 + `DBTX` 인터페이스 생성 → `*sql.DB` / `*sql.Tx` 양쪽 투명하게 처리.
- 생성 코드는 패키지 `gen` 아래 격리 → 도메인 레이어 (`domain/gacha`) 는 SQL 을 모름.

## 결과

**좋은 점**
- 쿼리가 단일 파일에 모여 있어 DBA / 운영 리뷰 대상이 명확함.
- 타입 안전 — 컬럼 추가 · 타입 변경이 컴파일 에러로 드러남.
- 트랜잭션 조합이 자연스러움 — [InsertRollsAndUpdatePity](../../internal/persistence/mysql/gacha_repo.go) 에서 BEGIN/INSERT/UPSERT/COMMIT 가 한 함수에.
- `DBTX` 덕분에 `go-sqlmock` 으로 완전한 단위 테스트 가능 ([gacha_repo_test.go](../../internal/persistence/mysql/gacha_repo_test.go)).

**나쁜 점**
- 동적 쿼리 (조건이 런타임에 조합되는 WHERE) 는 sqlc 가 약함. 필요시 `sq` / `squirrel` 같은 빌더 병용 여지.
- Spanner 용 별도 쿼리 파일을 Phase 12 에서 준비해야 함 — 이중 관리 부담.

**후속 작업**
- Phase 12 에서 Spanner 방언 쿼리 추가 시 sqlc engine 을 Spanner 용으로 분기하거나, Spanner 레이어는 Go 코드 기반 빌더로 쓸지 결정.
