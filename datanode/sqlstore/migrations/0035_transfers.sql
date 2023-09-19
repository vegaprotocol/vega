-- +goose Up
alter table transfers add column dispatch_strategy jsonb;
update transfers set dispatch_strategy = json_build_object('assetForMetric', dispatch_metric_asset, 'metric', dispatch_metric, 'markets', dispatch_markets,'entityScope',1, 'individualScope',1,'stakingRequirement','0','notionalTimeWeightedAveragePositionRequirement','0','windowLength',1,'lockPeriod',0,'distributionStrategy',1);
DROP VIEW IF EXISTS transfers_current;
alter table transfers drop column dispatch_metric_asset;
alter table transfers drop column dispatch_metric;
alter table transfers drop column dispatch_markets;
CREATE VIEW transfers_current AS ( SELECT DISTINCT ON (id, from_account_id, to_account_id) * FROM transfers ORDER BY id, from_account_id, to_account_id, vega_time DESC);
