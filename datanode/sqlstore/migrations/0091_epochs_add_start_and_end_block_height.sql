-- +goose Up

-- We will create a materialized view with the API data we need as it only needs to be refreshed when the epoch table is updated
-- we can trigger a refresh when that happens instead of having to query the same data every time.
CREATE MATERIALIZED VIEW IF NOT EXISTS current_epochs AS
    WITH epochs_current AS (
        SELECT DISTINCT ON (id) * FROM epochs ORDER BY id, vega_time DESC
    )
        SELECT e.id, e.start_time, e.expire_time, e.end_time, e.tx_hash, e.vega_time, bs.height first_block, be.height last_block
        FROM epochs_current AS e
        LEFT JOIN blocks bs on e.start_time = bs.vega_time
        LEFT JOIN blocks be on e.end_time = be.vega_time
WITH DATA;

-- We need to have a unique index on the materialized view in order to refresh it concurrently.
CREATE UNIQUE INDEX idx_uq_current_epochs_id on current_epochs(id);

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION refresh_current_epochs()
       RETURNS TRIGGER
       LANGUAGE plpgsql AS
$$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY current_epochs;
    RETURN NULL;
END;
$$
-- +goose StatementEnd

-- When an insert, update or delete happens on the epochs table, we want to refresh the materialized view.
-- This should be safe to do as an epoch only happens periodically, but the view may be queried very frequently.
CREATE OR REPLACE TRIGGER refresh_current_epochs
AFTER INSERT OR UPDATE OR DELETE ON epochs
FOR EACH STATEMENT
EXECUTE FUNCTION refresh_current_epochs();

-- +goose Down
DROP TRIGGER IF EXISTS refresh_current_epochs ON epochs;
DROP FUNCTION IF EXISTS refresh_current_epochs;
DROP MATERIALIZED VIEW IF EXISTS current_epochs;
