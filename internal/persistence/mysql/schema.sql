-- schema.sql — sqlc 가 읽는 현재 상태 DDL.
-- goose migrations/ 의 최종 상태와 항상 일치해야 함.
-- 새 마이그레이션 추가 시 이 파일도 같이 갱신.

CREATE TABLE profiles (
    id         VARCHAR(64)  NOT NULL PRIMARY KEY,
    name       VARCHAR(128) NOT NULL,
    created_at TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Phase 5 — ガチャ
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

-- Phase 6 — イベント
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

-- Phase 19 — HP 同時減算デッドロックラボ (battle domain)
CREATE TABLE player_hp (
    player_id  VARCHAR(64) NOT NULL PRIMARY KEY,
    hp         INT         NOT NULL,
    updated_at TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
