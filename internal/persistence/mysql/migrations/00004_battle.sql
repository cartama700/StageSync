-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS player_hp (
    player_id VARCHAR(64) PRIMARY KEY,
    hp        INT NOT NULL,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS player_hp;
-- +goose StatementEnd
