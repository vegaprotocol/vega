-- +goose Up

CREATE TYPE position_status_type AS enum('POSITION_STATUS_UNSPECIFIED', 'POSITION_STATUS_ORDERS_CLOSED', 'POSITION_STATUS_CLOSED_OUT');

ALTER TABLE positions
  ADD COLUMN loss_socialisation_amount   NUMERIC,
  ADD COLUMN distressed_status position_status_type;


UPDATE positions SET
  loss_socialisation_amount = ABS(loss) - adjustment,
  distressed_status = 'POSITION_STATUS_UNSPECIFIED';


ALTER TABLE positions
  ALTER COLUMN loss_socialisation_amount   SET NOT NULL,
  ALTER COLUMN distressed_status           SET NOT NULL;

ALTER TABLE positions_current
  ADD COLUMN loss_socialisation_amount   NUMERIC,
  ADD COLUMN distressed_status position_status_type;

UPDATE positions_current SET
  loss_socialisation_amount = ABS(loss) - adjustment,
  distressed_status = 'POSITION_STATUS_UNSPECIFIED';

ALTER TABLE positions_current
  ALTER COLUMN loss_socialisation_amount   SET NOT NULL,
  ALTER COLUMN distressed_status           SET NOT NULL;

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION update_current_positions()
    RETURNS TRIGGER
    LANGUAGE PLPGSQL AS
$$
BEGIN
    INSERT INTO positions_current(market_id,party_id,open_volume,realised_pnl,unrealised_pnl,average_entry_price,average_entry_market_price,loss,adjustment,tx_hash,vega_time,pending_open_volume,pending_realised_pnl,pending_unrealised_pnl,pending_average_entry_price,pending_average_entry_market_price, loss_socialisation_amount, distressed_status)
    VALUES(NEW.market_id,NEW.party_id,NEW.open_volume,NEW.realised_pnl,NEW.unrealised_pnl,NEW.average_entry_price,NEW.average_entry_market_price,NEW.loss,NEW.adjustment,NEW.tx_hash,NEW.vega_time,NEW.pending_open_volume,NEW.pending_realised_pnl,NEW.pending_unrealised_pnl,NEW.pending_average_entry_price,NEW.pending_average_entry_market_price, NEW.loss_socialisation_amount, NEW.distressed_status)
    ON CONFLICT(party_id, market_id) DO UPDATE SET
                                                   open_volume=EXCLUDED.open_volume,
                                                   realised_pnl=EXCLUDED.realised_pnl,
                                                   unrealised_pnl=EXCLUDED.unrealised_pnl,
                                                   average_entry_price=EXCLUDED.average_entry_price,
                                                   average_entry_market_price=EXCLUDED.average_entry_market_price,
                                                   loss=EXCLUDED.loss,
                                                   adjustment=EXCLUDED.adjustment,
                                                   tx_hash=EXCLUDED.tx_hash,
                                                   vega_time=EXCLUDED.vega_time,
                                                   pending_open_volume=EXCLUDED.pending_open_volume,
                                                   pending_realised_pnl=EXCLUDED.pending_realised_pnl,
                                                   pending_unrealised_pnl=EXCLUDED.pending_unrealised_pnl,
                                                   pending_average_entry_price=EXCLUDED.pending_average_entry_price,
                                                   pending_average_entry_market_price=EXCLUDED.pending_average_entry_market_price,
                                                   loss_socialisation_amount=EXCLUDED.loss_socialisation_amount,
                                                   distressed_status=EXCLUDED.distressed_status;
    RETURN NULL;
END;
$$;
-- +goose StatementEnd


-- +goose Down

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION update_current_positions()
    RETURNS TRIGGER
    LANGUAGE PLPGSQL AS
$$
BEGIN
    INSERT INTO positions_current(market_id,party_id,open_volume,realised_pnl,unrealised_pnl,average_entry_price,average_entry_market_price,loss,adjustment,tx_hash,vega_time,pending_open_volume,pending_realised_pnl,pending_unrealised_pnl,pending_average_entry_price,pending_average_entry_market_price)
    VALUES(NEW.market_id,NEW.party_id,NEW.open_volume,NEW.realised_pnl,NEW.unrealised_pnl,NEW.average_entry_price,NEW.average_entry_market_price,NEW.loss,NEW.adjustment,NEW.tx_hash,NEW.vega_time,NEW.pending_open_volume,NEW.pending_realised_pnl,NEW.pending_unrealised_pnl,NEW.pending_average_entry_price,NEW.pending_average_entry_market_price)
    ON CONFLICT(party_id, market_id) DO UPDATE SET
                                                   open_volume=EXCLUDED.open_volume,
                                                   realised_pnl=EXCLUDED.realised_pnl,
                                                   unrealised_pnl=EXCLUDED.unrealised_pnl,
                                                   average_entry_price=EXCLUDED.average_entry_price,
                                                   average_entry_market_price=EXCLUDED.average_entry_market_price,
                                                   loss=EXCLUDED.loss,
                                                   adjustment=EXCLUDED.adjustment,
                                                   tx_hash=EXCLUDED.tx_hash,
                                                   vega_time=EXCLUDED.vega_time,
                                                   pending_open_volume=EXCLUDED.pending_open_volume,
                                                   pending_realised_pnl=EXCLUDED.pending_realised_pnl,
                                                   pending_unrealised_pnl=EXCLUDED.pending_unrealised_pnl,
                                                   pending_average_entry_price=EXCLUDED.pending_average_entry_price,
                                                   pending_average_entry_market_price=EXCLUDED.pending_average_entry_market_price;
    RETURN NULL;
END;
$$;
-- +goose StatementEnd

ALTER TABLE positions
  DROP COLUMN IF EXISTS loss_socialisation_amount,
  DROP COLUMN IF EXISTS distressed_status;

ALTER TABLE positions_current
  DROP COLUMN IF EXISTS loss_socialisation_amount,
  DROP COLUMN IF EXISTS distressed_status;

DROP TYPE IF EXISTS position_status_type;
