-- sqlc 쿼리 — Phase 5 ガチャ.

-- name: GetPity :one
SELECT counter
FROM gacha_pity
WHERE player_id = ? AND pool_id = ?;

-- name: UpsertPity :exec
INSERT INTO gacha_pity (player_id, pool_id, counter)
VALUES (?, ?, ?)
ON DUPLICATE KEY UPDATE counter = VALUES(counter);

-- name: InsertRoll :exec
INSERT INTO gacha_rolls (id, player_id, pool_id, card_id, rarity, is_pity, pulled_at)
VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: ListRollsByPlayer :many
SELECT id, player_id, pool_id, card_id, rarity, is_pity, pulled_at
FROM gacha_rolls
WHERE player_id = ?
ORDER BY pulled_at DESC
LIMIT ?;
