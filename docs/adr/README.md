# Architecture Decision Records

> 프로젝트의 중요한 기술적 선택 기록. Michael Nygard 템플릿 기반.
> 각 ADR 은 일자 · 맥락 · 결정 · 결과를 명시 — 미래의 리뷰어·협업자가
> "왜 이렇게 했나" 를 코드·커밋로그 재구성 없이 확인 가능하도록.

## 목록

| ADR | 제목 | 상태 |
|---|---|---|
| [0001](./0001-chi-over-gin.md) | Router 로 chi 선택 (vs Gin/Echo) | Accepted |
| [0002](./0002-sqlc-over-orm.md) | DB 접근 레이어로 sqlc 선택 (vs GORM/ent) | Accepted |
| [0003](./0003-h2c-for-websocket-coexistence.md) | HTTP/2 cleartext (h2c) 로 WebSocket 과 공존 | Accepted |

## 포맷

```markdown
# ADR-NNNN: 제목

- 상태: Proposed | Accepted | Superseded by ADR-XXXX
- 일자: YYYY-MM-DD

## 맥락 (Context)
무엇이 문제인가. 왜 지금 결정해야 하는가.

## 결정 (Decision)
무엇을 선택했는가. 동사로 시작.

## 결과 (Consequences)
좋은 점 · 나쁜 점 · 따라오는 작업.
```
