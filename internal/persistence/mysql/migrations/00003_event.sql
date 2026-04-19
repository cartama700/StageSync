-- +goose Up
-- +goose StatementBegin
CREATE TABLE events (
    id         VARCHAR(64)  NOT NULL PRIMARY KEY,
    name       VARCHAR(128) NOT NULL,
    start_at   TIMESTAMP    NOT NULL,
    end_at     TIMESTAMP    NOT NULL,
    created_at TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_event_window (start_at, end_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE event_scores (
    event_id   VARCHAR(64) NOT NULL,
    player_id  VARCHAR(64) NOT NULL,
    points     BIGINT      NOT NULL DEFAULT 0,
    updated_at TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (event_id, player_id),
    INDEX idx_event_points (event_id, points DESC)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE event_rewards (
    event_id    VARCHAR(64)  NOT NULL,
    tier        INT          NOT NULL,
    min_points  BIGINT       NOT NULL,
    reward_name VARCHAR(128) NOT NULL,
    PRIMARY KEY (event_id, tier)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE event_rewards;
DROP TABLE event_scores;
DROP TABLE events;
-- +goose StatementEnd
