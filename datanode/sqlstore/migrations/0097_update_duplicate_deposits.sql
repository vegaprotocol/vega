-- +goose Up
WITH finalized_tx AS (
    SELECT foreign_tx_hash AS ftx,
    vega_time as vt
    FROM deposits
    WHERE status = 'STATUS_FINALIZED'
) UPDATE deposits
SET status = 'STATUS_DUPLICATE_REJECTED'
WHERE status = 'STATUS_OPEN'
AND foreign_tx_hash IN (SELECT ftx FROM finalized_tx WHERE vt < vega_time);

-- +goose Down

-- Nothing to do
