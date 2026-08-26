-- +goose Up
ALTER TABLE entities ADD COLUMN IF NOT EXISTS current_market_price double precision DEFAULT 0 NOT NULL;
ALTER TABLE entities ADD COLUMN IF NOT EXISTS last_price_sync_at timestamptz;
ALTER TABLE entities ADD COLUMN IF NOT EXISTS price_tracking_enabled boolean DEFAULT false NOT NULL;
ALTER TABLE entities ADD COLUMN IF NOT EXISTS price_tracking_source text DEFAULT '' NOT NULL;
ALTER TABLE entities ADD COLUMN IF NOT EXISTS price_tracking_id text DEFAULT '' NOT NULL;

CREATE TABLE IF NOT EXISTS entity_price_histories
(
    id          uuid PRIMARY KEY NOT NULL,
    created_at  timestamptz      NOT NULL,
    updated_at  timestamptz      NOT NULL,
    price       double precision NOT NULL,
    market_low  double precision DEFAULT 0 NOT NULL,
    market_mid  double precision DEFAULT 0 NOT NULL,
    market_high double precision DEFAULT 0 NOT NULL,
    direct_low  double precision DEFAULT 0 NOT NULL,
    source      text DEFAULT ''  NOT NULL,
    source_id   text DEFAULT ''  NOT NULL,
    recorded_at timestamptz      NOT NULL,
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
ALTER TABLE entities DROP COLUMN IF EXISTS current_market_price;
ALTER TABLE entities DROP COLUMN IF EXISTS last_price_sync_at;
ALTER TABLE entities DROP COLUMN IF EXISTS price_tracking_enabled;
ALTER TABLE entities DROP COLUMN IF EXISTS price_tracking_source;
ALTER TABLE entities DROP COLUMN IF EXISTS price_tracking_id;
