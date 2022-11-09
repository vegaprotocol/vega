-- +goose Up
ALTER TABLE trades ADD COLUMN testref text;
Update trades set testref = 'nondefaulttestref';

ALTER TABLE trades ALTER COLUMN testref SET DEFAULT 'defaulttestref';


-- +goose Down
ALTER TABLE trades DROP COLUMN testref;