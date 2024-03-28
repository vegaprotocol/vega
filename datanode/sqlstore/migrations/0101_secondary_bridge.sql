-- +goose Up

DROP VIEW IF EXISTS assets_current;

ALTER TABLE assets
    ADD COLUMN IF NOT EXISTS chain_id VARCHAR NOT NULL default '';

ALTER TABLE erc20_multisig_signer_events 
    ADD COLUMN IF NOT EXISTS chain_id VARCHAR NOT NULL default '';

CREATE VIEW assets_current AS
(
SELECT DISTINCT ON (id) *
FROM assets
ORDER BY id, vega_time DESC
    );

-- +goose StatementBegin
DO
$$
    DECLARE
        primary_chain_id VARCHAR;
    BEGIN
        -- All existing assets come have been enable on the primary bridge.
        -- So it's safe to update all assets with the chain ID configured in the
        -- network parameters.
        SELECT value::JSONB ->> 'chain_id' as chain_id
        INTO primary_chain_id
        FROM network_parameters_current
        WHERE key = 'blockchains.ethereumConfig';

        UPDATE assets SET chain_id = primary_chain_id;
        UPDATE erc20_multisig_signer_events SET chain_id = primary_chain_id;
    END;
$$;
-- +goose StatementEnd


-- +goose Down
DROP VIEW IF EXISTS assets_current;

ALTER TABLE assets
    DROP COLUMN IF EXISTS chain_id;

CREATE VIEW assets_current AS
(
SELECT DISTINCT ON (id) *
FROM assets
ORDER BY id, vega_time DESC
    );


ALTER TABLE erc20_multisig_signer_events 
    DROP COLUMN IF EXISTS chain_id;