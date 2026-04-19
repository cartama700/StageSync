-- sqlc 쿼리 — Phase 6 イベント.

-- name: CreateEvent :exec
INSERT INTO events (id, name, start_at, end_at)
VALUES (?, ?, ?, ?);

-- name: GetEvent :one
SELECT id, name, start_at, end_at, created_at
FROM events
WHERE id = ?;

-- name: ListCurrentEvents :many
SELECT id, name, start_at, end_at, created_at
FROM events
WHERE start_at <= ? AND end_at >= ?
ORDER BY end_at ASC;

-- name: AddEventScore :exec
INSERT INTO event_scores (event_id, player_id, points)
VALUES (?, ?, ?)
ON DUPLICATE KEY UPDATE points = points + VALUES(points);

-- name: GetEventScore :one
SELECT event_id, player_id, points, updated_at
FROM event_scores
WHERE event_id = ? AND player_id = ?;

-- name: ListRewardTiers :many
SELECT event_id, tier, min_points, reward_name
FROM event_rewards
WHERE event_id = ?
ORDER BY tier ASC;

-- name: InsertRewardTier :exec
INSERT INTO event_rewards (event_id, tier, min_points, reward_name)
VALUES (?, ?, ?, ?);
