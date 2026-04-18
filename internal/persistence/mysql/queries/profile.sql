-- sqlc 쿼리 파일 — 각 쿼리는 '-- name: Xxx :type' 주석으로 이름·반환 형태 선언.
-- :one  → 단일 row · 없으면 sql.ErrNoRows
-- :many → 여러 row
-- :exec → 실행만 (INSERT/UPDATE/DELETE)
-- :execrows → 실행 + 영향받은 row 수 반환

-- name: GetProfile :one
SELECT id, name, created_at
FROM profiles
WHERE id = ?;

-- name: CreateProfile :exec
INSERT INTO profiles (id, name, created_at)
VALUES (?, ?, ?);
