-- +goose Up
-- +goose StatementBegin
CREATE TABLE gacha_rolls (
    id         CHAR(36)    NOT NULL PRIMARY KEY,
    player_id  VARCHAR(64) NOT NULL,
    pool_id    VARCHAR(64) NOT NULL,
    card_id    VARCHAR(64) NOT NULL,
    rarity     VARCHAR(8)  NOT NULL,
    is_pity    TINYINT(1)  NOT NULL DEFAULT 0,
    pulled_at  TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_player_pulled (player_id, pulled_at DESC),
    INDEX idx_player_pool (player_id, pool_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE gacha_pity (
    player_id  VARCHAR(64) NOT NULL,
    pool_id    VARCHAR(64) NOT NULL,
    counter    INT         NOT NULL DEFAULT 0,
    updated_at TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (player_id, pool_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE gacha_pity;
DROP TABLE gacha_rolls;
-- +goose StatementEnd
