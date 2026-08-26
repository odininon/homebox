-- +goose Up
ALTER TABLE entities ADD COLUMN current_market_price real DEFAULT 0 NOT NULL;
ALTER TABLE entities ADD COLUMN last_price_sync_at datetime;
ALTER TABLE entities ADD COLUMN price_tracking_enabled integer DEFAULT 0 NOT NULL;
ALTER TABLE entities ADD COLUMN price_tracking_source text DEFAULT '' NOT NULL;
ALTER TABLE entities ADD COLUMN price_tracking_id text DEFAULT '' NOT NULL;

CREATE TABLE IF NOT EXISTS entity_price_histories
(
    id          uuid PRIMARY KEY NOT NULL,
    created_at  datetime         NOT NULL,
    updated_at  datetime         NOT NULL,
    price       real             NOT NULL,
    market_low  real DEFAULT 0   NOT NULL,
    market_mid  real DEFAULT 0   NOT NULL,
    market_high real DEFAULT 0   NOT NULL,
    direct_low  real DEFAULT 0   NOT NULL,
    source      text DEFAULT ''  NOT NULL,
    source_id   text DEFAULT ''  NOT NULL,
    recorded_at datetime         NOT NULL,
    notes       text DEFAULT ''  NOT NULL,
    entity_id   uuid             NOT NULL
        CONSTRAINT entity_price_histories_entities_price_history
            REFERENCES entities
            ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS entity_price_history_entity_id
    ON entity_price_histories (entity_id);

CREATE INDEX IF NOT EXISTS entity_price_history_recorded_at
    ON entity_price_histories (recorded_at);

-- +goose Down
DROP TABLE IF EXISTS entity_price_histories;
ALTER TABLE entities DROP COLUMN current_market_price;
ALTER TABLE entities DROP COLUMN last_price_sync_at;
ALTER TABLE entities DROP COLUMN price_tracking_enabled;
ALTER TABLE entities DROP COLUMN price_tracking_source;
ALTER TABLE entities DROP COLUMN price_tracking_id;
