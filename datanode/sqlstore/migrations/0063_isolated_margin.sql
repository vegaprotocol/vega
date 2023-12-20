-- +goose Up
DROP TYPE IF EXISTS margin_mode_type;
create type margin_mode_type as enum('MARGIN_MODE_UNSPECIFIED', 'MARGIN_MODE_CROSS_MARGIN', 'MARGIN_MODE_ISOLATED_MARGIN');

alter table margin_levels
    add column if not exists margin_mode margin_mode_type,
    add column if not exists margin_factor NUMERIC,
    add column if not exists order_margin HUGEINT,
    add column if not exists order_margin_account_id bytea;

update margin_levels
    set margin_mode='MARGIN_MODE_CROSS_MARGIN',
        margin_factor=0,
        order_margin=0;

alter table margin_levels
    ALTER COLUMN margin_mode SET NOT NULL,
    ALTER COLUMN margin_factor SET NOT NULL,
    ALTER COLUMN order_margin SET NOT NULL;

alter table current_margin_levels
    add column if not exists margin_mode margin_mode_type,
    add column if not exists margin_factor NUMERIC,
    add column if not exists order_margin HUGEINT,
    add column if not exists order_margin_account_id bytea;

update current_margin_levels
    set margin_mode='MARGIN_MODE_CROSS_MARGIN',
        margin_factor=0,
        order_margin=0;

alter table current_margin_levels
    ALTER COLUMN margin_mode SET NOT NULL,
    ALTER COLUMN margin_factor SET NOT NULL,
    ALTER COLUMN order_margin SET NOT NULL;

-- +goose StatementBegin
drop trigger if exists update_current_margin_levels on margin_levels;
CREATE OR REPLACE FUNCTION update_current_margin_levels()
   RETURNS TRIGGER
   LANGUAGE PLPGSQL AS
$$
BEGIN
INSERT INTO current_margin_levels(account_id,
                                  order_margin_account_id,
                                  timestamp,
                                  maintenance_margin,
                                  search_level,
                                  initial_margin,
                                  collateral_release_level,
                                  order_margin,
                                  tx_hash,
                                  vega_time,
                                  margin_mode,
                                  margin_factor) VALUES(NEW.account_id,
                                                    NEW.order_margin_account_id,
                                                    NEW.timestamp,
                                                    NEW.maintenance_margin,
                                                    NEW.search_level,
                                                    NEW.initial_margin,
                                                    NEW.collateral_release_level,
                                                    NEW.order_margin,
                                                    NEW.tx_hash,
                                                    NEW.vega_time,
                                                    NEW.margin_mode,
                                                    NEW.margin_factor)
    ON CONFLICT(account_id) DO UPDATE SET
                                   order_margin_account_id=EXCLUDED.order_margin_account_id,
                                   timestamp=EXCLUDED.timestamp,
                                   maintenance_margin=EXCLUDED.maintenance_margin,
                                   search_level=EXCLUDED.search_level,
                                   initial_margin=EXCLUDED.initial_margin,
                                   collateral_release_level=EXCLUDED.collateral_release_level,
                                   order_margin=EXCLUDED.order_margin,
                                   tx_hash=EXCLUDED.tx_hash,
                                   vega_time=EXCLUDED.vega_time,
                                   margin_mode=EXCLUDED.margin_mode,
                                   margin_factor=EXCLUDED.margin_factor;


RETURN NULL;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER update_current_margin_levels AFTER INSERT ON margin_levels FOR EACH ROW EXECUTE function update_current_margin_levels();

DROP VIEW all_margin_levels;
DROP MATERIALIZED VIEW conflated_margin_levels;


CREATE MATERIALIZED VIEW conflated_margin_levels
            WITH (timescaledb.continuous, timescaledb.materialized_only = true) AS
SELECT account_id,
       order_margin_account_id,
       time_bucket('1 minute', vega_time) AS bucket,
       last(maintenance_margin, vega_time) AS maintenance_margin,
       last(search_level, vega_time) AS search_level,
       last(initial_margin, vega_time) AS initial_margin,
       last(collateral_release_level, vega_time) AS collateral_release_level,
       last(order_margin, vega_time) AS order_margin,
       last(timestamp, vega_time) AS timestamp,
       last(tx_hash, vega_time) AS tx_hash,
       last(vega_time, vega_time) AS vega_time,
       last(margin_mode, vega_time) AS margin_mode,
       last(margin_factor, vega_time) AS margin_factor
FROM margin_levels
GROUP BY account_id, order_margin_account_id, bucket WITH NO DATA;

-- start_offset is set to a day, as data is append only this does not impact the processing time and ensures
-- that the CAGG data will be correct on recovery in the event of a transient outage ( < 1 day )
SELECT add_continuous_aggregate_policy('conflated_margin_levels', start_offset => INTERVAL '1 day',
    end_offset => INTERVAL '1 minute', schedule_interval => INTERVAL '1 minute');

CREATE VIEW all_margin_levels AS
(
SELECT margin_levels.account_id,
       margin_levels.order_margin_account_id,
       margin_levels."timestamp",
       margin_levels.maintenance_margin,
       margin_levels.search_level,
       margin_levels.initial_margin,
       margin_levels.collateral_release_level,
       margin_levels.order_margin,
       margin_levels.tx_hash,
       margin_levels.vega_time,
       margin_levels.margin_mode,
       margin_levels.margin_factor
FROM margin_levels
UNION ALL
SELECT conflated_margin_levels.account_id,
       conflated_margin_levels.order_margin_account_id,
       conflated_margin_levels."timestamp",
       conflated_margin_levels.maintenance_margin,
       conflated_margin_levels.search_level,
       conflated_margin_levels.initial_margin,
       conflated_margin_levels.collateral_release_level,
       conflated_margin_levels.order_margin,
       conflated_margin_levels.tx_hash,
       conflated_margin_levels.vega_time,
       conflated_margin_levels.margin_mode,
       conflated_margin_levels.margin_factor
FROM conflated_margin_levels
WHERE conflated_margin_levels.vega_time < (SELECT coalesce(min(margin_levels.vega_time), 'infinity') FROM margin_levels));

-- +goose Down
-- nothing to do, we're not going to convert it back
