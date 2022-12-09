-- +goose Up
ALTER TABLE trades ADD COLUMN testref text;
Update trades set testref = 'nondefaulttestref';

ALTER TABLE trades ALTER COLUMN testref SET DEFAULT 'defaulttestref';


create table transfers2 (
                            id bytea not null,
                            tx_hash bytea not null,
                            vega_time timestamp with time zone not null,
                            from_account_id bytea NOT NULL REFERENCES accounts(id),
                            to_account_id bytea NOT NULL REFERENCES accounts(id),
                            primary key (id, vega_time)
);

create index on transfers2 (from_account_id);
create index on transfers2 (to_account_id);

-- +goose Down
ALTER TABLE trades DROP COLUMN testref;
DROP TABLE transfers2;