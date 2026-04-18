-- schema.sql — sqlc 가 읽는 현재 상태 DDL.
-- goose migrations/ 의 최종 상태와 항상 일치해야 함.
-- 새 마이그레이션 추가 시 이 파일도 같이 갱신.

CREATE TABLE profiles (
    id         VARCHAR(64)  NOT NULL PRIMARY KEY,
    name       VARCHAR(128) NOT NULL,
    created_at TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
